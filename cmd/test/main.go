package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paulscoder/webrtc-poc/internal/logger"
	"github.com/pion/webrtc/v3"
)

func main() {
	logger.Init()
	logger.Info("Starting WebRTC connection test")

	// Create a new SettingEngine for the server
	serverSettingEngine := webrtc.SettingEngine{}

	// Configure ICE to use only local candidates (no STUN/TURN)
	// Disable mDNS
	serverSettingEngine.SetICEMulticastDNSMode(0) // 0 = Disabled

	// Allow all interfaces for direct connection
	serverSettingEngine.SetInterfaceFilter(func(interfaceName string) bool {
		return true // Allow all interfaces
	})

	// Create a new RTCPeerConnection configuration with no STUN servers
	// We're using only local candidates for direct connection
	serverConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{}, // Empty ICE servers list - no STUN/TURN
	}

	// Create a new API with the custom settings
	serverAPI := webrtc.NewAPI(webrtc.WithSettingEngine(serverSettingEngine))

	// Create a new peer connection
	serverPC, err := serverAPI.NewPeerConnection(serverConfig)
	if err != nil {
		logger.Error("Failed to create server peer connection: %v", err)
		os.Exit(1)
	}
	defer serverPC.Close()

	// Create a new SettingEngine for the client
	clientSettingEngine := webrtc.SettingEngine{}

	// Configure ICE to use only local candidates (no STUN/TURN)
	// Disable mDNS
	clientSettingEngine.SetICEMulticastDNSMode(0) // 0 = Disabled

	// Allow all interfaces for direct connection
	clientSettingEngine.SetInterfaceFilter(func(interfaceName string) bool {
		return true // Allow all interfaces
	})

	// Create a new RTCPeerConnection configuration with no STUN servers
	// We're using only local candidates for direct connection
	clientConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{}, // Empty ICE servers list - no STUN/TURN
	}

	// Create a new API with the custom settings
	clientAPI := webrtc.NewAPI(webrtc.WithSettingEngine(clientSettingEngine))

	// Create a new peer connection
	clientPC, err := clientAPI.NewPeerConnection(clientConfig)
	if err != nil {
		logger.Error("Failed to create client peer connection: %v", err)
		os.Exit(1)
	}
	defer clientPC.Close()

	// Monitor connection state changes for the server
	serverPC.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		logger.Info("Server connection state changed: %s", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			logger.Info("Server WebRTC connection established successfully!")
		case webrtc.PeerConnectionStateFailed:
			logger.Error("Server WebRTC connection failed")
		case webrtc.PeerConnectionStateClosed:
			logger.Info("Server WebRTC connection closed")
		}
	})

	// Monitor connection state changes for the client
	clientPC.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		logger.Info("Client connection state changed: %s", state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			logger.Info("Client WebRTC connection established successfully!")
		case webrtc.PeerConnectionStateFailed:
			logger.Error("Client WebRTC connection failed")
		case webrtc.PeerConnectionStateClosed:
			logger.Info("Client WebRTC connection closed")
		}
	})

	// Create a data channel on the server
	dataChannel, err := serverPC.CreateDataChannel("test", nil)
	if err != nil {
		logger.Error("Failed to create data channel: %v", err)
		os.Exit(1)
	}

	// Set up data channel handlers on the server
	dataChannel.OnOpen(func() {
		logger.Info("Server data channel opened")

		// Send a test message
		if err := dataChannel.SendText("Hello from server!"); err != nil {
			logger.Error("Failed to send message: %v", err)
		}
	})

	dataChannel.OnClose(func() {
		logger.Info("Server data channel closed")
	})

	// Set up data channel handler on the client
	clientPC.OnDataChannel(func(d *webrtc.DataChannel) {
		logger.Info("Client received data channel: %s", d.Label())

		d.OnOpen(func() {
			logger.Info("Client data channel opened")
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			logger.Info("Client received message: %s", string(msg.Data))
		})

		d.OnClose(func() {
			logger.Info("Client data channel closed")
		})
	})

	// Create an offer from the server
	offer, err := serverPC.CreateOffer(nil)
	if err != nil {
		logger.Error("Failed to create offer: %v", err)
		os.Exit(1)
	}

	// Set the local description on the server
	if err := serverPC.SetLocalDescription(offer); err != nil {
		logger.Error("Failed to set local description on server: %v", err)
		os.Exit(1)
	}

	// Wait for ICE gathering to complete on the server
	logger.Info("Waiting for server ICE gathering to complete...")
	<-webrtc.GatheringCompletePromise(serverPC)
	logger.Info("Server ICE gathering complete")

	// Get the local description after ICE gathering is complete
	offer = *serverPC.LocalDescription()

	// Set the remote description on the client
	if err := clientPC.SetRemoteDescription(offer); err != nil {
		logger.Error("Failed to set remote description on client: %v", err)
		os.Exit(1)
	}

	// Create an answer from the client
	answer, err := clientPC.CreateAnswer(nil)
	if err != nil {
		logger.Error("Failed to create answer: %v", err)
		os.Exit(1)
	}

	// Set the local description on the client
	if err := clientPC.SetLocalDescription(answer); err != nil {
		logger.Error("Failed to set local description on client: %v", err)
		os.Exit(1)
	}

	// Wait for ICE gathering to complete on the client
	logger.Info("Waiting for client ICE gathering to complete...")
	<-webrtc.GatheringCompletePromise(clientPC)
	logger.Info("Client ICE gathering complete")

	// Get the local description after ICE gathering is complete
	answer = *clientPC.LocalDescription()

	// Set the remote description on the server
	if err := serverPC.SetRemoteDescription(answer); err != nil {
		logger.Error("Failed to set remote description on server: %v", err)
		os.Exit(1)
	}

	// Create a channel to signal shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or timeout
	select {
	case <-shutdown:
		logger.Info("Shutting down...")
	case <-time.After(30 * time.Second):
		logger.Info("Test completed")
	}

	// Close the peer connections
	if err := serverPC.Close(); err != nil {
		logger.Error("Error closing server peer connection: %v", err)
	}

	if err := clientPC.Close(); err != nil {
		logger.Error("Error closing client peer connection: %v", err)
	}

	logger.Info("Test shutdown complete")
}
