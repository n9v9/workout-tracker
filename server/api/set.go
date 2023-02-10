package api

import (
	"net/http"

	"github.com/n9v9/workout-tracker/server/repository"
	"github.com/rs/zerolog/hlog"
)

type setResponse struct {
	ID                   int64   `json:"id"`
	ExerciseID           int64   `json:"exerciseId"`
	ExerciseName         string  `json:"exerciseName"`
	DoneSecondsUnixEpoch int     `json:"doneSecondsUnixEpoch"`
	Repetitions          int     `json:"repetitions"`
	Weight               int     `json:"weight"`
	Note                 *string `json:"note"`
}

func (a *API) handleGetSetsByWorkoutID(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramWorkoutID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	sets, err := a.sets.FindByWorkoutID(r.Context(), id)
	if err != nil {
		l.Err(err).Msg("Failed to get sets for workout ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	results := make([]setResponse, 0, len(sets))

	for _, v := range sets {
		results = append(results, setResponse(v))
	}

	writeJSON(w, r, results)
}

func (a *API) handleGetSetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramSetID)
	if !ok {
		return
	}

	set, err := a.sets.FindByID(r.Context(), id)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get set by ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, setResponse(set))
}

func (a *API) handleNewSetRecommendation(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramWorkoutID)
	if !ok {
		return
	}

	result, err := a.workouts.RecommendNewSet(r.Context(), id)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get recommendation for new set.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	type response struct {
		ExerciseID  int64 `json:"exerciseId"`
		Repetitions int   `json:"repetitions"`
		Weight      int   `json:"weight"`
	}

	writeJSON(w, r, response(result))
}

func (a *API) handleCreateSet(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, ok := paramInt64(w, r, paramWorkoutID)
	if !ok {
		return
	}

	type body struct {
		ExerciseID  int64  `json:"exerciseId"`
		Repetitions int    `json:"repetitions"`
		Weight      int    `json:"weight"`
		Note        string `json:"note"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	err := a.sets.Create(r.Context(), repository.CreateSetEntity{
		WorkoutID:   id,
		ExerciseID:  b.ExerciseID,
		Repetitions: b.Repetitions,
		Weight:      b.Weight,
		Note:        b.Note,
	})
	if err != nil {
		l.Err(err).Msg("Failed to create new set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *API) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, ok := paramInt64(w, r, paramSetID)
	if !ok {
		return
	}

	type body struct {
		ExerciseID  int64  `json:"exerciseId"`
		Repetitions int    `json:"repetitions"`
		Weight      int    `json:"weight"`
		Note        string `json:"note"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	err := a.sets.Update(r.Context(), repository.UpdateSetEntity{
		ID:          id,
		ExerciseID:  b.ExerciseID,
		Repetitions: b.Repetitions,
		Weight:      b.Weight,
		Note:        b.Note,
	})
	if err != nil {
		l.Err(err).Msg("Failed to update existing set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *API) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramSetID)
	if !ok {
		return
	}

	if err := a.sets.Delete(r.Context(), id); err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to delete set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
