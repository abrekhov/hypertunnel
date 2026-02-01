//go:build integration

/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */

package main

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/datachannel"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

type webrtcPeer struct {
	api      *webrtc.API
	gatherer *webrtc.ICEGatherer
	ice      *webrtc.ICETransport
	dtls     *webrtc.DTLSTransport
	sctp     *webrtc.SCTPTransport
	signal   datachannel.Signal
}

func newWebRTCPeer(t *testing.T) *webrtcPeer {
	t.Helper()

	api := webrtc.NewAPI()
	gatherer, err := api.NewICEGatherer(webrtc.ICEGatherOptions{})
	require.NoError(t, err)

	ice := api.NewICETransport(gatherer)
	dtls, err := api.NewDTLSTransport(ice, nil)
	require.NoError(t, err)

	sctp := api.NewSCTPTransport(dtls)

	gatherFinished := make(chan struct{})
	gatherer.OnLocalCandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			close(gatherFinished)
		}
	})
	require.NoError(t, gatherer.Gather())
	<-gatherFinished

	iceCandidates, err := gatherer.GetLocalCandidates()
	require.NoError(t, err)

	iceParams, err := gatherer.GetLocalParameters()
	require.NoError(t, err)

	dtlsParams, err := dtls.GetLocalParameters()
	require.NoError(t, err)

	return &webrtcPeer{
		api:      api,
		gatherer: gatherer,
		ice:      ice,
		dtls:     dtls,
		sctp:     sctp,
		signal: datachannel.Signal{
			ICECandidates:    iceCandidates,
			ICEParameters:    iceParams,
			DTLSParameters:   dtlsParams,
			SCTPCapabilities: sctp.GetCapabilities(),
		},
	}
}

func startPeer(t *testing.T, peer *webrtcPeer, remote datachannel.Signal, role webrtc.ICERole) {
	t.Helper()

	require.NoError(t, peer.ice.SetRemoteCandidates(remote.ICECandidates))
	require.NoError(t, peer.ice.Start(peer.gatherer, remote.ICEParameters, &role))
	require.NoError(t, peer.dtls.Start(remote.DTLSParameters))
	require.NoError(t, peer.sctp.Start(remote.SCTPCapabilities))
}

func TestWebRTCFileTransferLocalE2E(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.txt")
	receiverDir := filepath.Join(tempDir, "receiver")
	require.NoError(t, os.MkdirAll(receiverDir, 0750))

	payload := []byte("local webrtc transfer payload")
	require.NoError(t, os.WriteFile(sourcePath, payload, 0644))

	sender := newWebRTCPeer(t)
	receiver := newWebRTCPeer(t)

	done := make(chan error, 1)
	var doneOnce sync.Once
	sendDone := func(err error) {
		doneOnce.Do(func() {
			done <- err
		})
	}

	receiver.sctp.OnDataChannel(func(dc *webrtc.DataChannel) {
		targetPath := filepath.Join(receiverDir, dc.Label())
		fd, err := os.Create(targetPath)
		if err != nil {
			sendDone(err)
			return
		}

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if _, err := fd.Write(msg.Data); err != nil {
				sendDone(err)
			}
		})

		dc.OnClose(func() {
			sendDone(fd.Close())
		})
	})

	startPeer(t, receiver, sender.signal, webrtc.ICERoleControlled)
	startPeer(t, sender, receiver.signal, webrtc.ICERoleControlling)

	var id uint16 = 1
	dcParams := &webrtc.DataChannelParameters{
		Label:   filepath.Base(sourcePath),
		ID:      &id,
		Ordered: true,
	}
	dataChannel, err := sender.api.NewDataChannel(sender.sctp, dcParams)
	require.NoError(t, err)

	dataChannel.OnOpen(func() {
		fd, err := os.Open(sourcePath)
		if err != nil {
			sendDone(err)
			return
		}
		defer fd.Close()

		chunk := make([]byte, 65534)
		for {
			n, readErr := fd.Read(chunk)
			if n > 0 {
				if err := dataChannel.Send(chunk[:n]); err != nil {
					sendDone(err)
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					sendDone(readErr)
					return
				}
				break
			}
		}
		sendDone(dataChannel.Close())
	})

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for transfer to complete")
	}

	receivedPath := filepath.Join(receiverDir, filepath.Base(sourcePath))
	receivedData, err := os.ReadFile(receivedPath)
	require.NoError(t, err)
	require.Equal(t, payload, receivedData)
}
