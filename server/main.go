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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"
)

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
			app := newApplication(ctx.String("static-files"), ctx.String("db"))
			app.run(ctx.Context, ctx.String("addr"))

			return nil
		},
	}
}

type application struct {
	staticFilesDir string
	router         chi.Router
	db             *database
}

// newApplication initializes all dependencies, registers routes and returns
// an initialized application.
//
// Any error that happens will be logged, and the application will exit.
func newApplication(staticFilesDir, db string) *application {
	app := &application{
		staticFilesDir: staticFilesDir,
		router:         chi.NewRouter(),
		db:             newDatabase(db),
	}
	app.routes()
	app.db.runMigrations()
	return app
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
	// but instead has to be attached to a router with the `With` method. Otherwise the URL params
	// woul only be visible after the handler has executed.
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

// exerciseMustExist checks that the requested URL has an URL parameter with the given name,
// extracts it and checks if an exercise with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the exercise does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *application) exerciseMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt(w, r, parameter)
			if !ok {
				return
			}

			exists, err := a.db.existsExerciseID(r.Context(), id)
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

// workoutMustExist checks that the requested URL has an URL parameter with the given name,
// extracts it and checks if a workout with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the workout does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *application) workoutMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt(w, r, parameter)
			if !ok {
				return
			}

			exists, err := a.db.workoutExists(r.Context(), id)
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

// setMustExist checks that the requested URL has an URL parameter with the given name,
// extracts it and checks if a set with the extracted ID exists. If it does, the wrapped
// handler will be called.
//
// If the parameter does not exist, can not be parsed, or the set does not exist, then
// [net/http.StatusBadRequest] will be set and the wrapped handler will not be called.
func (a *application) setMustExist(parameter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := paramInt(w, r, parameter)
			if !ok {
				return
			}

			_, err := a.db.setById(r.Context(), id)
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
	exercises, err := a.db.exercises(r.Context())
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

	exists, err := a.db.existsExerciseName(r.Context(), b.Name)
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

	exercise, err := a.db.createExercise(r.Context(), b.Name)
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

	exists, err := a.db.existsExerciseName(r.Context(), b.Name)
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
	id, ok := paramInt(w, r, paramExerciseID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	if err := a.db.deleteExercise(r.Context(), id); err != nil {
		if errors.Is(err, errExerciseExists) {
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
	id, ok := paramInt(w, r, paramExerciseID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	exists, err := a.db.existsExerciseID(r.Context(), id)
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

	count, err := a.db.exerciseCountInSets(r.Context(), id)
	if err != nil {
		l.Err(err).Msg("Failed to get count of exercise with given ID in sets.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	type response struct {
		Count int64 `json:"count"`
	}

	writeJSON(w, r, response{Count: count})
}

func (a *application) handleGetWorkoutList(w http.ResponseWriter, r *http.Request) {
	workouts, err := a.db.workoutList(r.Context())
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

	id, err := a.db.createWorkout(r.Context())
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

	id, ok := paramInt(w, r, paramWorkoutID)
	if !ok {
		return
	}

	if err := a.db.deleteWorkout(r.Context(), id); err != nil {
		l.Err(err).Msg("Failed to delete workout.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type setResponse struct {
	ID                   int    `json:"id"`
	ExerciseID           int    `json:"exerciseId"`
	ExerciseName         string `json:"exerciseName"`
	DoneSecondsUnixEpoch int    `json:"doneSecondsUnixEpoch"`
	Repetitions          int    `json:"repetitions"`
	Weight               int    `json:"weight"`
}

func (a *application) handleGetSetsByWorkoutId(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt(w, r, paramWorkoutID)
	if !ok {
		return
	}

	l := hlog.FromRequest(r)

	sets, err := a.db.setsByWorkoutID(r.Context(), id)
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
	id, ok := paramInt(w, r, paramSetID)
	if !ok {
		return
	}

	set, err := a.db.setById(r.Context(), id)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get set by ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, setResponse(set))
}

func (a *application) handleNewSetRecommendation(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt(w, r, paramWorkoutID)
	if !ok {
		return
	}

	result, err := a.db.setRecommendationByWorkoutID(r.Context(), id)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get recommendation for new set.")
		w.WriteHeader(http.StatusInternalServerError)
	}

	type response struct {
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	writeJSON(w, r, response(result))
}

func (a *application) handleCreateSet(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, ok := paramInt(w, r, paramWorkoutID)
	if !ok {
		return
	}

	type body struct {
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	if err := a.db.createSet(
		r.Context(), id, b.ExerciseID, b.Repetitions, b.Weight,
	); err != nil {
		l.Err(err).Msg("Failed to create new set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	id, ok := paramInt(w, r, paramSetID)
	if !ok {
		return
	}

	type body struct {
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	if err := a.db.updateSet(
		r.Context(), id, b.ExerciseID, b.Repetitions, b.Weight,
	); err != nil {
		l.Err(err).Msg("Failed to update existing set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	id, ok := paramInt(w, r, paramSetID)
	if !ok {
		return
	}

	if err := a.db.deleteSet(r.Context(), id); err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to delete set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *application) handleStatistics(w http.ResponseWriter, r *http.Request) {
	statistics, err := a.db.statistics(r.Context())
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
		TotalWorkouts:        statistics.totalWorkouts,
		TotalDurationSeconds: int64(statistics.totalDuration.Seconds()),
		AvgDurationSeconds:   int64(statistics.avgDuration.Seconds()),
		TotalSets:            statistics.totalSets,
		TotalReps:            statistics.totalReps,
		AvgRepsPerSet:        statistics.avgRepsPerSet,
	}

	writeJSON(w, r, resp)
}

// paramInt tries to parse the URL parameter with the given name as an integer.
// If parsing fails, http.StatusBadRequest will be set.
func paramInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	v, err := strconv.Atoi(chi.URLParam(r, name))
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
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		hlog.FromRequest(r).Warn().Err(err).Msg("Failed to decode JSON body.")
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

type database struct {
	db *sqlx.DB
}

// newDatabase opens the SQLite database at the given path and tries to connect to it.
//
// If the database could not be opened, or connecting to the database failed,
// the error will be logged and the application exits.
func newDatabase(path string) *database {
	args := []string{
		"_pragma=foreign_keys(1)", // Enable foreign key checking.
	}

	db, err := sqlx.Open("sqlite", path+"?"+strings.Join(args, "&"))
	if err != nil {
		log.Err(err).
			Str("path", path).
			Msg("Failed to open sqlite database.")

		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		log.Err(err).
			Str("path", path).
			Msg("Failed to test connection to database.")

		os.Exit(1)
	}

	return &database{db}
}

//go:embed migrations/*.sql
var migrations embed.FS

// runMigrations tries to run all remaining up migrations.
// If an error happens, the error will be logged and the application exits.
func (d *database) runMigrations() {
	log.Info().Msg("Running migrations.")
	start := time.Now()
	defer func() {
		log.Info().Dur("duration", time.Since(start)).Msg("Running migrations done.")
	}()

	driver, err := sqlite.WithInstance(d.db.DB, new(sqlite.Config))
	if err != nil {
		log.Err(err).Msg("Failed to create migration instance.")
		os.Exit(1)
	}

	files, err := iofs.New(migrations, "migrations")
	if err != nil {
		log.Err(err).Msg("Failed to create iofs source driver for migrations.")
		os.Exit(1)
	}

	m, err := migrate.NewWithInstance("iofs", files, "workout-tracker", driver)
	if err != nil {
		log.Err(err).Msg("Failed to create migration instance.")
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("All migrations are already applied.")
		} else {
			log.Err(err).Msg("Failed to run migrations.")
		}
	}
}

type workoutRow struct {
	ID                    uint64 `db:"id"`
	StartSecondsUnixEpoch uint64 `db:"start_seconds_unix_epoch"`
}

// workoutList returns all workouts.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) workoutList(ctx context.Context) ([]workoutRow, error) {
	const query = `
		SELECT id,
			   UNIXEPOCH(start_date_utc) AS start_seconds_unix_epoch
		  FROM workout
		 ORDER BY start_date_utc DESC
	`

	var result []workoutRow

	if err := d.db.SelectContext(ctx, &result, query); err != nil {
		return nil, err
	}

	return result, nil
}

// workoutExists checks whether a workout with the given ID exist.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) workoutExists(ctx context.Context, id int) (bool, error) {
	const query = `
		SELECT COUNT(id)
		  FROM workout
		 WHERE id = ?
	`

	var count int

	if err := d.db.GetContext(ctx, &count, query, id); err != nil {
		return false, err
	}

	return count == 1, nil
}

// createWorkout tries to create a new workout.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) createWorkout(ctx context.Context) (int64, error) {
	const query = `
		INSERT INTO workout (start_date_utc)
		VALUES (DATETIME('now'))
	`

	result, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// deleteWorkout tries to delete the workout with the given ID.
//
// # Errors
//
// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
func (d *database) deleteWorkout(ctx context.Context, id int) error {
	const query = `
		DELETE
		  FROM workout
		 WHERE id = ?
	`

	result, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

type setRow struct {
	ID                   int    `db:"id"`
	ExerciseID           int    `db:"exercise_id"`
	ExerciseName         string `db:"exercise_name"`
	DoneSecondsUnixEpoch int    `db:"done_seconds_unix_epoch"`
	Repetitions          int    `db:"repetitions"`
	Weight               int    `db:"weight"`
}

// setsByWorkoutID returns all sets that belong to the workout with the given ID.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) setsByWorkoutID(ctx context.Context, id int) ([]setRow, error) {
	const query = `
		SELECT es.id,
			   es.exercise_id,
			   e.name                 AS exercise_name,
			   UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			   es.repetitions,
			   es.weight
		  FROM exercise_set AS es
			   JOIN
			   exercise     AS e ON es.exercise_id = e.id
		 WHERE es.workout_id = ?
		 ORDER BY es.date_utc DESC
	`

	var sets []setRow

	if err := d.db.SelectContext(ctx, &sets, query, id); err != nil {
		return nil, err
	}

	return sets, nil
}

type exerciseRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// exercises returns all exercises.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) exercises(ctx context.Context) ([]exerciseRow, error) {
	const query = `
		SELECT id,
			   name
		  FROM exercise
		 ORDER BY name ASC
	`

	var exercises []exerciseRow

	if err := d.db.SelectContext(ctx, &exercises, query); err != nil {
		return nil, err
	}

	return exercises, nil
}

// setById returns the set that has the given ID.
//
// # Errors
//
// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
func (d *database) setById(ctx context.Context, id int) (setRow, error) {
	const query = `
		SELECT es.id,
			   es.exercise_id,
			   e.name                 AS exercise_name,
			   UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			   es.repetitions,
			   es.weight
		  FROM exercise_set AS es
			   JOIN
			   exercise     AS e ON es.exercise_id = e.id
		 WHERE es.id = ?
		 ORDER BY es.date_utc DESC
	`

	var set setRow

	if err := d.db.GetContext(ctx, &set, query, id); err != nil {
		return setRow{}, err
	}

	return set, nil
}

// deleteSet tries to delete a set with the given ID.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) deleteSet(ctx context.Context, id int) error {
	const query = `
		DELETE
		  FROM exercise_set
		 WHERE id = ?
	`

	if _, err := d.db.ExecContext(ctx, query, id); err != nil {
		return err
	}

	return nil
}

// createSet tries to create a set with the given values.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) createSet(
	ctx context.Context,
	workoutID,
	exerciseID,
	repetitions,
	weight int,
) error {
	const query = `
		INSERT INTO exercise_set (exercise_id,
								  workout_id,
								  date_utc,
								  repetitions,
								  weight)
		VALUES (?,
				?,
				DATETIME('now'),
				?,
				?)
	`

	if _, err := d.db.ExecContext(
		ctx, query, exerciseID, workoutID, repetitions, weight,
	); err != nil {
		return err
	}

	return nil
}

// updateSet tries to update the set with the given ID.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) updateSet(
	ctx context.Context,
	id,
	exerciseID,
	repetitions,
	weight int,
) error {
	const query = `
		UPDATE
			exercise_set
		   SET exercise_id = ?,
			   repetitions = ?,
			   weight      = ?
		 WHERE id = ?
	`

	if _, err := d.db.ExecContext(ctx, query, exerciseID, repetitions, weight, id); err != nil {
		return err
	}

	return nil
}

type setRecommendationRow struct {
	ExerciseID  int `db:"exercise_id"`
	Repetitions int `db:"repetitions"`
	Weight      int `db:"weight"`
}

// setRecommendationByWorkoutID returns recommended values for a new set.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) setRecommendationByWorkoutID(
	ctx context.Context,
	id int,
) (setRecommendationRow, error) {
	// Very simple recommendation, just recommend the last set.
	const lastSetQuery = `
		SELECT exercise_id,
			   repetitions,
			   weight
		  FROM exercise_set
		 WHERE workout_id = ?
		 ORDER BY date_utc DESC
		 LIMIT 1
	`

	var recommendation setRecommendationRow

	err := d.db.GetContext(ctx, &recommendation, lastSetQuery, id)
	if err == nil {
		return recommendation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return setRecommendationRow{}, err
	}

	// Suggest the first set of the last workout that has sets.
	const firstSetQuery = `
		SELECT exercise_id,
			   repetitions,
			   weight
		  FROM exercise_set
		 WHERE workout_id = (SELECT MAX(w.id)
							   FROM workout           w
									JOIN exercise_set es ON w.id = es.workout_id)
		 ORDER BY date_utc
		 LIMIT 1;
	`

	err = d.db.GetContext(ctx, &recommendation, firstSetQuery)
	if err == nil {
		return recommendation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return setRecommendationRow{}, err
	}

	// There are no workouts with sets, so we just set some defaults.
	recommendation.ExerciseID = -1
	recommendation.Repetitions = 0
	recommendation.Weight = 0

	return recommendation, nil
}

type statisticsRow struct {
	totalWorkouts int64
	totalDuration time.Duration
	avgDuration   time.Duration
	totalReps     int64
	totalSets     int64
	avgRepsPerSet int64
}

// statistics returns various statistics as described below.
//
// # statistics
//
//   - Total number of workouts.
//   - Total duration of workouts.
//   - Average duration of a workout.
//   - Total number of repetitions.
//   - Total number of sets.
//   - Average number of repetitions per set.
//
// # Errors
//
// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
func (d *database) statistics(ctx context.Context) (statisticsRow, error) {
	const datesQuery = `
		SELECT UNIXEPOCH(w.start_date_utc) AS start_date_utc,
			   UNIXEPOCH(MAX(es.date_utc)) AS end_date_utc
		  FROM exercise_set es
			   JOIN
			   workout      w ON es.workout_id = w.id
		 GROUP BY w.id
	`

	type datesRow struct {
		StartUTC int64 `db:"start_date_utc"`
		EndUTC   int64 `db:"end_date_utc"`
	}

	var workouts []datesRow

	if err := d.db.SelectContext(ctx, &workouts, datesQuery); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return statisticsRow{}, nil
		}
		return statisticsRow{}, err
	}

	result := statisticsRow{
		totalWorkouts: int64(len(workouts)),
	}

	for _, v := range workouts {
		result.totalDuration += time.Unix(v.EndUTC, 0).Sub(time.Unix(v.StartUTC, 0))
	}

	result.avgDuration = time.Duration(int64(result.totalDuration) / result.totalWorkouts)

	const setsRepsQuery = `
		SELECT COUNT(id)                    AS total_sets,
			   SUM(repetitions)             AS total_reps,
			   SUM(repetitions) / COUNT(id) AS avg_reps_per_set
		  FROM exercise_set;
	`

	type setsRepsRow struct {
		TotalSets     int64 `db:"total_sets"`
		TotalReps     int64 `db:"total_reps"`
		AvgRepsPerSet int64 `db:"avg_reps_per_set"`
	}

	var setsRepsResult setsRepsRow

	if err := d.db.GetContext(ctx, &setsRepsResult, setsRepsQuery); err != nil {
		return statisticsRow{}, err
	}

	result.totalSets = setsRepsResult.TotalSets
	result.totalReps = setsRepsResult.TotalReps
	result.avgRepsPerSet = setsRepsResult.AvgRepsPerSet

	return result, nil
}

// createExercise creates an exercise with the given name.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) createExercise(ctx context.Context, name string) (exerciseRow, error) {
	const query = `
		INSERT INTO exercise (name)
		VALUES (?)
	`

	result, err := d.db.ExecContext(ctx, query, name)
	if err != nil {
		return exerciseRow{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return exerciseRow{}, err
	}

	return exerciseRow{ID: id, Name: name}, nil
}

// existsExerciseName returns whether an exercise with the given name exists.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) existsExerciseName(ctx context.Context, name string) (bool, error) {
	const query = `
		SELECT 1
		  FROM exercise
		 WHERE LOWER(name) = LOWER(?)
	`

	// Don't care about this value, just care about the existence.
	var tmp string

	err := d.db.QueryRowxContext(ctx, query, name).Scan(&tmp)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

// existsExerciseID checks whether an exercise with the given id exists.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) existsExerciseID(ctx context.Context, id int) (bool, error) {
	const query = `
		SELECT 1
		  FROM exercise
		 WHERE id = ?
	`

	// Don't care about this value, just care about the existence.
	var tmp string

	err := d.db.QueryRowxContext(ctx, query, id).Scan(&tmp)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

var errExerciseExists = errors.New("exercise exists in at least one set")

// deleteExercise tries to delete the exercise with the given id.
// If the exercise is used in any sets, errExerciseExists will be returned.
//
// # Errors
//
// Returns errExerciseExists if the exercise exists, or an underlying SQL error.
func (d *database) deleteExercise(ctx context.Context, id int) error {
	const checkQuery = `
		SELECT COUNT(*)
		  FROM exercise     e
			   JOIN
			   exercise_set es ON e.id = es.exercise_id
		 WHERE e.id = ?;
	`

	var count int64
	err := d.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errExerciseExists
	}

	const deleteQuery = `
		DELETE
		  FROM exercise
		 WHERE id = ?
	`
	_, err = d.db.ExecContext(ctx, deleteQuery, id)
	return err
}

// exerciseCountInSets returns the number of times the exercise with
// the given id is used in sets.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) exerciseCountInSets(ctx context.Context, id int) (int64, error) {
	const checkQuery = `
		SELECT COUNT(*)
		  FROM exercise     e
			   JOIN
			   exercise_set es ON e.id = es.exercise_id
		 WHERE e.id = ?;
	`

	var count int64

	err := d.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return 0, err
	}

	return count, nil
}
