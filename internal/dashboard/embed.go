package dashboard

import "embed"

//go:generate sh -c "rm -rf static && cp -r ../../web static"

//go:embed static
var staticFS embed.FS
