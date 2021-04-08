/*
Copyright © 2021 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/abrekhov/hypertunnel/pkg/datachannel"
	webrtc "github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// Flags
var (
	cfgFile string
	verbose bool
	key     string
	isOffer bool
	file    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// Use:   "hypertunnel",
	Use:   "ht",
	Short: "P2P secure copy",
	Long:  `HyperTunnel - is P2P secure copy tool. Inspired by magic-wormhole, gfile and so on...`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: Connection,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hypertunnel.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity")
	// rootCmd.Flags().BoolVarP(&isOffer, "offerer", "o", false, "IsOfferer?")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "File to transfer")
	// rootCmd.PersistentFlags().StringVarP(&key, "stun", "s", "stun.l.google.com:19302", "Default stun server")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".hypertunnel" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".hypertunnel")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func Connection(cmd *cobra.Command, args []string) {
	// Who receiver and who sender?
	if file == "" {
		isOffer = false
		log.Infoln("Receiver started...")
	} else {
		isOffer = true
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			log.Panicln("File does not exist.")
		}
		if info.IsDir() {
			log.Panicln("Directory is not yet supported")
		} else {
			log.Infoln("Sender started...")
			log.Debugf("Fileinfo: %#v\n", info)
		}
	}
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
	cobra.CheckErr(err)
	// Construct the ICE transport
	ice := api.NewICETransport(gatherer)
	// Construct the DTLS transport
	dtls, err := api.NewDTLSTransport(ice, nil)
	cobra.CheckErr(err)
	// Construct the SCTP transport
	sctp := api.NewSCTPTransport(dtls)
	log.Debugf("SCTP: %#v\n", sctp)

	// Handle incoming data channels (receiver)
	sctp.OnDataChannel(func(channel *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", channel.Label(), channel.ID())
		log.Debugf("DataChannel Opts: %#v\n", channel)
		_, err := os.Stat(channel.Label())
		if os.IsExist(err) {
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
		// channel.OnOpen(datachannel.HandleOnOpen(channel))
		// channel.OnOpen(func() {
		// 	fmt.Printf("Data channel '%s'-'%d' open. Transfering starting...\n", channel.Label(), channel.ID())
		// })
		channel.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
			fd.Write(msg.Data)
		})
		channel.OnClose(func() {
			fmt.Printf("Data channel '%s'-'%d' closed. Transfering ended...\n", channel.Label(), channel.ID())
			fd.Close()
			os.Exit(0)
			return
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
	cobra.CheckErr(err)

	<-gatherFinished
	iceCandidates, err := gatherer.GetLocalCandidates()
	cobra.CheckErr(err)

	iceParams, err := gatherer.GetLocalParameters()
	cobra.CheckErr(err)

	dtlsParams, err := dtls.GetLocalParameters()
	cobra.CheckErr(err)

	sctpCapabilities := sctp.GetCapabilities()

	s := datachannel.Signal{
		ICECandidates:    iceCandidates,
		ICEParameters:    iceParams,
		DTLSParameters:   dtlsParams,
		SCTPCapabilities: sctpCapabilities,
	}
	// Exchange the information
	fmt.Println(datachannel.Encode(s))
	remoteSignal := datachannel.Signal{}
	datachannel.Decode(datachannel.MustReadStdin(), &remoteSignal)

	iceRole := webrtc.ICERoleControlled
	if isOffer {
		iceRole = webrtc.ICERoleControlling
	}
	err = ice.SetRemoteCandidates(remoteSignal.ICECandidates)
	cobra.CheckErr(err)

	log.Debugln("Start ICE TR")
	// Start the ICE transport
	// err = ice.Start(nil, remoteSignal.ICEParameters, &iceRole)
	err = ice.Start(gatherer, remoteSignal.ICEParameters, &iceRole)
	cobra.CheckErr(err)

	log.Debugln("Start DTLS")
	// Start the DTLS transport
	err = dtls.Start(remoteSignal.DTLSParameters)
	cobra.CheckErr(err)

	log.Debugln("Start SCTP")
	// Start the SCTP transport
	err = sctp.Start(remoteSignal.SCTPCapabilities)
	cobra.CheckErr(err)
	// Construct the data channel as the offerer
	if isOffer {
		var id uint16 = 1
		info, err := os.Stat(file)
		cobra.CheckErr(err)

		dcParams := &webrtc.DataChannelParameters{
			Label: info.Name(),
			ID:    &id,
		}
		// log.Debugf("%#v\n", dcParams)
		log.Debugf("Fileinfo: %#v\n", info)
		var channel *webrtc.DataChannel
		channel, err = api.NewDataChannel(sctp, dcParams)
		cobra.CheckErr(err)

		var fd *os.File
		channel.OnOpen(func() {
			fd, err := os.Open(file)
			defer fd.Close()
			cobra.CheckErr(err)
			r := bufio.NewReader(fd)
			chunk := make([]byte, 1024)
			for {
				nbytes, err := r.Read(chunk)
				log.Debugln("nbytes:", nbytes)
				if err != nil {
					break
				}
				channel.Send(chunk[:nbytes])
			}
			cobra.CheckErr(err)
			err = fd.Close()
			cobra.CheckErr(err)
			channel.Close()
		})
		channel.OnClose(func() {
			fmt.Printf("Chunks from DataChannel '%s' transfered.\n", channel.Label())
			os.Exit(0)
			return
		})
		// Register the handlers
		// channel.OnOpen(handleOnOpen(channel)) // TODO: OnOpen on handle ChannelAck
		// go datachannel.HandleOnOpen(channel)() // Temporary alternative
		// channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// 	fmt.Printf("Chunk from DataChannel '%s': '%s'\n", channel.Label(), string(msg.Data))
		// 	w.Write(msg.Data)
		// })
		defer fd.Close()
	}

	select {}
}

func askForConfirmation(s string, in io.Reader) bool {
	return true
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
