package generate

import (
	"github.com/containers/common/pkg/sysinfo"
	"github.com/containers/podman/v2/pkg/cgroups"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/pkg/errors"
)

// Verify resource limits are sanely set, removing any limits that are not
// possible with the current cgroups config.
func verifyContainerResources(s *specgen.SpecGenerator) ([]string, error) {
	warnings := []string{}

	cgroup2, err := cgroups.IsCgroup2UnifiedMode()
	if err != nil || cgroup2 {
		return warnings, err
	}

	sysInfo := sysinfo.New(true)

	if s.ResourceLimits == nil {
		return warnings, nil
	}

	// Memory checks
	if s.ResourceLimits.Memory != nil {
		memory := s.ResourceLimits.Memory
		if memory.Limit != nil && !sysInfo.MemoryLimit {
			warnings = append(warnings, "Your kernel does not support memory limit capabilities or the cgroup is not mounted. Limitation discarded.")
			memory.Limit = nil
			memory.Swap = nil
		}
		if memory.Limit != nil && memory.Swap != nil && !sysInfo.SwapLimit {
			warnings = append(warnings, "Your kernel does not support swap limit capabilities,or the cgroup is not mounted. Memory limited without swap.")
			memory.Swap = nil
		}
		if memory.Limit != nil && memory.Swap != nil && *memory.Swap < *memory.Limit {
			return warnings, errors.New("minimum memoryswap limit should be larger than memory limit, see usage")
		}
		if memory.Limit == nil && memory.Swap != nil {
			return warnings, errors.New("you should always set a memory limit when using a memoryswap limit, see usage")
		}
		if memory.Swappiness != nil {
			if !sysInfo.MemorySwappiness {
				warnings = append(warnings, "Your kernel does not support memory swappiness capabilities, or the cgroup is not mounted. Memory swappiness discarded.")
				memory.Swappiness = nil
			} else {
				if *memory.Swappiness < 0 || *memory.Swappiness > 100 {
					return warnings, errors.Errorf("invalid value: %v, valid memory swappiness range is 0-100", *memory.Swappiness)
				}
			}
		}
		if memory.Reservation != nil && !sysInfo.MemoryReservation {
			warnings = append(warnings, "Your kernel does not support memory soft limit capabilities or the cgroup is not mounted. Limitation discarded.")
			memory.Reservation = nil
		}
		if memory.Limit != nil && memory.Reservation != nil && *memory.Limit < *memory.Reservation {
			return warnings, errors.New("minimum memory limit cannot be less than memory reservation limit, see usage")
		}
		if memory.Kernel != nil && !sysInfo.KernelMemory {
			warnings = append(warnings, "Your kernel does not support kernel memory limit capabilities or the cgroup is not mounted. Limitation discarded.")
			memory.Kernel = nil
		}
		if memory.DisableOOMKiller != nil && *memory.DisableOOMKiller && !sysInfo.OomKillDisable {
			warnings = append(warnings, "Your kernel does not support OomKillDisable. OomKillDisable discarded.")
			memory.DisableOOMKiller = nil
		}
	}

	// Pids checks
	if s.ResourceLimits.Pids != nil {
		pids := s.ResourceLimits.Pids
		// TODO: Should this be 0, or checking that ResourceLimits.Pids
		// is set at all?
		if pids.Limit > 0 && !sysInfo.PidsLimit {
			warnings = append(warnings, "Your kernel does not support pids limit capabilities or the cgroup is not mounted. PIDs limit discarded.")
			s.ResourceLimits.Pids = nil
		}
	}

	// CPU Checks
	if s.ResourceLimits.CPU != nil {
		cpu := s.ResourceLimits.CPU
		if cpu.Shares != nil && !sysInfo.CPUShares {
			warnings = append(warnings, "Your kernel does not support CPU shares or the cgroup is not mounted. Shares discarded.")
			cpu.Shares = nil
		}
		if cpu.Period != nil && !sysInfo.CPUCfsPeriod {
			warnings = append(warnings, "Your kernel does not support CPU cfs period or the cgroup is not mounted. Period discarded.")
			cpu.Period = nil
		}
		if cpu.Period != nil && (*cpu.Period < 1000 || *cpu.Period > 1000000) {
			return warnings, errors.New("CPU cfs period cannot be less than 1ms (i.e. 1000) or larger than 1s (i.e. 1000000)")
		}
		if cpu.Quota != nil && !sysInfo.CPUCfsQuota {
			warnings = append(warnings, "Your kernel does not support CPU cfs quota or the cgroup is not mounted. Quota discarded.")
			cpu.Quota = nil
		}
		if cpu.Quota != nil && *cpu.Quota < 1000 {
			return warnings, errors.New("CPU cfs quota cannot be less than 1ms (i.e. 1000)")
		}
		if (cpu.Cpus != "" || cpu.Mems != "") && !sysInfo.Cpuset {
			warnings = append(warnings, "Your kernel does not support cpuset or the cgroup is not mounted. CPUset discarded.")
			cpu.Cpus = ""
			cpu.Mems = ""
		}

		cpusAvailable, err := sysInfo.IsCpusetCpusAvailable(cpu.Cpus)
		if err != nil {
			return warnings, errors.Errorf("invalid value %s for cpuset cpus", cpu.Cpus)
		}
		if !cpusAvailable {
			return warnings, errors.Errorf("requested CPUs are not available - requested %s, available: %s", cpu.Cpus, sysInfo.Cpus)
		}

		memsAvailable, err := sysInfo.IsCpusetMemsAvailable(cpu.Mems)
		if err != nil {
			return warnings, errors.Errorf("invalid value %s for cpuset mems", cpu.Mems)
		}
		if !memsAvailable {
			return warnings, errors.Errorf("requested memory nodes are not available - requested %s, available: %s", cpu.Mems, sysInfo.Mems)
		}
	}

	// Blkio checks
	if s.ResourceLimits.BlockIO != nil {
		blkio := s.ResourceLimits.BlockIO
		if blkio.Weight != nil && !sysInfo.BlkioWeight {
			warnings = append(warnings, "Your kernel does not support Block I/O weight or the cgroup is not mounted. Weight discarded.")
			blkio.Weight = nil
		}
		if blkio.Weight != nil && (*blkio.Weight > 1000 || *blkio.Weight < 10) {
			return warnings, errors.New("range of blkio weight is from 10 to 1000")
		}
		if len(blkio.WeightDevice) > 0 && !sysInfo.BlkioWeightDevice {
			warnings = append(warnings, "Your kernel does not support Block I/O weight_device or the cgroup is not mounted. Weight-device discarded.")
			blkio.WeightDevice = nil
		}
		if len(blkio.ThrottleReadBpsDevice) > 0 && !sysInfo.BlkioReadBpsDevice {
			warnings = append(warnings, "Your kernel does not support BPS Block I/O read limit or the cgroup is not mounted. Block I/O BPS read limit discarded")
			blkio.ThrottleReadBpsDevice = nil
		}
		if len(blkio.ThrottleWriteBpsDevice) > 0 && !sysInfo.BlkioWriteBpsDevice {
			warnings = append(warnings, "Your kernel does not support BPS Block I/O write limit or the cgroup is not mounted. Block I/O BPS write limit discarded.")
			blkio.ThrottleWriteBpsDevice = nil
		}
		if len(blkio.ThrottleReadIOPSDevice) > 0 && !sysInfo.BlkioReadIOpsDevice {
			warnings = append(warnings, "Your kernel does not support IOPS Block read limit or the cgroup is not mounted. Block I/O IOPS read limit discarded.")
			blkio.ThrottleReadIOPSDevice = nil
		}
		if len(blkio.ThrottleWriteIOPSDevice) > 0 && !sysInfo.BlkioWriteIOpsDevice {
			warnings = append(warnings, "Your kernel does not support IOPS Block I/O write limit or the cgroup is not mounted. Block I/O IOPS write limit discarded.")
			blkio.ThrottleWriteIOPSDevice = nil
		}
	}

	return warnings, nil
}
