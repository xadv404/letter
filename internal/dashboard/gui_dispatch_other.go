//go:build !windows

package dashboard

func dispatchOnMain(fn func()) {
	fn()
}
