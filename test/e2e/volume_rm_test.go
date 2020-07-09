package integration

import (
	"os"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman volume rm", func() {
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
		podmanTest.CleanupVolume()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)

	})

	It("podman volume rm", func() {
		session := podmanTest.Podman([]string{"volume", "create", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "rm", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(len(session.OutputToStringArray())).To(Equal(0))
	})

	It("podman volume rm with --force flag", func() {
		session := podmanTest.Podman([]string{"create", "-v", "myvol:/myvol", ALPINE, "ls"})
		cid := session.OutputToString()
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "rm", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
		Expect(session.ErrorToString()).To(ContainSubstring(cid))

		session = podmanTest.Podman([]string{"volume", "rm", "-f", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(len(session.OutputToStringArray())).To(Equal(0))

		podmanTest.Cleanup()
	})

	It("podman rm with --all flag", func() {
		session := podmanTest.Podman([]string{"volume", "create", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "create"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "rm", "-a"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(len(session.OutputToStringArray())).To(Equal(0))
	})

	It("podman volume rm by partial name", func() {
		session := podmanTest.Podman([]string{"volume", "create", "myvol"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "rm", "myv"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(len(session.OutputToStringArray())).To(Equal(0))
	})

	It("podman volume rm by nonunique partial name", func() {
		session := podmanTest.Podman([]string{"volume", "create", "myvol1"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "create", "myvol2"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"volume", "rm", "myv"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())

		session = podmanTest.Podman([]string{"volume", "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(len(session.OutputToStringArray()) >= 2).To(BeTrue())
	})
})
