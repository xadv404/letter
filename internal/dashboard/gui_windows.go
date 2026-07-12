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

const (
	mbYesNo        = 0x00000004
	mbIconWarning  = 0x00000030
	mbIconError    = 0x00000010
	mbIconInfo     = 0x00000040
	idYes          = 6
)

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
		return promptWebView2Install(onClose)
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

func promptWebView2Install(onClose func()) error {
	msg := "Microsoft Edge WebView2 n'est pas installé sur ce PC.\n\n" +
		"Letter Recon en a besoin pour afficher l'interface.\n\n" +
		"Voulez-vous ouvrir la page de téléchargement pour l'installer ?"
	if confirmMessageBox("Letter Recon — WebView2 requis", msg) {
		_ = openDefaultBrowser(webView2Download)
	}
	if onClose != nil {
		onClose()
	}
	return nil
}

func openDefaultBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func confirmMessageBox(title, message string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return false
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return false
	}
	ret, _, _ := messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(mbYesNo|mbIconWarning),
	)
	return ret == idYes
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
	flags := uintptr(mbIconInfo)
	if isError {
		flags = mbIconError
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

func failStartup(err error) {
	showMessageBox("Letter Recon — erreur", fmt.Sprintf("Impossible de démarrer :\n\n%v", err), true)
}
