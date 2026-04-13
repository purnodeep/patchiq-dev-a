package inventory

import "time"

// LiveMetrics contains real-time system performance data.
type LiveMetrics struct {
	CollectedAt          time.Time      `json:"collected_at"`
	CPUUsagePct          float64        `json:"cpu_usage_pct"`
	CPUPerCore           []CoreMetric   `json:"cpu_per_core"`
	CPUTempCelsius       float64        `json:"cpu_temp_celsius"`
	LoadAvg1             float64        `json:"load_avg_1"`
	LoadAvg5             float64        `json:"load_avg_5"`
	LoadAvg15            float64        `json:"load_avg_15"`
	MemoryUsedPct        float64        `json:"memory_used_pct"`
	MemoryUsedBytes      uint64         `json:"memory_used_bytes"`
	MemoryTotalBytes     uint64         `json:"memory_total_bytes"`
	MemoryCachedBytes    uint64         `json:"memory_cached_bytes"`
	MemoryBuffersBytes   uint64         `json:"memory_buffers_bytes"`
	MemoryAvailableBytes uint64         `json:"memory_available_bytes"`
	SwapUsedBytes        uint64         `json:"swap_used_bytes"`
	SwapTotalBytes       uint64         `json:"swap_total_bytes"`
	Filesystems          []FSMetric     `json:"filesystems"`
	UptimeSeconds        uint64         `json:"uptime_seconds"`
	ProcessCount         int            `json:"process_count"`
	DiskIO               []DiskIOMetric `json:"disk_io"`
	NetworkIO            []NetIOMetric  `json:"network_io"`
	GPUUsagePct          float64        `json:"gpu_usage_pct"`
}

// CoreMetric holds per-core CPU usage and frequency.
type CoreMetric struct {
	CoreID   int     `json:"core_id"`
	UsagePct float64 `json:"usage_pct"`
	FreqMHz  float64 `json:"freq_mhz"`
}

// DiskIOMetric holds per-device disk I/O rates.
type DiskIOMetric struct {
	Device       string  `json:"device"`
	ReadBytesPS  float64 `json:"read_bytes_per_sec"`
	WriteBytesPS float64 `json:"write_bytes_per_sec"`
	IOUtilPct    float64 `json:"io_util_pct"`
}

// FSMetric holds filesystem usage data for a single mount point.
type FSMetric struct {
	Mount      string  `json:"mount"`
	Device     string  `json:"device"`
	FSType     string  `json:"fs_type"`
	TotalBytes uint64  `json:"total_bytes"`
	UsedBytes  uint64  `json:"used_bytes"`
	AvailBytes uint64  `json:"avail_bytes"`
	UsePct     float64 `json:"use_pct"`
}

// NetIOMetric holds per-interface network I/O rates.
type NetIOMetric struct {
	Interface   string  `json:"interface"`
	RxBytesPS   float64 `json:"rx_bytes_per_sec"`
	TxBytesPS   float64 `json:"tx_bytes_per_sec"`
	RxPacketsPS float64 `json:"rx_packets_per_sec"`
	TxPacketsPS float64 `json:"tx_packets_per_sec"`
}
