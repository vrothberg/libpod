package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman pod create", func() {
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

	It("podman create pod", func() {
		_, ec, podID := podmanTest.CreatePod("")
		Expect(ec).To(Equal(0))

		check := podmanTest.Podman([]string{"pod", "ps", "-q", "--no-trunc"})
		check.WaitWithDefaultTimeout()
		match, _ := check.GrepString(podID)
		Expect(match).To(BeTrue())
		Expect(len(check.OutputToStringArray())).To(Equal(1))
	})

	It("podman create pod with name", func() {
		name := "test"
		_, ec, _ := podmanTest.CreatePod(name)
		Expect(ec).To(Equal(0))

		check := podmanTest.Podman([]string{"pod", "ps", "--no-trunc"})
		check.WaitWithDefaultTimeout()
		match, _ := check.GrepString(name)
		Expect(match).To(BeTrue())
	})

	It("podman create pod with doubled name", func() {
		name := "test"
		_, ec, _ := podmanTest.CreatePod(name)
		Expect(ec).To(Equal(0))

		_, ec2, _ := podmanTest.CreatePod(name)
		Expect(ec2).To(Not(Equal(0)))

		check := podmanTest.Podman([]string{"pod", "ps", "-q"})
		check.WaitWithDefaultTimeout()
		Expect(len(check.OutputToStringArray())).To(Equal(1))
	})

	It("podman create pod with same name as ctr", func() {
		name := "test"
		session := podmanTest.Podman([]string{"create", "--name", name, ALPINE, "ls"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		_, ec, _ := podmanTest.CreatePod(name)
		Expect(ec).To(Not(Equal(0)))

		check := podmanTest.Podman([]string{"pod", "ps", "-q"})
		check.WaitWithDefaultTimeout()
		Expect(len(check.OutputToStringArray())).To(Equal(0))
	})

	It("podman create pod without network portbindings", func() {
		name := "test"
		session := podmanTest.Podman([]string{"pod", "create", "--name", name})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		pod := session.OutputToString()

		webserver := podmanTest.Podman([]string{"run", "--pod", pod, "-dt", nginx})
		webserver.WaitWithDefaultTimeout()
		Expect(webserver.ExitCode()).To(Equal(0))

		check := SystemExec("nc", []string{"-z", "localhost", "80"})
		Expect(check.ExitCode()).To(Equal(1))
	})

	It("podman create pod with network portbindings", func() {
		name := "test"
		session := podmanTest.Podman([]string{"pod", "create", "--name", name, "-p", "8080:80"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		pod := session.OutputToString()

		webserver := podmanTest.Podman([]string{"run", "--pod", pod, "-dt", nginx})
		webserver.WaitWithDefaultTimeout()
		Expect(webserver.ExitCode()).To(Equal(0))

		check := SystemExec("nc", []string{"-z", "localhost", "8080"})
		Expect(check.ExitCode()).To(Equal(0))
	})

	It("podman create pod with no infra but portbindings should fail", func() {
		name := "test"
		session := podmanTest.Podman([]string{"pod", "create", "--infra=false", "--name", name, "-p", "80:80"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(125))
	})

	It("podman create pod with --no-hosts", func() {
		SkipIfRemote()
		name := "test"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--no-hosts", "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		alpineResolvConf := podmanTest.Podman([]string{"run", "-ti", "--rm", "--no-hosts", ALPINE, "cat", "/etc/hosts"})
		alpineResolvConf.WaitWithDefaultTimeout()
		Expect(alpineResolvConf.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "cat", "/etc/hosts"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(podResolvConf.OutputToString()).To(Equal(alpineResolvConf.OutputToString()))
	})

	It("podman create pod with --no-hosts and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--no-hosts", "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with --add-host", func() {
		SkipIfRemote()
		name := "test"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--add-host", "test.example.com:12.34.56.78", "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "cat", "/etc/hosts"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), "12.34.56.78 test.example.com")).To(BeTrue())
	})

	It("podman create pod with --add-host and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--add-host", "test.example.com:12.34.56.78", "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with DNS server set", func() {
		SkipIfRemote()
		name := "test"
		server := "12.34.56.78"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns", server, "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "cat", "/etc/resolv.conf"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), fmt.Sprintf("nameserver %s", server))).To(BeTrue())
	})

	It("podman create pod with DNS server set and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		server := "12.34.56.78"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns", server, "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with DNS option set", func() {
		SkipIfRemote()
		name := "test"
		option := "attempts:5"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns-opt", option, "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "cat", "/etc/resolv.conf"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), fmt.Sprintf("options %s", option))).To(BeTrue())
	})

	It("podman create pod with DNS option set and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		option := "attempts:5"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns-opt", option, "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with DNS search domain set", func() {
		SkipIfRemote()
		name := "test"
		search := "example.com"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns-search", search, "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "cat", "/etc/resolv.conf"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), fmt.Sprintf("search %s", search))).To(BeTrue())
	})

	It("podman create pod with DNS search domain set and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		search := "example.com"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--dns-search", search, "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with IP address", func() {
		SkipIfRootless()
		name := "test"
		ip := GetRandomIPAddress()
		podCreate := podmanTest.Podman([]string{"pod", "create", "--ip", ip, "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "ip", "addr"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), ip)).To(BeTrue())
	})

	It("podman create pod with IP address and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		ip := GetRandomIPAddress()
		podCreate := podmanTest.Podman([]string{"pod", "create", "--ip", ip, "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod with MAC address", func() {
		SkipIfRemote()
		SkipIfRootless()
		name := "test"
		mac := "92:d0:c6:0a:29:35"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--mac-address", mac, "--name", name})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(0))

		podResolvConf := podmanTest.Podman([]string{"run", "--pod", name, "-ti", "--rm", ALPINE, "ip", "addr"})
		podResolvConf.WaitWithDefaultTimeout()
		Expect(podResolvConf.ExitCode()).To(Equal(0))
		Expect(strings.Contains(podResolvConf.OutputToString(), mac)).To(BeTrue())
	})

	It("podman create pod with MAC address and no infra should fail", func() {
		SkipIfRemote()
		name := "test"
		mac := "92:d0:c6:0a:29:35"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--mac-address", mac, "--name", name, "--infra=false"})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate.ExitCode()).To(Equal(125))
	})

	It("podman create pod and print id to external file", func() {
		// Switch to temp dir and restore it afterwards
		cwd, err := os.Getwd()
		Expect(err).To(BeNil())
		Expect(os.Chdir(os.TempDir())).To(BeNil())
		targetPath := filepath.Join(os.TempDir(), "dir")
		Expect(os.MkdirAll(targetPath, 0755)).To(BeNil())
		targetFile := filepath.Join(targetPath, "idFile")
		defer Expect(os.RemoveAll(targetFile)).To(BeNil())
		defer Expect(os.Chdir(cwd)).To(BeNil())

		session := podmanTest.Podman([]string{"pod", "create", "--name=abc", "--pod-id-file", targetFile})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		id, _ := ioutil.ReadFile(targetFile)
		check := podmanTest.Podman([]string{"pod", "inspect", "abc"})
		check.WaitWithDefaultTimeout()
		data := check.InspectPodToJSON()
		Expect(data.ID).To(Equal(string(id)))
	})

	It("podman pod create --replace", func() {
		// Make sure we error out with --name.
		session := podmanTest.Podman([]string{"pod", "create", "--replace", ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(125))

		// Create and replace 5 times in a row the "same" pod.
		podName := "testCtr"
		for i := 0; i < 5; i++ {
			session = podmanTest.Podman([]string{"pod", "create", "--replace", "--name", podName})
			session.WaitWithDefaultTimeout()
			Expect(session.ExitCode()).To(Equal(0))
		}
	})
})
