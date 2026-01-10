package datachannel

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func FileTransferHandler(channel *webrtc.DataChannel) {
	fmt.Printf("New DataChannel %s %d\n", channel.Label(), channel.ID())
	log.Debugf("DataChannel Opts: %#v\n", channel)
	_, err := os.Stat(channel.Label())
	if err == nil {
		if !AutoAccept {
			overwrite := askForConfirmation(fmt.Sprintf("File %s exists. Overwrite?", channel.Label()), os.Stdin)
			if !overwrite {
				fmt.Println("OK! Ignoring...")
				return
			}
		}
	} else if !os.IsNotExist(err) {
		log.Errorf("Failed to check existing file %s: %v", channel.Label(), err)
		return
	}
	if !AutoAccept {
		c := askForConfirmation(fmt.Sprintf("Do you want to receive the file %s?", channel.Label()), os.Stdin)
		if !c {
			fmt.Println("OK! Ignoring...")
			return
		}
	}

	var fd *os.File
	fd, err = os.Create(channel.Label())
	cobra.CheckErr(err)
	// Register the handlers
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// fmt.Printf("Message from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
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
