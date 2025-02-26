package cmd

import (
	"github.com/exo-framework/exo/gen"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"g", "gen"},
	Short:   "Generates the REST API glue code for the exo framework",
	Run: func(cmd *cobra.Command, args []string) {
		const path = "E:\\Development\\Sevenity\\z_otherstuff\\exo\\gentest"

		g := gen.NewGenerator()
		if err := g.Analyze(path); err != nil {
			panic(err)
		}

		if err := g.Generate(); err != nil {
			panic(err)
		}
	},
}
