package api

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

func (a *API) handleIndex() http.HandlerFunc {
	file, err := os.ReadFile(filepath.Join(a.staticFilesDir, "index.html"))
	if err != nil {
		log.Err(err).Msg("Failed to read index.html file.")
		os.Exit(1)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, bytes.NewReader(file)); err != nil {
			hlog.FromRequest(r).Err(err).Msg("Failed to serve index.html file")
		}
	}
}

func (a *API) handleAssets() http.HandlerFunc {
	server := http.FileServer(http.Dir(a.staticFilesDir))

	return func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}
}
