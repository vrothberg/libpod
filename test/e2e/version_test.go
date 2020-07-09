package integration

import (
	"os"

	. "github.com/containers/podman/v2/test/utils"
	"github.com/containers/podman/v2/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Podman version", func() {
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
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)
		podmanTest.SeedImages()

	})

	It("podman version", func() {
		session := podmanTest.Podman([]string{"version"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.Out.Contents()).Should(ContainSubstring(version.Version))
	})

	It("podman -v", func() {
		session := podmanTest.Podman([]string{"-v"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.Out.Contents()).Should(ContainSubstring(version.Version))
	})

	It("podman --version", func() {
		session := podmanTest.Podman([]string{"--version"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.Out.Contents()).Should(ContainSubstring(version.Version))
	})

	It("podman version --format json", func() {
		session := podmanTest.Podman([]string{"version", "--format", "json"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.IsJSONOutputValid()).To(BeTrue())
	})

	It("podman version --format json", func() {
		session := podmanTest.Podman([]string{"version", "--format", "{{ json .}}"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.IsJSONOutputValid()).To(BeTrue())
	})

	It("podman version --format GO template", func() {
		session := podmanTest.Podman([]string{"version", "--format", "{{ .Client.Version }}"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"version", "--format", "{{ .Server.Version }}"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"version", "--format", "{{ .Version }}"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
	})
})
