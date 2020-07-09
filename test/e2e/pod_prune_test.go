package integration

import (
	"os"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman pod prune", func() {
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

	It("podman pod prune empty pod", func() {
		_, ec, _ := podmanTest.CreatePod("")
		Expect(ec).To(Equal(0))

		result := podmanTest.Podman([]string{"pod", "prune", "--force"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
	})

	It("podman pod prune doesn't remove a pod with a running container", func() {
		_, ec, podid := podmanTest.CreatePod("")
		Expect(ec).To(Equal(0))

		ec2 := podmanTest.RunTopContainerInPod("", podid)
		ec2.WaitWithDefaultTimeout()
		Expect(ec2.ExitCode()).To(Equal(0))

		result := podmanTest.Podman([]string{"pod", "prune", "-f"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To((Equal(0)))

		result = podmanTest.Podman([]string{"ps", "-qa"})
		result.WaitWithDefaultTimeout()
		Expect(len(result.OutputToStringArray())).To(Equal(1))
	})

	It("podman pod prune removes a pod with a stopped container", func() {
		_, ec, podid := podmanTest.CreatePod("")
		Expect(ec).To(Equal(0))

		_, ec2, _ := podmanTest.RunLsContainerInPod("", podid)
		Expect(ec2).To(Equal(0))

		result := podmanTest.Podman([]string{"pod", "prune", "-f"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		result = podmanTest.Podman([]string{"ps", "-qa"})
		result.WaitWithDefaultTimeout()
		Expect(len(result.OutputToStringArray())).To(Equal(0))
	})
})
