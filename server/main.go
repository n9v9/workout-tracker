package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
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

	a.router.Get("/exercises", a.handleGetExercises)

	a.router.Get("/workouts", a.handleGetWorkoutList)
	a.router.Post("/workouts", a.handleCreateWorkout)
	a.router.Delete("/workouts/{workoutId}", a.handleDeleteWorkout)

	a.router.Get("/workouts/{workoutId}/sets", a.handleGetSetsByWorkoutId)
	a.router.Get("/workouts/{workoutId}/sets/{setId}", a.handleGetSetById)
	a.router.Put("/workouts/{workoutId}/sets/{setId}", a.handleSaveSet)
	a.router.Delete("/workouts/{workoutId}/sets/{setId}", a.handleDeleteSet)
}

func (a *application) handleGetExercises(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleGetWorkoutList(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleCreateWorkout(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleDeleteWorkout(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleGetSetsByWorkoutId(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleGetSetById(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleSaveSet(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
}

func (a *application) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusTeapot)
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
