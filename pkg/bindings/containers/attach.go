package containers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"time"

	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/bindings"
	sig "github.com/containers/podman/v2/pkg/signal"
	"github.com/containers/podman/v2/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// Attach attaches to a running container
func Attach(ctx context.Context, nameOrID string, detachKeys *string, logs, stream *bool, stdin io.Reader, stdout io.Writer, stderr io.Writer, attachReady chan bool) error {
	isSet := struct {
		stdin  bool
		stdout bool
		stderr bool
	}{
		stdin:  !(stdin == nil || reflect.ValueOf(stdin).IsNil()),
		stdout: !(stdout == nil || reflect.ValueOf(stdout).IsNil()),
		stderr: !(stderr == nil || reflect.ValueOf(stderr).IsNil()),
	}
	// Ensure golang can determine that interfaces are "really" nil
	if !isSet.stdin {
		stdin = (io.Reader)(nil)
	}
	if !isSet.stdout {
		stdout = (io.Writer)(nil)
	}
	if !isSet.stderr {
		stderr = (io.Writer)(nil)
	}

	conn, err := bindings.GetClient(ctx)
	if err != nil {
		return err
	}

	// Do we need to wire in stdin?
	ctnr, err := Inspect(ctx, nameOrID, bindings.PFalse)
	if err != nil {
		return err
	}

	params := url.Values{}
	if detachKeys != nil {
		params.Add("detachKeys", *detachKeys)
	}
	if logs != nil {
		params.Add("logs", fmt.Sprintf("%t", *logs))
	}
	if stream != nil {
		params.Add("stream", fmt.Sprintf("%t", *stream))
	}
	if isSet.stdin {
		params.Add("stdin", "true")
	}
	if isSet.stdout {
		params.Add("stdout", "true")
	}
	if isSet.stderr {
		params.Add("stderr", "true")
	}

	// Unless all requirements are met, don't use "stdin" is a terminal
	file, ok := stdin.(*os.File)
	needTTY := ok && terminal.IsTerminal(int(file.Fd())) && ctnr.Config.Tty
	if needTTY {
		state, err := setRawTerminal(file)
		if err != nil {
			return err
		}
		defer func() {
			if err := terminal.Restore(int(file.Fd()), state); err != nil {
				logrus.Errorf("unable to restore terminal: %q", err)
			}
			logrus.SetFormatter(&logrus.TextFormatter{})
		}()
	}

	headers := make(map[string]string)
	headers["Connection"] = "Upgrade"
	headers["Upgrade"] = "tcp"

	var socket net.Conn
	socketSet := false
	dialContext := conn.Client.Transport.(*http.Transport).DialContext
	t := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			c, err := dialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			if !socketSet {
				socket = c
				socketSet = true
			}
			return c, err
		},
		IdleConnTimeout: time.Duration(0),
	}
	conn.Client.Transport = t
	response, err := conn.DoRequest(nil, http.MethodPost, "/containers/%s/attach", params, headers, nameOrID)
	if err != nil {
		return err
	}
	if !(response.IsSuccess() || response.IsInformational()) {
		return response.Process(nil)
	}

	if needTTY {
		winChange := make(chan os.Signal, 1)
		signal.Notify(winChange, sig.SIGWINCH)
		winCtx, winCancel := context.WithCancel(ctx)
		defer winCancel()

		go attachHandleResize(ctx, winCtx, winChange, false, nameOrID, file)
	}

	// If we are attaching around a start, we need to "signal"
	// back that we are in fact attached so that started does
	// not execute before we can attach.
	if attachReady != nil {
		attachReady <- true
	}

	if isSet.stdin {
		go func() {
			logrus.Debugf("Copying STDIN to socket")
			_, err := utils.CopyDetachable(socket, stdin, []byte{})
			if err != nil {
				logrus.Error("failed to write input to service: " + err.Error())
			}
		}()
	}

	buffer := make([]byte, 1024)
	if ctnr.Config.Tty {
		logrus.Debugf("Copying STDOUT of container in terminal mode")

		if !isSet.stdout {
			return fmt.Errorf("container %q requires stdout to be set", ctnr.ID)
		}
		// If not multiplex'ed, read from server and write to stdout
		_, err := io.Copy(stdout, socket)
		if err != nil {
			return err
		}
	} else {
		logrus.Debugf("Copying standard streams of container in non-terminal mode")
		for {
			// Read multiplexed channels and write to appropriate stream
			fd, l, err := DemuxHeader(socket, buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			frame, err := DemuxFrame(socket, buffer, l)
			if err != nil {
				return err
			}

			switch {
			case fd == 0:
				if isSet.stdout {
					if _, err := stdout.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 1:
				if isSet.stdout {
					if _, err := stdout.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 2:
				if isSet.stderr {
					if _, err := stderr.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 3:
				return fmt.Errorf("error from service from stream: %s", frame)
			default:
				return fmt.Errorf("unrecognized channel '%d' in header, 0-3 supported", fd)
			}
		}
	}
	return nil
}

// DemuxHeader reads header for stream from server multiplexed stdin/stdout/stderr/2nd error channel
func DemuxHeader(r io.Reader, buffer []byte) (fd, sz int, err error) {
	n, err := io.ReadFull(r, buffer[0:8])
	if err != nil {
		return
	}
	if n < 8 {
		err = io.ErrUnexpectedEOF
		return
	}

	fd = int(buffer[0])
	if fd < 0 || fd > 3 {
		err = errors.Wrapf(ErrLostSync, fmt.Sprintf(`channel "%d" found, 0-3 supported`, fd))
		return
	}

	sz = int(binary.BigEndian.Uint32(buffer[4:8]))
	return
}

// DemuxFrame reads contents for frame from server multiplexed stdin/stdout/stderr/2nd error channel
func DemuxFrame(r io.Reader, buffer []byte, length int) (frame []byte, err error) {
	if len(buffer) < length {
		buffer = append(buffer, make([]byte, length-len(buffer)+1)...)
	}

	n, err := io.ReadFull(r, buffer[0:length])
	if err != nil {
		return nil, nil
	}
	if n < length {
		err = io.ErrUnexpectedEOF
		return
	}

	return buffer[0:length], nil
}

// ResizeContainerTTY sets container's TTY height and width in characters
func ResizeContainerTTY(ctx context.Context, nameOrID string, height *int, width *int) error {
	return resizeTTY(ctx, bindings.JoinURL("containers", nameOrID, "resize"), height, width)
}

// ResizeExecTTY sets session's TTY height and width in characters
func ResizeExecTTY(ctx context.Context, nameOrID string, height *int, width *int) error {
	return resizeTTY(ctx, bindings.JoinURL("exec", nameOrID, "resize"), height, width)
}

// resizeTTY set size of TTY of container
func resizeTTY(ctx context.Context, endpoint string, height *int, width *int) error {
	conn, err := bindings.GetClient(ctx)
	if err != nil {
		return err
	}

	params := url.Values{}
	if height != nil {
		params.Set("h", strconv.Itoa(*height))
	}
	if width != nil {
		params.Set("w", strconv.Itoa(*width))
	}
	rsp, err := conn.DoRequest(nil, http.MethodPost, endpoint, params, nil)
	if err != nil {
		return err
	}
	return rsp.Process(nil)
}

type rawFormatter struct {
	logrus.TextFormatter
}

func (f *rawFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	buffer, err := f.TextFormatter.Format(entry)
	if err != nil {
		return buffer, err
	}
	return append(buffer, '\r'), nil
}

// This is intended to be run as a goroutine, handling resizing for a container
// or exec session.
func attachHandleResize(ctx, winCtx context.Context, winChange chan os.Signal, isExec bool, id string, file *os.File) {
	// Prime the pump, we need one reset to ensure everything is ready
	winChange <- sig.SIGWINCH
	for {
		select {
		case <-winCtx.Done():
			return
		case <-winChange:
			h, w, err := terminal.GetSize(int(file.Fd()))
			if err != nil {
				logrus.Warnf("failed to obtain TTY size: " + err.Error())
			}

			var resizeErr error
			if isExec {
				resizeErr = ResizeExecTTY(ctx, id, &h, &w)
			} else {
				resizeErr = ResizeContainerTTY(ctx, id, &h, &w)
			}
			if resizeErr != nil {
				logrus.Warnf("failed to resize TTY: " + resizeErr.Error())
			}
		}
	}
}

// Configure the given terminal for raw mode
func setRawTerminal(file *os.File) (*terminal.State, error) {
	state, err := terminal.MakeRaw(int(file.Fd()))
	if err != nil {
		return nil, err
	}

	logrus.SetFormatter(&rawFormatter{})

	return state, err
}

// ExecStartAndAttach starts and attaches to a given exec session.
func ExecStartAndAttach(ctx context.Context, sessionID string, streams *define.AttachStreams) error {
	conn, err := bindings.GetClient(ctx)
	if err != nil {
		return err
	}

	// TODO: Make this configurable (can't use streams' InputStream as it's
	// buffered)
	terminalFile := os.Stdin

	logrus.Debugf("Starting & Attaching to exec session ID %q", sessionID)

	// We need to inspect the exec session first to determine whether to use
	// -t.
	resp, err := conn.DoRequest(nil, http.MethodGet, "/exec/%s/json", nil, nil, sessionID)
	if err != nil {
		return err
	}

	respStruct := new(define.InspectExecSession)
	if err := resp.Process(respStruct); err != nil {
		return err
	}
	isTerm := true
	if respStruct.ProcessConfig != nil {
		isTerm = respStruct.ProcessConfig.Tty
	}

	// If we are in TTY mode, we need to set raw mode for the terminal.
	// TODO: Share all of this with Attach() for containers.
	needTTY := terminalFile != nil && terminal.IsTerminal(int(terminalFile.Fd())) && isTerm
	if needTTY {
		state, err := setRawTerminal(terminalFile)
		if err != nil {
			return err
		}
		defer func() {
			if err := terminal.Restore(int(terminalFile.Fd()), state); err != nil {
				logrus.Errorf("unable to restore terminal: %q", err)
			}
			logrus.SetFormatter(&logrus.TextFormatter{})
		}()
	}

	body := struct {
		Detach bool `json:"Detach"`
	}{
		Detach: false,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var socket net.Conn
	socketSet := false
	dialContext := conn.Client.Transport.(*http.Transport).DialContext
	t := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			c, err := dialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			if !socketSet {
				socket = c
				socketSet = true
			}
			return c, err
		},
		IdleConnTimeout: time.Duration(0),
	}
	conn.Client.Transport = t
	response, err := conn.DoRequest(bytes.NewReader(bodyJSON), http.MethodPost, "/exec/%s/start", nil, nil, sessionID)
	if err != nil {
		return err
	}
	if !(response.IsSuccess() || response.IsInformational()) {
		return response.Process(nil)
	}

	if needTTY {
		winChange := make(chan os.Signal, 1)
		signal.Notify(winChange, sig.SIGWINCH)
		winCtx, winCancel := context.WithCancel(ctx)
		defer winCancel()

		go attachHandleResize(ctx, winCtx, winChange, true, sessionID, terminalFile)
	}

	if streams.AttachInput {
		go func() {
			logrus.Debugf("Copying STDIN to socket")
			_, err := utils.CopyDetachable(socket, streams.InputStream, []byte{})
			if err != nil {
				logrus.Error("failed to write input to service: " + err.Error())
			}
		}()
	}

	buffer := make([]byte, 1024)
	if isTerm {
		logrus.Debugf("Handling terminal attach to exec")
		if !streams.AttachOutput {
			return fmt.Errorf("exec session %s has a terminal and must have STDOUT enabled", sessionID)
		}
		// If not multiplex'ed, read from server and write to stdout
		_, err := utils.CopyDetachable(streams.OutputStream, socket, []byte{})
		if err != nil {
			return err
		}
	} else {
		logrus.Debugf("Handling non-terminal attach to exec")
		for {
			// Read multiplexed channels and write to appropriate stream
			fd, l, err := DemuxHeader(socket, buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			frame, err := DemuxFrame(socket, buffer, l)
			if err != nil {
				return err
			}

			switch {
			case fd == 0:
				if streams.AttachOutput {
					if _, err := streams.OutputStream.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 1:
				if streams.AttachInput {
					// Write STDIN to STDOUT (echoing characters
					// typed by another attach session)
					if _, err := streams.OutputStream.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 2:
				if streams.AttachError {
					if _, err := streams.ErrorStream.Write(frame[0:l]); err != nil {
						return err
					}
				}
			case fd == 3:
				return fmt.Errorf("error from service from stream: %s", frame)
			default:
				return fmt.Errorf("unrecognized channel '%d' in header, 0-3 supported", fd)
			}
		}
	}
	return nil
}
