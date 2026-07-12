//go:build windows

package dashboard

import webview2 "github.com/jchv/go-webview2"

var mainWebView webview2.WebView

func runGUI(url string, onClose func()) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Letter Recon",
			Width:  1000,
			Height: 600,
			Center: true,
		},
	})
	if w == nil {
		return
	}
	mainWebView = w
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
	mainWebView = nil
	if onClose != nil {
		onClose()
	}
}

func dispatchOnMain(fn func()) {
	if mainWebView != nil {
		mainWebView.Dispatch(fn)
		return
	}
	fn()
}
