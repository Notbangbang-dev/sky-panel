package runtime

// dockerStatsResponse mirrors the subset of the Docker Engine API's
// "/containers/{id}/stats" response we need. Field names/casing follow the
// Engine API exactly (see Docker's ContainerStats type).
type dockerStatsResponse struct {
	CPUStats    dockerCPUStats    `json:"cpu_stats"`
	PreCPUStats dockerCPUStats    `json:"precpu_stats"`
	MemoryStats dockerMemoryStats `json:"memory_stats"`
	Networks    map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

type dockerCPUStats struct {
	CPUUsage struct {
		TotalUsage uint64 `json:"total_usage"`
	} `json:"cpu_usage"`
	SystemUsage uint64 `json:"system_cpu_usage"`
	OnlineCPUs  uint64 `json:"online_cpus"`
}

type dockerMemoryStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
	// Stats.cache is subtracted from Usage on Linux cgroups v1 to match what
	// `docker stats` displays (raw cgroup "usage" double-counts page cache).
	Stats struct {
		Cache uint64 `json:"cache"`
	} `json:"stats"`
}

// parseDockerStats converts a raw Engine API stats payload into our Stats
// type. This is a pure function so the CPU-percent math (the one genuinely
// fiddly part of the Docker stats API) can be unit tested without a daemon.
func parseDockerStats(raw dockerStatsResponse) Stats {
	var rx, tx uint64
	for _, n := range raw.Networks {
		rx += n.RxBytes
		tx += n.TxBytes
	}

	var read, write uint64
	for _, entry := range raw.BlkioStats.IoServiceBytesRecursive {
		switch entry.Op {
		case "Read", "read":
			read += entry.Value
		case "Write", "write":
			write += entry.Value
		}
	}

	memUsed := raw.MemoryStats.Usage
	if raw.MemoryStats.Stats.Cache > 0 && memUsed > raw.MemoryStats.Stats.Cache {
		memUsed -= raw.MemoryStats.Stats.Cache
	}

	return Stats{
		CPUPercent:       cpuPercent(raw.CPUStats, raw.PreCPUStats),
		MemoryUsedBytes:  memUsed,
		MemoryLimitBytes: raw.MemoryStats.Limit,
		NetworkRxBytes:   rx,
		NetworkTxBytes:   tx,
		DiskReadBytes:    read,
		DiskWriteBytes:   write,
	}
}

// cpuPercent implements the same formula the official `docker stats` CLI
// uses: the container's share of total CPU delta since the previous sample,
// scaled by the number of online CPUs.
func cpuPercent(cur, prev dockerCPUStats) float64 {
	cpuDelta := float64(cur.CPUUsage.TotalUsage) - float64(prev.CPUUsage.TotalUsage)
	systemDelta := float64(cur.SystemUsage) - float64(prev.SystemUsage)

	if systemDelta <= 0 || cpuDelta < 0 {
		return 0
	}

	onlineCPUs := float64(cur.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	return (cpuDelta / systemDelta) * onlineCPUs * 100.0
}
