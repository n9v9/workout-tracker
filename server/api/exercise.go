package api

import (
	"errors"
	"net/http"

	"github.com/n9v9/workout-tracker/server/repository"
	"github.com/rs/zerolog/hlog"
)

func (a *API) handleGetExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := a.exercises.FindAll(r.Context())
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get exercises.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	results := make([]response, 0, len(exercises))

	for _, v := range exercises {
		results = append(results, response(v))
	}

	writeJSON(w, r, results)
}

func (a *API) handleCreateExercise(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	type body struct {
		Name string `json:"name"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	exists, err := a.exercises.ExistsName(r.Context(), b.Name)
	if err != nil {
		l.Err(err).Msg("Failed to check if exercise exists.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if exists {
		l.Warn().Msg("Invalid request tries to create existing exercise.")
		http.Error(w, "exercise already exists", http.StatusConflict)
		return
	}

	exercise, err := a.exercises.Create(r.Context(), b.Name)
	if err != nil {
		l.Err(err).Msg("Failed to create new exercise.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	writeJSON(w, r, response(exercise))
}

func (a *API) handleExistsExercise(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Name string `json:"name"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	exists, err := a.exercises.ExistsName(r.Context(), b.Name)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to query if exercise exists.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		Exists bool `json:"exists"`
	}

	writeJSON(w, r, response{Exists: exists})
}

func (a *API) handleDeleteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramExerciseID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	if err := a.exercises.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrExerciseExists) {
			l.Warn().Err(err).Msg("Invalid request tries to delete exercise that is used in sets.")
			http.Error(w, "exercise is used in sets", http.StatusConflict)
			return
		}
		l.Err(err).Msg("Failed to delete exercise with given ID.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) handleGetExerciseCountInSets(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramExerciseID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	exists, err := a.exercises.ExistsID(r.Context(), id)
	if err != nil {
		l.Err(err).Msg("Failed to check if exercise with given ID exists.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		l.Warn().Msg("Invalid request tries to get count in sets for exercise that does not exist.")
		http.Error(w, "exercise does not exist", http.StatusNotFound)
		return
	}

	count, err := a.exercises.UsageInSets(r.Context(), id)
	if err != nil {
		l.Err(err).Msg("Failed to get count of exercise with given ID in sets.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	type response struct {
		Count int64 `json:"count"`
	}

	writeJSON(w, r, response{Count: count})
}

func (a *API) handleUpdateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramExerciseID)
	if !ok {
		return
	}

	type body struct {
		Name string `json:"name"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	exercise, err := a.exercises.Update(r.Context(), id, b.Name)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to update exercise.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	writeJSON(w, r, response(exercise))
}
