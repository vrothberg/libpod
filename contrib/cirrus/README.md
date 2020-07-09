![PODMAN logo](../../logo/podman-logo-source.svg)

# Cirrus-CI

Similar to other integrated github CI/CD services, Cirrus utilizes a simple
YAML-based configuration/description file: ``.cirrus.yml``.  Ref: https://cirrus-ci.org/


## Workflow

All tasks execute in parallel, unless there are conditions or dependencies
which alter this behavior.  Within each task, each script executes in sequence,
so long as any previous script exited successfully.  The overall state of each
task (pass or fail) is set based on the exit status of the last script to execute.

### ``gating`` Task

***N/B: Steps below are performed by automation***

1. Launch a purpose-built container in Cirrus's community cluster.
   For container image details, please see
   [the contributors guide](https://github.com/containers/podman/blob/master/CONTRIBUTING.md#go-format-and-lint).

3. ``validate``: Perform standard `make validate` source verification,
   Should run for less than a minute or two.

4. ``lint``: Execute regular `make lint` to check for any code cruft.
   Should also run for less than a few minutes.

5. ``vendor``: runs `make vendor-in-container` followed by `./hack/tree_status.sh` to check
   whether the git tree is clean. The reasoning for that is to make sure that
   the vendor.conf, the code and the vendored packages in ./vendor are in sync
   at all times.

### ``meta`` Task

***N/B: Steps below are performed by automation***

1. Launch a container built from definition in ``./contrib/imgts``.

2. Update VM Image metadata to help track usage across all automation.

4. Always exits successfully unless there's a major problem.


### ``testing`` Task

***N/B: Steps below are performed by automation***

1. After `gating` passes, spin up one VM per
   `matrix: image_name` item. Once accessible, ``ssh``
   into each VM as the `root` user.

2. ``setup_environment.sh``: Configure root's `.bash_profile`
    for all subsequent scripts (each run in a new shell).  Any
    distribution-specific environment variables are also defined
    here.  For example, setting tags/flags to use compiling.

5. ``integration_test.sh``: Execute integration-testing.  This is
   much more involved, and relies on access to external
   resources like container images and code from other repositories.
   Total execution time is capped at 2-hours (includes all the above)
   but this script normally completes in less than an hour.


### ``special_testing_cross`` Task

Confirm that cross-compile of podman-remote functions for both `windows`
and `darwin` targets.


### ``special_testing_cgroupv2`` Task

Use the latest Fedora release with the required kernel options pre-set for
exercising cgroups v2 with Podman integration tests.  Also depends on
having `SPECIALMODE` set to 'cgroupv2`


### ``test_build_cache_images_task`` Task

Modifying the contents of cache-images is tested by making changes to
one or more of the ``./contrib/cirrus/packer/*_setup.sh`` files.  Then
in the PR description, add the magic string:  ``[CI:IMG]``

***N/B: Steps below are performed by automation***

1. ``setup_environment.sh``: Same as for other tasks.

2. ``build_vm_images.sh``: Utilize [the packer tool](http://packer.io/docs/)
   to produce new VM images.  Create a new VM from each base-image, connect
   to them with ``ssh``, and perform the steps as defined by the
   ``$PACKER_BASE/libpod_images.yml`` file:

    1. On a base-image VM, as root, copy the current state of the repository
       into ``/tmp/libpod``.
    2. Execute distribution-specific scripts to prepare the image for
       use.  For example, ``fedora_setup.sh``.
    3. If successful, shut down each VM and record the names, and dates
       into a json manifest file.
    4. Move the manifest file, into a google storage bucket object.
       This is a retained as a secondary method for tracking/auditing
       creation of VM images, should it ever be needed.

### ``verify_test_built_images`` Task

Only runs following successful ``test_build_cache_images_task`` task.  Uses
images following the standard naming format; ***however, only runs a limited
sub-set of automated tests***.  Validating newly built images fully, requires
updating ``.cirrus.yml``.

***N/B: Steps below are performed by automation***

1. Using the just build VM images, launch VMs and wait for them to boot.

2. Execute the `setup_environment.sh` as in the `testing` task.

2. Execute the `integration_test.sh` as in the `testing` task.


***Manual Steps:***  Assuming the automated steps pass, then
you'll find the new image names displayed at the end of the
`test_build_cache_images`.  For example:


```
...cut...

[+0747s] ==> Builds finished. The artifacts of successful builds are:
[+0747s] --> ubuntu-18: A disk image was created: ubuntu-18-libpod-5664838702858240
[+0747s] --> fedora-29: A disk image was created: fedora-29-libpod-5664838702858240
[+0747s] --> fedora-30: A disk image was created: fedora-30-libpod-5664838702858240
[+0747s] --> ubuntu-19: A disk image was created: ubuntu-19-libpod-5664838702858240
```

Notice the suffix on all the image names comes from the env. var. set in
*.cirrus.yml*: `BUILT_IMAGE_SUFFIX: "-${CIRRUS_REPO_NAME}-${CIRRUS_BUILD_ID}"`.
Edit `.cirrus.yml`, in the top-level `env` section, update the suffix variable
used at runtime to launch VMs for testing:


```yaml
env:
    ...cut...
    ####
    #### Cache-image names to test with (double-quotes around names are critical)
    ###
    _BUILT_IMAGE_SUFFIX: "libpod-5664838702858240"
    FEDORA_CACHE_IMAGE_NAME: "fedora-30-${_BUILT_IMAGE_SUFFIX}"
    PRIOR_FEDORA_CACHE_IMAGE_NAME: "fedora-29-${_BUILT_IMAGE_SUFFIX}"
    ...cut...
```

***NOTES:***
* If re-using the same PR with new images in `.cirrus.yml`,
  take care to also *update the PR description* to remove
  the magic ``[CI:IMG]`` string.  Keeping it and
  `--force` pushing would needlessly cause Cirrus-CI to build
  and test images again.
* In the future, if you need to review the log from the build that produced
  the referenced image:

  * Note the Build ID from the image name (for example `5664838702858240`).
  * Go to that build in the Cirrus-CI WebUI, using the build ID in the URL.
    (For example `https://cirrus-ci.com/build/5664838702858240`.
  * Choose the *test_build_cache_images* task.
  * Open the *build_vm_images* script section.

### `docs` Task

Builds swagger API documentation YAML and uploads to google storage (an online
service for storing unstructured data) for both
PR's (for testing the process) and the master branch.  For PR's
the YAML is uploaded into a [dedicated short-pruning cycle
bucket.](https://storage.googleapis.com/libpod-pr-releases/) for testing purposes
only.  For the master branch, a [separate bucket is
used](https://storage.googleapis.com/libpod-master-releases) and provides the
content rendered on [the API Reference page](https://docs.podman.io/en/latest/_static/api.html)

The online API reference is presented by javascript to the client.  To prevent hijacking
of the client by malicious data, the [javascript utilises CORS](https://cloud.google.com/storage/docs/cross-origin).
This CORS metadata is served by `https://storage.googleapis.com` when configured correctly.
It will appear in [the request and response headers from the
client](https://cloud.google.com/storage/docs/configuring-cors#troubleshooting) when accessing
the API reference page.

However, when the CORS metadata is missing or incorrectly configured, clients will receive an
error-message similar to:

![Javascript Stack Trace Image](swagger_stack_trace.png)

For documentation built by Read The Docs from the master branch, CORS metadata is
set on the `libpod-master-releases` storage bucket.  Viewing or setting the CORS
metadata on the bucket requires having locally [installed and
configured the google-cloud SDK](https://cloud.google.com/sdk/docs).  It also requires having
admin access to the google-storage bucket.  Contact a project owner for help if you are
unsure of your permissions or need help resolving an error similar to the picture above.

Assuming the SDK is installed, and you have the required admin access, the following command
will display the current CORS metadata:

```
gsutil cors get gs://libpod-master-releases
```

To function properly (allow client "trust" of content from `storage.googleapis.com`) the followiing
metadata JSON should be used.  Following the JSON, is an example of the command used to set this
metadata on the libpod-master-releases bucket.  For additional information about configuring CORS
please referr to [the google-storage documentation](https://cloud.google.com/storage/docs/configuring-cors).

```JSON
[
    {
      "origin": ["http://docs.podman.io", "https://docs.podman.io"],
      "responseHeader": ["Content-Type"],
      "method": ["GET"],
      "maxAgeSeconds": 600
    }
]
```

```
gsutil cors set /path/to/file.json gs://libpod-master-releases
```

***Note:*** The CORS metadata does _NOT_ change after the `docs` task uploads a new swagger YAML
file.  Therefore, if it is not functioning or misconfigured, a person must have altered it or
changes were made to the referring site (e.g. `docs.podman.io`).

## Base-images

Base-images are VM disk-images specially prepared for executing as GCE VMs.
In particular, they run services on startup similar in purpose/function
as the standard 'cloud-init' services.

*  The google services are required for full support of ssh-key management
   and GCE OAuth capabilities.  Google provides native images in GCE
   with services pre-installed, for many platforms. For example,
   RHEL, CentOS, and Ubuntu.

*  Google does ***not*** provide any images for Fedora (as of 5/2019), nor do
   they provide a base-image prepared to run packer for creating other images
   in the ``test_build_vm_images`` Task (above).

*  Base images do not need to be produced often, but doing so completely
   manually would be time-consuming and error-prone.  Therefore a special
   semi-automatic *Makefile* target is provided to assist with producing
   all the base-images: ``libpod_base_images``

To produce new base-images, including an `image-builder-image` (used by
the ``cache_images`` Task) some input parameters are required:

* ``GCP_PROJECT_ID``: The complete GCP project ID string e.g. foobar-12345
  identifying where the images will be stored.

* ``GOOGLE_APPLICATION_CREDENTIALS``: A *JSON* file containing
  credentials for a GCE service account.  This can be [a service
  account](https://cloud.google.com/docs/authentication/production#obtaining_and_providing_service_account_credentials_manually)
  or [end-user
  credentials](https://cloud.google.com/docs/authentication/end-user#creating_your_client_credentials)

*  Optionally, CSV's may be specified to ``PACKER_BUILDS``
   to limit the base-images produced.  For example,
   ``PACKER_BUILDS=fedora,image-builder-image``.

If there is no existing 'image-builder-image' within GCE, a new
one may be bootstrapped by creating a CentOS 7 VM with support for
nested-virtualization, and with elevated cloud privileges (to access
GCE, from within the GCE VM).  For example:

```
$ alias pgcloud='sudo podman run -it --rm -e AS_ID=$UID
    -e AS_USER=$USER -v $HOME:$HOME:z quay.io/cevich/gcloud_centos:latest'

$ URL=https://www.googleapis.com/auth
$ SCOPES=$URL/userinfo.email,$URL/compute,$URL/devstorage.full_control

# The --min-cpu-platform is critical for nested-virt.
$ pgcloud compute instances create $USER-image-builder \
    --image-family centos-7 \
    --boot-disk-size "200GB" \
    --min-cpu-platform "Intel Haswell" \
    --machine-type n1-standard-2 \
    --scopes $SCOPES
```

Then from that VM, execute the
``contrib/cirrus/packer/image-builder-image_base_setup.sh`` script.
Shutdown the VM, and convert it into a new image-builder-image.

Building new base images is done by first creating a VM from an
image-builder-image and copying the credentials json file to it.

```
$ hack/get_ci_vm.sh image-builder-image-1541772081
...in another terminal...
$ pgcloud compute scp /path/to/gac.json $USER-image-builder-image-1541772081:.
```

Then, on the VM, change to the ``packer`` sub-directory, and build the images:

```
$ cd libpod/contrib/cirrus/packer
$ make libpod_base_images GCP_PROJECT_ID=<VALUE> \
    GOOGLE_APPLICATION_CREDENTIALS=/path/to/gac.json \
    PACKER_BUILDS=<OPTIONAL>
```

Assuming this is successful (hence the semi-automatic part), packer will
produce a ``packer-manifest.json`` output file.  This contains the base-image
names suitable for updating in ``.cirrus.yml``, `env` keys ``*_BASE_IMAGE``.

On failure, it should be possible to determine the problem from the packer
output.  Sometimes that means setting `PACKER_LOG=1` and troubleshooting
the nested virt calls.  It's also possible to observe the (nested) qemu-kvm
console output.  Simply set the ``TTYDEV`` parameter, for example:

```
$ make libpod_base_images ... TTYDEV=$(tty)
  ...
```

## `$SPECIALMODE`

Some tasks alter their behavior based on this value.  A summary of supported
values follows:

* `none`: Operate as normal, this is the default value if unspecified.
* `rootless`: Causes a random, ordinary user account to be created
              and utilized for testing.
* `in_podman`: Causes testing to occur within a container executed by
* `windows`: See **darwin**
* `darwin`: Signals the ``special_testing_cross`` task to cross-compile the remote client.
