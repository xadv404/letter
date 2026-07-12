package dashboard

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

// Run starts the embedded HTML dashboard and blocks until shutdown.
func Run() error {
	app := newApp()

	content, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/config", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.handleConfigGet(w, r)
		case http.MethodPut:
			app.handleConfigPut(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/domains", corsMiddleware(http.HandlerFunc(app.handleDomainsUpload)))
	mux.Handle("/api/start", corsMiddleware(http.HandlerFunc(app.handleStart)))
	mux.Handle("/api/pause", corsMiddleware(http.HandlerFunc(app.handlePause)))
	mux.Handle("/api/resume", corsMiddleware(http.HandlerFunc(app.handleResume)))
	mux.Handle("/api/stop", corsMiddleware(http.HandlerFunc(app.handleStop)))
	mux.Handle("/api/state", corsMiddleware(http.HandlerFunc(app.handleState)))
	mux.Handle("/api/events", corsMiddleware(http.HandlerFunc(app.handleEvents)))
	mux.Handle("/api/dorks", corsMiddleware(http.HandlerFunc(app.handleDorks)))
	mux.Handle("/", http.FileServer(http.FS(content)))

	ln, err := net.Listen("tcp", "127.0.0.1:9477")
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
	}

	url := fmt.Sprintf("http://%s/index.html", ln.Addr().String())
	fmt.Println("Letter Recon →", url)
	_ = openBrowser(url)

	srv := &http.Server{Handler: mux}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		return srv.Close()
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
