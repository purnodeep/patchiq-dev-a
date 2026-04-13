package inventory

import (
	"runtime"
	"strings"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// windowsUpdate represents a single Windows Update result from IUpdateSearcher.
type windowsUpdate struct {
	KBID       string
	Title      string
	Severity   string
	Categories []string
}

// mapWindowsUpdates converts WUA search results to PackageInfo protos.
// Entries with empty KBID are skipped.
func mapWindowsUpdates(updates []windowsUpdate) []*pb.PackageInfo {
	var pkgs []*pb.PackageInfo
	for _, u := range updates {
		if u.KBID == "" {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:         u.Title,
			Version:      u.KBID,
			Architecture: runtime.GOARCH,
			Source:       "wua",
			KbArticle:    u.KBID,
			Severity:     u.Severity,
			Category:     strings.Join(u.Categories, ", "),
		})
	}
	return pkgs
}
