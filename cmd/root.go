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

// Package cmd implements the CLI commands for HyperTunnel.
package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/archive"

	"github.com/abrekhov/hypertunnel/pkg/datachannel"
	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/abrekhov/hypertunnel/pkg/tui"
	"github.com/atotto/clipboard"
	webrtc "github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// Flags
var (
	cfgFile    string
	verbose    bool
	isOffer    bool
	file       string
	autoAccept bool
	noTUI      bool
	useTUI     bool
	autoCopy   bool
	noCopy     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	// Use:   "hypertunnel",
	Use:   "ht",
	Short: "P2P secure copy",
	Long:  `HyperTunnel - is P2P secure copy tool. Inspired by magic-wormhole, gfile and so on...`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
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
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "File to transfer")
	rootCmd.Flags().BoolVar(&autoAccept, "auto-accept", false, "Automatically accept incoming files and overwrites")
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Disable the terminal UI (use plain text output)")
	rootCmd.Flags().BoolVar(&useTUI, "tui", false, "Enable the terminal UI")
	rootCmd.Flags().BoolVar(&autoCopy, "copy", true, "Automatically copy the connection signal when possible")
	rootCmd.Flags().BoolVar(&noCopy, "no-copy", false, "Disable automatic signal copy")
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

// Connection handles the main WebRTC P2P connection logic for file transfer.
func maybeCopySignal(signal string) {
	if !autoCopy {
		return
	}
	if err := clipboard.WriteAll(signal); err != nil {
		fmt.Fprintln(os.Stderr, "Clipboard unavailable; copy manually.")
		return
	}
	fmt.Fprintln(os.Stderr, "Copied to clipboard.")
}

func renderProgress(prefix string, progress *transfer.Progress, stop <-chan struct{}) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			metrics := progress.Metrics()
			fmt.Printf("\r%s\n", transfer.FormatProgressLine(prefix, metrics))
			return
		case <-ticker.C:
			metrics := progress.Metrics()
			fmt.Printf("\r%s", transfer.FormatProgressLine(prefix, metrics))
		}
	}
}

func printTransferSummary(action string, total int64, progress *transfer.Progress) {
	elapsed := progress.Elapsed()
	avgSpeed := float64(total) / elapsed.Seconds()
	fmt.Println()
	fmt.Printf("%s transfer complete\n", action)
	fmt.Printf("File size: %s\n", transfer.FormatSize(total))
	fmt.Printf("Time: %s, Avg: %s\n", transfer.FormatDuration(elapsed), transfer.FormatSpeed(avgSpeed))
}

// Connection handles the main WebRTC P2P connection logic for file transfer.
func Connection(_ *cobra.Command, _ []string) {
	datachannel.AutoAccept = autoAccept
	if noCopy || !term.IsTerminal(int(os.Stdout.Fd())) {
		autoCopy = false
	}

	// Who receiver and who sender?
	var isDirectory bool
	var filesize int64
	if file == "" {
		isOffer = false
		if verbose {
			log.Infoln("Receiver started...")
		} else {
			fmt.Println("Receiver started")
		}
	} else {
		isOffer = true
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			log.Panicln("File does not exist.")
		}
		isDirectory = info.IsDir()
		if !isDirectory {
			filesize = info.Size()
		}
		if isDirectory {
			if verbose {
				log.Infoln("Sender started (directory mode)...")
			} else {
				fmt.Println("Sender started (directory mode)")
			}
			log.Debugf("Directory: %s\n", file)
		} else {
			if verbose {
				log.Infoln("Sender started...")
			} else {
				fmt.Println("Sender started")
			}
			log.Debugf("Fileinfo: %#v\n", info)
		}
	}

	// If TUI is enabled and not in verbose mode, show TUI welcome
	if useTUI && !noTUI && !verbose {
		showTUIWelcome(isOffer, file, filesize)
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
	sctp.OnDataChannel(datachannel.FileTransferHandler)
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
	encodedSignal := datachannel.Encode(s)
	fmt.Printf("Your connection signal:\n\n")
	fmt.Println(encodedSignal)
	fmt.Printf("\n")
	fmt.Println("Paste the peer signal and press Ctrl+D.")
	maybeCopySignal(encodedSignal)

	// Waiting for encoded signal from other side
	remoteSignal := datachannel.Signal{}
	datachannel.Decode(datachannel.MustReadStdin(), &remoteSignal)
	if verbose {
		log.Infoln("Remote signal received.")
	}

	iceRole := webrtc.ICERoleControlled
	if isOffer {
		iceRole = webrtc.ICERoleControlling
	}
	err = ice.SetRemoteCandidates(remoteSignal.ICECandidates)
	cobra.CheckErr(err)

	log.Debugln("Start ICE TR")
	// Start the ICE transport
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
	// Construct the data channel as the sender
	if isOffer {
		var id uint16 = 1
		info, err := os.Stat(file)
		cobra.CheckErr(err)

		// Determine the label (filename to send)
		label := info.Name()
		if isDirectory {
			// For directories, append .tar.gz to indicate it's archived
			label += ".tar.gz"
		}

		dcParams := &webrtc.DataChannelParameters{
			Label:   label,
			ID:      &id,
			Ordered: true,
		}
		log.Debugf("Fileinfo: %#v\n", info)
		var channel *webrtc.DataChannel
		channel, err = api.NewDataChannel(sctp, dcParams)
		cobra.CheckErr(err)

		channel.OnOpen(func() {
			var r io.Reader
			var totalSize int64

			if isDirectory {
				// Create archive on-the-fly
				log.Infoln("Creating archive...")
				var buf bytes.Buffer
				opts := archive.DefaultOptions()
				bytesWritten, err := archive.CreateTarGz(&buf, file, opts)
				if err != nil {
					log.Errorf("Failed to create archive: %v", err)
					if err := channel.Close(); err != nil {
						log.Debugln(err)
					}
					return
				}
				log.Infof("Archive created: %d bytes", bytesWritten)
				r = &buf
				totalSize = bytesWritten
			} else {
				// Regular file transfer
				fd, err := os.Open(file) // #nosec G304 - file path is from user-provided flag
				if err != nil {
					log.Errorf("Failed to open file: %v", err)
					if err := channel.Close(); err != nil {
						log.Debugln(err)
					}
					return
				}
				defer func() {
					if closeErr := fd.Close(); closeErr != nil {
						log.Errorf("Failed to close file: %v", closeErr)
					}
				}()
				r = fd
				totalSize = filesize
			}

			progress := transfer.NewProgress(totalSize)
			progressStop := make(chan struct{})
			go renderProgress("Sending", progress, progressStop)

			// Stream data in chunks
			bufReader := bufio.NewReader(r)
			chunk := make([]byte, 65534)
			totalSent := int64(0)
			for {
				nbytes, readErr := bufReader.Read(chunk)
				log.Debugln("nbytes:", nbytes)
				if nbytes > 0 {
					totalSent += int64(nbytes)
					progress.Update(int64(nbytes))
					if sendErr := channel.Send(chunk[:nbytes]); sendErr != nil {
						log.Debugln(sendErr)
						if err := channel.Close(); err != nil {
							log.Debugln(err)
						}
						break
					}
				}
				if readErr == io.EOF {
					close(progressStop)
					if verbose {
						log.Infof("Transfer complete: %d bytes sent", totalSent)
					}
					printTransferSummary("Sent", totalSent, progress)
					if err := channel.Close(); err != nil {
						log.Debugln(err)
					}
					break
				}
				if readErr != nil {
					close(progressStop)
					log.Errorf("Failed reading: %v", readErr)
					if err := channel.Close(); err != nil {
						log.Debugln(err)
					}
					fmt.Println("Transfer failed. Check the file path and try again.")
					break
				}
			}
		})
		channel.OnClose(func() {
			if verbose {
				fmt.Printf("Ready state of channel: %s", channel.ReadyState().String())
				fmt.Printf("Chunks from DataChannel '%s' transferred.\n", channel.Label())
			}
			os.Exit(0)
		})
	}

	select {}
}

// showTUIWelcome displays a welcome screen using the TUI
func showTUIWelcome(isOffer bool, filename string, filesize int64) {
	// Create a simple welcome model
	m := tui.NewModel(isOffer, filename, filesize)

	// Just show the initial view without running the full program
	// This is a simple static display for Phase 2
	// Full integration with real-time updates will come in future phases
	fmt.Println(m.View())
	fmt.Println()
}
