package abi

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/docker"
	dockerarchive "github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/libpod/libpod/define"
	"github.com/containers/libpod/libpod/image"
	libpodImage "github.com/containers/libpod/libpod/image"
	"github.com/containers/libpod/pkg/domain/entities"
	domainUtils "github.com/containers/libpod/pkg/domain/utils"
	"github.com/containers/libpod/pkg/util"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (ir *ImageEngine) Exists(_ context.Context, nameOrId string) (*entities.BoolReport, error) {
	_, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil && errors.Cause(err) != define.ErrNoSuchImage {
		return nil, err
	}
	return &entities.BoolReport{Value: err == nil}, nil
}

func (ir *ImageEngine) Prune(ctx context.Context, opts entities.ImagePruneOptions) (*entities.ImagePruneReport, error) {
	results, err := ir.Libpod.ImageRuntime().PruneImages(ctx, opts.All, opts.Filter)
	if err != nil {
		return nil, err
	}

	report := entities.ImagePruneReport{
		Report: entities.Report{
			Id:  results,
			Err: nil,
		},
	}
	return &report, nil
}

func (ir *ImageEngine) History(ctx context.Context, nameOrId string, opts entities.ImageHistoryOptions) (*entities.ImageHistoryReport, error) {
	image, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil {
		return nil, err
	}
	results, err := image.History(ctx)
	if err != nil {
		return nil, err
	}

	history := entities.ImageHistoryReport{
		Layers: make([]entities.ImageHistoryLayer, len(results)),
	}

	for i, layer := range results {
		history.Layers[i] = ToDomainHistoryLayer(layer)
	}
	return &history, nil
}

func ToDomainHistoryLayer(layer *libpodImage.History) entities.ImageHistoryLayer {
	l := entities.ImageHistoryLayer{}
	l.ID = layer.ID
	l.Created = *layer.Created
	l.CreatedBy = layer.CreatedBy
	copy(l.Tags, layer.Tags)
	l.Size = layer.Size
	l.Comment = layer.Comment
	return l
}

func (ir *ImageEngine) Pull(ctx context.Context, rawImage string, options entities.ImagePullOptions) (*entities.ImagePullReport, error) {
	var writer io.Writer
	if !options.Quiet {
		writer = os.Stderr
	}

	dockerPrefix := fmt.Sprintf("%s://", docker.Transport.Name())
	imageRef, err := alltransports.ParseImageName(rawImage)
	if err != nil {
		imageRef, err = alltransports.ParseImageName(fmt.Sprintf("%s%s", dockerPrefix, rawImage))
		if err != nil {
			return nil, errors.Errorf("invalid image reference %q", rawImage)
		}
	}

	// Special-case for docker-archive which allows multiple tags.
	if imageRef.Transport().Name() == dockerarchive.Transport.Name() {
		newImage, err := ir.Libpod.ImageRuntime().LoadFromArchiveReference(ctx, imageRef, options.SignaturePolicy, writer)
		if err != nil {
			return nil, err
		}
		return &entities.ImagePullReport{Images: []string{newImage[0].ID()}}, nil
	}

	var registryCreds *types.DockerAuthConfig
	if options.Credentials != "" {
		creds, err := util.ParseRegistryCreds(options.Credentials)
		if err != nil {
			return nil, err
		}
		registryCreds = creds
	}
	dockerRegistryOptions := image.DockerRegistryOptions{
		DockerRegistryCreds:         registryCreds,
		DockerCertPath:              options.CertDir,
		OSChoice:                    options.OverrideOS,
		ArchitectureChoice:          options.OverrideArch,
		DockerInsecureSkipTLSVerify: options.SkipTLSVerify,
	}

	if !options.AllTags {
		newImage, err := ir.Libpod.ImageRuntime().New(ctx, rawImage, options.SignaturePolicy, options.Authfile, writer, &dockerRegistryOptions, image.SigningOptions{}, nil, util.PullImageAlways)
		if err != nil {
			return nil, err
		}
		return &entities.ImagePullReport{Images: []string{newImage.ID()}}, nil
	}

	// --all-tags requires the docker transport
	if imageRef.Transport().Name() != docker.Transport.Name() {
		return nil, errors.New("--all-tags requires docker transport")
	}

	// Trim the docker-transport prefix.
	rawImage = strings.TrimPrefix(rawImage, docker.Transport.Name())

	// all-tags doesn't work with a tagged reference, so let's check early
	namedRef, err := reference.Parse(rawImage)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing %q", rawImage)
	}
	if _, isTagged := namedRef.(reference.Tagged); isTagged {
		return nil, errors.New("--all-tags requires a reference without a tag")

	}

	systemContext := image.GetSystemContext("", options.Authfile, false)
	tags, err := docker.GetRepositoryTags(ctx, systemContext, imageRef)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting repository tags")
	}

	var foundIDs []string
	for _, tag := range tags {
		name := rawImage + ":" + tag
		newImage, err := ir.Libpod.ImageRuntime().New(ctx, name, options.SignaturePolicy, options.Authfile, writer, &dockerRegistryOptions, image.SigningOptions{}, nil, util.PullImageAlways)
		if err != nil {
			logrus.Errorf("error pulling image %q", name)
			continue
		}
		foundIDs = append(foundIDs, newImage.ID())
	}

	if len(tags) != len(foundIDs) {
		return nil, err
	}
	return &entities.ImagePullReport{Images: foundIDs}, nil
}

func (ir *ImageEngine) Inspect(ctx context.Context, namesOrIDs []string, opts entities.InspectOptions) ([]*entities.ImageInspectReport, error) {
	reports := []*entities.ImageInspectReport{}
	for _, i := range namesOrIDs {
		img, err := ir.Libpod.ImageRuntime().NewFromLocal(i)
		if err != nil {
			return nil, err
		}
		result, err := img.Inspect(ctx)
		if err != nil {
			return nil, err
		}
		report := entities.ImageInspectReport{}
		if err := domainUtils.DeepCopy(&report, result); err != nil {
			return nil, err
		}
		reports = append(reports, &report)
	}
	return reports, nil
}

func (ir *ImageEngine) Push(ctx context.Context, source string, destination string, options entities.ImagePushOptions) error {
	var writer io.Writer
	if !options.Quiet {
		writer = os.Stderr
	}

	var manifestType string
	switch options.Format {
	case "":
		// Default
	case "oci":
		manifestType = imgspecv1.MediaTypeImageManifest
	case "v2s1":
		manifestType = manifest.DockerV2Schema1SignedMediaType
	case "v2s2", "docker":
		manifestType = manifest.DockerV2Schema2MediaType
	default:
		return fmt.Errorf("unknown format %q. Choose on of the supported formats: 'oci', 'v2s1', or 'v2s2'", options.Format)
	}

	var registryCreds *types.DockerAuthConfig
	if options.Credentials != "" {
		creds, err := util.ParseRegistryCreds(options.Credentials)
		if err != nil {
			return err
		}
		registryCreds = creds
	}
	dockerRegistryOptions := image.DockerRegistryOptions{
		DockerRegistryCreds:         registryCreds,
		DockerCertPath:              options.CertDir,
		DockerInsecureSkipTLSVerify: options.TLSVerify,
	}

	signOptions := image.SigningOptions{
		RemoveSignatures: options.RemoveSignatures,
		SignBy:           options.SignBy,
	}

	newImage, err := ir.Libpod.ImageRuntime().NewFromLocal(source)
	if err != nil {
		return err
	}

	return newImage.PushImageToHeuristicDestination(
		ctx,
		destination,
		manifestType,
		options.Authfile,
		options.DigestFile,
		options.SignaturePolicy,
		writer,
		options.Compress,
		signOptions,
		&dockerRegistryOptions,
		nil)
}

// func (r *imageRuntime) Delete(ctx context.Context, nameOrId string, opts entities.ImageDeleteOptions) (*entities.ImageDeleteReport, error) {
// 	image, err := r.libpod.ImageEngine().NewFromLocal(nameOrId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	results, err := r.libpod.RemoveImage(ctx, image, opts.Force)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	report := entities.ImageDeleteReport{}
// 	if err := domainUtils.DeepCopy(&report, results); err != nil {
// 		return nil, err
// 	}
// 	return &report, nil
// }
//
// func (r *imageRuntime) Prune(ctx context.Context, opts entities.ImagePruneOptions) (*entities.ImagePruneReport, error) {
// 	// TODO: map FilterOptions
// 	id, err := r.libpod.ImageEngine().PruneImages(ctx, opts.All, []string{})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// TODO: Determine Size
// 	report := entities.ImagePruneReport{}
// 	copy(report.Report.Id, id)
// 	return &report, nil
// }

func (ir *ImageEngine) Tag(ctx context.Context, nameOrId string, tags []string, options entities.ImageTagOptions) error {
	newImage, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if err := newImage.TagImage(tag); err != nil {
			return err
		}
	}
	return nil
}

func (ir *ImageEngine) Untag(ctx context.Context, nameOrId string, tags []string, options entities.ImageUntagOptions) error {
	newImage, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil {
		return err
	}
	// If only one arg is provided, all names are to be untagged
	if len(tags) == 0 {
		tags = newImage.Names()
	}
	for _, tag := range tags {
		if err := newImage.UntagImage(tag); err != nil {
			return err
		}
	}
	return nil
}

func (ir *ImageEngine) Load(ctx context.Context, opts entities.ImageLoadOptions) (*entities.ImageLoadReport, error) {
	var (
		writer io.Writer
	)
	if !opts.Quiet {
		writer = os.Stderr
	}
	name, err := ir.Libpod.LoadImage(ctx, opts.Name, opts.Input, writer, opts.SignaturePolicy)
	if err != nil {
		return nil, err
	}
	names := strings.Split(name, ",")
	if len(names) <= 1 {
		newImage, err := ir.Libpod.ImageRuntime().NewFromLocal(name)
		if err != nil {
			return nil, errors.Wrap(err, "image loaded but no additional tags were created")
		}
		if len(opts.Name) > 0 {
			if err := newImage.TagImage(fmt.Sprintf("%s:%s", opts.Name, opts.Tag)); err != nil {
				return nil, errors.Wrapf(err, "error adding %q to image %q", opts.Name, newImage.InputName)
			}
		}
	}
	return &entities.ImageLoadReport{Names: names}, nil
}

func (ir *ImageEngine) Import(ctx context.Context, opts entities.ImageImportOptions) (*entities.ImageImportReport, error) {
	id, err := ir.Libpod.Import(ctx, opts.Source, opts.Reference, opts.Changes, opts.Message, opts.Quiet)
	if err != nil {
		return nil, err
	}
	return &entities.ImageImportReport{Id: id}, nil
}

func (ir *ImageEngine) Save(ctx context.Context, nameOrId string, tags []string, options entities.ImageSaveOptions) error {
	newImage, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil {
		return err
	}
	return newImage.Save(ctx, nameOrId, options.Format, options.Output, tags, options.Quiet, options.Compress)
}

func (ir *ImageEngine) Diff(_ context.Context, nameOrId string, _ entities.DiffOptions) (*entities.DiffReport, error) {
	changes, err := ir.Libpod.GetDiff("", nameOrId)
	if err != nil {
		return nil, err
	}
	return &entities.DiffReport{Changes: changes}, nil
}

func (ir *ImageEngine) Search(ctx context.Context, term string, opts entities.ImageSearchOptions) ([]entities.ImageSearchReport, error) {
	filter, err := image.ParseSearchFilter(opts.Filters)
	if err != nil {
		return nil, err
	}

	searchOpts := image.SearchOptions{
		Authfile:              opts.Authfile,
		Filter:                *filter,
		Limit:                 opts.Limit,
		NoTrunc:               opts.NoTrunc,
		InsecureSkipTLSVerify: opts.TLSVerify,
	}

	searchResults, err := image.SearchImages(term, searchOpts)
	if err != nil {
		return nil, err
	}

	// Convert from image.SearchResults to entities.ImageSearchReport. We don't
	// want to leak any low-level packages into the remote client, which
	// requires converting.
	reports := make([]entities.ImageSearchReport, len(searchResults))
	for i := range searchResults {
		reports[i].Index = searchResults[i].Index
		reports[i].Name = searchResults[i].Name
		reports[i].Description = searchResults[i].Index
		reports[i].Stars = searchResults[i].Stars
		reports[i].Official = searchResults[i].Official
		reports[i].Automated = searchResults[i].Automated
	}

	return reports, nil
}

// GetConfig returns a copy of the configuration used by the runtime
func (ir *ImageEngine) Config(_ context.Context) (*config.Config, error) {
	return ir.Libpod.GetConfig()
}

func (ir *ImageEngine) Build(ctx context.Context, containerFiles []string, opts entities.BuildOptions) (*entities.BuildReport, error) {
	id, _, err := ir.Libpod.Build(ctx, opts.BuildOptions, containerFiles...)
	if err != nil {
		return nil, err
	}
	return &entities.BuildReport{ID: id}, nil
}

func (ir *ImageEngine) Tree(ctx context.Context, nameOrId string, opts entities.ImageTreeOptions) (*entities.ImageTreeReport, error) {
	img, err := ir.Libpod.ImageRuntime().NewFromLocal(nameOrId)
	if err != nil {
		return nil, err
	}
	results, err := img.GenerateTree(opts.WhatRequires)
	if err != nil {
		return nil, err
	}
	return &entities.ImageTreeReport{Tree: results}, nil
}

// Remove removes one or more images from local storage.
func (ir *ImageEngine) Remove(ctx context.Context, images []string, opts entities.ImageRemoveOptions) (report *entities.ImageRemoveReport, finalError error) {
	var (
		// noSuchImageErrors indicates that at least one image was not found.
		noSuchImageErrors bool
		// inUseErrors indicates that at least one image is being used by a
		// container.
		inUseErrors bool
		// otherErrors indicates that at least one error other than the two
		// above occured.
		otherErrors bool
		// deleteError is a multierror to conveniently collect errors during
		// removal. We really want to delete as many images as possible and not
		// error out immediately.
		deleteError *multierror.Error
	)

	report = &entities.ImageRemoveReport{}

	// Set the removalCode and the error after all work is done.
	defer func() {
		switch {
		// 2
		case inUseErrors:
			// One of the specified images has child images or is
			// being used by a container.
			report.ExitCode = 2
		// 1
		case noSuchImageErrors && !(otherErrors || inUseErrors):
			// One of the specified images did not exist, and no other
			// failures.
			report.ExitCode = 1
		// 0
		default:
			// Nothing to do.
		}
		if deleteError != nil {
			// go-multierror has a trailing new line which we need to remove to normalize the string.
			finalError = deleteError.ErrorOrNil()
			finalError = errors.New(strings.TrimSpace(finalError.Error()))
		}
	}()

	// deleteImage is an anonymous function to conveniently delete an image
	// without having to pass all local data around.
	deleteImage := func(img *image.Image) error {
		results, err := ir.Libpod.RemoveImage(ctx, img, opts.Force)
		switch errors.Cause(err) {
		case nil:
			break
		case storage.ErrImageUsedByContainer:
			inUseErrors = true // Important for exit codes in Podman.
			return errors.New(
				fmt.Sprintf("A container associated with containers/storage, i.e. via Buildah, CRI-O, etc., may be associated with this image: %-12.12s\n", img.ID()))
		case define.ErrImageInUse:
			inUseErrors = true
			return err
		default:
			otherErrors = true // Important for exit codes in Podman.
			return err
		}

		report.Deleted = append(report.Deleted, results.Deleted)
		report.Untagged = append(report.Untagged, results.Untagged...)
		return nil
	}

	// Delete all images from the local storage.
	if opts.All {
		previousImages := 0
		// Remove all images one-by-one.
		for {
			storageImages, err := ir.Libpod.ImageRuntime().GetRWImages()
			if err != nil {
				deleteError = multierror.Append(deleteError,
					errors.Wrapf(err, "unable to query local images"))
				otherErrors = true // Important for exit codes in Podman.
				return
			}
			// No images (left) to remove, so we're done.
			if len(storageImages) == 0 {
				return
			}
			// Prevent infinity loops by making a delete-progress check.
			if previousImages == len(storageImages) {
				otherErrors = true // Important for exit codes in Podman.
				deleteError = multierror.Append(deleteError,
					errors.New("unable to delete all images, check errors and re-run image removal if needed"))
				break
			}
			previousImages = len(storageImages)
			// Delete all "leaves" (i.e., images without child images).
			for _, img := range storageImages {
				isParent, err := img.IsParent(ctx)
				if err != nil {
					otherErrors = true // Important for exit codes in Podman.
					deleteError = multierror.Append(deleteError, err)
				}
				// Skip parent images.
				if isParent {
					continue
				}
				if err := deleteImage(img); err != nil {
					deleteError = multierror.Append(deleteError, err)
				}
			}
		}

		return
	}

	// Delete only the specified images.
	for _, id := range images {
		img, err := ir.Libpod.ImageRuntime().NewFromLocal(id)
		switch errors.Cause(err) {
		case nil:
			break
		case image.ErrNoSuchImage:
			noSuchImageErrors = true // Important for exit codes in Podman.
			fallthrough
		default:
			deleteError = multierror.Append(deleteError, err)
			continue
		}

		err = deleteImage(img)
		if err != nil {
			otherErrors = true // Important for exit codes in Podman.
			deleteError = multierror.Append(deleteError, err)
		}
	}

	return
}

// Shutdown Libpod engine
func (ir *ImageEngine) Shutdown(_ context.Context) {
	shutdownSync.Do(func() {
		_ = ir.Libpod.Shutdown(false)
	})
}
