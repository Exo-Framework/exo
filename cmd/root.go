package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "exo",
	Short: "The exo CLI is the command line interface for the exo framework",
	Long:  "exo CLI is the command line interface for the exo framework allowing easy access to the frameworks most powerful features",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
}
