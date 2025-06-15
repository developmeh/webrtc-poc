package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/developmeh/webrtc-poc/internal/logger"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	cfgFile string

	// Server command flags
	serverAddr  string
	serverFile  string
	serverDelay int
	stunServer  string

	// Client command flags
	clientServer string
	clientOutput string
	clientStun   string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "webrtc-poc",
	Short: "WebRTC File Streaming Proof of Concept",
	Long: `A proof of concept for using WebRTC to stream a file line by line.
The implementation is kept as succinct as possible while still being functional.`,
}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the WebRTC file streaming server",
	Long: `Start the WebRTC file streaming server that will stream a file line by line.
The server will listen for WebRTC connections and stream the specified file.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start the WebRTC file streaming client",
	Long: `Start the WebRTC file streaming client that will connect to a server and receive a file.
The client will connect to the specified server and receive the file line by line.`,
	Run: func(cmd *cobra.Command, args []string) {
		runClient()
	},
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
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)

	// Server flags
	serverCmd.Flags().StringVar(&serverAddr, "addr", ":8080", "HTTP service address")
	serverCmd.Flags().StringVar(&serverFile, "file", "sample.txt", "File to stream")
	serverCmd.Flags().IntVar(&serverDelay, "delay", 1000, "Delay between lines in milliseconds")
	serverCmd.Flags().StringVar(&stunServer, "stun", "", "STUN server address (leave empty for direct connection)")

	// Client flags
	clientCmd.Flags().StringVar(&clientServer, "server", "http://localhost:8080/offer", "WebRTC server URL")
	clientCmd.Flags().StringVar(&clientOutput, "output", "", "Output file (leave empty for stdout)")
	clientCmd.Flags().StringVar(&clientStun, "stun", "", "STUN server address (leave empty for direct connection)")

	// Bind flags to viper
	viper.BindPFlag("server.addr", serverCmd.Flags().Lookup("addr"))
	viper.BindPFlag("server.file", serverCmd.Flags().Lookup("file"))
	viper.BindPFlag("server.delay", serverCmd.Flags().Lookup("delay"))
	viper.BindPFlag("server.stun", serverCmd.Flags().Lookup("stun"))
	viper.BindPFlag("client.server", clientCmd.Flags().Lookup("server"))
	viper.BindPFlag("client.output", clientCmd.Flags().Lookup("output"))
	viper.BindPFlag("client.stun", clientCmd.Flags().Lookup("stun"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory with name "config" (without extension).
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runServer() {
	// Get configuration from viper
	addr := viper.GetString("server.addr")
	filename := viper.GetString("server.file")
	delay := viper.GetInt("server.delay")
	stunServerURL := viper.GetString("server.stun")

	logger.Info("Starting WebRTC file streaming server on %s", addr)
	logger.Info("Will stream file: %s with delay: %dms", filename, delay)

	// Ensure the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		logger.Error("File does not exist: %s", filename)
		os.Exit(1)
	}

	// Create a new SettingEngine
	settingEngine := webrtc.SettingEngine{}

	// Configure ICE based on whether STUN server is provided
	if stunServerURL == "" {
		// No STUN server - use only local candidates
		logger.Info("No STUN server provided, using direct connection only")

		// Disable mDNS
		settingEngine.SetICEMulticastDNSMode(0) // 0 = Disabled

		// Allow all interfaces for direct connection
		settingEngine.SetInterfaceFilter(func(interfaceName string) bool {
			return true // Allow all interfaces
		})
	} else {
		logger.Info("Using STUN server: %s", stunServerURL)
	}

	// Create a new RTCPeerConnection configuration
	config := webrtc.Configuration{}

	// Add ICE servers if STUN server is provided
	if stunServerURL != "" {
		config.ICEServers = []webrtc.ICEServer{
			{
				URLs: []string{stunServerURL},
			},
		}
	}

	// Create a new API with the custom settings
	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	// Create a wait group to wait for all connections to complete
	var wg sync.WaitGroup

	// Create a channel to signal shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Handle HTTP requests
	http.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the raw offer from the request body
		offerBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read offer: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Log the raw offer for debugging
		logger.Debug("Raw offer received: %s", string(offerBytes))

		// Parse the offer from the request
		var offer webrtc.SessionDescription
		if err := json.Unmarshal(offerBytes, &offer); err != nil {
			http.Error(w, "Failed to parse offer: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Log the parsed offer for debugging
		logger.Debug("Parsed offer type: %s", offer.Type.String())

		// Log the parsed offer for debugging
		offerJSON, _ := json.Marshal(offer)
		logger.Debug("Parsed offer: %s", string(offerJSON))

		// Create a new peer connection
		peerConnection, err := api.NewPeerConnection(config)
		if err != nil {
			http.Error(w, "Failed to create peer connection: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Monitor connection state changes
		peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			logger.Info("Connection state changed: %s", state.String())

			switch state {
			case webrtc.PeerConnectionStateConnected:
				logger.Info("WebRTC connection established successfully!")
			case webrtc.PeerConnectionStateFailed:
				logger.Error("WebRTC connection failed")
			case webrtc.PeerConnectionStateClosed:
				logger.Info("WebRTC connection closed")
			}
		})

		// Set the remote description
		if err := peerConnection.SetRemoteDescription(offer); err != nil {
			http.Error(w, "Failed to set remote description: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Create a data channel
		dataChannel, err := peerConnection.CreateDataChannel("fileStream", nil)
		if err != nil {
			http.Error(w, "Failed to create data channel: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set up data channel handlers
		dataChannel.OnOpen(func() {
			logger.Info("Data channel opened")

			// Increment the wait group
			wg.Add(1)

			// Start streaming the file in a goroutine
			go func() {
				defer wg.Done()
				defer dataChannel.Close()

				streamFile(dataChannel, filename, delay)
			}()
		})

		dataChannel.OnClose(func() {
			logger.Info("Data channel closed")
		})

		// Create an answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			http.Error(w, "Failed to create answer: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set the local description
		if err := peerConnection.SetLocalDescription(answer); err != nil {
			http.Error(w, "Failed to set local description: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Wait for ICE gathering to complete
		logger.Info("Waiting for ICE gathering to complete...")
		<-webrtc.GatheringCompletePromise(peerConnection)
		logger.Info("ICE gathering complete")

		// Get the local description after ICE gathering is complete
		answer = *peerConnection.LocalDescription()

		// Return the answer
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(answer); err != nil {
			logger.Error("Failed to encode answer: %v", err)
		}
	})

	// Start the HTTP server
	server := &http.Server{Addr: addr}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	// Print the server's PID
	fmt.Printf("SERVER_PID=%d\n", os.Getpid())

	// Wait for shutdown signal
	<-shutdown
	logger.Info("Shutting down server...")

	// Shutdown the HTTP server
	if err := server.Close(); err != nil {
		logger.Error("Error shutting down HTTP server: %v", err)
	}

	// Wait for all connections to complete
	wg.Wait()
	logger.Info("Server shutdown complete")
}

func runClient() {
	// Get configuration from viper
	serverURL := viper.GetString("client.server")
	output := viper.GetString("client.output")
	stunServerURL := viper.GetString("client.stun")

	logger.Info("Starting WebRTC file streaming client")
	logger.Info("Connecting to server: %s", serverURL)

	// Create a new SettingEngine
	settingEngine := webrtc.SettingEngine{}

	// Configure ICE based on whether STUN server is provided
	if stunServerURL == "" {
		// No STUN server - use only local candidates
		logger.Info("No STUN server provided, using direct connection only")

		// Disable mDNS
		settingEngine.SetICEMulticastDNSMode(0) // 0 = Disabled

		// Allow all interfaces for direct connection
		settingEngine.SetInterfaceFilter(func(interfaceName string) bool {
			return true // Allow all interfaces
		})
	} else {
		logger.Info("Using STUN server: %s", stunServerURL)
	}

	// Create a new RTCPeerConnection configuration
	config := webrtc.Configuration{}

	// Add ICE servers if STUN server is provided
	if stunServerURL != "" {
		config.ICEServers = []webrtc.ICEServer{
			{
				URLs: []string{stunServerURL},
			},
		}
	}

	// Create a new API with the custom settings
	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	// Create a new peer connection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		logger.Error("Failed to create peer connection: %v", err)
		os.Exit(1)
	}

	// Monitor connection state changes
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		logger.Info("Connection state changed: %s", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			logger.Info("WebRTC connection established successfully!")
		case webrtc.PeerConnectionStateFailed:
			logger.Error("WebRTC connection failed")
		case webrtc.PeerConnectionStateClosed:
			logger.Info("WebRTC connection closed")
		}
	})

	// Create a channel to receive data
	dataChan := make(chan string)

	// Create a data channel to ensure media section in SDP
	_, err = peerConnection.CreateDataChannel("initChannel", nil)
	if err != nil {
		logger.Error("Failed to create init data channel: %v", err)
		os.Exit(1)
	}

	// Set up data channel handler
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		logger.Info("New data channel: %s", d.Label())

		d.OnOpen(func() {
			logger.Info("Data channel opened")
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			data := string(msg.Data)
			dataChan <- data
		})

		d.OnClose(func() {
			logger.Info("Data channel closed")
			close(dataChan)
		})
	})

	// Create an offer
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		logger.Error("Failed to create offer: %v", err)
		os.Exit(1)
	}

	// Set the local description
	if err := peerConnection.SetLocalDescription(offer); err != nil {
		logger.Error("Failed to set local description: %v", err)
		os.Exit(1)
	}

	// Wait for ICE gathering to complete
	logger.Info("Waiting for ICE gathering to complete...")
	<-webrtc.GatheringCompletePromise(peerConnection)
	logger.Info("ICE gathering complete")

	// Get the local description after ICE gathering is complete
	offer = *peerConnection.LocalDescription()

	// Log the SDP for debugging
	logger.Debug("Offer SDP: %s", offer.SDP)

	// Send the offer to the server
	offerJSON, err := json.Marshal(offer)
	if err != nil {
		logger.Error("Failed to marshal offer: %v", err)
		os.Exit(1)
	}

	// Log the raw offer for debugging
	logger.Debug("Raw offer: %s", string(offerJSON))

	resp, err := http.Post(serverURL, "application/json", strings.NewReader(string(offerJSON)))
	if err != nil {
		logger.Error("Failed to send offer: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Error("Server returned non-OK status: %d %s, body: %s",
			resp.StatusCode, resp.Status, string(bodyBytes))
		os.Exit(1)
	}

	// Read the answer
	answerJSON, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read answer: %v", err)
		os.Exit(1)
	}

	// Log the raw response for debugging
	logger.Debug("Raw server response: %s", string(answerJSON))

	// Parse the answer
	var answer webrtc.SessionDescription
	if err := json.Unmarshal(answerJSON, &answer); err != nil {
		logger.Error("Failed to parse answer: %v, raw response: %s", err, string(answerJSON))
		os.Exit(1)
	}

	// Set the remote description
	if err := peerConnection.SetRemoteDescription(answer); err != nil {
		logger.Error("Failed to set remote description: %v", err)
		os.Exit(1)
	}

	// Print the client's PID
	fmt.Printf("CLIENT_PID=%d\n", os.Getpid())

	// Create a channel to signal shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Open the output file if specified
	var outputFile *os.File
	if output != "" {
		outputFile, err = os.Create(output)
		if err != nil {
			logger.Error("Failed to create output file: %v", err)
			os.Exit(1)
		}
		defer outputFile.Close()
		logger.Info("Writing output to file: %s", output)
	} else {
		logger.Info("Writing output to stdout")
	}

	// Start receiving data
	go func() {
		lineCount := 0
		startTime := time.Now()

		for line := range dataChan {
			lineCount++

			if outputFile != nil {
				fmt.Fprintln(outputFile, line)
			} else {
				fmt.Println(line)
			}

			logger.Debug("Received line %d: %s", lineCount, line)
		}

		elapsed := time.Since(startTime)
		logger.Info("Received %d lines in %v (%.2f lines/sec)",
			lineCount, elapsed, float64(lineCount)/elapsed.Seconds())
	}()

	// Wait for shutdown signal
	<-shutdown
	logger.Info("Shutting down client...")

	// Close the peer connection
	if err := peerConnection.Close(); err != nil {
		logger.Error("Error closing peer connection: %v", err)
	}

	logger.Info("Client shutdown complete")
}

// streamFile streams a file line by line over a data channel
func streamFile(dataChannel *webrtc.DataChannel, filename string, delayMs int) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Recovered from panic in streamFile: %v", r)
		}
	}()

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Send the line over the data channel
		if err := dataChannel.SendText(line); err != nil {
			logger.Error("Failed to send line %d: %v", lineCount, err)
			return
		}

		logger.Debug("Sent line %d: %s", lineCount, line)

		// Delay between lines
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading file: %v", err)
	}

	logger.Info("Finished streaming file, sent %d lines", lineCount)
}

func main() {
	Execute()
}
