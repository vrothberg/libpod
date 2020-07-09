package server

import (
	"net/http"

	"github.com/containers/podman/v2/pkg/api/handlers/libpod"
	"github.com/gorilla/mux"
)

func (s *APIServer) registerPlayHandlers(r *mux.Router) error {
	// swagger:operation POST /libpod/play/kube libpod libpodPlayKube
	// ---
	// tags:
	//  - containers
	//  - pods
	// summary: Play a Kubernetes YAML file.
	// description: Create and run pods based on a Kubernetes YAML file (pod or service kind).
	// parameters:
	//  - in: query
	//    name: network
	//    type: string
	//    description: Connect the pod to this network.
	//  - in: query
	//    name: tlsVerify
	//    type: boolean
	//    default: true
	//    description: Require HTTPS and verify signatures when contating registries.
	//  - in: body
	//    name: request
	//    description: Kubernetes YAML file.
	//    schema:
	//      type: string
	// produces:
	// - application/json
	// responses:
	//   200:
	//     $ref: "#/responses/DocsLibpodPlayKubeResponse"
	//   500:
	//     $ref: "#/responses/InternalError"
	r.HandleFunc(VersionedPath("/libpod/play/kube"), s.APIHandler(libpod.PlayKube)).Methods(http.MethodPost)
	return nil
}
