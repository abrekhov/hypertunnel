//go:build integration
// +build integration

package datachannel

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

type webrtcPeer struct {
	api      *webrtc.API
	gatherer *webrtc.ICEGatherer
	ice      *webrtc.ICETransport
	dtls     *webrtc.DTLSTransport
	sctp     *webrtc.SCTPTransport
}

func newWebRTCPeer(t *testing.T) *webrtcPeer {
	t.Helper()

	se := webrtc.SettingEngine{}
	se.SetIncludeLoopbackCandidate(true)
	se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})

	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))
	gatherer, err := api.NewICEGatherer(webrtc.ICEGatherOptions{})
	require.NoError(t, err)

	ice := api.NewICETransport(gatherer)
	dtls, err := api.NewDTLSTransport(ice, nil)
	require.NoError(t, err)
	sctp := api.NewSCTPTransport(dtls)

	return &webrtcPeer{
		api:      api,
		gatherer: gatherer,
		ice:      ice,
		dtls:     dtls,
		sctp:     sctp,
	}
}

func gatherSignal(ctx context.Context, t *testing.T, p *webrtcPeer) Signal {
	t.Helper()

	gatherFinished := make(chan struct{})
	p.gatherer.OnLocalCandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			close(gatherFinished)
		}
	})

	require.NoError(t, p.gatherer.Gather())

	select {
	case <-ctx.Done():
		t.Fatalf("ICE gather timeout: %v", ctx.Err())
	case <-gatherFinished:
	}

	iceCandidates, err := p.gatherer.GetLocalCandidates()
	require.NoError(t, err)
	iceParams, err := p.gatherer.GetLocalParameters()
	require.NoError(t, err)
	dtlsParams, err := p.dtls.GetLocalParameters()
	require.NoError(t, err)

	return Signal{
		ICECandidates:    iceCandidates,
		ICEParameters:    iceParams,
		DTLSParameters:   dtlsParams,
		SCTPCapabilities: p.sctp.GetCapabilities(),
	}
}

func startTransports(t *testing.T, p *webrtcPeer, remote Signal, role webrtc.ICERole) {
	t.Helper()

	require.NoError(t, p.ice.SetRemoteCandidates(remote.ICECandidates))
	require.NoError(t, p.ice.Start(p.gatherer, remote.ICEParameters, &role))
	require.NoError(t, p.dtls.Start(remote.DTLSParameters))
	require.NoError(t, p.sctp.Start(remote.SCTPCapabilities))
}

func TestLocalWebRTCFileTransfer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	payload := make([]byte, 32*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)
	payloadHash := sha256.Sum256(payload)

	receiver := newWebRTCPeer(t)
	sender := newWebRTCPeer(t)

	received := bytes.NewBuffer(nil)
	recvDone := make(chan struct{})
	recvErr := make(chan error, 1)

	receiver.sctp.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if _, werr := received.Write(msg.Data); werr != nil {
				recvErr <- werr
			}
		})
		dc.OnClose(func() {
			close(recvDone)
		})
	})

	receiverSignal := gatherSignal(ctx, t, receiver)
	senderSignal := gatherSignal(ctx, t, sender)

	startTransports(t, receiver, senderSignal, webrtc.ICERoleControlled)
	startTransports(t, sender, receiverSignal, webrtc.ICERoleControlling)

	var dcID uint16 = 1
	dc, err := sender.api.NewDataChannel(sender.sctp, &webrtc.DataChannelParameters{
		Label:   "payload.bin",
		ID:      &dcID,
		Ordered: true,
	})
	require.NoError(t, err)

	sendErr := make(chan error, 1)
	dc.OnOpen(func() {
		chunkSize := 4096
		for offset := 0; offset < len(payload); offset += chunkSize {
			end := offset + chunkSize
			if end > len(payload) {
				end = len(payload)
			}
			if err := dc.Send(payload[offset:end]); err != nil {
				sendErr <- err
				return
			}
		}
		if err := dc.Close(); err != nil {
			sendErr <- err
			return
		}
		close(sendErr)
	})

	select {
	case <-ctx.Done():
		t.Fatalf("transfer timeout: %v", ctx.Err())
	case err := <-sendErr:
		if err != nil {
			t.Fatalf("send failed: %v", err)
		}
	case err := <-recvErr:
		t.Fatalf("receive failed: %v", err)
	case <-recvDone:
	}

	select {
	case <-ctx.Done():
		t.Fatalf("receive completion timeout: %v", ctx.Err())
	case <-recvDone:
	}

	receivedBytes := received.Bytes()
	require.Equalf(t, len(payload), len(receivedBytes), "received size mismatch: want=%d got=%d", len(payload), len(receivedBytes))

	receivedHash := sha256.Sum256(receivedBytes)
	require.Equalf(t, payloadHash, receivedHash, "payload hash mismatch: want=%s got=%s",
		fmt.Sprintf("%x", payloadHash[:]), fmt.Sprintf("%x", receivedHash[:]))
}
