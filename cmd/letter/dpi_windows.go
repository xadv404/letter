//go:build windows

package main

import "syscall"

func init() {
	// Per-monitor DPI v2 — évite le rendu flou sur écrans 125%/150%.
	user32 := syscall.NewLazyDLL("user32.dll")
	setCtx := user32.NewProc("SetProcessDpiAwarenessContext")
	if err := setCtx.Find(); err == nil {
		const dpiAwareV2 = ^uintptr(3) // -4
		if r, _, _ := setCtx.Call(dpiAwareV2); r != 0 {
			return
		}
	}
	shcore := syscall.NewLazyDLL("shcore.dll")
	setAware := shcore.NewProc("SetProcessDpiAwareness")
	if err := setAware.Find(); err == nil {
		const perMonitor = 2
		_, _, _ = setAware.Call(uintptr(perMonitor))
	}
}
