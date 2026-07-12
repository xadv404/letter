//go:build !windows

package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

func runGUI(url string, onClose func()) error {
	openBrowser(url)
	fmt.Println("Letter Recon →", url)
	fmt.Println("Ctrl+C pour quitter")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	if onClose != nil {
		onClose()
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", url)
		}
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}

func failStartup(err error) {
	fmt.Fprintln(os.Stderr, "Letter Recon — erreur:", err)
}

func dispatchOnMain(fn func()) {
	fn()
}
