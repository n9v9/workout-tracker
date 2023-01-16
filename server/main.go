package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

func setupGlobalLogger() {
	out := zerolog.ConsoleWriter{Out: os.Stderr}
	logger := zerolog.New(out).With().Timestamp().Logger()
	log.Logger = logger
}

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
	db             *Database
}

func newApplication(staticFilesDir string, db string) *application {
	app := &application{
		staticFilesDir: staticFilesDir,
		router:         chi.NewRouter(),
		db:             newDatabase(db),
	}
	app.routes()
	return app
}

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

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Err(err).Msg("HTTP Server ListenAndServe")
	}
}

func (a *application) routes() {
	logging := alice.New(
		hlog.NewHandler(log.Logger),
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Int("size", size).
				Int("status", status).
				Dur("duration", duration).
				Msg("")
		}),
		hlog.MethodHandler("method"),
		hlog.URLHandler("url"),
		hlog.RemoteAddrHandler("ip"),
	)

	a.router.Use(func(h http.Handler) http.Handler {
		return logging.Then(h)
	})

	a.router.Get("/api/exercises", a.handleGetExercises)

	a.router.Get("/api/workouts", a.handleGetWorkoutList)
	a.router.Post("/api/workouts", a.handleCreateWorkout)
	a.router.Delete("/api/workouts/{workoutID}", a.handleDeleteWorkout)

	a.router.Get("/api/workouts/{workoutID}/sets", a.handleGetSetsByWorkoutId)
	a.router.Get("/api/workouts/{workoutID}/sets/{setID}", a.handleGetSetById)
	a.router.Post("/api/workouts/{workoutID}/sets", a.handleCreateSet)
	a.router.Put("/api/workouts/{workoutID}/sets/{setID}", a.handleUpdateSet)
	a.router.Delete("/api/workouts/{workoutID}/sets/{setID}", a.handleDeleteSet)
}

func (a *application) handleGetExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := a.db.exercises(r.Context())
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get exercises.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	results := make([]response, 0, len(exercises))

	for _, v := range exercises {
		results = append(results, response(v))
	}

	writeJSON(w, r, results)
}

func (a *application) handleGetWorkoutList(w http.ResponseWriter, r *http.Request) {
	workouts, err := a.db.workoutList(r.Context())
	if err != nil {
		hlog.FromRequest(r).Err(err).Msg("Failed to get workout list.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID         int    `json:"id"`
		StartedUTC string `json:"startedUtc"`
	}

	results := make([]response, 0, len(workouts))

	for _, v := range workouts {
		results = append(results, response{
			ID:         int(v.ID),
			StartedUTC: v.StartedUTC,
		})
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

	l.Info().Int("workout_id", int(id)).Msg("Created new workout.")

	type response struct {
		ID int `json:"id"`
	}

	writeJSON(w, r, response{
		ID: int(id),
	})
}

func (a *application) handleDeleteWorkout(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	id, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to delete workout with invalid ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	l = l.With().Int("workout_id", id).Logger()

	if err := a.db.deleteWorkout(r.Context(), id); err != nil {
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

func (a *application) handleGetSetsByWorkoutId(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	id, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get sets for workout with invalid ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sets, err := a.db.setsForWorkout(r.Context(), id)
	if err != nil {
		l.Err(err).
			Int("workout_id", id).
			Msg("Failed to get sets for workout ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID           int    `json:"id"`
		ExerciseID   int    `json:"exerciseId"`
		ExerciseName string `json:"exerciseName"`
		DateUtc      string `json:"dateUtc"`
		Repetitions  int    `json:"repetitions"`
		Weight       int    `json:"weight"`
	}

	results := make([]response, 0, len(sets))

	for _, v := range sets {
		results = append(results, response(v))
	}

	writeJSON(w, r, results)
}

func (a *application) handleGetSetById(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	workoutID, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid workout ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	setID, err := strconv.Atoi(chi.URLParam(r, "setID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid set ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	set, err := a.db.setByIds(r.Context(), workoutID, setID)
	if err != nil {
		l.Err(err).
			Int("workout_id", workoutID).
			Int("set_id", setID).
			Msg("Failed to get set by workout ID and set ID.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		ID           int    `json:"id"`
		ExerciseID   int    `json:"exerciseId"`
		ExerciseName string `json:"exerciseName"`
		DateUtc      string `json:"dateUtc"`
		Repetitions  int    `json:"repetitions"`
		Weight       int    `json:"weight"`
	}

	writeJSON(w, r, response(set))
}

func (a *application) handleCreateSet(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	workoutID, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid workout ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	l = l.With().Int("workout_id", workoutID).Logger()

	type body struct {
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	var b body

	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		l.Err(err).Msg("Failed to read JSON body.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := a.db.createSet(r.Context(), workoutID, b.ExerciseID, b.Repetitions, b.Weight); err != nil {
		l.Err(err).Msg("Failed to create new set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	workoutID, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid workout ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	setID, err := strconv.Atoi(chi.URLParam(r, "setID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid workout ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	l = l.With().Int("workout_id", workoutID).Int("set_id", setID).Logger()

	type body struct {
		SetID       int `json:"setId"`
		ExerciseID  int `json:"exerciseId"`
		Repetitions int `json:"repetitions"`
		Weight      int `json:"weight"`
	}

	var b body

	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		l.Err(err).Msg("Failed to read JSON body.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	l.Info().Interface("data", b).Msg("UPDATING")

	if err := a.db.updateSet(r.Context(), workoutID, b.SetID, b.ExerciseID, b.Repetitions, b.Weight); err != nil {
		l.Err(err).Msg("Failed to update existing set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (a *application) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	l := *hlog.FromRequest(r)

	workoutID, err := strconv.Atoi(chi.URLParam(r, "workoutID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid workout ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	setID, err := strconv.Atoi(chi.URLParam(r, "setID"))
	if err != nil {
		l.Warn().Err(err).Msg("Request to get set with invalid set ID.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := a.db.deleteSet(r.Context(), workoutID, setID); err != nil {
		l.Err(err).
			Int("workout_id", workoutID).
			Int("set_id", setID).
			Msg("Failed to delete set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

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

type Database struct {
	db *sqlx.DB
}

func newDatabase(path string) *Database {
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

	return &Database{db}
}

type WorkoutList struct {
	ID         uint64 `db:"id"`
	StartedUTC string `db:"start_date_utc"`
}

func (d *Database) workoutList(ctx context.Context) ([]WorkoutList, error) {
	const query = `
		SELECT
			id,
			start_date_utc
		FROM
			workout
		ORDER BY
			start_date_utc DESC`

	var result []WorkoutList

	if err := d.db.SelectContext(ctx, &result, query); err != nil {
		return nil, err
	}

	return result, nil
}

type CreateWorkoutID int

func (d *Database) createWorkout(ctx context.Context) (CreateWorkoutID, error) {
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

	return CreateWorkoutID(id), nil
}

func (d *Database) deleteWorkout(ctx context.Context, workoutID int) error {
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

type WorkoutSet struct {
	ID           int    `db:"id"`
	ExerciseID   int    `db:"exercise_id"`
	ExerciseName string `db:"exercise_name"`
	DateUtc      string `db:"date_utc"`
	Repetitions  int    `db:"repetitions"`
	Weight       int    `db:"weight"`
}

func (d *Database) setsForWorkout(ctx context.Context, workoutID int) ([]WorkoutSet, error) {
	const query = `
		SELECT
			es.id,
			es.exercise_id,
			e.name AS exercise_name,
			es.date_utc,
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

	var sets []WorkoutSet

	if err := d.db.SelectContext(ctx, &sets, query, workoutID); err != nil {
		return nil, err
	}

	return sets, nil
}

type Exercise struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func (d *Database) exercises(ctx context.Context) ([]Exercise, error) {
	const query = `
		SELECT
			id,
			name
		FROM
			exercise
		ORDER BY
			name ASC
	`

	var exercises []Exercise

	if err := d.db.SelectContext(ctx, &exercises, query); err != nil {
		return nil, err
	}

	return exercises, nil
}

func (d *Database) setByIds(ctx context.Context, workoutID, setID int) (WorkoutSet, error) {
	const query = `
		SELECT
			es.id,
			es.exercise_id,
			e.name AS exercise_name,
			es.date_utc,
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

	var set WorkoutSet

	if err := d.db.GetContext(ctx, &set, query, workoutID, setID); err != nil {
		return WorkoutSet{}, err
	}

	return set, nil
}

func (d *Database) deleteSet(ctx context.Context, workoutID, setID int) error {
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

func (d *Database) createSet(
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

	if _, err := d.db.ExecContext(ctx, query, exerciseID, workoutID, repetitions, weight); err != nil {
		return err
	}

	return nil
}

func (d *Database) updateSet(
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

	if _, err := d.db.ExecContext(ctx, query, exerciseID, repetitions, weight, setID, workoutID); err != nil {
		return err
	}

	return nil
}
