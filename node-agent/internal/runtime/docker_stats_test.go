package runtime

import "testing"

func TestParseDockerStatsCPUPercent(t *testing.T) {
	raw := dockerStatsResponse{
		CPUStats: dockerCPUStats{
			SystemUsage: 200_000_000,
			OnlineCPUs:  4,
		},
		PreCPUStats: dockerCPUStats{
			SystemUsage: 100_000_000,
		},
	}
	raw.CPUStats.CPUUsage.TotalUsage = 50_000_000
	raw.PreCPUStats.CPUUsage.TotalUsage = 25_000_000

	stats := parseDockerStats(raw)

	// cpuDelta=25_000_000, systemDelta=100_000_000 -> 0.25 * 4 * 100 = 100%
	if stats.CPUPercent != 100 {
		t.Errorf("expected 100%% CPU, got %v", stats.CPUPercent)
	}
}

func TestParseDockerStatsZeroSystemDelta(t *testing.T) {
	raw := dockerStatsResponse{
		CPUStats:    dockerCPUStats{SystemUsage: 100, OnlineCPUs: 2},
		PreCPUStats: dockerCPUStats{SystemUsage: 100},
	}

	stats := parseDockerStats(raw)
	if stats.CPUPercent != 0 {
		t.Errorf("expected 0%% CPU when system delta is zero, got %v", stats.CPUPercent)
	}
}

func TestParseDockerStatsMemorySubtractsCache(t *testing.T) {
	raw := dockerStatsResponse{}
	raw.MemoryStats.Usage = 1000
	raw.MemoryStats.Limit = 2000
	raw.MemoryStats.Stats.Cache = 300

	stats := parseDockerStats(raw)
	if stats.MemoryUsedBytes != 700 {
		t.Errorf("expected memory usage 700 (1000-300 cache), got %d", stats.MemoryUsedBytes)
	}
	if stats.MemoryLimitBytes != 2000 {
		t.Errorf("expected memory limit 2000, got %d", stats.MemoryLimitBytes)
	}
}

func TestParseDockerStatsNetworkAndDiskAggregation(t *testing.T) {
	raw := dockerStatsResponse{
		Networks: map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		}{
			"eth0": {RxBytes: 100, TxBytes: 50},
			"eth1": {RxBytes: 200, TxBytes: 75},
		},
	}
	raw.BlkioStats.IoServiceBytesRecursive = []struct {
		Op    string `json:"op"`
		Value uint64 `json:"value"`
	}{
		{Op: "Read", Value: 10},
		{Op: "Write", Value: 20},
		{Op: "Read", Value: 5},
	}

	stats := parseDockerStats(raw)
	if stats.NetworkRxBytes != 300 || stats.NetworkTxBytes != 125 {
		t.Errorf("unexpected network totals: rx=%d tx=%d", stats.NetworkRxBytes, stats.NetworkTxBytes)
	}
	if stats.DiskReadBytes != 15 || stats.DiskWriteBytes != 20 {
		t.Errorf("unexpected disk totals: read=%d write=%d", stats.DiskReadBytes, stats.DiskWriteBytes)
	}
}
