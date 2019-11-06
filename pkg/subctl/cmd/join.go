package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(joinCmd)
}

var joinCmd = &cobra.Command{
	Use: "join",
}
