package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/justinas/alice"
	"github.com/n9v9/workout-tracker/server/repository"
	"github.com/n9v9/workout-tracker/server/repository/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// Strongly typed URL parameter names.
// So we don't need string replace when changing a parameter name.
const (
	paramWorkoutID  = "workout_id"
	paramSetID      = "set_id"
	paramExerciseID = "exercise_id"
)

type API struct {
	staticFilesDir string
	router         chi.Router
	db             *sqlite.DB
	workouts       repository.WorkoutRepository
	exercises      repository.ExerciseRepository
	sets           repository.SetRepository
	stats          repository.StatisticsRepository
}

func New(staticFilesDir string, db *sqlite.DB) *API {
	api := &API{
		staticFilesDir: staticFilesDir,
		router:         chi.NewRouter(),
		workouts:       repository.NewWorkoutRepository(db.DB),
		exercises:      repository.NewExerciseRepository(db.DB),
		sets:           repository.NewSetRepository(db.DB),
		stats:          repository.NewStatisticsRepository(db.DB),
		db:             db,
	}
	api.routes()
	return api
}

// Run runs the HTTP server listening on the given address.
//
// Upon cancellation of ctx, the server will be shutdown and the method will return.
func (a *API) Run(ctx context.Context, addr string) {
	done := make(chan struct{})

	server := http.Server{
		Addr:    addr,
		Handler: a.router,
	}

	go func() {
		defer close(done)
		<-ctx.Done()
		server.Shutdown(context.TODO())
	}()

	log.Info().Str("addr", addr).Msg("Serving REST API on given address.")

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Err(err).Msg("Failed running HTTP Server ListenAndServe.")
	}

	if err := a.db.Close(); err != nil {
		log.Err(err).Msg("Failed to close database connection.")
	}
}

func (a *API) routes() {
	// Setup logging middleware.
	logging := alice.New(
		hlog.NewHandler(log.Logger),
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			//
			// This function will be called after the request has been served.
			//
			logParams := zerolog.Dict()
			urlParams := chi.RouteContext(r.Context()).URLParams

			for i, key := range urlParams.Keys {
				// Ignore the asterisk which stands for the complete route.
				if key == "*" {
					continue
				}
				value := urlParams.Values[i]
				logParams.Str(key, value)
			}

			hlog.FromRequest(r).Info().
				Int("size", size).
				Int("status", status).
				Dur("duration", duration).
				Dict("url_params", logParams).
				Send()
		}),
		hlog.MethodHandler("method"),
		hlog.URLHandler("url"),
		hlog.RemoteAddrHandler("ip"),
		hlog.RequestIDHandler("request_id", ""),
	)

	a.router.Use(func(h http.Handler) http.Handler {
		return logging.Then(h)
	})

	//
	// Static files handlers
	//
	a.router.Get("/", a.handleIndex())
	a.router.Get("/index.html", a.handleIndex())
	a.router.Get("/assets/*", a.handleAssets())
	// This makes SPA routing requests work, otherwise 404 would be returned.
	a.router.Get("/*", a.handleIndex())

	//
	// API handlers
	//
	api := chi.NewRouter()
	a.router.Mount("/api", api)

	//
	// Exercises
	//
	api.Get("/exercises", a.handleGetExercises)
	api.Post("/exercises", a.handleCreateExercise)
	api.Post("/exercises/exists", a.handleExistsExercise)

	api.Group(func(r chi.Router) {
		r.Use(a.exerciseMustExist(paramExerciseID))

		r.Put(fmt.Sprintf("/exercises/{%s}", paramExerciseID), a.handleUpdateExercise)
		r.Delete(fmt.Sprintf("/exercises/{%s}", paramExerciseID), a.handleDeleteExercise)
		r.Get(fmt.Sprintf("/exercises/{%s}/count", paramExerciseID), a.handleGetExerciseCountInSets)
	})

	//
	// Workouts
	//
	api.Get("/workouts", a.handleGetWorkoutList)
	api.Post("/workouts", a.handleCreateWorkout)

	api.Group(func(r chi.Router) {
		r.Use(a.workoutMustExist(paramWorkoutID))

		r.Delete(fmt.Sprintf("/workouts/{%s}", paramWorkoutID), a.handleDeleteWorkout)
		r.Get(
			fmt.Sprintf("/workouts/{%s}/sets/recommendation", paramWorkoutID),
			a.handleNewSetRecommendation,
		)

		r.Get(fmt.Sprintf("/workouts/{%s}/sets", paramWorkoutID), a.handleGetSetsByWorkoutID)
		r.Post(fmt.Sprintf("/workouts/{%s}/sets", paramWorkoutID), a.handleCreateSet)
	})

	//
	// Sets
	//
	api.Group(func(r chi.Router) {
		r.Use(a.setMustExist(paramSetID))

		r.Get(fmt.Sprintf("/sets/{%s}", paramSetID), a.handleGetSetByID)
		r.Put(fmt.Sprintf("/sets/{%s}", paramSetID), a.handleUpdateSet)
		r.Delete(fmt.Sprintf("/sets/{%s}", paramSetID), a.handleDeleteSet)
	})

	api.Get("/statistics", a.handleStatistics)
}

// exerciseMustExist checks that the requested URL has a URL parameter with the given name,
// extracts it and checks if an exercise with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the exercise does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *API) exerciseMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt64(w, r, parameter)
			if !ok {
				return
			}

			exists, err := a.exercises.ExistsID(r.Context(), id)
			if err != nil {
				hlog.FromRequest(r).Err(err).Msg("Failed to check if exercise with given ID exists.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !exists {
				hlog.FromRequest(r).Warn().Msg("Invalid request for exercise with non existing ID.")
				http.Error(w, "exercise does not exist", http.StatusNotFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// workoutMustExist checks that the requested URL has a URL parameter with the given name,
// extracts it and checks if a workout with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the workout does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *API) workoutMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt64(w, r, parameter)
			if !ok {
				return
			}

			exists, err := a.workouts.Exists(r.Context(), id)
			if err != nil {
				hlog.FromRequest(r).Err(err).Msg("Failed to check if workout with given ID exists.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !exists {
				hlog.FromRequest(r).Warn().Msg("Invalid request for workout with non existing ID.")
				http.Error(w, "workout id does not exist", http.StatusNotFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// setMustExist checks that the requested URL has a URL parameter with the given name,
// extracts it and checks if a set with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the set does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *API) setMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt64(w, r, parameter)
			if !ok {
				return
			}

			_, err := a.sets.FindByID(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
				hlog.FromRequest(r).Warn().Msg("Invalid request for set with non existing ID.")
				http.Error(w, "set does not exist", http.StatusNotFound)
				return
			}
			if err != nil {
				hlog.FromRequest(r).Err(err).Msg("Failed to check if set with given ID exists.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
