package compat

import (
	"net/http"
	"strings"

	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/pkg/api/handlers"
	"github.com/containers/podman/v2/pkg/api/handlers/utils"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

func TopContainer(w http.ResponseWriter, r *http.Request) {
	runtime := r.Context().Value("runtime").(*libpod.Runtime)
	decoder := r.Context().Value("decoder").(*schema.Decoder)

	defaultValue := "-ef"
	if utils.IsLibpodRequest(r) {
		defaultValue = ""
	}
	query := struct {
		PsArgs string `schema:"ps_args"`
	}{
		PsArgs: defaultValue,
	}
	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
			errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}

	name := utils.GetName(r)
	c, err := runtime.LookupContainer(name)
	if err != nil {
		utils.ContainerNotFound(w, name, err)
		return
	}

	output, err := c.Top([]string{query.PsArgs})
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}

	var body = handlers.ContainerTopOKBody{}
	if len(output) > 0 {
		body.Titles = strings.Split(output[0], "\t")
		for _, line := range output[1:] {
			body.Processes = append(body.Processes, strings.Split(line, "\t"))
		}
	}
	utils.WriteJSON(w, http.StatusOK, body)
}
