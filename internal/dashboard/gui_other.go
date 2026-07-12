//go:build !windows

package dashboard

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func runGUI(url string, onClose func()) {
	fmt.Println("Letter Recon (dev) →", url)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	if onClose != nil {
		onClose()
	}
}
