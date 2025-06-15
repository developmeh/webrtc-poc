package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pion/ice/v2"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/paulscoder/webrtc-poc/internal/logger"
	"github.com/pion/webrtc/v3"
)

var (
	serverURL = flag.String("server", "http://localhost:8080/offer", "WebRTC server URL")
	output    = flag.String("output", "", "Output file (leave empty for stdout)")
)

func main() {
	flag.Parse()

	logger.Init()
	logger.Info("Starting WebRTC file streaming client")
	logger.Info("Connecting to server: %s", *serverURL)

	// Create a new SettingEngine
	settingEngine := webrtc.SettingEngine{}

	// Configure ICE to use only local candidates (no STUN/TURN)
	// Disable mDNS
	settingEngine.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)

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

	resp, err := http.Post(*serverURL, "application/json", strings.NewReader(string(offerJSON)))
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
	if *output != "" {
		outputFile, err = os.Create(*output)
		if err != nil {
			logger.Error("Failed to create output file: %v", err)
			os.Exit(1)
		}
		defer outputFile.Close()
		logger.Info("Writing output to file: %s", *output)
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
