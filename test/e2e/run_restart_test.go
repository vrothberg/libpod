// +build !remote

package integration

import (
	"os"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman run restart containers", func() {
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

	It("Podman start after successful run", func() {
		session := podmanTest.Podman([]string{"run", ALPINE, "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session2 := podmanTest.Podman([]string{"start", "--attach", "--latest"})
		session2.WaitWithDefaultTimeout()
		Expect(session2.ExitCode()).To(Equal(0))
	})

	It("Podman start after signal kill", func() {
		_ = podmanTest.RunTopContainer("test1")
		ok := WaitForContainer(podmanTest)
		Expect(ok).To(BeTrue())

		killSession := podmanTest.Podman([]string{"kill", "-s", "9", "test1"})
		killSession.WaitWithDefaultTimeout()
		Expect(killSession.ExitCode()).To(Equal(0))

		session2 := podmanTest.Podman([]string{"start", "test1"})
		session2.WaitWithDefaultTimeout()
		Expect(session2.ExitCode()).To(Equal(0))
	})
})
