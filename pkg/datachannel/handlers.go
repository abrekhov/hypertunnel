package datachannel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/abrekhov/hypertunnel/pkg/archive"
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
	if err == nil {
		if !OverwriteExisting {
			overwrite := askForConfirmation(fmt.Sprintf("%s %s exists. Overwrite?",
				map[bool]string{true: "Directory", false: "File"}[isArchive],
				targetPath), os.Stdin)
			if !overwrite {
				fmt.Println("OK! Ignoring...")
				return
			}
		}

		if isArchive {
			if err := os.RemoveAll(targetPath); err != nil {
				log.Errorf("Failed to remove existing directory: %v", err)
				return
			}
		}
	} else if !os.IsNotExist(err) {
		log.Errorf("Failed to check existing %s: %v", targetPath, err)
		return
	}

	if !AutoAccept {
		c := askForConfirmation(fmt.Sprintf("Do you want to receive %s %s?",
			map[bool]string{true: "directory", false: "file"}[isArchive],
			targetPath), os.Stdin)
		if !c {
			fmt.Println("OK! Ignoring...")
			return
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

	// Register the handlers
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if _, err := fd.Write(msg.Data); err != nil {
			log.Errorf("Failed to write data: %v", err)
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
var OverwriteExisting bool

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
