/*
 *   Copyright (c) 2021 Anton Brekhov anton.brekhov@rsc-tech.ru
 *   All rights reserved.
 */
package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chzyer/readline"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	isOffer bool
)

func init() {
	rootCmd.AddCommand(webrtcCmd)

	// webrtcCmd.Flags().StringVarP(&offerAddr, "offer-address", "o", ":50000", "Offerer addr")
	// webrtcCmd.Flags().StringVarP(&answerAddr, "answer-address", "a", ":60000", "Answer addr")
	webrtcCmd.Flags().BoolVarP(&isOffer, "offerer", "o", false, "IsOfferer?")
}

// webrtcCmd represents the webrtc command
var webrtcCmd = &cobra.Command{
	Use:   "webrtc",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		// Prepare ICE gathering options
		iceOptions := webrtc.ICEGatherOptions{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		}
		// Create an API object
		api := webrtc.NewAPI()
		// Create the ICE gatherer
		gatherer, err := api.NewICEGatherer(iceOptions)
		if err != nil {
			panic(err)
		}
		// Construct the ICE transport
		ice := api.NewICETransport(gatherer)
		// Construct the DTLS transport
		dtls, err := api.NewDTLSTransport(ice, nil)
		if err != nil {
			panic(err)
		}
		// Construct the SCTP transport
		sctp := api.NewSCTPTransport(dtls)
		logrus.Debugf("%#v\n", sctp)

		// Handle incoming data channels
		sctp.OnDataChannel(func(channel *webrtc.DataChannel) {
			fmt.Printf("New DataChannel %s %d\n", channel.Label(), channel.ID())

			// Register the handlers
			channel.OnOpen(handleOnOpen(channel))
			channel.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("Message from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
			})
		})
		gatherFinished := make(chan struct{})
		gatherer.OnLocalCandidate(func(i *webrtc.ICECandidate) {
			if i == nil {
				close(gatherFinished)
			}
		})

		// Gather candidates
		err = gatherer.Gather()
		if err != nil {
			panic(err)
		}

		<-gatherFinished
		iceCandidates, err := gatherer.GetLocalCandidates()
		if err != nil {
			panic(err)
		}

		iceParams, err := gatherer.GetLocalParameters()
		if err != nil {
			panic(err)
		}

		dtlsParams, err := dtls.GetLocalParameters()
		if err != nil {
			panic(err)
		}

		sctpCapabilities := sctp.GetCapabilities()

		s := Signal{
			ICECandidates:    iceCandidates,
			ICEParameters:    iceParams,
			DTLSParameters:   dtlsParams,
			SCTPCapabilities: sctpCapabilities,
		}
		// Exchange the information
		fmt.Println(encode(s))
		remoteSignal := Signal{}
		decode(mustReadStdin(), &remoteSignal)

		iceRole := webrtc.ICERoleControlled
		if isOffer {
			logrus.Debugln("Offer iceRoleControll")
			iceRole = webrtc.ICERoleControlling
			logrus.Debugf("%#v\n", iceRole)
		}
		err = ice.SetRemoteCandidates(remoteSignal.ICECandidates)
		if err != nil {
			panic(err)
		}

		logrus.Debugln("Start ICE TR")
		// Start the ICE transport
		// err = ice.Start(nil, remoteSignal.ICEParameters, &iceRole)
		err = ice.Start(gatherer, remoteSignal.ICEParameters, &iceRole)
		if err != nil {
			panic(err)
		}

		logrus.Debugln("Start DTLS")
		// Start the DTLS transport
		err = dtls.Start(remoteSignal.DTLSParameters)
		if err != nil {
			panic(err)
		}

		logrus.Debugln("Start SCTP")
		// Start the SCTP transport
		err = sctp.Start(remoteSignal.SCTPCapabilities)
		if err != nil {
			panic(err)
		}
		// Construct the data channel as the offerer
		if isOffer {
			logrus.Debugln("Offer data channel started")
			var id uint16 = 1

			dcParams := &webrtc.DataChannelParameters{
				Label: "Foo",
				ID:    &id,
			}
			var channel *webrtc.DataChannel
			channel, err = api.NewDataChannel(sctp, dcParams)
			if err != nil {
				panic(err)
			}

			// Register the handlers
			// channel.OnOpen(handleOnOpen(channel)) // TODO: OnOpen on handle ChannelAck
			go handleOnOpen(channel)() // Temporary alternative
			channel.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Printf("Message from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
			})
		}

		select {}

	},
}

func signalCandidate(addr string, c *webrtc.ICECandidate) error {
	payload := []byte(c.ToJSON().Candidate)
	resp, err := http.Post(fmt.Sprintf("http://%s/candidate", addr), // nolint:noctx
		"application/json; charset=utf-8", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	if closeErr := resp.Body.Close(); closeErr != nil {
		return closeErr
	}

	return nil
}

func onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		logrus.Infof("ICE Connection State has changed: %s\n", connectionState.String())
	}
}

// Signal is used to exchange signaling info.
// This is not part of the ORTC spec. You are free
// to exchange this information any way you want.
type Signal struct {
	ICECandidates    []webrtc.ICECandidate   `json:"iceCandidates"`
	ICEParameters    webrtc.ICEParameters    `json:"iceParameters"`
	DTLSParameters   webrtc.DTLSParameters   `json:"dtlsParameters"`
	SCTPCapabilities webrtc.SCTPCapabilities `json:"sctpCapabilities"`
}

func handleOnOpen(channel *webrtc.DataChannel) func() {
	return func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", channel.Label(), channel.ID())

		for range time.NewTicker(5 * time.Second).C {
			message := "hello"
			fmt.Printf("Sending '%s' \n", message)

			err := channel.SendText(message)
			if err != nil {
				panic(err)
			}
		}
	}
}

func encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	logrus.Debugf("%#v\n", string(b))
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

func decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	logrus.Debugf("%#v\n", string(b))
	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func mustReadStdin() string {
	rl, err := readline.New("Insert SDP: ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	var in string
	line, err := rl.Readline()
	readline.Stdin.Close()
	in = line
	in = strings.TrimSpace(in)
	logrus.Trace("in:", in)
	return in
}
