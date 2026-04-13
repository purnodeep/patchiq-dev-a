//go:build windows

package patcher

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// validUpdateID matches a UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// to prevent WQL injection via UpdateID interpolation.
var validUpdateID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type wuaUpdate struct {
	Title    string
	UpdateID string
}

type wuaInstallResult struct {
	ResultCode     int // 2=succeeded, 3=succeeded with errors, 4=failed, 5=aborted
	RebootRequired bool
}

type wuaClientIface interface {
	SearchUpdates(ctx context.Context, criteria string) ([]wuaUpdate, error)
	DownloadUpdates(ctx context.Context, updates []wuaUpdate) error
	InstallUpdates(ctx context.Context, updates []wuaUpdate) (wuaInstallResult, error)
}

type wuaInstaller struct {
	client wuaClientIface
	logger *slog.Logger
}

func (w *wuaInstaller) Name() string { return "wua" }

func (w *wuaInstaller) Install(ctx context.Context, pkg PatchTarget, dryRun bool) (InstallResult, error) {
	// Pre-flight: verify admin privileges and WUA service availability.
	if err := checkAdmin(); err != nil {
		return InstallResult{}, fmt.Errorf("wua install %s: %w", pkg.Name, err)
	}
	if err := checkWUAService(); err != nil {
		return InstallResult{}, fmt.Errorf("wua install %s: %w", pkg.Name, err)
	}

	criteria := "IsInstalled=0"
	updates, err := w.client.SearchUpdates(ctx, criteria)
	if err != nil {
		return InstallResult{}, fmt.Errorf("wua search: %w", err)
	}

	var matched []wuaUpdate
	for _, u := range updates {
		if strings.Contains(u.Title, pkg.Name) || strings.Contains(u.UpdateID, pkg.Name) {
			matched = append(matched, u)
		}
	}
	if len(matched) == 0 {
		return InstallResult{}, fmt.Errorf("wua install %s: no matching update found", pkg.Name)
	}

	if err := w.client.DownloadUpdates(ctx, matched); err != nil {
		return InstallResult{}, fmt.Errorf("wua download %s: %w", pkg.Name, err)
	}

	if dryRun {
		return InstallResult{
			Stdout: []byte(fmt.Sprintf("dry-run: downloaded %d update(s) for %s", len(matched), pkg.Name)),
		}, nil
	}

	result, err := w.client.InstallUpdates(ctx, matched)
	if err != nil {
		return InstallResult{}, fmt.Errorf("wua install %s: %w", pkg.Name, err)
	}

	exitCode := 0
	if result.ResultCode == 4 || result.ResultCode == 5 {
		exitCode = result.ResultCode
	}

	return InstallResult{
		Stdout:         []byte(fmt.Sprintf("WUA result code: %d", result.ResultCode)),
		ExitCode:       exitCode,
		RebootRequired: result.RebootRequired,
	}, nil
}

// comWUAClient implements wuaClientIface using real Windows Update Agent COM API.
type comWUAClient struct {
	logger *slog.Logger
}

func (c *comWUAClient) SearchUpdates(_ context.Context, criteria string) ([]wuaUpdate, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return nil, fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return nil, fmt.Errorf("create update session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("query session interface: %w", err)
	}
	defer session.Release()

	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return nil, fmt.Errorf("create searcher: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	resultDisp, err := oleutil.CallMethod(searcher, "Search", criteria)
	if err != nil {
		return nil, classifyCOMError(fmt.Errorf("search updates: %w", err))
	}
	result := resultDisp.ToIDispatch()
	defer result.Release()

	updatesDisp, err := oleutil.GetProperty(result, "Updates")
	if err != nil {
		return nil, fmt.Errorf("get updates collection: %w", err)
	}
	updatesCol := updatesDisp.ToIDispatch()
	defer updatesCol.Release()

	countVar, err := oleutil.GetProperty(updatesCol, "Count")
	if err != nil {
		return nil, fmt.Errorf("get updates count: %w", err)
	}
	count := int(countVar.Val)

	var results []wuaUpdate
	for i := range count {
		itemDisp, err := oleutil.GetProperty(updatesCol, "Item", i)
		if err != nil {
			c.logger.Warn("skip update item", "index", i, "error", err)
			continue
		}
		item := itemDisp.ToIDispatch()

		u := wuaUpdate{}
		if title, err := oleutil.GetProperty(item, "Title"); err == nil {
			u.Title = title.ToString()
		}
		if idProp, err := oleutil.GetProperty(item, "Identity"); err == nil {
			idDisp := idProp.ToIDispatch()
			if uid, err := oleutil.GetProperty(idDisp, "UpdateID"); err == nil {
				u.UpdateID = uid.ToString()
			}
			idDisp.Release()
		}
		item.Release()
		results = append(results, u)
	}

	return results, nil
}

func (c *comWUAClient) DownloadUpdates(_ context.Context, updates []wuaUpdate) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return fmt.Errorf("create update session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query session interface: %w", err)
	}
	defer session.Release()

	collUnknown, err := oleutil.CreateObject("Microsoft.Update.UpdateColl")
	if err != nil {
		return fmt.Errorf("create update collection: %w", err)
	}
	coll, err := collUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query collection interface: %w", err)
	}
	defer coll.Release()

	// Search for each update by UpdateID to get IUpdate objects to add to collection.
	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return fmt.Errorf("create searcher: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	for _, u := range updates {
		if !validUpdateID.MatchString(u.UpdateID) {
			c.logger.Warn("skipping update with invalid UpdateID", "updateID", u.UpdateID)
			continue
		}
		criteria := fmt.Sprintf("UpdateID='%s'", u.UpdateID)
		resultDisp, err := oleutil.CallMethod(searcher, "Search", criteria)
		if err != nil {
			c.logger.Warn("search update by ID failed", "updateID", u.UpdateID, "error", err)
			continue
		}
		result := resultDisp.ToIDispatch()

		updatesDisp, err := oleutil.GetProperty(result, "Updates")
		if err != nil {
			result.Release()
			continue
		}
		updatesCol := updatesDisp.ToIDispatch()

		countVar, err := oleutil.GetProperty(updatesCol, "Count")
		if err == nil && int(countVar.Val) > 0 {
			if itemDisp, err := oleutil.GetProperty(updatesCol, "Item", 0); err == nil {
				if _, err := oleutil.CallMethod(coll, "Add", itemDisp.ToIDispatch()); err != nil {
					c.logger.Warn("add update to collection failed", "updateID", u.UpdateID, "error", err)
				}
			}
		}
		updatesCol.Release()
		result.Release()
	}

	downloaderDisp, err := oleutil.CallMethod(session, "CreateUpdateDownloader")
	if err != nil {
		return fmt.Errorf("create downloader: %w", err)
	}
	downloader := downloaderDisp.ToIDispatch()
	defer downloader.Release()

	if _, err := oleutil.PutProperty(downloader, "Updates", coll); err != nil {
		return fmt.Errorf("set downloader updates: %w", err)
	}

	if _, err := oleutil.CallMethod(downloader, "Download"); err != nil {
		return classifyCOMError(fmt.Errorf("download updates: %w", err))
	}

	return nil
}

func (c *comWUAClient) InstallUpdates(_ context.Context, updates []wuaUpdate) (wuaInstallResult, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return wuaInstallResult{}, fmt.Errorf("COM init: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create update session: %w", err)
	}
	session, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("query session interface: %w", err)
	}
	defer session.Release()

	collUnknown, err := oleutil.CreateObject("Microsoft.Update.UpdateColl")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create update collection: %w", err)
	}
	coll, err := collUnknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("query collection interface: %w", err)
	}
	defer coll.Release()

	searcherDisp, err := oleutil.CallMethod(session, "CreateUpdateSearcher")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create searcher: %w", err)
	}
	searcher := searcherDisp.ToIDispatch()
	defer searcher.Release()

	for _, u := range updates {
		if !validUpdateID.MatchString(u.UpdateID) {
			c.logger.Warn("skipping update with invalid UpdateID", "updateID", u.UpdateID)
			continue
		}
		criteria := fmt.Sprintf("UpdateID='%s'", u.UpdateID)
		resultDisp, err := oleutil.CallMethod(searcher, "Search", criteria)
		if err != nil {
			c.logger.Warn("search update by ID failed", "updateID", u.UpdateID, "error", err)
			continue
		}
		result := resultDisp.ToIDispatch()

		updatesDisp, err := oleutil.GetProperty(result, "Updates")
		if err != nil {
			result.Release()
			continue
		}
		updatesCol := updatesDisp.ToIDispatch()

		countVar, err := oleutil.GetProperty(updatesCol, "Count")
		if err == nil && int(countVar.Val) > 0 {
			if itemDisp, err := oleutil.GetProperty(updatesCol, "Item", 0); err == nil {
				if _, err := oleutil.CallMethod(coll, "Add", itemDisp.ToIDispatch()); err != nil {
					c.logger.Warn("add update to collection failed", "updateID", u.UpdateID, "error", err)
				}
			}
		}
		updatesCol.Release()
		result.Release()
	}

	installerDisp, err := oleutil.CallMethod(session, "CreateUpdateInstaller")
	if err != nil {
		return wuaInstallResult{}, fmt.Errorf("create installer: %w", err)
	}
	installer := installerDisp.ToIDispatch()
	defer installer.Release()

	if _, err := oleutil.PutProperty(installer, "Updates", coll); err != nil {
		return wuaInstallResult{}, fmt.Errorf("set installer updates: %w", err)
	}

	installResultDisp, err := oleutil.CallMethod(installer, "Install")
	if err != nil {
		return wuaInstallResult{}, classifyCOMError(fmt.Errorf("install updates: %w", err))
	}
	installResult := installResultDisp.ToIDispatch()
	defer installResult.Release()

	var res wuaInstallResult

	if rc, err := oleutil.GetProperty(installResult, "ResultCode"); err == nil {
		res.ResultCode = int(rc.Val)
	}
	if rb, err := oleutil.GetProperty(installResult, "RebootRequired"); err == nil {
		res.RebootRequired = rb.Value().(bool)
	}

	return res, nil
}

// classifyCOMError attempts to map common COM/WUA HRESULT codes to
// descriptive errors. Falls back to the original error if unrecognized.
func classifyCOMError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	// E_ACCESSDENIED (0x80070005)
	if strings.Contains(msg, "80070005") {
		return fmt.Errorf("%w: %v", errNotAdmin, err)
	}
	// WU_E_INSTALL_NOT_ALLOWED (0x80240016)
	if strings.Contains(msg, "80240016") {
		return fmt.Errorf("patcher: Windows Update installation not allowed (policy or reboot pending): %w", err)
	}
	// WU_E_NO_SERVICE (0x8024001E)
	if strings.Contains(msg, "8024001E") {
		return fmt.Errorf("%w: %v", errWUAServiceStopped, err)
	}
	return err
}
