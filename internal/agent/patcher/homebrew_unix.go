//go:build !windows

package patcher

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// brewOwnerUser returns the username that owns the brew binary.
func (h *homebrewInstaller) brewOwnerUser() string {
	if os.Getuid() != 0 {
		return ""
	}
	info, err := os.Stat(h.brew())
	if err != nil {
		return ""
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	u, err := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		return ""
	}
	return u.Username
}
