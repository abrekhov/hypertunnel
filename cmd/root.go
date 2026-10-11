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
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
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
	cfgFile       string
	verbose       bool
	isOffer       bool
	file          string
	autoAccept    bool
	noTUI         bool
	useTUI        bool
	autoCopy      bool
	noCopy        bool
	autoAcceptTTL time.Duration
	icePortRange  string
	iceLoopback   bool
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
	RunE: Connection,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.SilenceUsage = true
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
	rootCmd.Flags().DurationVar(&autoAcceptTTL, "accept-timeout", 20*time.Second, "Auto-accept incoming files after this timeout (0 disables)")
	rootCmd.Flags().StringVar(&icePortRange, "ice-port-range", "", "Restrict local ICE UDP ports, e.g. 443:443 (for firewalls that allow only specific UDP ports)")
	rootCmd.Flags().BoolVar(&iceLoopback, "ice-loopback", false, "Use loopback ICE candidates only")
	_ = rootCmd.Flags().MarkHidden("ice-loopback")
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

func decideICERole(localICE webrtc.ICEParameters, localDTLS webrtc.DTLSParameters, remoteICE webrtc.ICEParameters, remoteDTLS webrtc.DTLSParameters) (webrtc.ICERole, error) {
	localKey := iceRoleKey(localICE, localDTLS)
	remoteKey := iceRoleKey(remoteICE, remoteDTLS)
	switch {
	case localKey > remoteKey:
		return webrtc.ICERoleControlling, nil
	case localKey < remoteKey:
		return webrtc.ICERoleControlled, nil
	default:
		return webrtc.ICERoleControlled, fmt.Errorf("peer signal matches this node; paste the other peer's signal")
	}
}

func iceRoleKey(iceParams webrtc.ICEParameters, dtlsParams webrtc.DTLSParameters) string {
	var buf bytes.Buffer
	buf.WriteString(iceParams.UsernameFragment)
	buf.WriteByte('|')
	buf.WriteString(iceParams.Password)
	buf.WriteByte('|')
	fps := make([]string, 0, len(dtlsParams.Fingerprints))
	for _, fp := range dtlsParams.Fingerprints {
		fps = append(fps, fp.Algorithm+":"+fp.Value)
	}
	sort.Strings(fps)
	for i, fp := range fps {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(fp)
	}
	return buf.String()
}

// Connection handles the main WebRTC P2P connection logic for file transfer.
func Connection(_ *cobra.Command, _ []string) error {
	datachannel.AutoAccept = autoAccept
	datachannel.AcceptTimeout = autoAcceptTTL
	if noCopy || !term.IsTerminal(int(os.Stdout.Fd())) {
		autoCopy = false
	}

	// Who receiver and who sender?
	var isDirectory bool
	var filesize int64
	var info os.FileInfo
	if file == "" {
		isOffer = false
		if verbose {
			log.Infoln("Receiver started...")
		} else {
			fmt.Println("Receiver started")
		}
	} else {
		isOffer = true
		var err error
		info, err = os.Stat(file)
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
	iceServers := []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}
	if iceLoopback {
		iceServers = nil
	}
	iceOptions := webrtc.ICEGatherOptions{
		ICEServers: iceServers,
	}
	// Create an API object
	api := webrtc.NewAPI()
	if icePortRange != "" || iceLoopback {
		se := webrtc.SettingEngine{}
		if iceLoopback {
			se.SetIncludeLoopbackCandidate(true)
			se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
			se.SetInterfaceFilter(func(name string) bool { return strings.HasPrefix(name, "lo") })
		}
		if icePortRange != "" {
			lo, hi, err := parsePortRange(icePortRange)
			cobra.CheckErr(err)
			cobra.CheckErr(se.SetEphemeralUDPPortRange(lo, hi))
		}
		api = webrtc.NewAPI(webrtc.WithSettingEngine(se))
	}
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

	done := make(chan error, 1)
	finish := func(err error) {
		select {
		case done <- err:
		default:
		}
	}
	defer func() {
		if err := dtls.Stop(); err != nil {
			log.Debugln(err)
		}
		if err := sctp.Stop(); err != nil {
			log.Debugln(err)
		}
		if err := ice.Stop(); err != nil {
			log.Debugln(err)
		}
	}()
	sctp.OnDataChannel(func(channel *webrtc.DataChannel) {
		datachannel.FileTransferHandler(channel, finish)
	})
	ice.OnConnectionStateChange(func(state webrtc.ICETransportState) {
		switch state {
		case webrtc.ICETransportStateFailed:
			finish(fmt.Errorf("ICE connection state: %s", state))
		default:
		}
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
	encodedSignal := datachannel.Encode(s)
	fmt.Printf("Your connection signal:\n\n")
	fmt.Println(encodedSignal)
	fmt.Printf("\n")
	fmt.Println("Paste the peer signal and press Ctrl+D.")
	maybeCopySignal(encodedSignal)

	// Waiting for encoded signal from other side
	remoteSignal := datachannel.Signal{}
	datachannel.Decode(datachannel.MustReadStdin(), &remoteSignal)
	log.Infoln("Remote signal received.")
	iceRole, err := decideICERole(iceParams, dtlsParams, remoteSignal.ICEParameters, remoteSignal.DTLSParameters)
	cobra.CheckErr(err)
	log.Debugf("Selected ICE role: %s", iceRole)
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
		var md *transfer.Metadata
		var src io.Reader
		var totalSize int64
		if isDirectory {
			log.Infoln("Creating archive...")
			var buf bytes.Buffer
			opts := archive.DefaultOptions()
			_, err := archive.CreateTarGz(&buf, file, opts)
			if err != nil {
				return fmt.Errorf("create archive: %w", err)
			}
			log.Infof("Archive created: %d bytes", buf.Len())
			sum := sha256.Sum256(buf.Bytes())
			md = transfer.NewMetadata(info.Name()+".tar.gz", int64(buf.Len())).WithChecksum(transfer.ChecksumToHex(sum[:]))
			md.IsArchive = true
			src = &buf
			totalSize = int64(buf.Len())
		} else {
			sum, err := transfer.CalculateFileChecksum(file)
			if err != nil {
				return fmt.Errorf("calculate checksum: %w", err)
			}
			md = transfer.NewMetadata(info.Name(), filesize).WithChecksum(transfer.ChecksumToHex(sum))
			fd, err := os.Open(file) // #nosec G304 - file path is from user-provided flag
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer func() {
				if closeErr := fd.Close(); closeErr != nil {
					log.Errorf("Failed to close file: %v", closeErr)
				}
			}()
			src = fd
			totalSize = filesize
		}

		var id uint16 = 1
		dcParams := &webrtc.DataChannelParameters{
			Label:   md.Filename,
			ID:      &id,
			Ordered: true,
		}
		var channel *webrtc.DataChannel
		channel, err = api.NewDataChannel(sctp, dcParams)
		cobra.CheckErr(err)

		progress := transfer.NewProgress(totalSize)
		progressStop := make(chan struct{})
		go renderProgress("Sending", progress, progressStop)
		channel.OnOpen(func() {
			datachannel.SendFile(channel, md, &progressReader{r: src, progress: progress}, func(err error) {
				close(progressStop)
				if err == nil {
					printTransferSummary("Sent", progress.Metrics().TransferredBytes, progress)
				}
				finish(err)
			})
		})
	}

	return <-done
}

type progressReader struct {
	r        io.Reader
	progress *transfer.Progress
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.progress.Update(int64(n))
	}
	return n, err
}

func parsePortRange(s string) (uint16, uint16, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid -ice-port-range %q: want min:max", s)
	}
	lo, err := strconv.Atoi(parts[0])
	if err != nil || lo < 0 || lo > 65535 {
		return 0, 0, fmt.Errorf("invalid -ice-port-range %q: bad min port", s)
	}
	hi, err := strconv.Atoi(parts[1])
	if err != nil || hi < 0 || hi > 65535 || hi < lo {
		return 0, 0, fmt.Errorf("invalid -ice-port-range %q: bad max port", s)
	}
	return uint16(lo), uint16(hi), nil
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
