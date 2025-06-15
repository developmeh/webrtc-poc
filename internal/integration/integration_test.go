package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/paulscoder/webrtc-poc/internal/client"
	"github.com/paulscoder/webrtc-poc/internal/logger"
	"github.com/pion/webrtc/v3"
)

// TestEndToEndFileTransfer tests the end-to-end file transfer functionality
// This test creates a server and client in the same process and transfers a file
func TestEndToEndFileTransfer(t *testing.T) {
	// Initialize logger
	logger.Init()

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-transfer-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content to the file
	testContent := []string{
		"Line 1 of the test file",
		"Line 2 of the test file",
		"Line 3 of the test file",
		"This is a longer line with some special characters: !@#$%^&*()",
		"WebRTC is a free, open-source project that provides real-time communication",
		"It allows audio and video communication to work inside web pages",
		"In this test, we're using WebRTC data channels to stream a text file",
	}
	for _, line := range testContent {
		tmpFile.WriteString(line + "\n")
	}
	tmpFile.Close()

	// Create a temporary output file
	outputFile, err := os.CreateTemp("", "test-output-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp output file: %v", err)
	}
	outputFile.Close()
	defer os.Remove(outputFile.Name())

	// Start a test HTTP server for signaling
	serverOfferChan := make(chan webrtc.SessionDescription)
	clientAnswerChan := make(chan webrtc.SessionDescription)
	signalDone := make(chan struct{})

	// Create a mutex to protect the channels
	var mu sync.Mutex

	// Create an HTTP server for signaling
	http.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		// Read the offer from the request
		var offer webrtc.SessionDescription
		err := readJSON(r, &offer)
		if err != nil {
			http.Error(w, "Failed to read offer: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Send the offer to the server
		serverOfferChan <- offer

		// Wait for the answer from the server
		answer := <-clientAnswerChan

		// Send the answer to the client
		writeJSON(w, answer)
	})

	// Start the HTTP server
	server := &http.Server{Addr: ":18080"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", err)
		}
	}()
	defer server.Close()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Create a wait group to wait for the test to complete
	var wg sync.WaitGroup
	wg.Add(2)

	// Start the server in a goroutine
	go func() {
		defer wg.Done()

		// Create a new peer connection
		peerConnection, err := createPeerConnection()
		if err != nil {
			t.Errorf("Failed to create server peer connection: %v", err)
			return
		}
		defer peerConnection.Close()

		// Wait for the offer from the client
		offer := <-serverOfferChan

		// Set the remote description
		if err := peerConnection.SetRemoteDescription(offer); err != nil {
			t.Errorf("Failed to set remote description on server: %v", err)
			return
		}

		// Create a data channel
		dataChannel, err := peerConnection.CreateDataChannel("fileStream", nil)
		if err != nil {
			t.Errorf("Failed to create data channel: %v", err)
			return
		}

		// Set up data channel handlers
		dataChannel.OnOpen(func() {
			t.Log("Server data channel opened")

			// Stream the file
			go func() {
				// Create a LineWriter adapter for the data channel
				writer := &webrtcLineWriter{dataChannel: dataChannel}

				// Stream the file with minimal delay for testing
				err := StreamFile(writer, tmpFile.Name(), 1)
				if err != nil {
					t.Errorf("StreamFile returned error: %v", err)
				}

				// Close the data channel when done
				dataChannel.Close()
			}()
		})

		// Create an answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			t.Errorf("Failed to create answer: %v", err)
			return
		}

		// Set the local description
		if err := peerConnection.SetLocalDescription(answer); err != nil {
			t.Errorf("Failed to set local description on server: %v", err)
			return
		}

		// Wait for ICE gathering to complete
		<-webrtc.GatheringCompletePromise(peerConnection)

		// Get the local description after ICE gathering is complete
		answer = *peerConnection.LocalDescription()

		// Send the answer to the client
		mu.Lock()
		clientAnswerChan <- answer
		mu.Unlock()

		// Wait for the signal that the test is done
		<-signalDone
	}()

	// Start the client in a goroutine
	go func() {
		defer wg.Done()

		// Create a new peer connection
		peerConnection, err := createPeerConnection()
		if err != nil {
			t.Errorf("Failed to create client peer connection: %v", err)
			return
		}
		defer peerConnection.Close()

		// Create a channel to receive data
		linesChan := make(chan string)
		errChan := make(chan error, 1)

		// Set up data channel handler
		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			t.Logf("Client received data channel: %s", d.Label())

			d.OnOpen(func() {
				t.Log("Client data channel opened")
			})

			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				data := string(msg.Data)
				linesChan <- data
			})

			d.OnClose(func() {
				t.Log("Client data channel closed")
				close(linesChan)
			})
		})

		// Create an offer
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			t.Errorf("Failed to create offer: %v", err)
			return
		}

		// Set the local description
		if err := peerConnection.SetLocalDescription(offer); err != nil {
			t.Errorf("Failed to set local description on client: %v", err)
			return
		}

		// Wait for ICE gathering to complete
		<-webrtc.GatheringCompletePromise(peerConnection)

		// Get the local description after ICE gathering is complete
		offer = *peerConnection.LocalDescription()

		// Create a LineReceiver adapter for the channels
		receiver := &channelLineReceiver{
			linesChan: linesChan,
			errChan:   errChan,
		}

		// Process the lines in a goroutine
		go func() {
			lineCount, _, err := client.ProcessLines(receiver, outputFile.Name())
			if err != nil {
				t.Errorf("ProcessLines returned error: %v", err)
			}

			if lineCount != len(testContent) {
				t.Errorf("Expected %d lines, got %d", len(testContent), lineCount)
			}

			// Signal that the test is done
			close(signalDone)
		}()

		// Send the offer to the server via HTTP
		resp, err := http.Post("http://localhost:18080/offer", "application/json", strings.NewReader(fmt.Sprintf(`{"type":"%s","sdp":"%s"}`, offer.Type.String(), offer.SDP)))
		if err != nil {
			t.Errorf("Failed to send offer: %v", err)
			return
		}
		defer resp.Body.Close()

		// Read the answer
		var answer webrtc.SessionDescription
		err = readJSONFromReader(resp.Body, &answer)
		if err != nil {
			t.Errorf("Failed to read answer: %v", err)
			return
		}

		// Set the remote description
		if err := peerConnection.SetRemoteDescription(answer); err != nil {
			t.Errorf("Failed to set remote description on client: %v", err)
			return
		}
	}()

	// Wait for both goroutines to complete with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out")
	}

	// Verify the output file
	content, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Split the content into lines
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Check that all lines were received
	if len(lines) != len(testContent) {
		t.Errorf("Expected %d lines in output file, got %d", len(testContent), len(lines))
	}

	// Check content of lines
	for i, line := range testContent {
		if i < len(lines) && lines[i] != line {
			t.Errorf("Line %d: expected '%s', got '%s'", i+1, line, lines[i])
		}
	}
}

// createPeerConnection creates a new WebRTC peer connection for testing
func createPeerConnection() (*webrtc.PeerConnection, error) {
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
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{}, // Empty ICE servers list - no STUN/TURN
	}

	// Create a new API with the custom settings
	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	// Create a new peer connection
	return api.NewPeerConnection(config)
}

// webrtcLineWriter adapts a WebRTC data channel to the LineWriter interface
type webrtcLineWriter struct {
	dataChannel *webrtc.DataChannel
}

// SendText implements the server.LineWriter interface
func (w *webrtcLineWriter) SendText(text string) error {
	return w.dataChannel.SendText(text)
}

// channelLineReceiver adapts channels to the LineReceiver interface
type channelLineReceiver struct {
	linesChan <-chan string
	errChan   <-chan error
}

// ReceiveLines implements the client.LineReceiver interface
func (r *channelLineReceiver) ReceiveLines() (<-chan string, <-chan error) {
	return r.linesChan, r.errChan
}

// Helper functions for JSON handling
func readJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func readJSONFromReader(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func writeJSON(w http.ResponseWriter, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

// StreamFile streams a file line by line to the provided writer
// This is a simplified version for testing
func StreamFile(writer LineWriter, filename string, delayMs int) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if err := writer.SendText(line); err != nil {
			return err
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	return scanner.Err()
}

// LineWriter is an interface for writing lines of text
type LineWriter interface {
	SendText(text string) error
}
