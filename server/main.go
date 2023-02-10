package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"

	"github.com/n9v9/workout-tracker/server/api"
	"github.com/n9v9/workout-tracker/server/repository/sqlite"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

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
			dbFile := ctx.String("db")
			addr := ctx.String("addr")

			if err := run(ctx.Context, addr, staticFiles, dbFile); err != nil {
				log.Err(err).Str("static_files", staticFiles).Str("db", dbFile).Send()
				os.Exit(1)
			}

			return nil
		},
	}
}

func run(ctx context.Context, addr, staticFilesDir, dbFile string) error {
	db, err := sqlite.NewDB(dbFile)
	if err != nil {
		return fmt.Errorf("failed to create database connection: %w", err)
	}

	if err := db.RunMigrations(migrations); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	api.New(staticFilesDir, db).Run(ctx, addr)

	return nil
}
