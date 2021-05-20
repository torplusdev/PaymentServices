package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var embeddedWebUI embed.FS

func Handler() (http.Handler, error) {
	webUI, err := fs.Sub(embeddedWebUI, "static")
	if err != nil {
		return nil, err

	}
	return http.FileServer(http.FS(webUI)), nil
}
