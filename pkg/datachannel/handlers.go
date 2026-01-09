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
		// File exists
		log.Panicln("File with same name exists in current directory.")
	}
	c := askForConfirmation(fmt.Sprintf("Do you want to receive the file %s?", channel.Label()), os.Stdin)
	if !c {
		fmt.Println("OK! Ignoring...")
		return
	}

	var fd *os.File
	fd, err = os.Create(channel.Label())
	cobra.CheckErr(err)
	// Register the handlers
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// fmt.Printf("Message from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
		fd.Write(msg.Data)
	})
	channel.OnClose(func() {
		fmt.Printf("Data channel '%s'-'%d' closed. Transfering ended...\n", channel.Label(), channel.ID())
		fd.Close()
		os.Exit(0)
	})
}

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
