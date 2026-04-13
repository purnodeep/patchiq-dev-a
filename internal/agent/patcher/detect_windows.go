//go:build windows

package patcher

import (
	"log/slog"
	"os/exec"
)

func init() {
	platformInstallerDetectors = []installerDetectorFunc{
		detectMSIInstaller,   // PRIMARY — best type detection, logging, version query
		detectWUAInstaller,   // Windows Update
		detectMSIXInstaller,  // Modern apps
		detectEXEInstaller,   // FALLBACK — always available
	}
}

func detectMSIInstaller(executor CommandExecutor) Installer {
	if _, err := exec.LookPath("msiexec"); err != nil {
		return nil
	}
	return &msiInstaller{executor: executor}
}

func detectMSIXInstaller(executor CommandExecutor) Installer {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return nil
	}
	return &msixInstaller{executor: executor}
}

func detectWUAInstaller(_ CommandExecutor) Installer {
	return &wuaInstaller{
		client: &comWUAClient{logger: slog.Default()},
		logger: slog.Default(),
	}
}

func detectEXEInstaller(executor CommandExecutor) Installer {
	return &exeInstaller{
		executor: executor,
		logger:   slog.Default(),
	}
}
