package utils_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common functions test", func() {
	var defaultOSPath string
	var defaultCgroupPath string

	BeforeEach(func() {
		defaultOSPath = OSReleasePath
		defaultCgroupPath = ProcessOneCgroupPath
	})

	AfterEach(func() {
		OSReleasePath = defaultOSPath
		ProcessOneCgroupPath = defaultCgroupPath
	})

	It("Test CreateTempDirInTempDir", func() {
		tmpDir, _ := CreateTempDirInTempDir()
		_, err := os.Stat(tmpDir)
		Expect(os.IsNotExist(err)).ShouldNot(BeTrue(), "Directory is not created as expect")
	})

	It("Test SystemExec", func() {
		session := SystemExec(GoechoPath, []string{})
		Expect(session.Command.Process).ShouldNot(BeNil(), "SystemExec cannot start a process")
	})

	It("Test StringInSlice", func() {
		testSlice := []string{"apple", "peach", "pear"}
		Expect(StringInSlice("apple", testSlice)).To(BeTrue(), "apple should in ['apple', 'peach', 'pear']")
		Expect(StringInSlice("banana", testSlice)).ShouldNot(BeTrue(), "banana should not in ['apple', 'peach', 'pear']")
		Expect(StringInSlice("anything", []string{})).ShouldNot(BeTrue(), "anything should not in empty slice")
	})

	DescribeTable("Test GetHostDistributionInfo",
		func(path, id, ver string, empty bool) {
			txt := fmt.Sprintf("ID=%s\nVERSION_ID=%s", id, ver)
			if !empty {
				f, _ := os.Create(path)
				f.WriteString(txt)
				f.Close()
			}

			OSReleasePath = path
			host := GetHostDistributionInfo()
			if empty {
				Expect(host).To(Equal(HostOS{}), "HostOs should be empty.")
			} else {
				Expect(host.Distribution).To(Equal(strings.Trim(id, "\"")))
				Expect(host.Version).To(Equal(strings.Trim(ver, "\"")))
			}
		},
		Entry("Configure file is not exist.", "/tmp/notexist", "", "", true),
		Entry("Item value with and without \"", "/tmp/os-release.test", "fedora", "\"28\"", false),
		Entry("Item empty with and without \"", "/tmp/os-release.test", "", "\"\"", false),
	)

	DescribeTable("Test IsKernelNewerThan",
		func(kv string, expect, isNil bool) {
			newer, err := IsKernelNewerThan(kv)
			Expect(newer).To(Equal(expect), "Version compare results is not as expect.")
			Expect(err == nil).To(Equal(isNil), "Error is not as expect.")
		},
		Entry("Invlid kernel version: 0", "0", false, false),
		Entry("Older kernel version:0.0", "0.0", true, true),
		Entry("Newer kernel version: 100.17.14", "100.17.14", false, true),
		Entry("Invlid kernel version: I am not a kernel version", "I am not a kernel version", false, false),
	)

	DescribeTable("Test TestIsCommandAvailable",
		func(cmd string, expect bool) {
			cmdExist := IsCommandAvailable(cmd)
			Expect(cmdExist).To(Equal(expect))
		},
		Entry("Command exist", GoechoPath, true),
		Entry("Command exist", "Fakecmd", false),
	)

	It("Test WriteJsonFile", func() {
		type testJson struct {
			Item1 int
			Item2 []string
		}
		compareData := &testJson{}

		testData := &testJson{
			Item1: 5,
			Item2: []string{"test"},
		}

		testByte, _ := json.Marshal(testData)
		err := WriteJsonFile(testByte, "/tmp/testJson")

		Expect(err).To(BeNil(), "Failed to write JSON to file.")

		read, err := os.Open("/tmp/testJson")
		defer read.Close()

		Expect(err).To(BeNil(), "Can not find the JSON file after we write it.")

		bytes, _ := ioutil.ReadAll(read)
		json.Unmarshal(bytes, compareData)

		Expect(reflect.DeepEqual(testData, compareData)).To(BeTrue(), "Data changed after we store it to file.")
	})

	DescribeTable("Test Containerized",
		func(path string, setEnv, createFile, expect bool) {
			if setEnv && (os.Getenv("container") == "") {
				os.Setenv("container", "test")
				defer os.Setenv("container", "")
			}
			if !setEnv && (os.Getenv("container") != "") {
				containerized := os.Getenv("container")
				os.Setenv("container", "")
				defer os.Setenv("container", containerized)
			}
			txt := "1:test:/"
			if expect {
				txt = "2:docker:/"
			}
			if createFile {
				f, _ := os.Create(path)
				f.WriteString(txt)
				f.Close()
			}
			ProcessOneCgroupPath = path
			Expect(Containerized()).To(Equal(expect))
		},
		Entry("Set container in env", "", true, false, true),
		Entry("Can not read from file", "/tmp/notexist", false, false, false),
		Entry("Docker in cgroup file", "/tmp/cgroup.test", false, true, true),
		Entry("Docker not in cgroup file", "/tmp/cgroup.test", false, true, false),
	)

})
