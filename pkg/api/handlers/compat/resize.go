package compat

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/api/handlers/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/remotecommand"
)

func ResizeTTY(w http.ResponseWriter, r *http.Request) {
	runtime := r.Context().Value("runtime").(*libpod.Runtime)
	decoder := r.Context().Value("decoder").(*schema.Decoder)

	// /containers/{id}/resize
	query := struct {
		height uint16 `schema:"h"`
		width  uint16 `schema:"w"`
	}{
		// override any golang type defaults
	}

	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
			errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}

	sz := remotecommand.TerminalSize{
		Width:  query.width,
		Height: query.height,
	}

	var status int
	switch {
	case strings.Contains(r.URL.Path, "/containers/"):
		name := utils.GetName(r)
		ctnr, err := runtime.LookupContainer(name)
		if err != nil {
			utils.ContainerNotFound(w, name, err)
			return
		}
		if state, err := ctnr.State(); err != nil {
			utils.InternalServerError(w, errors.Wrapf(err, "cannot obtain container state"))
			return
		} else if state != define.ContainerStateRunning {
			utils.Error(w, "Container not running", http.StatusConflict,
				fmt.Errorf("container %q in wrong state %q", name, state.String()))
			return
		}
		if err := ctnr.AttachResize(sz); err != nil {
			utils.InternalServerError(w, errors.Wrapf(err, "cannot resize container"))
			return
		}
		// This is not a 204, even though we write nothing, for compatibility
		// reasons.
		status = http.StatusOK
	case strings.Contains(r.URL.Path, "/exec/"):
		name := mux.Vars(r)["id"]
		ctnr, err := runtime.GetExecSessionContainer(name)
		if err != nil {
			utils.SessionNotFound(w, name, err)
			return
		}
		if state, err := ctnr.State(); err != nil {
			utils.InternalServerError(w, errors.Wrapf(err, "cannot obtain session container state"))
			return
		} else if state != define.ContainerStateRunning {
			utils.Error(w, "Container not running", http.StatusConflict,
				fmt.Errorf("container %q in wrong state %q", name, state.String()))
			return
		}
		if err := ctnr.ExecResize(name, sz); err != nil {
			utils.InternalServerError(w, errors.Wrapf(err, "cannot resize session"))
			return
		}
		// This is not a 204, even though we write nothing, for compatibility
		// reasons.
		status = http.StatusCreated
	}
	utils.WriteResponse(w, status, "")
}
