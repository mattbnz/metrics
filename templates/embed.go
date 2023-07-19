package templates

import (
	"embed"
	"text/template"
)

//go:embed *.tmpl
var files embed.FS

// Returns an http.Handler to serve the embedded JS
func Get(name string) (*template.Template, error) {
	return template.ParseFS(files, name)
}
