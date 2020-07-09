package generate

import (
	"testing"

	"github.com/containers/podman/v2/pkg/domain/entities"
)

func TestValidateRestartPolicyContainer(t *testing.T) {
	type containerInfo struct {
		restart string
	}
	tests := []struct {
		name          string
		containerInfo containerInfo
		wantErr       bool
	}{
		{"good-on", containerInfo{restart: "no"}, false},
		{"good-on-success", containerInfo{restart: "on-success"}, false},
		{"good-on-failure", containerInfo{restart: "on-failure"}, false},
		{"good-on-abnormal", containerInfo{restart: "on-abnormal"}, false},
		{"good-on-watchdog", containerInfo{restart: "on-watchdog"}, false},
		{"good-on-abort", containerInfo{restart: "on-abort"}, false},
		{"good-always", containerInfo{restart: "always"}, false},
		{"fail", containerInfo{restart: "foobar"}, true},
		{"failblank", containerInfo{restart: ""}, true},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRestartPolicy(test.containerInfo.restart); (err != nil) != test.wantErr {
				t.Errorf("ValidateRestartPolicy() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateContainerSystemdUnit(t *testing.T) {
	goodID := `# container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.service
# autogenerated by Podman CI

[Unit]
Description=Podman container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStart=/usr/bin/podman start 639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401
ExecStop=/usr/bin/podman stop -t 10 639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401
ExecStopPost=/usr/bin/podman stop -t 10 639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401
PIDFile=/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodName := `# container-foobar.service
# autogenerated by Podman CI

[Unit]
Description=Podman container-foobar.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStart=/usr/bin/podman start foobar
ExecStop=/usr/bin/podman stop -t 10 foobar
ExecStopPost=/usr/bin/podman stop -t 10 foobar
PIDFile=/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodNameBoundTo := `# container-foobar.service
# autogenerated by Podman CI

[Unit]
Description=Podman container-foobar.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target
BindsTo=a.service b.service c.service pod.service
After=a.service b.service c.service pod.service

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStart=/usr/bin/podman start foobar
ExecStop=/usr/bin/podman stop -t 10 foobar
ExecStopPost=/usr/bin/podman stop -t 10 foobar
PIDFile=/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodWithNameAndGeneric := `# jadda-jadda.service
# autogenerated by Podman CI

[Unit]
Description=Podman jadda-jadda.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStartPre=/bin/rm -f %t/jadda-jadda.pid %t/jadda-jadda.ctr-id
ExecStart=/usr/bin/podman run --conmon-pidfile %t/jadda-jadda.pid --cidfile %t/jadda-jadda.ctr-id --cgroups=no-conmon -d --replace --name jadda-jadda --hostname hello-world awesome-image:latest command arg1 ... argN
ExecStop=/usr/bin/podman stop --ignore --cidfile %t/jadda-jadda.ctr-id -t 42
ExecStopPost=/usr/bin/podman rm --ignore -f --cidfile %t/jadda-jadda.ctr-id
PIDFile=%t/jadda-jadda.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodWithExplicitShortDetachParam := `# jadda-jadda.service
# autogenerated by Podman CI

[Unit]
Description=Podman jadda-jadda.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStartPre=/bin/rm -f %t/jadda-jadda.pid %t/jadda-jadda.ctr-id
ExecStart=/usr/bin/podman run --conmon-pidfile %t/jadda-jadda.pid --cidfile %t/jadda-jadda.ctr-id --cgroups=no-conmon --replace -d --name jadda-jadda --hostname hello-world awesome-image:latest command arg1 ... argN
ExecStop=/usr/bin/podman stop --ignore --cidfile %t/jadda-jadda.ctr-id -t 42
ExecStopPost=/usr/bin/podman rm --ignore -f --cidfile %t/jadda-jadda.ctr-id
PIDFile=%t/jadda-jadda.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodNameNewWithPodFile := `# jadda-jadda.service
# autogenerated by Podman CI

[Unit]
Description=Podman jadda-jadda.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStartPre=/bin/rm -f %t/jadda-jadda.pid %t/jadda-jadda.ctr-id
ExecStart=/usr/bin/podman run --conmon-pidfile %t/jadda-jadda.pid --cidfile %t/jadda-jadda.ctr-id --cgroups=no-conmon --pod-id-file /tmp/pod-foobar.pod-id-file --replace -d --name jadda-jadda --hostname hello-world awesome-image:latest command arg1 ... argN
ExecStop=/usr/bin/podman stop --ignore --cidfile %t/jadda-jadda.ctr-id -t 42
ExecStopPost=/usr/bin/podman rm --ignore -f --cidfile %t/jadda-jadda.ctr-id
PIDFile=%t/jadda-jadda.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodNameNewDetach := `# jadda-jadda.service
# autogenerated by Podman CI

[Unit]
Description=Podman jadda-jadda.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStartPre=/bin/rm -f %t/jadda-jadda.pid %t/jadda-jadda.ctr-id
ExecStart=/usr/bin/podman run --conmon-pidfile %t/jadda-jadda.pid --cidfile %t/jadda-jadda.ctr-id --cgroups=no-conmon --replace --detach --name jadda-jadda --hostname hello-world awesome-image:latest command arg1 ... argN
ExecStop=/usr/bin/podman stop --ignore --cidfile %t/jadda-jadda.ctr-id -t 42
ExecStopPost=/usr/bin/podman rm --ignore -f --cidfile %t/jadda-jadda.ctr-id
PIDFile=%t/jadda-jadda.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	goodIDNew := `# container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.service
# autogenerated by Podman CI

[Unit]
Description=Podman container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Restart=always
ExecStartPre=/bin/rm -f %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.pid %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.ctr-id
ExecStart=/usr/bin/podman run --conmon-pidfile %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.pid --cidfile %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.ctr-id --cgroups=no-conmon -d awesome-image:latest
ExecStop=/usr/bin/podman stop --ignore --cidfile %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.ctr-id -t 10
ExecStopPost=/usr/bin/podman rm --ignore -f --cidfile %t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.ctr-id
PIDFile=%t/container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401.pid
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

	tests := []struct {
		name    string
		info    containerInfo
		want    string
		new     bool
		wantErr bool
	}{

		{"good with id",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401",
				ContainerNameOrID: "639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       10,
				PodmanVersion:     "CI",
				EnvVariable:       EnvVariable,
			},
			goodID,
			false,
			false,
		},
		{"good with name",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "container-foobar",
				ContainerNameOrID: "foobar",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       10,
				PodmanVersion:     "CI",
				EnvVariable:       EnvVariable,
			},
			goodName,
			false,
			false,
		},
		{"good with name and bound to",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "container-foobar",
				ContainerNameOrID: "foobar",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       10,
				PodmanVersion:     "CI",
				BoundToServices:   []string{"pod", "a", "b", "c"},
				EnvVariable:       EnvVariable,
			},
			goodNameBoundTo,
			false,
			false,
		},
		{"bad restart policy",
			containerInfo{
				Executable:    "/usr/bin/podman",
				ServiceName:   "639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401",
				RestartPolicy: "never",
				PIDFile:       "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:   10,
				PodmanVersion: "CI",
				EnvVariable:   EnvVariable,
			},
			"",
			false,
			true,
		},
		{"good with name and generic",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "jadda-jadda",
				ContainerNameOrID: "jadda-jadda",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       42,
				PodmanVersion:     "CI",
				CreateCommand:     []string{"I'll get stripped", "container", "run", "--name", "jadda-jadda", "--hostname", "hello-world", "awesome-image:latest", "command", "arg1", "...", "argN"},
				EnvVariable:       EnvVariable,
			},
			goodWithNameAndGeneric,
			true,
			false,
		},
		{"good with explicit short detach param",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "jadda-jadda",
				ContainerNameOrID: "jadda-jadda",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       42,
				PodmanVersion:     "CI",
				CreateCommand:     []string{"I'll get stripped", "container", "run", "-d", "--name", "jadda-jadda", "--hostname", "hello-world", "awesome-image:latest", "command", "arg1", "...", "argN"},
				EnvVariable:       EnvVariable,
			},
			goodWithExplicitShortDetachParam,
			true,
			false,
		},
		{"good with explicit short detach param and podInfo",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "jadda-jadda",
				ContainerNameOrID: "jadda-jadda",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       42,
				PodmanVersion:     "CI",
				CreateCommand:     []string{"I'll get stripped", "container", "run", "-d", "--name", "jadda-jadda", "--hostname", "hello-world", "awesome-image:latest", "command", "arg1", "...", "argN"},
				EnvVariable:       EnvVariable,
				pod: &podInfo{
					PodIDFile: "/tmp/pod-foobar.pod-id-file",
				},
			},
			goodNameNewWithPodFile,
			true,
			false,
		},
		{"good with explicit full detach param",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "jadda-jadda",
				ContainerNameOrID: "jadda-jadda",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       42,
				PodmanVersion:     "CI",
				CreateCommand:     []string{"I'll get stripped", "container", "run", "--detach", "--name", "jadda-jadda", "--hostname", "hello-world", "awesome-image:latest", "command", "arg1", "...", "argN"},
				EnvVariable:       EnvVariable,
			},
			goodNameNewDetach,
			true,
			false,
		},
		{"good with id and no param",
			containerInfo{
				Executable:        "/usr/bin/podman",
				ServiceName:       "container-639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401",
				ContainerNameOrID: "639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401",
				RestartPolicy:     "always",
				PIDFile:           "/var/run/containers/storage/overlay-containers/639c53578af4d84b8800b4635fa4e680ee80fd67e0e6a2d4eea48d1e3230f401/userdata/conmon.pid",
				StopTimeout:       10,
				PodmanVersion:     "CI",
				CreateCommand:     []string{"I'll get stripped", "container", "run", "awesome-image:latest"},
				EnvVariable:       EnvVariable,
			},
			goodIDNew,
			true,
			false,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			opts := entities.GenerateSystemdOptions{
				Files: false,
				New:   test.new,
			}
			got, err := executeContainerTemplate(&test.info, opts)
			if (err != nil) != test.wantErr {
				t.Errorf("CreateContainerSystemdUnit() error = \n%v, wantErr \n%v", err, test.wantErr)
				return
			}
			if got != test.want {
				t.Errorf("CreateContainerSystemdUnit() = \n%v\n---------> want\n%v", got, test.want)
			}
		})
	}
}
