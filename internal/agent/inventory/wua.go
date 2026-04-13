//go:build windows

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// updateSearcher abstracts the Windows Update Agent search for testability.
type updateSearcher interface {
	Search(ctx context.Context, criteria string) ([]windowsUpdate, error)
}

// wuaCollector collects available Windows updates via COM IUpdateSearcher.
type wuaCollector struct {
	searcher updateSearcher
	logger   *slog.Logger
	mu       sync.RWMutex
	lastPkgs []ExtendedPackageInfo
}

func (c *wuaCollector) Name() string { return "wua" }

func (c *wuaCollector) Collect(ctx context.Context) ([]*pb.PackageInfo, error) {
	updates, err := c.searcher.Search(ctx, "IsInstalled=0")
	if err != nil {
		return nil, fmt.Errorf("wua collector: %w", err)
	}
	pkgs := mapWindowsUpdates(updates)

	next := make([]ExtendedPackageInfo, 0, len(pkgs))
	for _, p := range pkgs {
		next = append(next, ExtendedPackageInfo{
			Name:        p.Name,
			Version:     p.Version,
			Source:      p.Source,
			Category:    p.Category,
			Status:      "available",
			Description: p.KbArticle,
		})
	}
	c.mu.Lock()
	c.lastPkgs = next
	c.mu.Unlock()
	return pkgs, nil
}

func (c *wuaCollector) ExtendedPackages() []ExtendedPackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ExtendedPackageInfo, len(c.lastPkgs))
	copy(out, c.lastPkgs)
	return out
}

// comSearcher implements updateSearcher using COM interop.
type comSearcher struct {
	logger *slog.Logger
}

func (s *comSearcher) Search(_ context.Context, criteria string) ([]windowsUpdate, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
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
		return nil, fmt.Errorf("search updates: %w", err)
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

	var results []windowsUpdate
	for i := range count {
		itemDisp, err := oleutil.GetProperty(updatesCol, "Item", i)
		if err != nil {
			s.logger.Warn("skip update item", "index", i, "error", err)
			continue
		}
		item := itemDisp.ToIDispatch()

		wu := windowsUpdate{}

		if title, err := oleutil.GetProperty(item, "Title"); err == nil {
			wu.Title = title.ToString()
		}

		if sev, err := oleutil.GetProperty(item, "MsrcSeverity"); err == nil {
			wu.Severity = sev.ToString()
		}

		if kbCol, err := oleutil.GetProperty(item, "KBArticleIDs"); err == nil {
			kbDisp := kbCol.ToIDispatch()
			if kbCount, err := oleutil.GetProperty(kbDisp, "Count"); err == nil && kbCount.Val > 0 {
				if kbItem, err := oleutil.GetProperty(kbDisp, "Item", 0); err == nil {
					wu.KBID = "KB" + kbItem.ToString()
				}
			}
			kbDisp.Release()
		}

		if catCol, err := oleutil.GetProperty(item, "Categories"); err == nil {
			catDisp := catCol.ToIDispatch()
			if catCount, err := oleutil.GetProperty(catDisp, "Count"); err == nil {
				for j := range int(catCount.Val) {
					if catItem, err := oleutil.GetProperty(catDisp, "Item", j); err == nil {
						catObj := catItem.ToIDispatch()
						if catName, err := oleutil.GetProperty(catObj, "Name"); err == nil {
							wu.Categories = append(wu.Categories, catName.ToString())
						}
						catObj.Release()
					}
				}
			}
			catDisp.Release()
		}

		item.Release()
		results = append(results, wu)
	}

	return results, nil
}
