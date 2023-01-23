package main

import (
	"bytes"
	"context"
	"database/sql"
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
	"github.com/jmoiron/sqlx"
	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"
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
// functions in [zerolog/log].
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
		log.Err(err).Msg("HTTP Server ListenAndServe")
	}
}

func (a *application) routes() {
	logging := alice.New(
		hlog.NewHandler(log.Logger),
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			event := hlog.FromRequest(r).Info().
				Int("size", size).
				Int("status", status).
				Dur("duration", duration)

			logParams := zerolog.Dict()
			urlParams := chi.RouteContext(r.Context()).URLParams

			for i, key := range urlParams.Keys {
				// XXX: I don't know why go-chi adds this parameter when using sub routers.
				//      Just filter this param out for now as it's value is always "".
				if key == "*" {
					continue
				}
				value := urlParams.Values[i]
				logParams.Str(key, value)
			}

			event.Dict("url_params", logParams)
			event.Send()
		}),
		hlog.MethodHandler("method"),
		hlog.URLHandler("url"),
		hlog.RemoteAddrHandler("ip"),
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

	api.Get("/exercises", a.handleGetExercises)
	api.Post("/exercises", a.handleCreateExercise)
	api.Delete("/exercises/{id}", a.handleDeleteExercise)
	api.Get("/exercises/{id}/count", a.handleGetExerciseCountInSets)
	api.Post("/exercises/exists", a.handleExistsExercise)

	api.Get("/workouts", a.handleGetWorkoutList)
	api.Post("/workouts", a.handleCreateWorkout)
	api.Delete("/workouts/{workoutID}", a.handleDeleteWorkout)

	api.Get("/workouts/{workoutID}/sets", a.handleGetSetsByWorkoutId)
	api.Get("/workouts/{workoutID}/sets/{setID}", a.handleGetSetById)
	api.Post("/workouts/{workoutID}/sets", a.handleCreateSet)
	api.Put("/workouts/{workoutID}/sets/{setID}", a.handleUpdateSet)
	api.Delete("/workouts/{workoutID}/sets/{setID}", a.handleDeleteSet)

	api.Get("/workouts/{workoutID}/sets/recommendation", a.handleNewSetRecommendation)

	api.Get("/statistics", a.handleStatistics)
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
	id, ok := paramInt(w, r, "id")
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
		l.Warn().Msg("Invalid request tries to delete exercise that does not exist.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	id, ok := paramInt(w, r, "id")
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
		ID int `json:"id"`
	}

	writeJSON(w, r, response{
		ID: int(id),
	})
}

func (a *application) handleDeleteWorkout(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	if err := a.db.deleteWorkout(r.Context(), workoutID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn().Msg("Request to delete workout with non existent ID.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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
	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	sets, err := a.db.setsForWorkout(r.Context(), workoutID)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get sets for workout ID.")
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
	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	setID, ok := paramInt(w, r, "setID")
	if !ok {
		return
	}

	set, err := a.db.setByIds(r.Context(), workoutID, setID)
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get set by workout ID and set ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, r, setResponse(set))
}

func (a *application) handleNewSetRecommendation(w http.ResponseWriter, r *http.Request) {
	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	result, err := a.db.newSetRecommendation(r.Context(), workoutID)
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

	workoutID, ok := paramInt(w, r, "workoutID")
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
		r.Context(), workoutID, b.ExerciseID, b.Repetitions, b.Weight,
	); err != nil {
		l.Err(err).Msg("Failed to create new set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
	l := hlog.FromRequest(r)

	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	type body struct {
		SetID       int `json:"setId"`
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	var b body

	if !readJSON(w, r, &b) {
		return
	}

	if err := a.db.updateSet(
		r.Context(), workoutID, b.SetID, b.ExerciseID, b.Repetitions, b.Weight,
	); err != nil {
		l.Err(err).Msg("Failed to update existing set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	workoutID, ok := paramInt(w, r, "workoutID")
	if !ok {
		return
	}

	setID, ok := paramInt(w, r, "setID")
	if !ok {
		return
	}

	if err := a.db.deleteSet(r.Context(), workoutID, setID); err != nil {
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
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		log.Err(err).
			Str("path", path).
			Msg("Failed to open sqlite database.")

		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		log.Err(err).
			Str("path", path).
			Msg("Failed to test connection to database")

		os.Exit(1)
	}

	return &database{db}
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
		SELECT
			id,
			UNIXEPOCH(start_date_utc) AS start_seconds_unix_epoch
		FROM
			workout
		ORDER BY
			start_date_utc DESC
	`

	var result []workoutRow

	if err := d.db.SelectContext(ctx, &result, query); err != nil {
		return nil, err
	}

	return result, nil
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
func (d *database) deleteWorkout(ctx context.Context, workoutID int) error {
	const query = "DELETE FROM workout WHERE id = ?"

	result, err := d.db.ExecContext(ctx, query, workoutID)
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

// setsForWorkout returns all sets that belong to the workout with the given ID.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) setsForWorkout(ctx context.Context, workoutID int) ([]setRow, error) {
	const query = `
		SELECT
			es.id,
			es.exercise_id,
			e.name AS exercise_name,
			UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			es.repetitions,
			es.weight
		FROM
			exercise_set AS es
		JOIN
			exercise AS e ON es.exercise_id = e.id
		WHERE
			es.workout_id = ?
		ORDER BY
			es.date_utc DESC
	`

	var sets []setRow

	if err := d.db.SelectContext(ctx, &sets, query, workoutID); err != nil {
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
		SELECT
			id,
			name
		FROM
			exercise
		ORDER BY
			name ASC
	`

	var exercises []exerciseRow

	if err := d.db.SelectContext(ctx, &exercises, query); err != nil {
		return nil, err
	}

	return exercises, nil
}

// setByIds returns the set that belongs to the given IDS.
//
// # Errors
//
// Returns either [database/sql.ErrNoRows] or another, underlying SQL error.
func (d *database) setByIds(ctx context.Context, workoutID, setID int) (setRow, error) {
	const query = `
		SELECT
			es.id,
			es.exercise_id,
			e.name AS exercise_name,
			UNIXEPOCH(es.date_utc) AS done_seconds_unix_epoch,
			es.repetitions,
			es.weight
		FROM
			exercise_set AS es
		JOIN
			exercise AS e ON es.exercise_id = e.id
		WHERE
			es.workout_id = ? AND
			es.id = ?
		ORDER BY
			es.date_utc DESC
	`

	var set setRow

	if err := d.db.GetContext(ctx, &set, query, workoutID, setID); err != nil {
		return setRow{}, err
	}

	return set, nil
}

// deleteSet tries to delete a set with the given IDs.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) deleteSet(ctx context.Context, workoutID, setID int) error {
	const query = `
		DELETE
		FROM
			exercise_set
		WHERE
			workout_id = ? AND
			id = ?
	`

	if _, err := d.db.ExecContext(ctx, query, workoutID, setID); err != nil {
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
		INSERT INTO exercise_set (
			exercise_id,
			workout_id,
			date_utc,
			repetitions,
			weight
		)
		VALUES (
			?,
			?,
			DATETIME('now'),
			?,
			?
		)
	`

	if _, err := d.db.ExecContext(
		ctx, query, exerciseID, workoutID, repetitions, weight,
	); err != nil {
		return err
	}

	return nil
}

// updateSet tries to update the set with the given IDs.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) updateSet(
	ctx context.Context,
	workoutID,
	setID,
	exerciseID,
	repetitions,
	weight int,
) error {
	const query = `
		UPDATE
			exercise_set
		SET
			exercise_id = ?,
			repetitions = ?,
			weight = ?
		WHERE
			id = ? AND
			workout_id = ?
	`

	if _, err := d.db.ExecContext(
		ctx, query, exerciseID, repetitions, weight, setID, workoutID,
	); err != nil {
		return err
	}

	return nil
}

type setRecommendationRow struct {
	ExerciseID  int `db:"exercise_id"`
	Repetitions int `db:"repetitions"`
	Weight      int `db:"weight"`
}

// newSetRecommendation returns recommended values for a new set.
//
// # Errors
//
// Returns an underlying SQL error.
func (d *database) newSetRecommendation(
	ctx context.Context,
	workoutID int,
) (setRecommendationRow, error) {
	// Very simple recommendation, just recommend the last set.
	const lastSetQuery = `
		SELECT
			exercise_id,
			repetitions,
			weight
		FROM
			exercise_set
		WHERE
			workout_id = ?
		ORDER BY
			date_utc DESC
		LIMIT 1
	`

	var recommendation setRecommendationRow

	err := d.db.GetContext(ctx, &recommendation, lastSetQuery, workoutID)
	if err == nil {
		return recommendation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return setRecommendationRow{}, err
	}

	// Suggest the first set of the last workout that has sets.
	const firstSetQuery = `
		SELECT
			exercise_id,
			repetitions,
			weight
		FROM
			exercise_set
		WHERE
			workout_id = (
				SELECT
					MAX(w.id)
				FROM
					workout w JOIN exercise_set es on w.id = es.workout_id
			)
		ORDER BY
			date_utc ASC
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
		SELECT
			UNIXEPOCH(w.start_date_utc) AS start_date_utc,
			UNIXEPOCH(MAX(es.date_utc)) AS end_date_utc
		FROM
			exercise_set es
		JOIN
			workout w on es.workout_id = w.id
		GROUP BY
			w.id;
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
		SELECT
			COUNT(id) AS total_sets,
			SUM(repetitions) AS total_reps,
			SUM(repetitions) / COUNT(id) AS avg_reps_per_set
		FROM
			exercise_set;
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
	const query = "INSERT INTO exercise (name) VALUES (?)"

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
	const query = "SELECT 1 FROM exercise WHERE lower(name) = lower(?)"

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
	const query = "SELECT 1 FROM exercise WHERE id = ?"

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
// If the exercise is used in any sets, [exerciseExistsErr] will be returned.
//
// # Errors
//
// Returns [exerciseExistsErr] if the exercise exists, or an underlying SQL error.
func (d *database) deleteExercise(ctx context.Context, id int) error {
	const checkQuery = `
		SELECT
			COUNT(*)
		FROM
			exercise e
		JOIN
			exercise_set es ON e.id = es.exercise_id
		WHERE
			e.id = ?;
	`

	var count int64
	err := d.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errExerciseExists
	}

	const deleteQuery = "DELETE FROM exercise WHERE id = ?"
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
		SELECT
			COUNT(*)
		FROM
			exercise e
		JOIN
			exercise_set es ON e.id = es.exercise_id
		WHERE
			e.id = ?;
	`

	var count int64

	err := d.db.GetContext(ctx, &count, checkQuery, id)
	if err != nil {
		return 0, err
	}

	return count, nil
}
