package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

	rootCmd = &cobra.Command{
		Use:   "filedown",
		Short: "A small cli tool to downlaod the files from remote locations",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
