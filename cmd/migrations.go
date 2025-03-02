package cmd

import (
	"fmt"
	"log"
	"slices"

	"github.com/exo-framework/exo/migrator"
	"github.com/spf13/cobra"
)

func getMigratorForCLI() (*migrator.Migrator, error) {
	mig := migrator.New()

	if err := mig.Initialize(nil); err != nil {
		return nil, err
	}

	return mig, nil
}

func requestYesNoUserConfirmationForMigrations(msg string) {
	println(msg + " (yes/no)")

	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil {
		return
	}

	if answer != "yes" && answer != "y" {
		log.Fatal("Aborted")
	}

	println("Continuing...")
}

var migrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "The main command for migrations. The base command lists all migrations.",
	Run: func(cmd *cobra.Command, args []string) {
		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		if err := mig.ListMigrations(); err != nil {
			panic(err)
		}
	},
}

var migrationsGenerateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generates a new pair of empty migration files for up and down.",
	Run: func(cmd *cobra.Command, args []string) {
		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		version, err := mig.GenereteEmptyMigration()
		if err != nil {
			panic(err)
		}

		println("Migration files generated with version:", version)
	},
}

var migrationsDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Generates a new pair of migration files for up and down based on the diff between the current schema and the database schema.",
	Run: func(cmd *cobra.Command, args []string) {
		initial := slices.Contains(args, "--init")

		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		schemaData, err := mig.LoadExternalGormSchema()
		if err != nil {
			panic(err)
		}

		version, err := mig.GenerateDiffMigration(initial, schemaData)
		if err != nil {
			panic(err)
		}

		println("Migration files generated with version:", version)
	},
}

var migrationsUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Runs all pending migrations.",
	Run: func(cmd *cobra.Command, args []string) {
		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		yes := slices.Contains(args, "--yes")
		if !yes {
			requestYesNoUserConfirmationForMigrations("Are you sure you want to run all pending migrations?")
		}

		if err := mig.ExecuteAll(migrator.Up); err != nil {
			panic(err)
		}

		println("Migrations executed successfully")
	},
}

var migrationsDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rolls back all executed migrations.",
	Run: func(cmd *cobra.Command, args []string) {
		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		yes := slices.Contains(args, "--yes")
		if !yes {
			requestYesNoUserConfirmationForMigrations("Are you sure you want to roll back all executed migrations?")
		}

		if err := mig.ExecuteAll(migrator.Down); err != nil {
			panic(err)
		}

		println("Migrations rolled back successfully")
	},
}

var migrationsExecuteCmd = &cobra.Command{
	Use:     "execute",
	Aliases: []string{"exec"},
	Short:   "Executes a specific migration up or down.",
	Run: func(cmd *cobra.Command, args []string) {
		var dir *migrator.MigrateDir
		for i, arg := range args {
			if arg == "--up" || arg == "--u" {
				dr := migrator.Up
				dir = &dr
				args = append(args[:i], args[i+1:]...)
				break
			}

			if arg == "--down" || arg == "--d" {
				dr := migrator.Down
				dir = &dr
				args = append(args[:i], args[i+1:]...)
				break
			}
		}

		if dir == nil {
			log.Fatal("Please specify a direction (--up or --down)")
		}

		if len(args) == 0 {
			log.Fatal("Please specify a migration version")
		}

		mig, err := getMigratorForCLI()
		if err != nil {
			panic(err)
		}

		version := args[0]
		if err := mig.Execute(version, *dir); err != nil {
			panic(err)
		}

		println("Migration executed successfully")
	},
}
