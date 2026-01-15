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

	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

func FileTransferHandler(channel *webrtc.DataChannel) {
	fmt.Printf("New DataChannel %s %d\n", channel.Label(), channel.ID())
	log.Debugf("DataChannel Opts: %#v\n", channel)

	// Detect if this is a directory archive
	isArchive := strings.HasSuffix(channel.Label(), ".tar.gz")
	targetPath := channel.Label()

	if isArchive {
		// Remove .tar.gz suffix to get directory name
		targetPath = strings.TrimSuffix(channel.Label(), ".tar.gz")
		fmt.Printf("Receiving directory: %s (archived)\n", targetPath)
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
			log.Infof("Auto-accept enabled: overwriting existing %s %s",
				map[bool]string{true: "directory", false: "file"}[isArchive],
				targetPath)
		} else {
			log.Infof("Auto-accept enabled: accepting incoming %s %s",
				map[bool]string{true: "directory", false: "file"}[isArchive],
				targetPath)
		}
	} else {
		log.Infof("Prompting to accept incoming %s %s",
			map[bool]string{true: "directory", false: "file"}[isArchive],
			targetPath)
		accept := askForConfirmation(fmt.Sprintf("Do you want to receive %s %s?",
			map[bool]string{true: "directory", false: "file"}[isArchive],
			targetPath), os.Stdin)
		if !accept {
			log.Infoln("Transfer declined; ignoring incoming data channel.")
			fmt.Println("OK! Ignoring...")
			return
		}
		log.Infoln("Transfer accepted; starting receive.")
		if targetExists {
			log.Infof("Existing %s detected: prompting for overwrite", targetPath)
			overwrite := askForConfirmation(fmt.Sprintf("%s %s exists. Overwrite?",
				map[bool]string{true: "Directory", false: "File"}[isArchive],
				targetPath), os.Stdin)
			if !overwrite {
				log.Infoln("Overwrite declined; ignoring incoming transfer.")
				fmt.Println("OK! Ignoring...")
				return
			}
			log.Infoln("Overwrite confirmed; proceeding with transfer.")
		} else {
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

	var fd *os.File
	fd, err = os.Create(channel.Label())
	cobra.CheckErr(err)
	progress := transfer.NewProgress(0)
	logInterval := 2 * time.Second
	nextLog := time.Now().Add(logInterval)
	// Register the handlers
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if _, err := fd.Write(msg.Data); err != nil {
			log.Errorf("Failed to write data: %v", err)
		}
		progress.Add(len(msg.Data))
		now := time.Now()
		if now.After(nextLog) {
			metrics := progress.Snapshot(now)
			log.WithFields(log.Fields{
				"received_bytes":   metrics.TransferredBytes,
				"bytes_per_second": metrics.BytesPerSecond,
				"elapsed":          metrics.Elapsed.String(),
			}).Infoln("Transfer progress (receive)")
			nextLog = now.Add(logInterval)
		}
	})

	channel.OnClose(func() {
		fmt.Printf("Data channel '%s'-'%d' closed. Transfer complete.\n", channel.Label(), channel.ID())
		if err := fd.Close(); err != nil {
			log.Errorf("Failed to close file: %v", err)
		}
		os.Exit(0)
	})
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
		fmt.Printf("Data channel '%s'-'%d' closed.\n", channel.Label(), channel.ID())
		fmt.Printf("Total received: %d bytes\n", totalReceived)
		fmt.Println("Extracting archive...")

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

		fmt.Printf("Directory extracted to: %s\n", destPath)
		os.Exit(0)
	})
}

var AutoAccept bool

func askForConfirmation(s string, in io.Reader) bool {
	tries := 3
	reader := bufio.NewReader(in)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [y/n]: ", s)

		res, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		return strings.ToLower(strings.TrimSpace(res))[0] == 'y'
	}

	return false
}
