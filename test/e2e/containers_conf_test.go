// +build !remote

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman run", func() {
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
		os.Setenv("CONTAINERS_CONF", "config/containers.conf")
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)
		os.Unsetenv("CONTAINERS_CONF")
	})

	It("podman run limits test", func() {
		SkipIfRootless()
		//containers.conf is set to "nofile=500:500"
		session := podmanTest.Podman([]string{"run", "--rm", fedoraMinimal, "ulimit", "-n"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("500"))

		session = podmanTest.Podman([]string{"run", "--rm", "--ulimit", "nofile=2048:2048", fedoraMinimal, "ulimit", "-n"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("2048"))
	})

	It("podman run with containers.conf having additional env", func() {
		//containers.conf default env includes foo
		session := podmanTest.Podman([]string{"run", ALPINE, "printenv"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("foo=bar"))
	})

	It("podman run with additional devices", func() {
		//containers.conf devices includes notone
		session := podmanTest.Podman([]string{"run", "--device", "/dev/null:/dev/bar", ALPINE, "ls", "/dev"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("bar"))
		Expect(session.OutputToString()).To(ContainSubstring("notone"))
	})

	It("podman run shm-size", func() {
		//containers.conf default sets shm-size=201k, which ends up as 200k
		session := podmanTest.Podman([]string{"run", ALPINE, "grep", "shm", "/proc/self/mounts"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("size=200k"))
	})

	It("podman Capabilities in containers.conf", func() {
		SkipIfRootless()
		os.Setenv("CONTAINERS_CONF", "config/containers.conf")
		cap := podmanTest.Podman([]string{"run", ALPINE, "grep", "CapEff", "/proc/self/status"})
		cap.WaitWithDefaultTimeout()
		Expect(cap.ExitCode()).To(Equal(0))

		os.Setenv("CONTAINERS_CONF", "config/containers-ns.conf")
		session := podmanTest.Podman([]string{"run", "busybox", "grep", "CapEff", "/proc/self/status"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).ToNot(Equal(cap.OutputToString()))
	})

	It("podman Regular capabilties", func() {
		SkipIfRootless()
		os.Setenv("CONTAINERS_CONF", "config/containers.conf")
		setup := podmanTest.RunTopContainer("test1")
		setup.WaitWithDefaultTimeout()
		result := podmanTest.Podman([]string{"top", "test1", "capeff"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.OutputToString()).To(ContainSubstring("SYS_CHROOT"))
		Expect(result.OutputToString()).To(ContainSubstring("NET_RAW"))
	})

	It("podman drop capabilties", func() {
		os.Setenv("CONTAINERS_CONF", "config/containers-caps.conf")
		setup := podmanTest.RunTopContainer("test1")
		setup.WaitWithDefaultTimeout()
		result := podmanTest.Podman([]string{"container", "top", "test1", "capeff"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.OutputToString()).ToNot(ContainSubstring("SYS_CHROOT"))
		Expect(result.OutputToString()).ToNot(ContainSubstring("NET_RAW"))
	})

	verifyNSHandling := func(nspath, option string) {
		os.Setenv("CONTAINERS_CONF", "config/containers-ns.conf")
		//containers.conf default ipcns to default to host
		session := podmanTest.Podman([]string{"run", ALPINE, "ls", "-l", nspath})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		fields := strings.Split(session.OutputToString(), " ")
		ctrNS := strings.TrimSuffix(fields[len(fields)-1], "\n")

		cmd := exec.Command("ls", "-l", nspath)
		res, err := cmd.Output()
		Expect(err).To(BeNil())
		fields = strings.Split(string(res), " ")
		hostNS := strings.TrimSuffix(fields[len(fields)-1], "\n")
		Expect(hostNS).To(Equal(ctrNS))

		session = podmanTest.Podman([]string{"run", option, "private", ALPINE, "ls", "-l", nspath})
		fields = strings.Split(session.OutputToString(), " ")
		ctrNS = fields[len(fields)-1]
		Expect(hostNS).ToNot(Equal(ctrNS))
	}

	It("podman compare netns", func() {
		verifyNSHandling("/proc/self/ns/net", "--network")
	})

	It("podman compare ipcns", func() {
		verifyNSHandling("/proc/self/ns/ipc", "--ipc")
	})

	It("podman compare utsns", func() {
		verifyNSHandling("/proc/self/ns/uts", "--uts")
	})

	It("podman compare pidns", func() {
		verifyNSHandling("/proc/self/ns/pid", "--pid")
	})

	It("podman compare cgroupns", func() {
		verifyNSHandling("/proc/self/ns/cgroup", "--cgroupns")
	})

	It("podman containers.conf additionalvolumes", func() {
		conffile := filepath.Join(podmanTest.TempDir, "container.conf")
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		err := ioutil.WriteFile(conffile, []byte(fmt.Sprintf("[containers]\nvolumes=[\"%s:%s:Z\",]\n", tempdir, tempdir)), 0755)
		if err != nil {
			os.Exit(1)
		}

		os.Setenv("CONTAINERS_CONF", conffile)
		result := podmanTest.Podman([]string{"run", ALPINE, "ls", tempdir})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
	})

	It("podman run containers.conf sysctl test", func() {
		SkipIfRootless()
		//containers.conf is set to   "net.ipv4.ping_group_range=0 1000"
		session := podmanTest.Podman([]string{"run", "--rm", fedoraMinimal, "cat", "/proc/sys/net/ipv4/ping_group_range"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("1000"))
	})

	It("podman run containers.conf search domain", func() {
		session := podmanTest.Podman([]string{"run", ALPINE, "cat", "/etc/resolv.conf"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		session.LineInOuputStartsWith("search foobar.com")
	})

	It("podman run add dns server", func() {
		session := podmanTest.Podman([]string{"run", ALPINE, "cat", "/etc/resolv.conf"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		session.LineInOuputStartsWith("server 1.2.3.4")
	})

	It("podman run add dns option", func() {
		session := podmanTest.Podman([]string{"run", ALPINE, "cat", "/etc/resolv.conf"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		session.LineInOuputStartsWith("options debug")
	})

	It("podman run containers.conf remove all search domain", func() {
		session := podmanTest.Podman([]string{"run", "--dns-search=.", ALPINE, "cat", "/etc/resolv.conf"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.LineInOuputStartsWith("search")).To(BeFalse())
	})

	It("podman run containers.conf timezone", func() {
		//containers.conf timezone set to Pacific/Honolulu
		session := podmanTest.Podman([]string{"run", ALPINE, "date", "+'%H %Z'"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("HST"))

	})
})
