package admin

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/fabiolb/fabio/admin/api"
	"github.com/fabiolb/fabio/admin/ui"
	"github.com/fabiolb/fabio/config"
	"github.com/fabiolb/fabio/proxy"
)

// Server provides the HTTP server for the admin UI and API.
type Server struct {
	Access   string
	Color    string
	Title    string
	Version  string
	Commands string
	Cfg      *config.Config
}

// ListenAndServe starts the admin server.
func (s *Server) ListenAndServe(l config.Listen, tlscfg *tls.Config) error {
	return proxy.ListenAndServeHTTP(l, s.handler(), tlscfg)
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()

	switch s.Access {
	case "ro":
		mux.HandleFunc("/fabio/api/paths", forbidden)
		mux.HandleFunc("/fabio/api/manual", forbidden)
		mux.HandleFunc("/fabio/api/manual/", forbidden)
		mux.HandleFunc("/fabio/manual", forbidden)
		mux.HandleFunc("/fabio/manual/", forbidden)
	case "rw":
		// for historical reasons the configured config path starts with a '/'
		// but Consul treats all KV paths without a leading slash.
		pathsPrefix := strings.TrimPrefix(s.Cfg.Registry.Consul.KVPath, "/")
		mux.Handle("/fabio/api/paths", &api.ManualPathsHandler{Prefix: pathsPrefix})
		mux.Handle("/fabio/api/manual", &api.ManualHandler{BasePath: "/api/manual"})
		mux.Handle("/fabio/api/manual/", &api.ManualHandler{BasePath: "/api/manual"})
		mux.Handle("/fabio/manual", &ui.ManualHandler{
			BasePath: "/fabio/manual",
			Color:    s.Color,
			Title:    s.Title,
			Version:  s.Version,
			Commands: s.Commands,
		})
		mux.Handle("/fabio/manual/", &ui.ManualHandler{
			BasePath: "/fabio/manual",
			Color:    s.Color,
			Title:    s.Title,
			Version:  s.Version,
			Commands: s.Commands,
		})
	}

	mux.Handle("/fabio/api/config", &api.ConfigHandler{Config: s.Cfg})
	mux.Handle("/fabio/api/routes", &api.RoutesHandler{})
	mux.Handle("/fabio/api/version", &api.VersionHandler{Version: s.Version})
	mux.Handle("/fabio/routes", &ui.RoutesHandler{Color: s.Color, Title: s.Title, Version: s.Version})
	mux.HandleFunc("/health", handleHealth)

	mux.Handle("/fabio/assets/", AssetHandler("/fabio/assets/", ui.Static, "./assets"))
	mux.HandleFunc("/fabio/favicon.ico", http.NotFound)

	mux.Handle("/fabio/", http.RedirectHandler("/fabio/routes", http.StatusSeeOther))
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler returns an http.Handler that will serve files from
// the Assets embed.FS. When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem.
func AssetHandler(prefix string, assets embed.FS, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		// If we can't find the asset, fs can handle the error
		file, err := assets.Open(assetPath)
		if err != nil {
			return nil, err
		}

		// Otherwise assume this is a legitimate request routed correctly
		return file, err
	})

	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}
