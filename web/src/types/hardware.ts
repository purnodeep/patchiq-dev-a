/** TypeScript interfaces matching the agent's HardwareInfo JSONB structure. */

export interface HardwareInfo {
  cpu: CPUInfo;
  memory: MemoryInfo;
  motherboard: MotherboardInfo;
  storage: StorageDevice[];
  gpu: GPUInfo[];
  network: NetworkInfo[];
  usb: USBDevice[];
  battery: BatteryInfo;
  tpm: TPMInfo;
  virtualization: VirtInfo;
}

export interface CPUInfo {
  model_name: string;
  vendor: string;
  family: string;
  model: string;
  stepping: string;
  architecture: string;
  cores_per_socket: number;
  threads_per_core: number;
  sockets: number;
  total_logical_cpus: number;
  max_mhz: number;
  min_mhz: number;
  bogomips: number;
  cache_l1d: string;
  cache_l1i: string;
  cache_l2: string;
  cache_l3: string;
  flags: string[];
  virtualization_type: string;
}

export interface MemoryInfo {
  total_bytes: number;
  available_bytes: number;
  max_capacity: string;
  num_slots: number;
  error_correction: string;
  dimms: DIMMInfo[];
}

export interface DIMMInfo {
  locator: string;
  bank_locator: string;
  size_mb: number;
  type: string;
  speed_mhz: number;
  manufacturer: string;
  serial_number: string;
  part_number: string;
  form_factor: string;
  rank: string;
}

export interface MotherboardInfo {
  board_manufacturer: string;
  board_product: string;
  board_version: string;
  board_serial: string;
  bios_vendor: string;
  bios_version: string;
  bios_release_date: string;
}

export interface StorageDevice {
  name: string;
  model: string;
  serial: string;
  size_bytes: number;
  type: string;
  firmware_version: string;
  transport: string;
  smart_status: string;
  temperature_celsius: number;
  partitions: PartitionInfo[];
}

export interface PartitionInfo {
  name: string;
  size_bytes: number;
  fstype: string;
  mountpoint: string;
  usage_pct: number;
}

export interface GPUInfo {
  model: string;
  vram_mb: number;
  driver_version: string;
  pci_slot: string;
  usage_pct?: number;
}

export interface NetworkInfo {
  name: string;
  mac_address: string;
  mtu: number;
  type: string;
  state: string;
  speed_mbps: number;
  ipv4_addresses: IPAddress[];
  ipv6_addresses: IPAddress[];
  driver: string;
}

export interface IPAddress {
  address: string;
  prefix_len: number;
}

export interface USBDevice {
  bus: string;
  device_num: string;
  vendor_id: string;
  product_id: string;
  description: string;
}

export interface BatteryInfo {
  present: boolean;
  status?: string;
  capacity_pct?: number;
  energy_full_wh?: number;
  energy_full_design_wh?: number;
  health_pct?: number;
  cycle_count?: number;
  technology?: string;
}

export interface TPMInfo {
  present: boolean;
  version?: string;
}

export interface VirtInfo {
  is_virtual: boolean;
  hypervisor_type?: string;
}

/** Software summary stats stored as JSONB on the endpoint. */
export interface SoftwareSummary {
  total_packages: number;
  by_source: Record<string, number>;
  by_arch: Record<string, number>;
  security_updates_available: number;
  last_updated: string;
}
