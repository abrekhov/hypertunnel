package datachannel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/archive"
	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// FileTransferHandler handles incoming file transfers on a WebRTC data channel.
func FileTransferHandler(channel *webrtc.DataChannel) {
	if log.IsLevelEnabled(log.DebugLevel) {
		fmt.Printf("New DataChannel %s %d\n", channel.Label(), channel.ID())
		log.Debugf("DataChannel Opts: %#v\n", channel)
	}

	// Detect if this is a directory archive
	isArchive := strings.HasSuffix(channel.Label(), ".tar.gz")
	targetPath := channel.Label()

	if isArchive {
		// Remove .tar.gz suffix to get directory name
		targetPath = strings.TrimSuffix(channel.Label(), ".tar.gz")
		if log.IsLevelEnabled(log.DebugLevel) {
			fmt.Printf("Receiving directory: %s (archived)\n", targetPath)
		}
	}

	// Check if target already exists
	_, err := os.Stat(targetPath)
	targetExists := err == nil
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to check existing %s: %v", targetPath, err)
		return
	}

	if AutoAccept {
		if targetExists {
			if log.IsLevelEnabled(log.DebugLevel) {
				log.Infof("Auto-accept enabled: overwriting existing %s %s",
					map[bool]string{true: "directory", false: "file"}[isArchive],
					targetPath)
			}
		} else {
			if log.IsLevelEnabled(log.DebugLevel) {
				log.Infof("Auto-accept enabled: accepting incoming %s %s",
					map[bool]string{true: "directory", false: "file"}[isArchive],
					targetPath)
			}
		}
	} else {
		if log.IsLevelEnabled(log.DebugLevel) {
			log.Infof("Prompting to accept incoming %s %s",
				map[bool]string{true: "directory", false: "file"}[isArchive],
				targetPath)
		}

		accept := askForConfirmation(fmt.Sprintf("Receive %s %s?",
			map[bool]string{true: "directory", false: "file"}[isArchive],
			targetPath), os.Stdin)
		if !accept {
			if log.IsLevelEnabled(log.InfoLevel) {
				log.Infoln("Transfer declined; ignoring incoming data channel.")
			}
			fmt.Println("Transfer declined.")

			return
		}
		if log.IsLevelEnabled(log.DebugLevel) {
			log.Infoln("Transfer accepted; starting receive.")
		}
		if targetExists {
			if log.IsLevelEnabled(log.DebugLevel) {
				log.Infof("Existing %s detected: prompting for overwrite", targetPath)
			}
			overwrite := askForConfirmation(fmt.Sprintf("%s %s exists. Overwrite?",
				map[bool]string{true: "directory", false: "file"}[isArchive],
				targetPath), os.Stdin)
			if !overwrite {
				if log.IsLevelEnabled(log.InfoLevel) {
					log.Infoln("Overwrite declined; ignoring incoming transfer.")
				}
				fmt.Println("Transfer declined.")

				return
			}
			if log.IsLevelEnabled(log.DebugLevel) {
				log.Infoln("Overwrite confirmed; proceeding with transfer.")
			}
		} else if log.IsLevelEnabled(log.DebugLevel) {
			log.Debugf("No existing target at %s", targetPath)
		}
	}

	if isArchive {
		// For archives, collect data in memory then extract
		handleArchiveTransfer(channel, targetPath)
	} else {
		// For regular files, write directly to disk
		handleFileTransfer(channel, targetPath)
	}
}

// handleFileTransfer handles receiving a regular file.
func handleFileTransfer(channel *webrtc.DataChannel, targetPath string) {
	fd, err := os.Create(targetPath) // #nosec G304 - targetPath is from datachannel label, validated before
	if err != nil {
		log.Errorf("Failed to create file: %v", err)
		return
	}

	progress := transfer.NewProgress(0)
	progressStop := make(chan struct{})
	go renderReceiveProgress(progress, progressStop)

	// Register the handlers
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if _, err := fd.Write(msg.Data); err != nil {
			log.Errorf("Failed to write data: %v", err)
			return
		}
		progress.Update(int64(len(msg.Data)))
	})

	channel.OnClose(func() {
		close(progressStop)
		if log.IsLevelEnabled(log.DebugLevel) {
			fmt.Printf("Data channel '%s'-'%d' closed. Transfer complete.\n", channel.Label(), channel.ID())
		}
		if err := fd.Close(); err != nil {
			log.Errorf("Failed to close file: %v", err)
		}
		printReceiveSummary(channel.Label(), progress)
		os.Exit(0)
	})
}

func renderReceiveProgress(progress *transfer.Progress, stop <-chan struct{}) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			metrics := progress.Metrics()
			fmt.Printf("\r%s\n", transfer.FormatProgressLine("Receiving", metrics))
			return
		case <-ticker.C:
			metrics := progress.Metrics()
			fmt.Printf("\r%s", transfer.FormatProgressLine("Receiving", metrics))
		}
	}
}

func printReceiveSummary(name string, progress *transfer.Progress) {
	metrics := progress.Metrics()
	elapsed := progress.Elapsed()
	avgSpeed := 0.0
	if elapsed.Seconds() > 0 {
		avgSpeed = float64(metrics.TransferredBytes) / elapsed.Seconds()
	}
	fmt.Println()
	fmt.Println("Receive complete")
	fmt.Printf("File: %s (%s)\n", name, transfer.FormatSize(metrics.TransferredBytes))
	fmt.Printf("Time: %s, Avg: %s\n", transfer.FormatDuration(elapsed), transfer.FormatSpeed(avgSpeed))
}

// handleArchiveTransfer handles receiving and extracting a directory archive.
func handleArchiveTransfer(channel *webrtc.DataChannel, targetPath string) {
	// Collect all data in a buffer
	var buf bytes.Buffer
	totalReceived := int64(0)

	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		n, err := buf.Write(msg.Data)
		if err != nil {
			log.Errorf("Failed to buffer data: %v", err)
			return
		}
		totalReceived += int64(n)
		log.Debugf("Received: %d bytes (total: %d)", n, totalReceived)
	})

	channel.OnClose(func() {
		if log.IsLevelEnabled(log.DebugLevel) {
			fmt.Printf("Data channel '%s'-'%d' closed.\n", channel.Label(), channel.ID())
			fmt.Printf("Total received: %d bytes\n", totalReceived)
			fmt.Println("Extracting archive...")
		}

		// Extract the archive
		opts := archive.DefaultOptions()

		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			log.Errorf("Failed to get current directory: %v", err)
			os.Exit(1)
		}

		// Create the target directory if it doesn't exist
		destPath := filepath.Join(currentDir, targetPath)
		if err := os.MkdirAll(destPath, 0750); err != nil {
			log.Errorf("Failed to create directory: %v", err)
			os.Exit(1)
		}

		// Extract archive
		if err := archive.ExtractTarGz(&buf, destPath, opts); err != nil {
			log.Errorf("Failed to extract archive: %v", err)
			os.Exit(1)
		}

		if log.IsLevelEnabled(log.DebugLevel) {
			fmt.Printf("Directory extracted to: %s\n", destPath)
		}
		os.Exit(0)
	})
}

// AutoAccept controls whether to automatically accept incoming file transfers without prompting.
var AutoAccept bool

func askForConfirmation(s string, in io.Reader) bool {
	tries := 3
	reader := bufio.NewReader(in)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [Y/n]: ", s)

		res, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Empty input (i.e. "\n") defaults to yes
		if len(res) < 2 {
			return true
		}

		trimmed := strings.ToLower(strings.TrimSpace(res))
		if trimmed == "y" || trimmed == "yes" {
			return true
		}
		if trimmed == "n" || trimmed == "no" {
			return false
		}
	}

	return false
}
