![PODMAN logo](logo/podman-logo-source.svg)
# Contributing to libpod

We'd love to have you join the community! Below summarizes the processes
that we follow.

## Topics

* [Reporting Issues](#reporting-issues)
* [Contributing to libpod](#contributing-to-libpod)
* [Continuous Integration](#continuous-integration) [![Build Status](https://api.cirrus-ci.com/github/containers/libpod.svg)](https://cirrus-ci.com/github/containers/libpod/master)
* [Submitting Pull Requests](#submitting-pull-requests)
* [Communications](#communications)

## Reporting Issues

Before reporting an issue, check our backlog of
[open issues](https://github.com/containers/podman/issues)
to see if someone else has already reported it. If so, feel free to add
your scenario, or additional information, to the discussion. Or simply
"subscribe" to it to be notified when it is updated.

If you find a new issue with the project we'd love to hear about it! The most
important aspect of a bug report is that it includes enough information for
us to reproduce it. So, please include as much detail as possible and try
to remove the extra stuff that doesn't really relate to the issue itself.
The easier it is for us to reproduce it, the faster it'll be fixed!

Please don't include any private/sensitive information in your issue!

## Contributing to libpod

This section describes how to start a contribution to libpod.

### Prepare your environment

Read the [install documentation to see how to install dependencies](install.md) .

The install documentation will illustrate the following steps:
- install libs and tools
- check installed versions
- configure network
- how to install libpod from sources

### Fork and clone libpod

First you need to fork this project on GitHub.

Be sure to have [defined your `$GOPATH` environment variable](https://github.com/golang/go/wiki/GOPATH).

Create a path that corresponds to the go import paths of libpod: `mkdir -p $GOPATH/src/github.com/containers`.

Then clone your fork locally:
```shell
$ git clone git@github.com:<you>/libpod $GOPATH/src/github.com/containers/podman
$ cd $GOPATH/src/github.com/containers/podman
```

### Deal with make

Libpod use a Makefile to realize common action like building etc...

You can list available actions by using:
```shell
$ make help
Usage: make <target>
...output...
```

### Install tools

Makefile allow you to install needed tools:
```shell
$ make install.tools
```

### Building binaries and test your changes

To test your changes do `make binaries` to generate your binaries.

Your binaries are created inside the `bin/` directory and you can test your changes:
```shell
$ bin/podman -h
bin/podman -h
NAME:
   podman - manage pods and images

USAGE:
   podman [global options] command [command options] [arguments...]

VERSION:
   1.0.1-dev

COMMANDS:
     attach           Attach to a running container
     build            Build an image using instructions from Dockerfiles
     commit           Create new image based on the changed container
     container        Manage Containers
     cp               Copy files/folders between a container and the local filesystem
```

Well, you can now create your own branch, apply changes on it, and then submitting your pull request.

For further reading about branching [you can read this document](https://herve.beraud.io/containers/linux/podman/isolate/environment/2019/02/06/how-to-hack-on-podman.html).

## Submitting Pull Requests

No Pull Request (PR) is too small! Typos, additional comments in the code,
new test cases, bug fixes, new features, more documentation, ... it's all
welcome!

While bug fixes can first be identified via an "issue", that is not required.
It's ok to just open up a PR with the fix, but make sure you include the same
information you would have included in an issue - like how to reproduce it.

PRs for new features should include some background on what use cases the
new code is trying to address. When possible and when it makes sense, try to break-up
larger PRs into smaller ones - it's easier to review smaller
code changes. But only if those smaller ones make sense as stand-alone PRs.

Regardless of the type of PR, all PRs should include:
* well documented code changes
* additional testcases. Ideally, they should fail w/o your code change applied
* documentation changes

Squash your commits into logical pieces of work that might want to be reviewed
separate from the rest of the PRs. But, squashing down to just one commit is ok
too since in the end the entire PR will be reviewed anyway. When in doubt,
squash.

PRs that fix issues should include a reference like `Closes #XXXX` in the
commit message so that GitHub will automatically close the referenced issue
when the PR is merged.

PRs will be approved by an [approver][owners] listed in [`OWNERS`](OWNERS).

### Describe your Changes in Commit Messages

Describe your problem. Whether your patch is a one-line bug fix or 5000 lines
of a new feature, there must be an underlying problem that motivated you to do
this work. Convince the reviewer that there is a problem worth fixing and that
it makes sense for them to read past the first paragraph.

Describe user-visible impact. Straight up crashes and lockups are pretty
convincing, but not all bugs are that blatant. Even if the problem was spotted
during code review, describe the impact you think it can have on users. Keep in
mind that the majority of users run packages provided by distributions, so
include anything that could help route your change downstream.

Quantify optimizations and trade-offs. If you claim improvements in
performance, memory consumption, stack footprint, or binary size, include
numbers that back them up. But also describe non-obvious costs. Optimizations
usually aren’t free but trade-offs between CPU, memory, and readability; or,
when it comes to heuristics, between different workloads. Describe the expected
downsides of your optimization so that the reviewer can weigh costs against
benefits.

Once the problem is established, describe what you are actually doing about it
in technical detail. It’s important to describe the change in plain English for
the reviewer to verify that the code is behaving as you intend it to.

Solve only one problem per patch. If your description starts to get long,
that’s a sign that you probably need to split up your patch.

If the patch fixes a logged bug entry, refer to that bug entry by number and
URL. If the patch follows from a mailing list discussion, give a URL to the
mailing list archive.

However, try to make your explanation understandable without external
resources. In addition to giving a URL to a mailing list archive or bug,
summarize the relevant points of the discussion that led to the patch as
submitted.

If you want to refer to a specific commit, don’t just refer to the SHA-1 ID of
the commit. Please also include the oneline summary of the commit, to make it
easier for reviewers to know what it is about. Example:

```
Commit f641c2d9384e ("fix bug in rm -fa parallel deletes") [...]
```

You should also be sure to use at least the first twelve characters of the
SHA-1 ID. The libpod repository holds a lot of objects, making collisions with
shorter IDs a real possibility. Bear in mind that, even if there is no
collision with your six-character ID now, that condition may change five years
from now.

If your patch fixes a bug in a specific commit, e.g. you found an issue using
git bisect, please use the ‘Fixes:’ tag with the first 12 characters of the
SHA-1 ID, and the one line summary. For example:

```
Fixes: f641c2d9384e ("fix bug in rm -fa parallel deletes")
```

The following git config settings can be used to add a pretty format for
outputting the above style in the git log or git show commands:

```
[core]
        abbrev = 12
[pretty]
        fixes = Fixes: %h (\"%s\")
```

### Sign your PRs

The sign-off is a line at the end of the explanation for the patch. Your
signature certifies that you wrote the patch or otherwise have the right to pass
it on as an open-source patch. The rules are simple: if you can certify
the below (from [developercertificate.org](http://developercertificate.org/)):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
660 York Street, Suite 102,
San Francisco, CA 94110 USA

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Then you just add a line to every git commit message:

    Signed-off-by: Joe Smith <joe.smith@email.com>

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.

### Go Format and lint

All code changes must pass ``make validate`` and ``make lint``, as
executed in a standard container.  The container image for this
purpose is provided at: ``quay.io/libpod/gate:master``.  With
other tags available for different branches as needed.  These
images are built automatically after merges to the branch.

#### Building the gate container locally

For local use, debugging, or experimentation, the gate image may
be built locally from the repository root, with the command:

```
podman build -t gate -f contrib/gate/Dockerfile .
```

***N/B:*** **don't miss the dot (.) at the end, it's really important**

#### Local use of gate container

The gate container's entry-point executes 'make' by default, on a copy of
the repository made at runtime.  This avoids the container changing or
leaving build artifacts in your hosts working directory.  It also guarantees
every execution is based upon pristine code provided from the host.

Execution does not require any special permissions from the host. However,
your libpod repository clone's root must be bind-mounted to the container at
'/usr/src/libpod'.  The copy will be made into /var/tmp/go (`$GOSRC` in container)
before running your make target.  For example, running `make lint` from a
repository clone at $HOME/devel/libpod could be done with the commands:

```bash
$ cd $HOME/devel/libpod
$ podman run -it --rm -v $PWD:/usr/src/libpod:ro \
    --security-opt label=disable quay.io/libpod/gate:master \
    lint
```

***N/B:*** Depending on your clone's git remotes-configuration,
(esp. for `validate` and `lint` targets), you may also need to reference the
commit which was your upstream fork-point.  Otherwise you may receive an error
similar to:

```
fatal: Not a valid object name master
Makefile:152: *** Required variable EPOCH_TEST_COMMIT value is undefined, whitespace, or empty.  Stop.
```

For example, assuming your have a remote called `upstream` running the
validate target should be done like this:

```bash
$ cd $HOME/devel/libpod
$ git remote update upstream
$ export EPOCH_TEST_COMMIT=$(git merge-base upstream/master HEAD)
$ podman run -it --rm -e EPOCH_TEST_COMMIT -v $PWD:/usr/src/libpod:ro \
    --security-opt label=disable quay.io/libpod/gate:master \
    validate
```

### Integration Tests

Our primary means of performing integration testing for libpod is with the
[Ginkgo](https://github.com/onsi/ginkgo) BDD testing framework. This allows
us to use native Golang to perform our tests and there is a strong affiliation
between Ginkgo and the Go test framework.  Adequate test cases are expected to
be provided with PRs.

For details on how to run the tests for Podman in your test environment, see the
Integration Tests [README.md](test/README.md).

## Continuous Integration

All pull requests and branch-merges automatically run:

* Go format/lint checking
* Unit testing
* Integration Testing
* Special testing (like running inside a container, or as a regular user)

For a more in-depth reference of the CI system, please [refer to it's dedicated
documentation.](contrib/cirrus/README.md)

There is always additional complexity added by automation, and so it sometimes
can fail for any number of reasons.  This includes post-merge testing on all
branches, which you may occasionally see [red bars on the status graph
.](https://cirrus-ci.com/github/containers/libpod/master)

When the graph shows mostly green bars on the right, it's a good indication
the master branch is currently stable.  Alternating red/green bars is indicative
of a testing "flake", and should be examined (anybody can do this):

* *One or a small handful of tests, on a single task, (i.e. specific distro/version)
  where all others ran successfully:*  Frequently the cause is networking or a brief
  external service outage.  The failed tasks may simply be re-run by pressing the
  corresponding button on the task details page.

* *Multiple tasks failing*: Logically this should be due to some shared/common element.
  If that element is identifiable as a networking or external service (e.g. packaging
  repository outage), a re-run should be attempted.

* *All tasks are failing*: If a common element is **not** identifiable as
  temporary (i.e. container registry outage), please seek assistance via
  [the methods below](#communications) as this may be early indication of
  a more serious problem.

In the (hopefully) rare case there are multiple, contiguous red bars, this is
a ***very bad*** sign.  It means additional merges are occurring despite an uncorrected
or persistently faulty condition.  This risks additional bugs being introduced
and further complication of necessary corrective measures.  Most likely people
are aware and working on this, but it doesn't hurt [to confirm and/or try and help
if possible.](#communications)

## Communications

For general questions and discussion, please use the
IRC `#podman` channel on `irc.freenode.net`.

For discussions around issues/bugs and features, you can use the GitHub
[issues](https://github.com/containers/podman/issues)
and
[PRs](https://github.com/containers/podman/pulls)
tracking system.

There is also a [mailing list](https://lists.podman.io/archives/) at `lists.podman.io`.
You can subscribe by sending a message to `podman@lists.podman.io` with the subject `subscribe`.

### Bot Interactions

The primary human-interface is through comments in pull-requests.  Some of these are outlined
below, along with their meaning and intended usage.  Some of them require the comment
author hold special privileges on the github repository.  Others can be used by anyone.

* ``/close``: Closes an issue or PR.

* ``/approve``: Mark a PR as appropriate to the project, and as close to meeting
  met all the contribution criteria above.  Adds the *approved* label, marking
  it as ready for review and possible future merging.

* ``/lgtm``: A literal "Stamp of approval", signaling okay-to-merge.  This causes
  the bot to ad the *lgtm* label, then attempt a merge.  In other words - Never,
  ever, ever comment ``/lgtm``, unless a PR has actually, really, been fully
  reviewed.  The bot isn't too smart about these things, and could merge
  unintentionally.  Instead, just write ``LGTM``, or
  spell it out.

* ``/hold`` and ``/unhold``: Override the automatic handling of a request.  Either
  put it on hold (no handling) or remove the hold (normal handling).

* ``[ci skip]``: [Adding `[ci skip]` within the HEAD commit](https://cirrus-ci.org/guide/writing-tasks/#conditional-task-execution)
  will cause Cirrus CI to ***NOT*** execute tests for the PR or after merge.  This
  is useful in only one instance:  Your changes are absolutely not exercised by
  any test.  For example, documentation changes.  ***IMPORTANT NOTE*** **Other
  automation may interpret the lack of test results as "PASSED" and unintentional
  merge a PR.  Consider also using `/hold` in a comment, to add additional
  protection.**

[The complete list may be found on the command-help page.](https://prow.k8s.io/command-help)
However, not all commands are implemented for this repository.  If in doubt, ask a maintainer.
