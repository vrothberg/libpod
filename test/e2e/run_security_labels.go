// +build !remote

package integration

import (
	"os"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman generate kube", func() {
	var (
		tempdir    string
		err        error
		podmanTest *PodmanTestIntegration
	)

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanTestCreate(tempdir)
		podmanTest.Setup()
		podmanTest.SeedImages()

	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)
	})

	It("podman security labels", func() {
		test1 := podmanTest.Podman([]string{"create", "--label", "io.containers.capabilities=setuid,setgid", "--name", "test1", "alpine", "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Equal("CAP_SETUID,CAP_SETGID"))
	})

	It("podman bad security labels", func() {
		test1 := podmanTest.Podman([]string{"create", "--label", "io.containers.capabilities=sys_admin", "--name", "test1", "alpine", "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Not(Equal("CAP_SYS_ADMIN")))
	})

	It("podman --cap-add sys_admin security labels", func() {
		test1 := podmanTest.Podman([]string{"create", "--cap-add", "SYS_ADMIN", "--label", "io.containers.capabilities=sys_admin", "--name", "test1", "alpine", "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Equal("CAP_SYS_ADMIN"))
	})

	It("podman --cap-drop all sys_admin security labels", func() {
		test1 := podmanTest.Podman([]string{"create", "--cap-drop", "all", "--label", "io.containers.capabilities=sys_admin", "--name", "test1", "alpine", "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Equal(""))
	})

	It("podman security labels from image", func() {
		test1 := podmanTest.Podman([]string{"create", "--name", "test1", "alpine", "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		commit := podmanTest.Podman([]string{"commit", "-c", "label=io.containers.capabilities=sys_chroot,net_raw", "test1", "image1"})
		commit.WaitWithDefaultTimeout()
		Expect(commit.ExitCode()).To(BeZero())

		image1 := podmanTest.Podman([]string{"create", "--name", "test2", "image1", "echo", "test1"})
		image1.WaitWithDefaultTimeout()
		Expect(image1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test2"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Equal("CAP_SYS_CHROOT,CAP_NET_RAW"))

	})

	It("podman --privileged security labels", func() {
		pull := podmanTest.Podman([]string{"create", "--privileged", "--label", "io.containers.capabilities=setuid,setgid", "--name", "test1", "alpine", "echo", "test"})
		pull.WaitWithDefaultTimeout()
		Expect(pull.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Not(Equal("CAP_SETUID,CAP_SETGID")))
	})

	It("podman container runlabel (podman --version)", func() {
		PodmanDockerfile := `
FROM  alpine:latest
LABEL io.containers.capabilities=chown,mknod`

		image := "podman-caps:podman"
		podmanTest.BuildImage(PodmanDockerfile, image, "false")

		test1 := podmanTest.Podman([]string{"create", "--name", "test1", image, "echo", "test1"})
		test1.WaitWithDefaultTimeout()
		Expect(test1.ExitCode()).To(BeZero())

		inspect := podmanTest.Podman([]string{"inspect", "test1"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		ctr := inspect.InspectContainerToJSON()
		caps := strings.Join(ctr[0].EffectiveCaps, ",")
		Expect(caps).To(Equal("CAP_CHOWN,CAP_MKNOD"))
	})

})
