//go:build windows

package inventory

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
)

// CollectHardware gathers deep hardware inventory from a Windows endpoint
// using PowerShell CIM queries. Each subsystem failure is logged as a warning
// but does not fail the overall collection — partial data is returned.
func CollectHardware(ctx context.Context, logger *slog.Logger) (*HardwareInfo, error) {
	hw := &HardwareInfo{}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_Processor | Select-Object Name, Manufacturer, Family, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed, L2CacheSize, L3CacheSize, Architecture, VirtualizationFirmwareEnabled | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: cpu query failed", "error", err)
	} else {
		hw.CPU = parseWinCPU(out)
	}

	memOS := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: memory OS query failed", "error", err)
	} else {
		memOS = out
	}
	memDIMM := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel, DeviceLocator, Capacity, SMBIOSMemoryType, ConfiguredClockSpeed, Manufacturer, SerialNumber, PartNumber, FormFactor | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: memory DIMM query failed", "error", err)
	} else {
		memDIMM = out
	}
	hw.Memory = parseWinMemory(memOS, memDIMM)

	if out, err := runPSCtx(ctx, `$b = Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer, Product, Version, SerialNumber; $i = Get-CimInstance Win32_BIOS | Select-Object Manufacturer, SMBIOSBIOSVersion, ReleaseDate; @{board=$b; bios=$i} | ConvertTo-Json -Compress -Depth 3`); err != nil {
		logger.Warn("hardware collector: motherboard query failed", "error", err)
	} else {
		hw.Motherboard = parseWinMotherboard(out)
	}

	diskRaw := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_DiskDrive | Select-Object DeviceID, Model, SerialNumber, Size, MediaType, InterfaceType, FirmwareRevision, Status, Partitions | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: disk drive query failed", "error", err)
	} else {
		diskRaw = out
	}
	logicalRaw := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Select-Object DeviceID, Size, FreeSpace, FileSystem, VolumeName | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: logical disk query failed", "error", err)
	} else {
		logicalRaw = out
	}
	hw.Storage = parseWinStorage(diskRaw, logicalRaw)

	gpuInfo := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_VideoController | Select-Object Name, AdapterRAM, DriverVersion, PNPDeviceID | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: gpu query failed", "error", err)
	} else {
		gpuInfo = out
	}
	gpuUsage := ""
	if out, err := runPSCtx(ctx, `try { $s=(Get-Counter '\GPU Engine(*engtype_3D*)\Utilization Percentage' -ErrorAction Stop).CounterSamples | Where-Object {$_.CookedValue -gt 0}; if($s){[math]::Min(100,[math]::Round(($s|Measure-Object CookedValue -Sum).Sum))}else{0} } catch { 0 }`); err != nil {
		logger.Debug("hardware collector: gpu utilization query failed", "error", err)
	} else {
		gpuUsage = out
	}
	hw.GPU = parseWinGPU(gpuInfo, gpuUsage)

	if out, err := runPSCtx(ctx, `$a = Get-NetAdapter | Where-Object { $_.Status -eq 'Up' -or $_.Status -eq 'Disconnected' } | Select-Object Name, MacAddress, MtuSize, Status, LinkSpeed, InterfaceDescription, DriverName; $i = Get-NetIPAddress -ErrorAction SilentlyContinue | Select-Object InterfaceAlias, IPAddress, PrefixLength, AddressFamily; @{adapters=$a; ips=$i} | ConvertTo-Json -Compress -Depth 3`); err != nil {
		logger.Warn("hardware collector: network query failed", "error", err)
	} else {
		hw.Network = parseWinNetwork(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_PnPEntity | Where-Object { $_.PNPDeviceID -like 'USB\*' } | Select-Object PNPDeviceID, Name | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: usb query failed", "error", err)
	} else {
		hw.USB = parseWinUSB(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_Battery | Select-Object BatteryStatus, EstimatedChargeRemaining, DesignCapacity, FullChargeCapacity, Chemistry | ConvertTo-Json -Compress`); err != nil {
		logger.Debug("hardware collector: battery query failed (expected on desktops)", "error", err)
	} else {
		hw.Battery = parseWinBattery(out)
	}

	if out, err := runPSCtx(ctx, `Get-Tpm -ErrorAction SilentlyContinue | Select-Object TpmPresent, ManufacturerVersion | ConvertTo-Json -Compress`); err != nil {
		logger.Debug("hardware collector: tpm query failed (may need elevation)", "error", err)
	} else {
		hw.TPM = parseWinTPM(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_ComputerSystem | Select-Object Model, HypervisorPresent | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: virtualization query failed", "error", err)
	} else {
		hw.Virtualization = parseWinVirtualization(out)
	}

	return hw, nil
}

// runPSCtx executes a PowerShell command with context and returns trimmed stdout.
func runPSCtx(ctx context.Context, cmd string) (string, error) {
	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
