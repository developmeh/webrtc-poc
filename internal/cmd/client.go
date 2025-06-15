package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Client command flags
	clientServer string
	clientOutput string
	clientStun   string
)

// ClientCmd represents the client command
var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start the WebRTC file streaming client",
	Long: `Start the WebRTC file streaming client that will connect to a server and receive a file.
The client will connect to the specified server and receive the file line by line.`,
	Run: func(cmd *cobra.Command, args []string) {
		runClient()
	},
}

func init() {
	// Client flags
	ClientCmd.Flags().StringVar(&clientServer, "server", "http://localhost:8080/offer", "WebRTC server URL")
	ClientCmd.Flags().StringVar(&clientOutput, "output", "", "Output file (leave empty for stdout)")
	ClientCmd.Flags().StringVar(&clientStun, "stun", "", "STUN server address (leave empty for direct connection)")

	// Bind flags to viper
	viper.BindPFlag("client.server", ClientCmd.Flags().Lookup("server"))
	viper.BindPFlag("client.output", ClientCmd.Flags().Lookup("output"))
	viper.BindPFlag("client.stun", ClientCmd.Flags().Lookup("stun"))
}

func runClient() {
	// This will be implemented later by refactoring the existing client code
}