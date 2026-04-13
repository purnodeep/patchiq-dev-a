//go:build windows

package inventory

import (
	"log/slog"
	"os/exec"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func init() {
	platformCollectorDetectors = []collectorDetectorFunc{
		detectHotFixCollector,
		detectWUACollector,
		detectWUAInstalledCollector,
		detectRegistryCollector,
		detectPendingRebootCollector,
		detectWindowsFeaturesCollector,
	}
}

func detectHotFixCollector() packageCollector {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		slog.Warn("hotfix collector unavailable: powershell.exe not found", "error", err)
		return nil
	}
	return &hotfixCollector{runner: &execRunner{}}
}

func detectWUACollector() packageCollector {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		slog.Warn("wua collector unavailable: COM initialization failed", "error", err)
		return nil
	}
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		ole.CoUninitialize()
		slog.Warn("wua collector unavailable: Microsoft.Update.Session COM object unavailable", "error", err)
		return nil
	}
	unknown.Release()
	ole.CoUninitialize()

	return &wuaCollector{
		searcher: &comSearcher{logger: slog.Default()},
		logger:   slog.Default(),
	}
}

func detectWUAInstalledCollector() packageCollector {
	// Reuse the same COM probe as detectWUACollector.
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		slog.Warn("wua installed collector unavailable: COM initialization failed", "error", err)
		return nil
	}
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		ole.CoUninitialize()
		slog.Warn("wua installed collector unavailable: Microsoft.Update.Session COM object unavailable", "error", err)
		return nil
	}
	unknown.Release()
	ole.CoUninitialize()

	return &wuaInstalledCollector{
		searcher: &comSearcher{logger: slog.Default()},
		logger:   slog.Default(),
	}
}

func detectPendingRebootCollector() packageCollector {
	return &pendingRebootCollector{
		checker: &winRebootChecker{logger: slog.Default()},
		logger:  slog.Default(),
	}
}

func detectRegistryCollector() packageCollector {
	return &registryCollector{
		reader: &winRegistryReader{logger: slog.Default()},
		logger: slog.Default(),
	}
}

func detectWindowsFeaturesCollector() packageCollector {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		slog.Warn("windows features collector unavailable: powershell.exe not found", "error", err)
		return nil
	}
	return &windowsFeaturesCollector{
		runner: &execRunner{},
		logger: slog.Default(),
	}
}
