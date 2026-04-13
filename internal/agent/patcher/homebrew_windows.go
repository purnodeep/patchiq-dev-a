//go:build windows

package patcher

// brewOwnerUser is a no-op on Windows. Homebrew does not exist on Windows.
func (h *homebrewInstaller) brewOwnerUser() string {
	return ""
}
