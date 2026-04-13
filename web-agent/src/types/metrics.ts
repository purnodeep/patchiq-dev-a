export interface LiveMetrics {
  collected_at: string;
  cpu_usage_pct: number;
  cpu_per_core: CoreMetric[];
  cpu_temp_celsius: number;
  load_avg_1: number;
  load_avg_5: number;
  load_avg_15: number;
  memory_used_pct: number;
  memory_used_bytes: number;
  memory_total_bytes: number;
  memory_cached_bytes: number;
  memory_buffers_bytes: number;
  memory_available_bytes: number;
  swap_used_bytes: number;
  swap_total_bytes: number;
  filesystems?: FilesystemMetric[];
  uptime_seconds: number;
  process_count: number;
  disk_io: DiskIO[];
  network_io: NetworkIO[];
  gpu_usage_pct: number;
}

export interface CoreMetric {
  core_id: number;
  usage_pct: number;
  freq_mhz: number;
}

export interface DiskIO {
  device: string;
  read_bytes_per_sec: number;
  write_bytes_per_sec: number;
  io_util_pct: number;
}

export interface FilesystemMetric {
  mount: string;
  device: string;
  fs_type: string;
  total_bytes: number;
  used_bytes: number;
  avail_bytes: number;
  use_pct: number;
}

export interface NetworkIO {
  interface: string;
  rx_bytes_per_sec: number;
  tx_bytes_per_sec: number;
  rx_packets_per_sec: number;
  tx_packets_per_sec: number;
}
