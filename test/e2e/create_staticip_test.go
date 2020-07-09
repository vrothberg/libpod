// +build !remote

package integration

import (
	"os"
	"time"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman create with --ip flag", func() {
	var (
		tempdir    string
		err        error
		podmanTest *PodmanTestIntegration
	)

	BeforeEach(func() {
		SkipIfRootless()
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanTestCreate(tempdir)
		podmanTest.Setup()
		podmanTest.SeedImages()
		// Cleanup the CNI networks used by the tests
		os.RemoveAll("/var/lib/cni/networks/podman")
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)

	})

	It("Podman create --ip with garbage address", func() {
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", "114232346", ALPINE, "ls"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
	})

	It("Podman create --ip with v6 address", func() {
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", "2001:db8:bad:beef::1", ALPINE, "ls"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
	})

	It("Podman create --ip with non-allocatable IP", func() {
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", "203.0.113.124", ALPINE, "ls"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		result = podmanTest.Podman([]string{"start", "test"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
	})

	It("Podman create with specified static IP has correct IP", func() {
		ip := GetRandomIPAddress()
		result := podmanTest.Podman([]string{"create", "--name", "test", "--ip", ip, ALPINE, "ip", "addr"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		result = podmanTest.Podman([]string{"start", "test"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		result = podmanTest.Podman([]string{"logs", "test"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.OutputToString()).To(ContainSubstring(ip + "/16"))
	})

	It("Podman create two containers with the same IP", func() {
		ip := GetRandomIPAddress()
		result := podmanTest.Podman([]string{"create", "--name", "test1", "--ip", ip, ALPINE, "sleep", "999"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		result = podmanTest.Podman([]string{"create", "--name", "test2", "--ip", ip, ALPINE, "ip", "addr"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		result = podmanTest.Podman([]string{"start", "test1"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		// race prevention: wait until IP address is assigned
		for i := 0; i < 5; i++ {
			result = podmanTest.Podman([]string{"inspect", "--format", "{{.NetworkSettings.IPAddress}}", "test1"})
			result.WaitWithDefaultTimeout()
			Expect(result.ExitCode()).To(Equal(0))
			if result.OutputToString() != "" {
				break
			}
			time.Sleep(1 * time.Second)
		}
		Expect(result.OutputToString()).To(Equal(ip))

		// test1 container is running with the given IP.
		result = podmanTest.Podman([]string{"start", "test2"})
		result.WaitWithDefaultTimeout()
		Expect(result).To(ExitWithError())
		Expect(result.ErrorToString()).To(ContainSubstring("requested IP address " + ip + " is not available"))
	})
})
