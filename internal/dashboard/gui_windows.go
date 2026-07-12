//go:build windows

package dashboard

import webview2 "github.com/jchv/go-webview2"

func runGUI(url string, onClose func()) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Letter Recon",
			Width:  920,
			Height: 600,
			Center: true,
		},
	})
	if w == nil {
		return
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	if onClose != nil {
		onClose()
	}
}
