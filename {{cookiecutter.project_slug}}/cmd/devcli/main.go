package main

import (
	"fmt"
	"log"
	"os"

	"{{ cookiecutter.go_module }}/configs"
{% if cookiecutter.use_asynq == "yes" %}
	"{{ cookiecutter.go_module }}/internal/core/asynctasks"
{% endif %}

	"github.com/urfave/cli/v2"
)

func main() {
	configs.Bootstrap()
	defer configs.Shutdown()

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "dropdb",
				Usage:  "reset db",
				Action: dropdb,
			},
			{
				Name:   "fakedb",
				Usage:  "fake db",
				Action: fakedb,
			},
{% if cookiecutter.use_asynq == "yes" %}
			{
				Name:   "debugtask",
				Usage:  "send a debug task",
				Action: sendDebugTask,
			},
{% endif %}
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func dropdb(cCtx *cli.Context) error {
	db := configs.GetDB()
	tbs, err := db.Migrator().GetTables()

	for _, table := range tbs {
		if err := db.Migrator().DropTable(table); err != nil {
			log.Printf("failed to drop table %s: %v", table, err)
		} else {
			log.Printf("successfully dropped table %s", table)
		}
	}

	log.Printf("drop schema atlas_schema_revisions")
	if err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", "atlas_schema_revisions")).Error; err != nil {
		log.Println("Error dropping schema:", err)
		return err
	}

	return err
}

func fakedb(cCtx *cli.Context) error {
	// db := configs.GetDB()
	// Fake db here
	return nil
}

{% if cookiecutter.use_asynq == "yes" %}
func sendDebugTask(c *cli.Context) error {
	return asynctasks.PublishDebugTaskTask()
}
{% endif %}
