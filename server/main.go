package main

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/justinas/alice"
	"github.com/n9v9/workout-tracker/server/repository"
	"github.com/n9v9/workout-tracker/server/repository/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Strongly typed URL parameter names.
// So we don't need string replace when changing a parameter name.
const (
	paramWorkoutID  = "workout_id"
	paramSetID      = "set_id"
	paramExerciseID = "exercise_id"
)

func main() {
	setupGlobalLogger()

	if err := setupCLI().RunContext(setupContext(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// setupGlobalLogger sets up the global logger for the application.
//
// After this function is called, logging can be done by using the package
// functions in [github.com/rs/zerolog/log].
func setupGlobalLogger() {
	out := zerolog.ConsoleWriter{Out: os.Stderr}
	logger := zerolog.New(out).With().Timestamp().Logger()
	log.Logger = logger
}

// setupContext provides the [context.Context] for the application and registers
// interrupt handler to signal cancellation.
func setupContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)

		<-ch
		log.Info().Msg("Shutdown requested.")
		cancel()
	}()

	return ctx
}

// setupCLI sets up the command line interface to parse flags when
// starting the application.
func setupCLI() *cli.App {
	return &cli.App{
		Name:            "server",
		Usage:           "Server binary for the `workout-tracker` application",
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Value: "127.0.0.1:8080",
				Usage: "address and port to listen on",
			},
			&cli.StringFlag{
				Name:     "static-files",
				Required: true,
				Usage:    "Path to the static files to serve",
			},
			&cli.StringFlag{
				Name:     "db",
				Required: true,
				Usage:    "Path to the sqlite database",
			},
		},
		Action: func(ctx *cli.Context) error {
			staticFiles := ctx.String("static-files")
			db := ctx.String("db")

			app, err := newApplication(staticFiles, db)
			if err != nil {
				log.Err(err).Str("static_files", staticFiles).Str("db", db).Send()
				os.Exit(1)
			}

			app.run(ctx.Context, ctx.String("addr"))

			return nil
		},
	}
}

type application struct {
	staticFilesDir string
	router         chi.Router
	db             *sqlite.DB
	workouts       repository.WorkoutRepository
	exercises      repository.ExerciseRepository
	sets           repository.SetRepository
	stats          repository.StatisticsRepository
}

// newApplication initializes all dependencies, runs database migrations,
// registers routes, and returns an initialized application.
func newApplication(staticFilesDir, dbFile string) (*application, error) {
	db, err := sqlite.NewDB(dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	if err := db.RunMigrations(migrations); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	app := &application{
		staticFilesDir: staticFilesDir,
		router:         chi.NewRouter(),
		workouts:       repository.NewWorkoutRepository(db.DB),
		exercises:      repository.NewExerciseRepository(db.DB),
		sets:           repository.NewSetRepository(db.DB),
		stats:          repository.NewStatisticsRepository(db.DB),
		db:             db,
	}
	app.routes()

	return app, nil
}

// run runs the HTTP server listening on the given address.
//
// Upon cancellation of ctx, the server will be shutdown and the method will return.
func (a *application) run(ctx context.Context, addr string) {
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

func (a *application) routes() {
	// Setup logging middleware.
	logging := alice.New(
		hlog.NewHandler(log.Logger),
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			// This function will be called after the request has been served.
			hlog.FromRequest(r).Info().
				Int("size", size).
				Int("status", status).
				Dur("duration", duration).
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

	// Add URL parameters to the logging context. That way, URL parameters are als logged
	// when emitting logs from the handlers.
	//
	// Because of the way routing in chi works, this function cannot be a middleware,
	// but instead has to be attached to a router with the `With` method. Otherwise, the URL params
	// would only be visible after the handler has executed.
	logURLParams := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			hlog.FromRequest(r).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Dict("url_params", logParams)
			})

			h.ServeHTTP(w, r)
		})
	}

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
	api := chi.NewRouter().With(logURLParams)
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

		r.Get(fmt.Sprintf("/workouts/{%s}/sets", paramWorkoutID), a.handleGetSetsByWorkoutId)
		r.Post(fmt.Sprintf("/workouts/{%s}/sets", paramWorkoutID), a.handleCreateSet)
	})

	//
	// Sets
	//
	api.Group(func(r chi.Router) {
		r.Use(a.setMustExist(paramSetID))

		r.Get(fmt.Sprintf("/sets/{%s}", paramSetID), a.handleGetSetById)
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
func (a *application) exerciseMustExist(parameter string) func(http.Handler) http.Handler {
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
				w.WriteHeader(http.StatusBadRequest)
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
func (a *application) workoutMustExist(parameter string) func(http.Handler) http.Handler {
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
				w.WriteHeader(http.StatusBadRequest)
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
func (a *application) setMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt64(w, r, parameter)
			if !ok {
				return
			}

			_, err := a.sets.ByID(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
				hlog.FromRequest(r).Warn().Msg("Invalid request for set with non existing ID.")
				w.WriteHeader(http.StatusBadRequest)
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

func (a *application) handleIndex() http.HandlerFunc {
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

func (a *application) handleAssets() http.HandlerFunc {
	server := http.FileServer(http.Dir(a.staticFilesDir))

	return func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	}
}

func (a *application) handleGetExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := a.exercises.All(r.Context())
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

func (a *application) handleCreateExercise(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusBadRequest)
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

func (a *application) handleExistsExercise(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleDeleteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramExerciseID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	if err := a.exercises.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrExerciseExists) {
			l.Warn().Err(err).Msg("Invalid request tries to delete exercise that is used in sets.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		l.Err(err).Msg("Failed to delete exercise with given ID.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (a *application) handleGetExerciseCountInSets(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusBadRequest)
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

func (a *application) handleUpdateExercise(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleGetWorkoutList(w http.ResponseWriter, r *http.Request) {
	workouts, err := a.workouts.All(r.Context())
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

func (a *application) handleCreateWorkout(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleDeleteWorkout(w http.ResponseWriter, r *http.Request) {
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

type setResponse struct {
	ID                   int64   `json:"id"`
	ExerciseID           int64   `json:"exerciseId"`
	ExerciseName         string  `json:"exerciseName"`
	DoneSecondsUnixEpoch int     `json:"doneSecondsUnixEpoch"`
	Repetitions          int     `json:"repetitions"`
	Weight               int     `json:"weight"`
	Note                 *string `json:"note"`
}

func (a *application) handleGetSetsByWorkoutId(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramWorkoutID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	sets, err := a.sets.ByWorkoutID(r.Context(), id)
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

func (a *application) handleGetSetById(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt64(w, r, paramSetID)
	if !ok {
		return
	}

	set, err := a.sets.ByID(r.Context(), id)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get set by ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, setResponse(set))
}

func (a *application) handleNewSetRecommendation(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleCreateSet(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
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

func (a *application) handleStatistics(w http.ResponseWriter, r *http.Request) {
	statistics, err := a.stats.Overview(r.Context())
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get statistics.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		TotalWorkouts        int64 `json:"totalWorkouts"`
		TotalDurationSeconds int64 `json:"totalDurationSeconds"`
		AvgDurationSeconds   int64 `json:"avgDurationSeconds"`
		TotalSets            int64 `json:"totalSets"`
		TotalReps            int64 `json:"totalReps"`
		AvgRepsPerSet        int64 `json:"avgRepsPerSet"`
	}

	resp := response{
		TotalWorkouts:        statistics.TotalWorkouts,
		TotalDurationSeconds: int64(statistics.TotalDuration.Seconds()),
		AvgDurationSeconds:   int64(statistics.AvgDuration.Seconds()),
		TotalSets:            statistics.TotalSets,
		TotalReps:            statistics.TotalReps,
		AvgRepsPerSet:        statistics.AvgRepsPerSet,
	}

	writeJSON(w, r, resp)
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

		w.WriteHeader(http.StatusBadRequest)
		return 0, false
	}
	return v, true
}

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
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}
