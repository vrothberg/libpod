#!/usr/bin/perl
#
# tests for logformatter
#
(our $ME = $0) =~ s|^.*/||;

use v5.14;
use strict;
use warnings;

use FindBin;
use File::Temp          qw(tempdir);
use Test::More;

#
# Read the test cases (see __END__ section below)
#
my @tests;
my $context = '';
while (my $line = <DATA>) {
    chomp $line;

    if ($line =~ /^==\s+(.*)/) {
        push @tests, { name => $1, input => [], expect => [] };
        $context = '';
    }
    elsif ($line =~ /^<<</) {
        $context = 'input';
    }
    elsif ($line =~ /^>>>/) {
        $context = 'expect';
    }
    elsif (@tests && $line) {
        push @{ $tests[-1]{$context} }, $line;
    }
}

plan tests => scalar(@tests);

my $tempdir = tempdir("logformatter-test.XXXXXX", TMPDIR => 1, CLEANUP => !$ENV{DEBUG});

chdir $tempdir
    or die "$ME: Could not cd $tempdir: $!\n";

for my $t (@tests) {
    my $name = $t->{name};
    (my $fname = $name) =~ s/\s+/_/g;

    open my $fh_out, '>', "$fname.txt"
        or die "$ME: Cannot create $tempdir/$fname.txt: $!\n";
    print { $fh_out } "$_\n" for @{$t->{input}};
    close $fh_out
        or die "$ME: Error writing $tempdir/$fname.txt: $!\n";

    system("$FindBin::Bin/logformatter $fname <$fname.txt >/dev/null");
    open my $fh_in, '<', "$fname.log.html"
        or die "$ME: Fatal: $fname: logformatter did not create .log.html\n";
    my @actual;
    while (my $line = <$fh_in>) {
        chomp $line;
        push @actual, $line  if $line =~ / begin processed output / .. $line =~ / end processed output /;
    }
    close $fh_in;

    # Strip off leading and trailing "<pre>"
    shift @actual; pop @actual;

    # For debugging: preserve expected results
    if ($ENV{DEBUG}) {
        open my $fh_out, '>', "$fname.expect";
        print { $fh_out } "$_\n" for @{$t->{expect}};
        close $fh_out;
    }

    is_deeply \@actual, $t->{expect}, $name;
}

chdir '/';



__END__

== simple bats

<<<
1..4
ok 1 hi
ok 2 bye # skip no reason
not ok 3 fail
# $ /path/to/podman foo -bar
# #| FAIL: exit code is 123; expected 321
ok 4 blah
>>>
1..4
<span class='bats-passed'><a name='t--00001'>ok 1 hi</a></span>
<span class='bats-skipped'><a name='t--00002'>ok 2 bye # skip no reason</a></span>
<span class='bats-failed'><a name='t--00003'>not ok 3 fail</a></span>
<span class='bats-log'># $ <span title="/path/to/podman">podman</span> foo -bar</span>
<span class='bats-log-esm'># #| FAIL: exit code is 123; expected 321</span>
<span class='bats-passed'><a name='t--00004'>ok 4 blah</a></span>
<hr/><span class='bats-summary'>Summary: <span class='bats-passed'>2 Passed</span>, <span class='bats-failed'>1 Failed</span>, <span class='bats-skipped'>1 Skipped</span>. Total tests: 4</span>







== simple ginkgo

<<<
$SCRIPT_BASE/integration_test.sh |& ${TIMESTAMP}
[08:26:19] START - All [+xxxx] lines that follow are relative to right now.
[+0002s] GO111MODULE=on go build -mod=vendor  -gcflags 'all=-trimpath=/var/tmp/go/src/github.com/containers/podman' -asmflags 'all=-trimpath=/var/tmp/go/src/github.com/containers/podman' -ldflags '-X github.com/containers/podman/libpod/define.gitCommit=40f5d8b1becd381c4e8283ed3940d09193e4fe06 -X github.com/containers/podman/libpod/define.buildInfo=1582809981 -X github.com/containers/podman/libpod/config._installPrefix=/usr/local -X github.com/containers/podman/libpod/config._etcDir=/etc -extldflags ""' -tags "   selinux systemd exclude_graphdriver_devicemapper seccomp varlink" -o bin/podman github.com/containers/podman/cmd/podman
[+0103s] •
[+0103s] ------------------------------
[+0103s] Podman pod restart
[+0103s]   podman pod restart single empty pod
[+0103s]   /var/tmp/go/src/github.com/containers/podman/test/e2e/pod_restart_test.go:41
[+0103s] [BeforeEach] Podman pod restart
[+0103s]   /var/tmp/go/src/github.com/containers/podman/test/e2e/pod_restart_test.go:18
[+0103s] [It] podman pod restart single empty pod
[+0103s]   /var/tmp/go/src/github.com/containers/podman/test/e2e/pod_restart_test.go:41
[+0103s] Running: /var/tmp/go/src/github.com/containers/podman/bin/podman --storage-opt vfs.imagestore=/tmp/podman/imagecachedir --root /tmp/podman_test553496330/crio --runroot /tmp/podman_test553496330/crio-run --runtime /usr/bin/runc --conmon /usr/bin/conmon --cni-config-dir /etc/cni/net.d --cgroup-manager systemd --tmpdir /tmp/podman_test553496330 --events-backend file --storage-driver vfs pod create --infra=false --share
[+0103s] 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
[+0103s] output: 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
[+0103s] Running: /var/tmp/go/src/github.com/containers/podman/bin/podman --storage-opt vfs.imagestore=/tmp/podman/imagecachedir --root /tmp/podman_test553496330/crio --runroot /tmp/podman_test553496330/crio-run --runtime /usr/bin/runc --conmon /usr/bin/conmon --cni-config-dir /etc/cni/net.d --cgroup-manager systemd --tmpdir /tmp/podman_test553496330 --events-backend file --storage-driver vfs pod restart 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
[+0103s] Error: no containers in pod 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89 have no dependencies, cannot start pod: no such container
[+0103s] output:
[+0103s] [AfterEach] Podman pod restart
[+0103s]   /var/tmp/go/src/github.com/containers/podman/test/e2e/pod_restart_test.go:28
[+0103s] Running: /var/tmp/go/src/github.com/containers/podman/bin/podman --storage-opt vfs.imagestore=/tmp/podman/imagecachedir --root /tmp/podman_test553496330/crio --runroot /tmp/podman_test553496330/crio-run --runtime /usr/bin/runc --conmon /usr/bin/conmon --cni-config-dir /etc/cni/net.d --cgroup-manager systemd --tmpdir /tmp/podman_test553496330 --events-backend file --storage-driver vfs pod rm -fa
[+0103s] 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
[+0107s] •
[+0107s] ------------------------------
[+0107s] podman system reset
>>>
$SCRIPT_BASE/integration_test.sh |&amp; ${TIMESTAMP}
[08:26:19] START - All [+xxxx] lines that follow are relative to right now.
<span class="timestamp">[+0002s] </span>GO111MODULE=on go build -mod=vendor  -gcflags &#39;all=-trimpath=/var/tmp/go/src/github.com/containers/podman&#39; -asmflags &#39;all=-trimpath=/var/tmp/go/src/github.com/containers/podman&#39; -ldflags &#39;-X github.com/containers/podman/libpod/define.gitCommit=40f5d8b1becd381c4e8283ed3940d09193e4fe06 -X github.com/containers/podman/libpod/define.buildInfo=1582809981 -X github.com/containers/podman/libpod/config._installPrefix=/usr/local -X github.com/containers/podman/libpod/config._etcDir=/etc -extldflags &quot;&quot;&#39; -tags &quot;   selinux systemd exclude_graphdriver_devicemapper seccomp varlink&quot; -o bin/podman github.com/containers/podman/cmd/podman
<span class="timestamp">[+0103s] </span>•
</pre>
<hr />
<pre>
<span class="timestamp">[+0103s] </span>Podman pod restart
<span class="timestamp">         </span><a name='t--podman-pod-restart-single-empty-pod--1'><h2>  podman pod restart single empty pod</h2></a>
<span class="timestamp">         </span>  /var/tmp/go/src/github.com<a class="codelink" href='https://github.com/containers/podman/blob/40f5d8b1becd381c4e8283ed3940d09193e4fe06/test/e2e/pod_restart_test.go#L41'>/containers/libpod/test/e2e/pod_restart_test.go:41</a>
<span class="timestamp">         </span>[BeforeEach] Podman pod restart
<span class="timestamp">         </span>  /var/tmp/go/src/github.com<a class="codelink" href='https://github.com/containers/podman/blob/40f5d8b1becd381c4e8283ed3940d09193e4fe06/test/e2e/pod_restart_test.go#L18'>/containers/libpod/test/e2e/pod_restart_test.go:18</a>
<span class="timestamp">         </span>[It] podman pod restart single empty pod
<span class="timestamp">         </span>  /var/tmp/go/src/github.com<a class="codelink" href='https://github.com/containers/podman/blob/40f5d8b1becd381c4e8283ed3940d09193e4fe06/test/e2e/pod_restart_test.go#L41'>/containers/libpod/test/e2e/pod_restart_test.go:41</a>
<span class="timestamp">         </span>Running: <span title="/var/tmp/go/src/github.com/containers/podman/bin/podman"><b>podman</b></span> <span class="boring" title="--storage-opt vfs.imagestore=/tmp/podman/imagecachedir
--root /tmp/podman_test553496330/crio
--runroot /tmp/podman_test553496330/crio-run
--runtime /usr/bin/runc
--conmon /usr/bin/conmon
--cni-config-dir /etc/cni/net.d
--cgroup-manager systemd
--tmpdir /tmp/podman_test553496330
--events-backend file
--storage-driver vfs">[options]</span><b> pod create --infra=false --share</b>
<span class="timestamp">         </span>4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
<span class="timestamp">         </span>Running: <span title="/var/tmp/go/src/github.com/containers/podman/bin/podman"><b>podman</b></span> <span class="boring" title="--storage-opt vfs.imagestore=/tmp/podman/imagecachedir
--root /tmp/podman_test553496330/crio
--runroot /tmp/podman_test553496330/crio-run
--runtime /usr/bin/runc
--conmon /usr/bin/conmon
--cni-config-dir /etc/cni/net.d
--cgroup-manager systemd
--tmpdir /tmp/podman_test553496330
--events-backend file
--storage-driver vfs">[options]</span><b> pod restart 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89</b>
<span class="timestamp">         </span><span class='log-warn'>Error: no containers in pod 4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89 have no dependencies, cannot start pod: no such container</span>
<span class="timestamp">         </span>output:
<span class="timestamp">         </span>[AfterEach] Podman pod restart
<span class="timestamp">         </span>  /var/tmp/go/src/github.com<a class="codelink" href='https://github.com/containers/podman/blob/40f5d8b1becd381c4e8283ed3940d09193e4fe06/test/e2e/pod_restart_test.go#L28'>/containers/libpod/test/e2e/pod_restart_test.go:28</a>
<span class="timestamp">         </span>Running: <span title="/var/tmp/go/src/github.com/containers/podman/bin/podman"><b>podman</b></span> <span class="boring" title="--storage-opt vfs.imagestore=/tmp/podman/imagecachedir
--root /tmp/podman_test553496330/crio
--runroot /tmp/podman_test553496330/crio-run
--runtime /usr/bin/runc
--conmon /usr/bin/conmon
--cni-config-dir /etc/cni/net.d
--cgroup-manager systemd
--tmpdir /tmp/podman_test553496330
--events-backend file
--storage-driver vfs">[options]</span><b> pod rm -fa</b>
<span class="timestamp">         </span>4810be0cfbd42241e349dbe7d50fbc54405cd320a6637c65fd5323f34d64af89
<span class="timestamp">[+0107s] </span>•
</pre>
<hr />
<pre>
<span class="timestamp">[+0107s] </span>podman system reset
