#!/usr/bin/env bats   -*- bats -*-
#
# tests for podman kill
#

load helpers

@test "podman kill - test signal handling in containers" {
    # podman-remote and crun interact poorly in f31: crun seems to gobble up
    # some signals.
    # Workaround: run 'env --default-signal sh' instead of just 'sh' in
    # the container. Since env on our regular alpine image doesn't support
    # that flag, we need to pull fedora-minimal. See:
    #    https://github.com/containers/podman/issues/5004
    # FIXME: remove this kludge once we get rid of podman-remote
    local _image=$IMAGE
    local _sh_cmd="sh"
    if is_remote; then
        _image=quay.io/libpod/fedora-minimal:latest
        _sh_cmd="env --default-signal sh"
    fi

    # Start a container that will handle all signals by emitting 'got: N'
    local -a signals=(1 2 3 4 5 6 8 10 12 13 14 15 16 20 21 22 23 24 25 26 64)
    run_podman run -d $_image $_sh_cmd -c \
        "for i in ${signals[*]}; do trap \"echo got: \$i\" \$i; done;
        echo READY;
        while ! test -e /stop; do sleep 0.05; done;
        echo DONE"
    # Ignore output regarding pulling/processing container images
    cid=$(echo "$output" | tail -1)

    # Run 'logs -f' on that container, but run it in the background with
    # redirection to a named pipe from which we (foreground job) read
    # and confirm that signals are received. We can't use run_podman here.
    local fifo=${PODMAN_TMPDIR}/podman-kill-fifo.$(random_string 10)
    mkfifo $fifo
    $PODMAN logs -f $cid >$fifo </dev/null &
    podman_log_pid=$!

    # Open the FIFO for reading, and keep it open. This prevents a race
    # condition in which the container can exit (e.g. if for some reason
    # it doesn't handle the signal) and we (this test) try to read from
    # the FIFO. Since there wouldn't be an active writer, the open()
    # would hang forever. With this exec we keep the FD open, allowing
    # 'read -t' to time out and report a useful error.
    exec 5<$fifo

    # First container emits READY when ready; wait for it.
    read -t 10 -u 5 ready
    is "$ready" "READY" "first log message from container"

    # Helper function: send the given signal, verify that it's received.
    kill_and_check() {
        local signal=$1
        local signum=${2:-$1}       # e.g. if signal=HUP, we expect to see '1'

        run_podman kill -s $signal $cid
        read -t 10 -u 5 actual || die "Timed out: no ACK for kill -s $signal"
        is "$actual" "got: $signum" "Signal $signal handled by container"
    }

    # Send signals in random order; make sure each one is received
    for s in $(fmt --width=2 <<< "${signals[*]}" | sort --random-sort);do
        kill_and_check $s
    done

    # Variations: with leading dash; by name, with/without dash or SIG
    kill_and_check -1        1
    kill_and_check -INT      2
    kill_and_check  FPE      8
    kill_and_check -SIGUSR1 10
    kill_and_check  SIGUSR2 12

    # Done. Tell the container to stop, and wait for final DONE
    run_podman exec $cid touch /stop
    read -t 5 -u 5 done || die "Timed out waiting for DONE from container"
    is "$done" "DONE" "final log message from container"

    # Clean up
    run_podman wait $cid
    run_podman rm $cid
    wait $podman_log_pid

    if [[ $_image != $IMAGE ]]; then
        run_podman rmi $_image
    fi
}

@test "podman kill - rejects invalid args" {
    # These errors are thrown by the imported docker/signal.ParseSignal()
    local -a bad_signal_names=(0 SIGBADSIG SIG BADSIG %% ! "''" '""' " ")
    for s in ${bad_signal_names[@]}; do
        # 'nosuchcontainer' is fine: podman should bail before it gets there
        run_podman 125 kill -s $s nosuchcontainer
        is "$output" "Error: invalid signal: $s" "Error from kill -s $s"

        run_podman 125 pod kill -s $s nosuchpod
        is "$output" "Error: invalid signal: $s" "Error from pod kill -s $s"
    done

    # Special case: these too are thrown by docker/signal.ParseSignal(),
    # but the dash sign is stripped by our wrapper in utils, so the
    # error message doesn't include the dash.
    local -a bad_dash_signals=(-0 -SIGBADSIG -SIG -BADSIG -)
    for s in ${bad_dash_signals[@]}; do
        run_podman 125 kill -s $s nosuchcontainer
        is "$output" "Error: invalid signal: ${s##-}" "Error from kill -s $s"
    done

    # This error (signal out of range) is thrown by our wrapper
    local -a bad_signal_nums=(65 -65 96 999 99999999)
    for s in ${bad_signal_nums[@]}; do
        run_podman 125 kill -s $s nosuchcontainer
        is "$output" "Error: valid signals are 1 through 64" \
           "Error from kill -s $s"
    done

    # 'podman create' uses the same parsing code
    run_podman 125 create --stop-signal=99 $IMAGE
    is "$output" "Error: valid signals are 1 through 64" "podman create"
}

# vim: filetype=sh
