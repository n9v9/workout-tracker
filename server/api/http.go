package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
)

// writeJSON encodes data as JSON and writes it to w.
// If writing fails, http.StatusInternalServerError will be set.
func writeJSON(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		hlog.FromRequest(r).
			Err(err).
			Interface("data", data).
			Msg("Failed to send JSON response.")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// readJSON decodes the request body into data.
// If reading fails, http.StatusBadRequest will be set and false will be returned.
func readJSON(w http.ResponseWriter, r *http.Request, data any) bool {
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		hlog.FromRequest(r).Warn().Err(err).Msg("Failed to decode JSON body.")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return false
	}
	return true
}

// paramInt64 tries to parse the URL parameter with the given name as an integer.
// If parsing fails, http.StatusBadRequest will be set.
func paramInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		hlog.FromRequest(r).
			Warn().
			Err(err).
			Str("param_name", name).
			Msg("Failed to parse URL parameter.")

		http.Error(w, "invalid query parameter", http.StatusBadRequest)
		return 0, false
	}
	return v, true
}
