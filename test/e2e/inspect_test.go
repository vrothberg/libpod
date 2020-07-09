package integration

import (
	"os"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/selinux/go-selinux"
)

var _ = Describe("Podman inspect", func() {
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

	It("podman inspect alpine image", func() {
		session := podmanTest.Podman([]string{"inspect", "--format=json", ALPINE})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.IsJSONOutputValid()).To(BeTrue())
		imageData := session.InspectImageJSON()
		Expect(imageData[0].RepoTags[0]).To(Equal("docker.io/library/alpine:latest"))
	})

	It("podman inspect bogus container", func() {
		session := podmanTest.Podman([]string{"inspect", "foobar4321"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
	})

	It("podman inspect with GO format", func() {
		session := podmanTest.Podman([]string{"inspect", "--format", "{{.ID}}", ALPINE})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		result := podmanTest.Podman([]string{"images", "-q", "--no-trunc", ALPINE})
		result.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(strings.Contains(result.OutputToString(), session.OutputToString()))
	})

	It("podman inspect specified type", func() {
		session := podmanTest.Podman([]string{"inspect", "--type", "image", ALPINE})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman inspect container with GO format for ConmonPidFile", func() {
		session, ec, _ := podmanTest.RunLsContainer("test1")
		Expect(ec).To(Equal(0))

		session = podmanTest.Podman([]string{"inspect", "--format", "{{.ConmonPidFile}}", "test1"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman inspect container with size", func() {
		_, ec, _ := podmanTest.RunLsContainer("sizetest")
		Expect(ec).To(Equal(0))

		result := podmanTest.Podman([]string{"inspect", "--size", "sizetest"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		conData := result.InspectContainerToJSON()
		Expect(conData[0].SizeRootFs).To(BeNumerically(">", 0))
		Expect(*conData[0].SizeRw).To(BeNumerically(">=", 0))
	})

	It("podman inspect container and image", func() {
		ls, ec, _ := podmanTest.RunLsContainer("")
		Expect(ec).To(Equal(0))
		cid := ls.OutputToString()

		result := podmanTest.Podman([]string{"inspect", "--format={{.ID}}", cid, ALPINE})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(len(result.OutputToStringArray())).To(Equal(2))
	})

	It("podman inspect container and filter for Image{ID}", func() {
		ls, ec, _ := podmanTest.RunLsContainer("")
		Expect(ec).To(Equal(0))
		cid := ls.OutputToString()

		result := podmanTest.Podman([]string{"inspect", "--format={{.ImageID}}", cid})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(len(result.OutputToStringArray())).To(Equal(1))

		result = podmanTest.Podman([]string{"inspect", "--format={{.Image}}", cid})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(len(result.OutputToStringArray())).To(Equal(1))
	})

	It("podman inspect container and filter for CreateCommand", func() {
		ls, ec, _ := podmanTest.RunLsContainer("")
		Expect(ec).To(Equal(0))
		cid := ls.OutputToString()

		result := podmanTest.Podman([]string{"inspect", "--format={{.Config.CreateCommand}}", cid})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(len(result.OutputToStringArray())).To(Equal(1))
	})

	It("podman inspect -l with additional input should fail", func() {
		SkipIfRemote()
		result := podmanTest.Podman([]string{"inspect", "-l", "1234foobar"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(125))
	})

	It("podman inspect with mount filters", func() {

		ctrSession := podmanTest.Podman([]string{"create", "--name", "test", "-v", "/tmp:/test1", ALPINE, "top"})
		ctrSession.WaitWithDefaultTimeout()
		Expect(ctrSession.ExitCode()).To(Equal(0))

		inspectSource := podmanTest.Podman([]string{"inspect", "test", "--format", "{{(index .Mounts 0).Source}}"})
		inspectSource.WaitWithDefaultTimeout()
		Expect(inspectSource.ExitCode()).To(Equal(0))
		Expect(inspectSource.OutputToString()).To(Equal("/tmp"))

		inspectSrc := podmanTest.Podman([]string{"inspect", "test", "--format", "{{(index .Mounts 0).Src}}"})
		inspectSrc.WaitWithDefaultTimeout()
		Expect(inspectSrc.ExitCode()).To(Equal(0))
		Expect(inspectSrc.OutputToString()).To(Equal("/tmp"))

		inspectDestination := podmanTest.Podman([]string{"inspect", "test", "--format", "{{(index .Mounts 0).Destination}}"})
		inspectDestination.WaitWithDefaultTimeout()
		Expect(inspectDestination.ExitCode()).To(Equal(0))
		Expect(inspectDestination.OutputToString()).To(Equal("/test1"))

		inspectDst := podmanTest.Podman([]string{"inspect", "test", "--format", "{{(index .Mounts 0).Dst}}"})
		inspectDst.WaitWithDefaultTimeout()
		Expect(inspectDst.ExitCode()).To(Equal(0))
		Expect(inspectDst.OutputToString()).To(Equal("/test1"))
	})

	It("podman inspect shows healthcheck on docker image", func() {
		pull := podmanTest.Podman([]string{"pull", healthcheck})
		pull.WaitWithDefaultTimeout()
		Expect(pull.ExitCode()).To(BeZero())

		session := podmanTest.Podman([]string{"inspect", "--format=json", healthcheck})
		session.WaitWithDefaultTimeout()
		imageData := session.InspectImageJSON()
		Expect(imageData[0].HealthCheck.Timeout).To(BeNumerically("==", 3000000000))
		Expect(imageData[0].HealthCheck.Interval).To(BeNumerically("==", 60000000000))
		Expect(imageData[0].HealthCheck.Test).To(Equal([]string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"}))
	})

	It("podman inspect --latest with no container fails", func() {
		SkipIfRemote()

		session := podmanTest.Podman([]string{"inspect", "--latest"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Not(Equal(0)))
	})

	It("podman [image,container] inspect on image", func() {
		baseInspect := podmanTest.Podman([]string{"inspect", ALPINE})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Equal(0))
		baseJSON := baseInspect.InspectImageJSON()
		Expect(len(baseJSON)).To(Equal(1))

		ctrInspect := podmanTest.Podman([]string{"container", "inspect", ALPINE})
		ctrInspect.WaitWithDefaultTimeout()
		Expect(ctrInspect.ExitCode()).To(Not(Equal(0)))

		imageInspect := podmanTest.Podman([]string{"image", "inspect", ALPINE})
		imageInspect.WaitWithDefaultTimeout()
		Expect(imageInspect.ExitCode()).To(Equal(0))
		imageJSON := imageInspect.InspectImageJSON()
		Expect(len(imageJSON)).To(Equal(1))

		Expect(baseJSON[0].ID).To(Equal(imageJSON[0].ID))
	})

	It("podman [image, container] inspect on container", func() {
		ctrName := "testCtr"
		create := podmanTest.Podman([]string{"create", "--name", ctrName, ALPINE, "sh"})
		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(Equal(0))

		baseInspect := podmanTest.Podman([]string{"inspect", ctrName})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Equal(0))
		baseJSON := baseInspect.InspectContainerToJSON()
		Expect(len(baseJSON)).To(Equal(1))

		ctrInspect := podmanTest.Podman([]string{"container", "inspect", ctrName})
		ctrInspect.WaitWithDefaultTimeout()
		Expect(ctrInspect.ExitCode()).To(Equal(0))
		ctrJSON := ctrInspect.InspectContainerToJSON()
		Expect(len(ctrJSON)).To(Equal(1))

		imageInspect := podmanTest.Podman([]string{"image", "inspect", ctrName})
		imageInspect.WaitWithDefaultTimeout()
		Expect(imageInspect.ExitCode()).To(Not(Equal(0)))

		Expect(baseJSON[0].ID).To(Equal(ctrJSON[0].ID))
	})

	It("podman inspect always produces a valid array", func() {
		baseInspect := podmanTest.Podman([]string{"inspect", "doesNotExist"})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Not(Equal(0)))
		emptyJSON := baseInspect.InspectContainerToJSON()
		Expect(len(emptyJSON)).To(Equal(0))
	})

	It("podman inspect one container with not exist returns 1-length valid array", func() {
		ctrName := "testCtr"
		create := podmanTest.Podman([]string{"create", "--name", ctrName, ALPINE, "sh"})
		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(Equal(0))

		baseInspect := podmanTest.Podman([]string{"inspect", ctrName, "doesNotExist"})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Not(Equal(0)))
		baseJSON := baseInspect.InspectContainerToJSON()
		Expect(len(baseJSON)).To(Equal(1))
		Expect(baseJSON[0].Name).To(Equal(ctrName))
	})

	It("podman inspect container + image with same name gives container", func() {
		ctrName := "testcontainer"
		create := podmanTest.PodmanNoCache([]string{"create", "--name", ctrName, ALPINE, "sh"})
		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(Equal(0))

		tag := podmanTest.PodmanNoCache([]string{"tag", ALPINE, ctrName + ":latest"})
		tag.WaitWithDefaultTimeout()
		Expect(tag.ExitCode()).To(Equal(0))

		baseInspect := podmanTest.Podman([]string{"inspect", ctrName})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Equal(0))
		baseJSON := baseInspect.InspectContainerToJSON()
		Expect(len(baseJSON)).To(Equal(1))
		Expect(baseJSON[0].Name).To(Equal(ctrName))
	})

	It("podman inspect - HostConfig.SecurityOpt ", func() {
		if !selinux.GetEnabled() {
			Skip("SELinux not enabled")
		}

		ctrName := "hugo"
		create := podmanTest.PodmanNoCache([]string{
			"create", "--name", ctrName,
			"--security-opt", "seccomp=unconfined",
			"--security-opt", "label=type:spc_t",
			"--security-opt", "label=level:s0",
			ALPINE, "sh"})

		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(Equal(0))

		baseInspect := podmanTest.Podman([]string{"inspect", ctrName})
		baseInspect.WaitWithDefaultTimeout()
		Expect(baseInspect.ExitCode()).To(Equal(0))
		baseJSON := baseInspect.InspectContainerToJSON()
		Expect(len(baseJSON)).To(Equal(1))
		Expect(baseJSON[0].HostConfig.SecurityOpt).To(Equal([]string{"label=type:spc_t,label=level:s0", "seccomp=unconfined"}))
	})

})
