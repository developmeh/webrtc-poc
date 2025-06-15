package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Server command flags
	serverAddr  string
	serverFile  string
	serverDelay int
	stunServer  string
)

// serverCmd represents the server command
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the WebRTC file streaming server",
	Long: `Start the WebRTC file streaming server that will stream a file line by line.
The server will listen for WebRTC connections and stream the specified file.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	// Server flags
	ServerCmd.Flags().StringVar(&serverAddr, "addr", ":8080", "HTTP service address")
	ServerCmd.Flags().StringVar(&serverFile, "file", "sample.txt", "File to stream")
	ServerCmd.Flags().IntVar(&serverDelay, "delay", 1000, "Delay between lines in milliseconds")
	ServerCmd.Flags().StringVar(&stunServer, "stun", "", "STUN server address (leave empty for direct connection)")

	// Bind flags to viper
	viper.BindPFlag("server.addr", ServerCmd.Flags().Lookup("addr"))
	viper.BindPFlag("server.file", ServerCmd.Flags().Lookup("file"))
	viper.BindPFlag("server.delay", ServerCmd.Flags().Lookup("delay"))
	viper.BindPFlag("server.stun", ServerCmd.Flags().Lookup("stun"))
}

func runServer() {
	// This will be implemented later by refactoring the existing server code
}
