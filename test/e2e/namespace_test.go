// +build !remote

package integration

import (
	"os"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman namespaces", func() {
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

	It("podman namespace test", func() {
		podman1 := podmanTest.Podman([]string{"--namespace", "test1", "run", "-d", ALPINE, "echo", "hello"})
		podman1.WaitWithDefaultTimeout()
		Expect(podman1.ExitCode()).To(Equal(0))

		podman2 := podmanTest.Podman([]string{"--namespace", "test2", "ps", "-aq"})
		podman2.WaitWithDefaultTimeout()
		Expect(podman2.ExitCode()).To(Equal(0))
		output := podman2.OutputToStringArray()
		numCtrs := 0
		for _, outputLine := range output {
			if outputLine != "" {
				numCtrs = numCtrs + 1
			}
		}
		Expect(numCtrs).To(Equal(0))

		numberOfCtrsNoNamespace := podmanTest.NumberOfContainers()
		Expect(numberOfCtrsNoNamespace).To(Equal(1))
	})
})
