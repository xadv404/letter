package dashboard

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"github.com/xadv404/letter/internal/throttle"
)

// Run starts the embedded HTML dashboard in a native GUI window.
func Run() error {
	err := runDashboard()
	if err != nil {
		failStartup(err)
	}
	return err
}

func runDashboard() error {
	app := newApp()

	content, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("dashboard assets: %w (run: go generate ./internal/dashboard/...)", err)
	}
	if _, err := fs.Stat(content, "index.html"); err != nil {
		return fmt.Errorf("index.html manquant — exécutez: go generate ./internal/dashboard/...")
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
	mux.Handle("/", http.FileServer(http.FS(content)))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/index.html", ln.Addr().String())
	srv := &http.Server{Handler: mux}

	go func() { _ = srv.Serve(ln) }()

	if err := runGUI(url, func() {
		ctx, cancel := context.WithTimeout(context.Background(), throttle.GracefulShutdown)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}); err != nil {
		return err
	}

	return nil
}
