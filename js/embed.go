package js

import (
	"embed"
	"net/http"
)

//go:embed *.js
var files embed.FS

// Returns an http.Handler to serve the embedded JS
func FileServer() http.Handler {
	return http.FileServer(http.FS(files))
}
