package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/exo-framework/exo/common"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Prints the version of the exo framework",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("v%s\n", common.VERSION)
	},
}

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Initializes a new exo configuration",
	Run: func(cmd *cobra.Command, args []string) {
		println("🧙‍♂️ Hi! I'm your friendly setup wizard. Let's first setup the postgres connection!")

		hostPrompt := promptui.Prompt{
			Label:   "Postgres Hostname",
			Default: "localhost",
		}

		host, err := hostPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted hostname. Sorry :(", err)
			return
		}

		portPrompt := promptui.Prompt{
			Label:   "Postgres Port",
			Default: "5432",
			Validate: func(input string) error {
				if _, err := strconv.Atoi(input); err != nil {
					return errors.New("port must be a number")
				}
				return nil
			},
		}

		port, err := portPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted port. Sorry :(", err)
			return
		}

		dbPrompt := promptui.Prompt{
			Label:   "Postgres Database",
			Default: "postgres",
		}

		db, err := dbPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted database. Sorry :(", err)
			return
		}

		userPrompt := promptui.Prompt{
			Label:   "Postgres Username",
			Default: "postgres",
		}

		user, err := userPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted username. Sorry :(", err)
			return
		}

		passwordPrompt := promptui.Prompt{
			Label:   "Postgres Password",
			Default: "postgres",
			Mask:    '*',
		}

		password, err := passwordPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted password. Sorry :(", err)
			return
		}

		customEnvPrompt := promptui.Select{
			Label: "🧙‍♂️ Do you want to select custom env var names",
			Items: []string{"No", "Yes"},
		}

		_, customEnv, err := customEnvPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted choice. Sorry :(", err)
			return
		}

		hostEnv := "DB_HOST"
		portEnv := "DB_PORT"
		dbEnv := "DB_NAME"
		userEnv := "DB_USER"
		passwordEnv := "DB_PASS"

		if customEnv == "Yes" {
			hostEnvPrompt := promptui.Prompt{
				Label:   "Postgres Hostname Env",
				Default: "DB_HOST",
			}

			hostEnv, err = hostEnvPrompt.Run()
			if err != nil {
				println("😒 Could not read your wanted hostname env. Sorry :(", err)
				return
			}

			portEnvPrompt := promptui.Prompt{
				Label:   "Postgres Port Env",
				Default: "DB_PORT",
			}

			portEnv, err = portEnvPrompt.Run()
			if err != nil {
				println("😒 Could not read your wanted port env. Sorry :(", err)
				return
			}

			dbEnvPrompt := promptui.Prompt{
				Label:   "Postgres Database Env",
				Default: "DB_NAME",
			}

			dbEnv, err = dbEnvPrompt.Run()
			if err != nil {
				println("😒 Could not read your wanted database env. Sorry :(", err)
				return
			}

			userEnvPrompt := promptui.Prompt{
				Label:   "Postgres Username Env",
				Default: "DB_USER",
			}

			userEnv, err = userEnvPrompt.Run()
			if err != nil {
				println("😒 Could not read your wanted username env. Sorry :(", err)
				return
			}

			passwordEnvPrompt := promptui.Prompt{
				Label:   "Postgres Password Env",
				Default: "DB_PASS",
			}

			passwordEnv, err = passwordEnvPrompt.Run()
			if err != nil {
				println("😒 Could not read your wanted password env. Sorry :(", err)
				return
			}
		}

		envDirty := hostEnv != "DB_HOST" || portEnv != "DB_PORT" || dbEnv != "DB_NAME" || userEnv != "DB_USER" || passwordEnv != "DB_PASS"

		println("🧙‍♂️ Great! Now one last thing, how should your Go package be called with the Gorm connection?")
		goPackagePrompt := promptui.Prompt{
			Label:   "Go Package Name",
			Default: "db",
		}

		goPackage, err := goPackagePrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted Go package name. Sorry :(", err)
			return
		}

		println("🧙‍♂️ Great! Here's your configuration:")
		println("Host:", host)
		println("Port:", port)
		println("Database:", db)
		println("Username:", user)
		println("Password:", strings.Repeat("*", len(password)))
		println("Host Env:", hostEnv)
		println("Port Env:", portEnv)
		println("Database Env:", dbEnv)
		println("Username Env:", userEnv)
		println("Password Env:", passwordEnv)
		println("Go Package:", goPackage)

		happyPrompt := promptui.Select{
			Label: "🧙‍♂️ Are you happy with this configuration",
			Items: []string{"Yes", "No"},
		}

		_, happy, err := happyPrompt.Run()
		if err != nil {
			println("😒 Could not read your wanted choice. Sorry :(", err)
			return
		}

		if happy == "No" {
			println("🧙‍♂️ Sorry to hear that. Let's start over.")
			return
		}

		println("🧙‍♂️ Great! Let's set it up then!")

		println("🚀 Writing .env...")
		envs := map[string]string{
			hostEnv:     host,
			portEnv:     port,
			dbEnv:       db,
			userEnv:     user,
			passwordEnv: password,
		}

		if _, err := os.Stat(".env"); err == nil {
			if err := os.Remove(".env"); err != nil {
				println("😒 Could not remove .env file. Sorry :(", err)
				return
			}
		}

		{
			f, err := os.Create(".env")
			if err != nil {
				println("😒 Could not create .env file. Sorry :(", err)
				return
			}

			defer f.Close()

			for k, v := range envs {
				if _, err := f.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
					println("😒 Could not write to .env file. Sorry :(", err)
					return
				}
			}

			println("🔥 .env written!")
		}

		println("🚀 Writing .env.example...")
		if _, err := os.Stat(".env.example"); err == nil {
			if err := os.Remove(".env.example"); err != nil {
				println("😒 Could not remove .env.example file. Sorry :(", err)
				return
			}
		}

		{
			f, err := os.Create(".env.example")
			if err != nil {
				println("😒 Could not create .env.example file. Sorry :(", err)
				return
			}

			defer f.Close()

			if _, err := f.WriteString("# Copy this file to .env and fill in the values\n\n## Postgres DB\n"); err != nil {
				println("😒 Could not write to .env.example file. Sorry :(", err)
				return
			}

			for k := range envs {
				if _, err := f.WriteString(fmt.Sprintf("%s=somevalue\n", k)); err != nil {
					println("😒 Could not write to .env.example file. Sorry :(", err)
					return
				}
			}

			println("🔥 .env.example written!")
		}

		if envDirty || goPackage != "db" {
			println("🚀 Writing .exorc file...")

			if _, err := os.Stat(".exorc"); err == nil {
				if err := os.Remove(".exorc"); err != nil {
					println("😒 Could not remove .exorc file. Sorry :(", err)
					return
				}
			}

			f, err := os.Create(".exorc")
			if err != nil {
				println("😒 Could not create .exorc file. Sorry :(", err)
				return
			}

			defer f.Close()

			if _, err := f.WriteString(fmt.Sprintf("DB_HOST->%s\n", hostEnv)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			if _, err := f.WriteString(fmt.Sprintf("DB_PORT->%s\n", portEnv)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			if _, err := f.WriteString(fmt.Sprintf("DB_NAME->%s\n", dbEnv)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			if _, err := f.WriteString(fmt.Sprintf("DB_USER->%s\n", userEnv)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			if _, err := f.WriteString(fmt.Sprintf("DB_PASS->%s\n", passwordEnv)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			if _, err := f.WriteString(fmt.Sprintf("DB_PACKAGE->%s\n", goPackage)); err != nil {
				println("😒 Could not write to .exorc file. Sorry :(", err)
				return
			}

			println("🔥 .exorc written!")
		}

		fmt.Printf("🚀 Writing %s/db.go...\n", goPackage)
		if _, err := os.Stat(goPackage); os.IsNotExist(err) {
			if err := os.Mkdir(goPackage, 0755); err != nil {
				println("😒 Could not create directory. Sorry :(", err)
				return
			}
		}

		dbFile := path.Join(goPackage, "db.go")
		if _, err := os.Stat(dbFile); err == nil {
			if err := os.Remove(dbFile); err != nil {
				println("😒 Could not remove db.go file. Sorry :(", err)
				return
			}
		}

		f, err := os.Create(dbFile)
		if err != nil {
			println("😒 Could not create db.go file. Sorry :(", err)
			return
		}

		defer f.Close()

		code := []string{
			fmt.Sprintf("package %s", goPackage),
			"",
			"import (",
			"\t\"fmt\"",
			"\t\"os\"",
			"\t\"time\"",
			"",
			"\t\"gorm.io/driver/postgres\"",
			"\t\"gorm.io/gorm\"",
			")",
			"",
			"var DB *gorm.DB",
			"",
			"func Connect() error {",
			fmt.Sprintf("\tdsn := fmt.Sprintf(\"host=%%s port=%%s user=%%s password=%%s dbname=%%s\", os.Getenv(\"%s\"), os.Getenv(\"%s\"), os.Getenv(\"%s\"), os.Getenv(\"%s\"), os.Getenv(\"%s\"))", hostEnv, portEnv, userEnv, passwordEnv, dbEnv),
			"",
			"\tdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})",
			"\tif err != nil {",
			"\t\treturn err",
			"\t}",
			"",
			"\tDB = db",
			"",
			"\tsqlDB, err := db.DB()",
			"\tif err != nil {",
			"\t\treturn err",
			"\t}",
			"",
			"\tsqlDB.SetMaxIdleConns(10)",
			"\tsqlDB.SetMaxOpenConns(100)",
			"\tsqlDB.SetConnMaxLifetime(time.Hour)",
			"",
			"\tif err := db.Exec(\"CREATE EXTENSION IF NOT EXISTS \\\"uuid-ossp\\\";\").Error; err != nil {",
			"\t\treturn err",
			"\t}",
			"",
			"\treturn nil",
			"}",
		}

		for _, line := range code {
			if _, err := f.WriteString(fmt.Sprintf("%s\n", line)); err != nil {
				println("😒 Could not write to db.go file. Sorry :(", err)
				return
			}
		}

		println("🔥 db.go written!")
		println("🧙‍♂️ All done! Happy coding! 🚀")
	},
}
