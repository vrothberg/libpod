package server

import (
	"net/http"

	"github.com/containers/podman/v2/pkg/api/handlers/compat"
	"github.com/gorilla/mux"
)

func (s *APIServer) registerExecHandlers(r *mux.Router) error {
	// swagger:operation POST /containers/{name}/exec compat createExec
	// ---
	// tags:
	//   - exec (compat)
	// summary: Create an exec instance
	// description: Create an exec session to run a command inside a running container. Exec sessions will be automatically removed 5 minutes after they exit.
	// parameters:
	//  - in: path
	//    name: name
	//    type: string
	//    required: true
	//    description: name of container
	//  - in: body
	//    name: control
	//    description: Attributes for create
	//    schema:
	//      type: object
	//      properties:
	//        AttachStdin:
	//          type: boolean
	//          description: Attach to stdin of the exec command
	//        AttachStdout:
	//          type: boolean
	//          description: Attach to stdout of the exec command
	//        AttachStderr:
	//          type: boolean
	//          description: Attach to stderr of the exec command
	//        DetachKeys:
	//          type: string
	//          description: |
	//           "Override the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-<value> where <value> is one of: a-z, @, ^, [, , or _."
	//        Tty:
	//          type: boolean
	//          description: Allocate a pseudo-TTY
	//        Env:
	//          type: array
	//          description: A list of environment variables in the form ["VAR=value", ...]
	//          items:
	//            type: string
	//        Cmd:
	//          type: array
	//          description: Command to run, as a string or array of strings.
	//          items:
	//            type: string
	//        Privileged:
	//          type: boolean
	//          default: false
	//          description: Runs the exec process with extended privileges
	//        User:
	//          type: string
	//          description: |
	//           "The user, and optionally, group to run the exec process inside the container. Format is one of: user, user:group, uid, or uid:gid."
	//        WorkingDir:
	//          type: string
	//          description: The working directory for the exec process inside the container.
	// produces:
	// - application/json
	// responses:
	//   201:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchContainer"
	//   409:
	//	   description: container is paused
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/containers/{name}/exec"), s.APIHandler(compat.ExecCreateHandler)).Methods(http.MethodPost)
	// Added non version path to URI to support docker non versioned paths
	r.Handle("/containers/{name}/exec", s.APIHandler(compat.ExecCreateHandler)).Methods(http.MethodPost)
	// swagger:operation POST /exec/{id}/start compat startExec
	// ---
	// tags:
	//   - exec (compat)
	// summary: Start an exec instance
	// description: Starts a previously set up exec instance. If detach is true, this endpoint returns immediately after starting the command. Otherwise, it sets up an interactive session with the command.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	//  - in: body
	//    name: control
	//    description: Attributes for start
	//    schema:
	//      type: object
	//      properties:
	//        Detach:
	//          type: boolean
	//          description: Detach from the command. Not presently supported.
	//        Tty:
	//          type: boolean
	//          description: Allocate a pseudo-TTY. Presently ignored.
	// produces:
	// - application/json
	// responses:
	//   200:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   409:
	//	   description: container is not running
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/exec/{id}/start"), s.APIHandler(compat.ExecStartHandler)).Methods(http.MethodPost)
	// Added non version path to URI to support docker non versioned paths
	r.Handle("/exec/{id}/start", s.APIHandler(compat.ExecStartHandler)).Methods(http.MethodPost)
	// swagger:operation POST /exec/{id}/resize compat resizeExec
	// ---
	// tags:
	//   - exec (compat)
	// summary: Resize an exec instance
	// description: |
	//  Resize the TTY session used by an exec instance. This endpoint only works if tty was specified as part of creating and starting the exec instance.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	//  - in: query
	//    name: h
	//    type: integer
	//    description: Height of the TTY session in characters
	//  - in: query
	//    name: w
	//    type: integer
	//    description: Width of the TTY session in characters
	// produces:
	// - application/json
	// responses:
	//   201:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/exec/{id}/resize"), s.APIHandler(compat.ResizeTTY)).Methods(http.MethodPost)
	// Added non version path to URI to support docker non versioned paths
	r.Handle("/exec/{id}/resize", s.APIHandler(compat.ResizeTTY)).Methods(http.MethodPost)
	// swagger:operation GET /exec/{id}/json compat inspectExec
	// ---
	// tags:
	//   - exec (compat)
	// summary: Inspect an exec instance
	// description: Return low-level information about an exec instance.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	// produces:
	// - application/json
	// responses:
	//   200:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/exec/{id}/json"), s.APIHandler(compat.ExecInspectHandler)).Methods(http.MethodGet)
	// Added non version path to URI to support docker non versioned paths
	r.Handle("/exec/{id}/json", s.APIHandler(compat.ExecInspectHandler)).Methods(http.MethodGet)

	/*
		libpod api follows
	*/

	// swagger:operation POST /libpod/containers/{name}/exec libpod libpodCreateExec
	// ---
	// tags:
	//   - exec
	// summary: Create an exec instance
	// description: Create an exec session to run a command inside a running container. Exec sessions will be automatically removed 5 minutes after they exit.
	// parameters:
	//  - in: path
	//    name: name
	//    type: string
	//    required: true
	//    description: name of container
	//  - in: body
	//    name: control
	//    description: Attributes for create
	//    schema:
	//      type: object
	//      properties:
	//        AttachStdin:
	//          type: boolean
	//          description: Attach to stdin of the exec command
	//        AttachStdout:
	//          type: boolean
	//          description: Attach to stdout of the exec command
	//        AttachStderr:
	//          type: boolean
	//          description: Attach to stderr of the exec command
	//        DetachKeys:
	//          type: string
	//          description: |
	//           "Override the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-<value> where <value> is one of: a-z, @, ^, [, , or _."
	//        Tty:
	//          type: boolean
	//          description: Allocate a pseudo-TTY
	//        Env:
	//          type: array
	//          description: A list of environment variables in the form ["VAR=value", ...]
	//          items:
	//            type: string
	//        Cmd:
	//          type: array
	//          description: Command to run, as a string or array of strings.
	//          items:
	//            type: string
	//        Privileged:
	//          type: boolean
	//          default: false
	//          description: Runs the exec process with extended privileges
	//        User:
	//          type: string
	//          description: |
	//           "The user, and optionally, group to run the exec process inside the container. Format is one of: user, user:group, uid, or uid:gid."
	//        WorkingDir:
	//          type: string
	//          description: The working directory for the exec process inside the container.
	// produces:
	// - application/json
	// responses:
	//   201:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchContainer"
	//   409:
	//	   description: container is paused
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/libpod/containers/{name}/exec"), s.APIHandler(compat.ExecCreateHandler)).Methods(http.MethodPost)
	// swagger:operation POST /libpod/exec/{id}/start libpod libpodStartExec
	// ---
	// tags:
	//   - exec
	// summary: Start an exec instance
	// description: Starts a previously set up exec instance. If detach is true, this endpoint returns immediately after starting the command. Otherwise, it sets up an interactive session with the command.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	//  - in: body
	//    name: control
	//    description: Attributes for start
	//    schema:
	//      type: object
	//      properties:
	//        Detach:
	//          type: boolean
	//          description: Detach from the command. Not presently supported.
	//        Tty:
	//          type: boolean
	//          description: Allocate a pseudo-TTY. Presently ignored.
	// produces:
	// - application/json
	// responses:
	//   200:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   409:
	//	   description: container is not running.
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/libpod/exec/{id}/start"), s.APIHandler(compat.ExecStartHandler)).Methods(http.MethodPost)
	// swagger:operation POST /libpod/exec/{id}/resize libpod libpodResizeExec
	// ---
	// tags:
	//   - exec
	// summary: Resize an exec instance
	// description: |
	//  Resize the TTY session used by an exec instance. This endpoint only works if tty was specified as part of creating and starting the exec instance.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	//  - in: query
	//    name: h
	//    type: integer
	//    description: Height of the TTY session in characters
	//  - in: query
	//    name: w
	//    type: integer
	//    description: Width of the TTY session in characters
	// produces:
	// - application/json
	// responses:
	//   201:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/libpod/exec/{id}/resize"), s.APIHandler(compat.ResizeTTY)).Methods(http.MethodPost)
	// swagger:operation GET /libpod/exec/{id}/json libpod libpodInspectExec
	// ---
	// tags:
	//   - exec
	// summary: Inspect an exec instance
	// description: Return low-level information about an exec instance.
	// parameters:
	//  - in: path
	//    name: id
	//    type: string
	//    required: true
	//    description: Exec instance ID
	// produces:
	// - application/json
	// responses:
	//   200:
	//     description: no error
	//   404:
	//     $ref: "#/responses/NoSuchExecInstance"
	//   500:
	//     $ref: "#/responses/InternalError"
	r.Handle(VersionedPath("/libpod/exec/{id}/json"), s.APIHandler(compat.ExecInspectHandler)).Methods(http.MethodGet)
	return nil
}
