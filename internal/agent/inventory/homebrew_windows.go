//go:build windows

package inventory

// brewOwnerUser is a no-op on Windows. Homebrew does not exist on Windows.
func (c *homebrewCollector) brewOwnerUser() string {
	return ""
}
