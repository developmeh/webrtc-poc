package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pion/webrtc/v3"

	"github.com/paulscoder/webrtc-poc/internal/logger"
)

var (
	addr     = flag.String("addr", ":8080", "HTTP service address")
	filename = flag.String("file", "sample.txt", "File to stream")
	delay    = flag.Int("delay", 1000, "Delay between lines in milliseconds")
)

func main() {
	flag.Parse()

	logger.Init()
	logger.Info("Starting WebRTC file streaming server on %s", *addr)
	logger.Info("Will stream file: %s with delay: %dms", *filename, *delay)

	// Ensure the file exists
	if _, err := os.Stat(*filename); os.IsNotExist(err) {
		logger.Error("File does not exist: %s", *filename)
		os.Exit(1)
	}

	// Create a new SettingEngine
	settingEngine := webrtc.SettingEngine{}

	// Configure ICE to use only local candidates (no STUN/TURN)
	// Disable mDNS
	settingEngine.SetICEMulticastDNSMode(0) // 0 = Disabled

	// Allow all interfaces for direct connection
	settingEngine.SetInterfaceFilter(func(interfaceName string) bool {
		return true // Allow all interfaces
	})

	// Create a new RTCPeerConnection configuration with no STUN servers
	// We're using only local candidates for direct connection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{}, // Empty ICE servers list - no STUN/TURN
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

				streamFile(dataChannel, *filename, *delay)
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
	server := &http.Server{Addr: *addr}
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
