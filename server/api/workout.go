package api

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
)

func (a *API) handleGetWorkoutList(w http.ResponseWriter, r *http.Request) {
	workouts, err := a.workouts.FindAll(r.Context())
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get workout list.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID                    uint64 `json:"id"`
		StartSecondsUnixEpoch uint64 `json:"startSecondsUnixEpoch"`
	}

	results := make([]response, 0, len(workouts))

	for _, v := range workouts {
		results = append(results, response(v))
	}

	writeJSON(w, r, results)
}

func (a *API) handleCreateWorkout(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, err := a.workouts.Create(r.Context())
	if err != nil {
		l.Err(err).Msg("Failed to create workout.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID int64 `json:"id"`
	}

	writeJSON(w, r, response{
		ID: id,
	})
}

func (a *API) handleDeleteWorkout(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, ok := paramInt64(w, r, paramWorkoutID)
	if !ok {
		return
	}

	if err := a.workouts.Delete(r.Context(), id); err != nil {
		l.Err(err).Msg("Failed to delete workout.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
