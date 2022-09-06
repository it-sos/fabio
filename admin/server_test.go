package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabiolb/fabio/config"
)

func TestAdminServerAccess(t *testing.T) {
	type test struct {
		uri  string
		code int
	}

	testAccess := func(access string, tests []test) {
		srv := &Server{
			Access: access,
			Cfg: &config.Config{
				Registry: config.Registry{
					Consul: config.Consul{
						KVPath: "/fabio/config",
					},
				},
			},
		}
		ts := httptest.NewServer(srv.handler())
		defer ts.Close()

		noRedirectClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		for _, tt := range tests {
			t.Run(access+tt.uri, func(t *testing.T) {
				resp, err := noRedirectClient.Get(ts.URL + tt.uri)
				if err != nil {
					t.Fatalf("got %v want nil", err)
				}
				if got, want := resp.StatusCode, tt.code; got != want {
					t.Fatalf("got code %d want %d", got, want)
				}
			})
		}
	}

	roTests := []test{
		{"/fabio/api/manual", 403},
		{"/fabio/api/paths", 403},
		{"/fabio/api/config", 200},
		{"/fabio/api/routes", 200},
		{"/fabio/api/version", 200},
		{"/fabio/manual", 403},
		{"/fabio/routes", 200},
		{"/health", 200},
		{"/fabio/assets/logo.svg", 200},
		{"/fabio/assets/logo.bw.svg", 200},
		{"/fabio/", 303},
	}

	rwTests := []test{
		{"/fabio/api/manual", 200},
		{"/fabio/api/paths", 200},
		{"/fabio/api/config", 200},
		{"/fabio/api/routes", 200},
		{"/fabio/api/version", 200},
		{"/fabio/manual", 200},
		{"/fabio/routes", 200},
		{"/health", 200},
		{"/fabio/assets/logo.svg", 200},
		{"/fabio/assets/logo.bw.svg", 200},
		{"/fabio/", 303},
	}

	testAccess("ro", roTests)
	testAccess("rw", rwTests)
}
