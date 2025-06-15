package main

import (
	"fmt"
	"github.com/paulscoder/webrtc-poc/internal/cmd"
	"github.com/paulscoder/webrtc-poc/internal/logger"
	"github.com/spf13/cobra"
	"os"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "webrtc-poc",
	Short: "WebRTC File Streaming Proof of Concept",
	Long: `A proof of concept for using WebRTC to stream a file line by line.
The implementation is kept as succinct as possible while still being functional.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

	// Initialize logger
	logger.Init()

	// Add commands
	rootCmd.AddCommand(cmd.ServerCmd)
	rootCmd.AddCommand(cmd.ClientCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Config file initialization will be implemented later
}

func main() {
	Execute()
}
