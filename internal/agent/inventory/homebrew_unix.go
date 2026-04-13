//go:build !windows

package inventory

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// brewOwnerUser returns the username that owns the brew binary.
// Homebrew refuses to run as root, so when the agent is a root daemon
// we must run brew commands as the owning user via sudo -u.
func (c *homebrewCollector) brewOwnerUser() string {
	if os.Getuid() != 0 {
		return "" // not root, no need to switch user
	}
	info, err := os.Stat(c.brew())
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
