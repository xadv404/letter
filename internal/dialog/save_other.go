//go:build !windows

package dialog

// SaveDorks is a no-op on non-Windows builds (dev / CI).
func SaveDorks(srcPath string) (dest string, cancelled bool, err error) {
	return srcPath, false, nil
}
