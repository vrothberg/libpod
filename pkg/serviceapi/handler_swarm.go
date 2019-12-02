package serviceapi

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (s *APIServer) registerSwarmHandlers(r *mux.Router) error {
	r.PathPrefix("/v{version:[0-9.]+}/swarm/").HandlerFunc(noSwarm)
	r.PathPrefix("/v{version:[0-9.]+}/services/").HandlerFunc(noSwarm)
	r.PathPrefix("/v{version:[0-9.]+}/nodes/").HandlerFunc(noSwarm)
	r.PathPrefix("/v{version:[0-9.]+}/tasks/").HandlerFunc(noSwarm)
	r.PathPrefix("/v{version:[0-9.]+}/secrets/").HandlerFunc(noSwarm)
	r.PathPrefix("/v{version:[0-9.]+}/configs/").HandlerFunc(noSwarm)
	return nil
}

// noSwarm returns http.StatusServiceUnavailable rather than something like http.StatusInternalServerError,
// this allows the client to decide if they still can talk to us
func noSwarm(w http.ResponseWriter, r *http.Request) {
	logrus.Errorf("%s is not a podman supported service", r.URL.String())
	Error(w, "node is not part of a swarm", http.StatusServiceUnavailable, errors.New("Podman does not support service: "+r.URL.String()))
}
