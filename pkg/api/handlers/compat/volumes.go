package compat

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/api/handlers/utils"
	"github.com/containers/podman/v2/pkg/domain/filters"
	"github.com/containers/podman/v2/pkg/domain/infra/abi/parse"
	docker_api_types "github.com/docker/docker/api/types"
	docker_api_types_volume "github.com/docker/docker/api/types/volume"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

func ListVolumes(w http.ResponseWriter, r *http.Request) {
	var (
		decoder = r.Context().Value("decoder").(*schema.Decoder)
		runtime = r.Context().Value("runtime").(*libpod.Runtime)
	)
	query := struct {
		Filters map[string][]string `schema:"filters"`
	}{
		// override any golang type defaults
	}

	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
			errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}

	// Reject any libpod specific filters since `GenerateVolumeFilters()` will
	// happily parse them for us.
	for filter := range query.Filters {
		if filter == "opts" {
			utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
				errors.Errorf("unsupported libpod filters passed to docker endpoint"))
			return
		}
	}
	volumeFilters, err := filters.GenerateVolumeFilters(query.Filters)
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}

	vols, err := runtime.Volumes(volumeFilters...)
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}
	volumeConfigs := make([]*docker_api_types.Volume, 0, len(vols))
	for _, v := range vols {
		config := docker_api_types.Volume{
			Name:       v.Name(),
			Driver:     v.Driver(),
			Mountpoint: v.MountPoint(),
			CreatedAt:  v.CreatedTime().Format(time.RFC3339),
			Labels:     v.Labels(),
			Scope:      v.Scope(),
			Options:    v.Options(),
		}
		volumeConfigs = append(volumeConfigs, &config)
	}
	response := docker_api_types_volume.VolumeListOKBody{
		Volumes:  volumeConfigs,
		Warnings: []string{},
	}
	utils.WriteResponse(w, http.StatusOK, response)
}

func CreateVolume(w http.ResponseWriter, r *http.Request) {
	var (
		volumeOptions []libpod.VolumeCreateOption
		runtime       = r.Context().Value("runtime").(*libpod.Runtime)
		decoder       = r.Context().Value("decoder").(*schema.Decoder)
	)
	/* No query string data*/
	query := struct{}{}
	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
			errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}
	// decode params from body
	input := docker_api_types_volume.VolumeCreateBody{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, "Something went wrong.", http.StatusInternalServerError, errors.Wrap(err, "Decode()"))
		return
	}

	if len(input.Name) > 0 {
		volumeOptions = append(volumeOptions, libpod.WithVolumeName(input.Name))
	}
	if len(input.Driver) > 0 {
		volumeOptions = append(volumeOptions, libpod.WithVolumeDriver(input.Driver))
	}
	if len(input.Labels) > 0 {
		volumeOptions = append(volumeOptions, libpod.WithVolumeLabels(input.Labels))
	}
	if len(input.DriverOpts) > 0 {
		parsedOptions, err := parse.VolumeOptions(input.DriverOpts)
		if err != nil {
			utils.InternalServerError(w, err)
			return
		}
		volumeOptions = append(volumeOptions, parsedOptions...)
	}
	vol, err := runtime.NewVolume(r.Context(), volumeOptions...)
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}
	config, err := vol.Config()
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}
	volResponse := docker_api_types.Volume{
		Name:       config.Name,
		Driver:     config.Driver,
		Mountpoint: config.MountPoint,
		CreatedAt:  config.CreatedTime.Format(time.RFC3339),
		Labels:     config.Labels,
		Options:    config.Options,
		Scope:      "local",
		// ^^ We don't have volume scoping so we'll just claim it's "local"
		// like we do in the `libpod.Volume.Scope()` method
		//
		// TODO: We don't include the volume `Status` or `UsageData`, but both
		// are nullable in the Docker engine API spec so that's fine for now
	}
	utils.WriteResponse(w, http.StatusCreated, volResponse)
}

func InspectVolume(w http.ResponseWriter, r *http.Request) {
	var (
		runtime = r.Context().Value("runtime").(*libpod.Runtime)
	)
	name := utils.GetName(r)
	vol, err := runtime.GetVolume(name)
	if err != nil {
		utils.VolumeNotFound(w, name, err)
		return
	}
	volResponse := docker_api_types.Volume{
		Name:       vol.Name(),
		Driver:     vol.Driver(),
		Mountpoint: vol.MountPoint(),
		CreatedAt:  vol.CreatedTime().Format(time.RFC3339),
		Labels:     vol.Labels(),
		Options:    vol.Options(),
		Scope:      vol.Scope(),
		// TODO: As above, we don't return `Status` or `UsageData` yet
	}
	utils.WriteResponse(w, http.StatusOK, volResponse)
}

func RemoveVolume(w http.ResponseWriter, r *http.Request) {
	var (
		runtime = r.Context().Value("runtime").(*libpod.Runtime)
		decoder = r.Context().Value("decoder").(*schema.Decoder)
	)
	query := struct {
		Force bool `schema:"force"`
	}{
		// override any golang type defaults
	}

	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest,
			errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}

	/* The implications for `force` differ between Docker and us, so we can't
	 * simply pass the `force` parameter to `runeimt.RemoveVolume()`.
	 * Specifically, Docker's behavior seems to be that `force` means "do not
	 * error on missing volume"; ours means "remove any not-running containers
	 * using the volume at the same time".
	 *
	 * With this in mind, we only consider the `force` query parameter when we
	 * hunt for specified volume by name, using it to seletively return a 204
	 * or blow up depending on `force` being truthy or falsey/unset
	 * respectively.
	 */
	name := utils.GetName(r)
	vol, err := runtime.LookupVolume(name)
	if err == nil {
		// As above, we do not pass `force` from the query parameters here
		if err := runtime.RemoveVolume(r.Context(), vol, false); err != nil {
			if errors.Cause(err) == define.ErrVolumeBeingUsed {
				utils.Error(w, "volumes being used", http.StatusConflict, err)
			} else {
				utils.InternalServerError(w, err)
			}
		} else {
			// Success
			utils.WriteResponse(w, http.StatusNoContent, "")
		}
	} else {
		if !query.Force {
			utils.VolumeNotFound(w, name, err)
		} else {
			// Volume does not exist and `force` is truthy - this emulates what
			// Docker would do when told to `force` removal of a nonextant
			// volume
			utils.WriteResponse(w, http.StatusNoContent, "")
		}
	}
}

func PruneVolumes(w http.ResponseWriter, r *http.Request) {
	var (
		runtime = r.Context().Value("runtime").(*libpod.Runtime)
		decoder = r.Context().Value("decoder").(*schema.Decoder)
	)
	// For some reason the prune filters are query parameters even though this
	// is a POST endpoint
	query := struct {
		Filters map[string][]string `schema:"filters"`
	}{
		// override any golang type defaults
	}

	if err := decoder.Decode(&query, r.URL.Query()); err != nil {
		utils.Error(w, "Something went wrong.", http.StatusBadRequest, errors.Wrapf(err, "Failed to parse parameters for %s", r.URL.String()))
		return
	}
	// TODO: We have no ability to pass pruning filters to `PruneVolumes()` so
	// we'll explicitly reject the request if we see any
	if len(query.Filters) > 0 {
		utils.InternalServerError(w, errors.New("filters for pruning volumes is not implemented"))
		return
	}

	pruned, err := runtime.PruneVolumes(r.Context())
	if err != nil {
		utils.InternalServerError(w, err)
		return
	}
	prunedIds := make([]string, 0, len(pruned))
	for k := range pruned {
		// XXX: This drops any pruning per-volume error messages on the floor
		prunedIds = append(prunedIds, k)
	}
	pruneResponse := docker_api_types.VolumesPruneReport{
		VolumesDeleted: prunedIds,
		// TODO: We don't have any insight into how much space was reclaimed
		// from `PruneVolumes()` but it's not nullable
		SpaceReclaimed: 0,
	}

	utils.WriteResponse(w, http.StatusOK, pruneResponse)
}
