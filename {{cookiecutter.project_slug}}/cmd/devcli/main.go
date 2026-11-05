// Command devcli offers the local development utilities: reset and fake the
// database, and enqueue a debug task.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"{{ cookiecutter.go_module }}/configs"
	"{{ cookiecutter.go_module }}/internal/core/models"
	"{{ cookiecutter.go_module }}/pkg/cryptography"
{% if cookiecutter.use_asynq == "yes" %}
	"{{ cookiecutter.go_module }}/internal/core/asynctasks"
{% endif %}

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// demoPassword is the password given to every user seeded by `fakedb`.
const demoPassword = "demo-password-change-me"

// devCli keeps the loaded configuration and logger instead of reading them
// from package level globals.
type devCli struct {
	cfg    *configs.Config
	logger *slog.Logger
}

func main() {
	cfg, err := configs.NewConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger, err := configs.NewLogger(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	a := &devCli{cfg: cfg, logger: logger}
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "dropdb",
				Usage:  "reset db",
				Action: a.dropdb,
			},
			{
				Name:   "fakedb",
				Usage:  "fake db",
				Action: a.fakedb,
			},
{% if cookiecutter.use_asynq == "yes" %}
			{
				Name:   "debugtask",
				Usage:  "send a debug task",
				Action: a.debugtask,
			},
{% endif %}
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Error("command failed", "err", err)
		os.Exit(1)
	}
}

func (a *devCli) dropdb(_ *cli.Context) error {
	db, err := configs.NewDB(a.cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := configs.CloseDB(db); err != nil {
			a.logger.Error("failed to close database connection", "err", err)
		}
	}()

	tables, err := db.Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	for _, table := range tables {
		if err := db.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		a.logger.Info("dropped table", "table", table)
	}

	if err := db.Exec("DROP SCHEMA IF EXISTS atlas_schema_revisions CASCADE").Error; err != nil {
		return fmt.Errorf("failed to drop atlas schema: %w", err)
	}
	a.logger.Info("dropped schema", "schema", "atlas_schema_revisions")

	return nil
}

func (a *devCli) fakedb(_ *cli.Context) error {
	db, err := configs.NewDB(a.cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := configs.CloseDB(db); err != nil {
			a.logger.Error("failed to close database connection", "err", err)
		}
	}()

	hashed, err := cryptography.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("failed to hash the demo password: %w", err)
	}

	users := []models.User{
		{ID: uuid.New(), Firstname: "Ada", Lastname: "Lovelace", Email: "ada@example.com", Password: string(hashed)},
		{ID: uuid.New(), Firstname: "Alan", Lastname: "Turing", Email: "alan@example.com", Password: string(hashed)},
		{ID: uuid.New(), Firstname: "Grace", Lastname: "Hopper", Email: "grace@example.com", Password: string(hashed)},
	}
	for _, user := range users {
		var existing int64
		if err := db.Model(&models.User{}).Where("email = ?", user.Email).Count(&existing).Error; err != nil {
			return fmt.Errorf("failed to look up user %s: %w", user.Email, err)
		}
		if existing > 0 {
			a.logger.Info("user already exists, skipped", "email", user.Email)
			continue
		}
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.Email, err)
		}
		a.logger.Info("created user", "email", user.Email)
	}
	a.logger.Info("log in with the seeded users using password " + demoPassword)
	return nil
}

{% if cookiecutter.use_asynq == "yes" %}
func (a *devCli) debugtask(_ *cli.Context) error {
	client := configs.NewAsynqClient(a.cfg)
	defer func() {
		if err := configs.CloseAsynqClient(client); err != nil {
			a.logger.Error("failed to close asynq client", "err", err)
		}
	}()
	return asynctasks.NewPublisher(client).DebugTask()
}
{% endif %}
