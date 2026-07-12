//go:build windows

package dashboard

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

var mainWebView webview2.WebView

const webView2Download = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"

func runGUI(url string, onClose func()) error {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Letter Recon",
			Width:  1680,
			Height: 1050,
			Center: true,
		},
	})
	if w == nil {
		return runBrowserFallback(url, onClose)
	}
	mainWebView = w
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	mainWebView = nil
	if onClose != nil {
		onClose()
	}
	return nil
}

func runBrowserFallback(url string, onClose func()) error {
	_ = openDefaultBrowser(url)
	msg := "WebView2 n'est pas installé sur ce PC.\n\n" +
		"Letter Recon a été ouvert dans votre navigateur par défaut.\n\n" +
		"Pour la fenêtre native, installez Microsoft Edge WebView2 Runtime :\n" +
		webView2Download + "\n\n" +
		"Cliquez OK pour fermer Letter Recon."
	showMessageBox("Letter Recon", msg, true)
	if onClose != nil {
		onClose()
	}
	return nil
}

func openDefaultBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func showMessageBox(title, message string, isError bool) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	flags := uintptr(0x40) // MB_ICONINFORMATION
	if isError {
		flags = 0x10 // MB_ICONERROR
	}
	messageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}

func dispatchOnMain(fn func()) {
	if mainWebView != nil {
		mainWebView.Dispatch(fn)
		return
	}
	fn()
}

// failStartup shows a modal error when the app cannot start at all.
func failStartup(err error) {
	showMessageBox("Letter Recon — erreur", fmt.Sprintf("Impossible de démarrer :\n\n%v", err), true)
}
