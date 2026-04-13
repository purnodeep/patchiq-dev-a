package inventory

// HardwareInfo contains deep hardware inventory collected from the endpoint.
type HardwareInfo struct {
	CPU            CPUInfo         `json:"cpu"`
	Memory         MemoryInfo      `json:"memory"`
	Motherboard    MotherboardInfo `json:"motherboard"`
	Storage        []StorageDevice `json:"storage"`
	GPU            []GPUInfo       `json:"gpu"`
	Network        []NetworkInfo   `json:"network"`
	USB            []USBDevice     `json:"usb"`
	Battery        BatteryInfo     `json:"battery"`
	TPM            TPMInfo         `json:"tpm"`
	Virtualization VirtInfo        `json:"virtualization"`
}

// CPUInfo describes the processor(s) installed on the endpoint.
type CPUInfo struct {
	ModelName      string   `json:"model_name"`
	Vendor         string   `json:"vendor"`
	Family         string   `json:"family"`
	Model          string   `json:"model"`
	Stepping       string   `json:"stepping"`
	Architecture   string   `json:"architecture"`
	CoresPerSocket int      `json:"cores_per_socket"`
	ThreadsPerCore int      `json:"threads_per_core"`
	Sockets        int      `json:"sockets"`
	TotalLogical   int      `json:"total_logical_cpus"`
	MaxMHz         float64  `json:"max_mhz"`
	MinMHz         float64  `json:"min_mhz"`
	BogoMIPS       float64  `json:"bogomips"`
	CacheL1d       string   `json:"cache_l1d"`
	CacheL1i       string   `json:"cache_l1i"`
	CacheL2        string   `json:"cache_l2"`
	CacheL3        string   `json:"cache_l3"`
	Flags          []string `json:"flags"`
	VirtType       string   `json:"virtualization_type"`
}

// MemoryInfo describes system RAM and DIMM slots.
type MemoryInfo struct {
	TotalBytes      uint64     `json:"total_bytes"`
	AvailableBytes  uint64     `json:"available_bytes"`
	MaxCapacity     string     `json:"max_capacity"`
	NumSlots        int        `json:"num_slots"`
	ErrorCorrection string     `json:"error_correction"`
	DIMMs           []DIMMInfo `json:"dimms"`
}

// DIMMInfo describes a single memory module.
type DIMMInfo struct {
	Locator      string `json:"locator"`
	BankLocator  string `json:"bank_locator"`
	SizeMB       int    `json:"size_mb"`
	Type         string `json:"type"`
	SpeedMHz     int    `json:"speed_mhz"`
	Manufacturer string `json:"manufacturer"`
	SerialNumber string `json:"serial_number"`
	PartNumber   string `json:"part_number"`
	FormFactor   string `json:"form_factor"`
	Rank         string `json:"rank"`
}

// MotherboardInfo describes the baseboard and BIOS.
type MotherboardInfo struct {
	BoardManufacturer string `json:"board_manufacturer"`
	BoardProduct      string `json:"board_product"`
	BoardVersion      string `json:"board_version"`
	BoardSerial       string `json:"board_serial"`
	BIOSVendor        string `json:"bios_vendor"`
	BIOSVersion       string `json:"bios_version"`
	BIOSReleaseDate   string `json:"bios_release_date"`
}

// StorageDevice describes a block storage device and its partitions.
type StorageDevice struct {
	Name            string          `json:"name"`
	Model           string          `json:"model"`
	Serial          string          `json:"serial"`
	SizeBytes       uint64          `json:"size_bytes"`
	Type            string          `json:"type"`
	FirmwareVersion string          `json:"firmware_version"`
	Transport       string          `json:"transport"`
	SmartStatus     string          `json:"smart_status"`
	TempCelsius     int             `json:"temperature_celsius"`
	Partitions      []PartitionInfo `json:"partitions"`
}

// PartitionInfo describes a partition on a storage device.
type PartitionInfo struct {
	Name       string `json:"name"`
	SizeBytes  uint64 `json:"size_bytes"`
	FSType     string `json:"fstype"`
	MountPoint string `json:"mountpoint"`
	UsagePct   int    `json:"usage_pct"`
}

// GPUInfo describes a graphics processing unit.
type GPUInfo struct {
	Model         string `json:"model"`
	VRAMMB        int    `json:"vram_mb"`
	DriverVersion string `json:"driver_version"`
	PCISlot       string `json:"pci_slot"`
	UsagePct      int    `json:"usage_pct"`
}

// NetworkInfo describes a network interface.
type NetworkInfo struct {
	Name          string      `json:"name"`
	MACAddress    string      `json:"mac_address"`
	MTU           int         `json:"mtu"`
	Type          string      `json:"type"`
	State         string      `json:"state"`
	SpeedMbps     int         `json:"speed_mbps"`
	IPv4Addresses []IPAddress `json:"ipv4_addresses"`
	IPv6Addresses []IPAddress `json:"ipv6_addresses"`
	Driver        string      `json:"driver"`
}

// IPAddress represents an IP address with its prefix length.
type IPAddress struct {
	Address   string `json:"address"`
	PrefixLen int    `json:"prefix_len"`
}

// USBDevice describes a USB device attached to the endpoint.
type USBDevice struct {
	Bus         string `json:"bus"`
	DeviceNum   string `json:"device_num"`
	VendorID    string `json:"vendor_id"`
	ProductID   string `json:"product_id"`
	Description string `json:"description"`
}

// BatteryInfo describes battery status (primarily for laptops).
type BatteryInfo struct {
	Present        bool    `json:"present"`
	Status         string  `json:"status,omitempty"`
	CapacityPct    int     `json:"capacity_pct,omitempty"`
	EnergyFullWh   float64 `json:"energy_full_wh,omitempty"`
	EnergyDesignWh float64 `json:"energy_full_design_wh,omitempty"`
	HealthPct      int     `json:"health_pct,omitempty"`
	CycleCount     int     `json:"cycle_count,omitempty"`
	Technology     string  `json:"technology,omitempty"`
}

// TPMInfo describes the Trusted Platform Module.
type TPMInfo struct {
	Present bool   `json:"present"`
	Version string `json:"version,omitempty"`
}

// VirtInfo describes the virtualization environment.
type VirtInfo struct {
	IsVirtual      bool   `json:"is_virtual"`
	HypervisorType string `json:"hypervisor_type,omitempty"`
}
