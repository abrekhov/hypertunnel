//go:build integration

package datachannel

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

type testPeer struct {
	api      *webrtc.API
	gatherer *webrtc.ICEGatherer
	ice      *webrtc.ICETransport
	dtls     *webrtc.DTLSTransport
	sctp     *webrtc.SCTPTransport
}

func newTestPeer(t *testing.T) *testPeer {
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
	return &testPeer{api: api, gatherer: gatherer, ice: ice, dtls: dtls, sctp: sctp}
}

func gatherTestSignal(t *testing.T, p *testPeer) Signal {
	t.Helper()
	done := make(chan struct{})
	p.gatherer.OnLocalCandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			close(done)
		}
	})
	require.NoError(t, p.gatherer.Gather())
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("gather timeout")
	}
	candidates, err := p.gatherer.GetLocalCandidates()
	require.NoError(t, err)
	params, err := p.gatherer.GetLocalParameters()
	require.NoError(t, err)
	dtlsParams, err := p.dtls.GetLocalParameters()
	require.NoError(t, err)
	return Signal{
		ICECandidates:    candidates,
		ICEParameters:    params,
		DTLSParameters:   dtlsParams,
		SCTPCapabilities: p.sctp.GetCapabilities(),
	}
}

func startTestTransportsErr(p *testPeer, remote Signal, role webrtc.ICERole) error {
	if err := p.ice.SetRemoteCandidates(remote.ICECandidates); err != nil {
		return err
	}
	if err := p.ice.Start(p.gatherer, remote.ICEParameters, &role); err != nil {
		return err
	}
	if err := p.dtls.Start(remote.DTLSParameters); err != nil {
		return err
	}
	return p.sctp.Start(remote.SCTPCapabilities)
}

func stopPeer(p *testPeer) {
	_ = p.sctp.Stop()
	_ = p.dtls.Stop()
	_ = p.ice.Stop()
}

func startConcurrent(t *testing.T, a *testPeer, aRemote Signal, aRole webrtc.ICERole, b *testPeer, bRemote Signal, bRole webrtc.ICERole) {
	t.Helper()
	t.Cleanup(func() { stopPeer(a) })
	t.Cleanup(func() { stopPeer(b) })
	errCh := make(chan error, 2)
	go func() { errCh <- startTestTransportsErr(a, aRemote, aRole) }()
	go func() { errCh <- startTestTransportsErr(b, bRemote, bRole) }()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(30 * time.Second):
			t.Fatal("transport start timeout")
		}
	}
}

func connectTestDataChannel(t *testing.T, sender *testPeer, receiver *testPeer) (senderChannel *webrtc.DataChannel, receiverChannel *webrtc.DataChannel) {
	t.Helper()
	opened := make(chan struct{})
	receiver.sctp.OnDataChannel(func(c *webrtc.DataChannel) {
		receiverChannel = c
		close(opened)
	})
	var id uint16 = 1
	senderChannel, err := sender.api.NewDataChannel(sender.sctp, &webrtc.DataChannelParameters{
		Label:   "payload.bin",
		ID:      &id,
		Ordered: true,
	})
	require.NoError(t, err)
	senderOpen := make(chan struct{})
	senderChannel.OnOpen(func() { close(senderOpen) })
	select {
	case <-senderOpen:
	case <-time.After(15 * time.Second):
		t.Fatal("data channel open timeout")
	}
	select {
	case <-opened:
	case <-time.After(15 * time.Second):
		t.Fatal("receiver data channel timeout")
	}
	return senderChannel, receiverChannel
}

func runTransferCase(t *testing.T, md *transfer.Metadata, payload []byte) (senderErr error, receiverErr error) {
	t.Helper()
	sender := newTestPeer(t)
	receiver := newTestPeer(t)
	receiverSignal := gatherTestSignal(t, receiver)
	senderSignal := gatherTestSignal(t, sender)
	startConcurrent(t, receiver, senderSignal, webrtc.ICERoleControlled, sender, receiverSignal, webrtc.ICERoleControlling)

	receiverDone := make(chan error, 1)
	senderDone := make(chan error, 1)

	channel, receiverChannel := connectTestDataChannel(t, sender, receiver)
	FileTransferHandler(receiverChannel, func(err error) { receiverDone <- err })
	SendFile(channel, md, bytes.NewReader(payload), func(err error) { senderDone <- err })

	select {
	case senderErr = <-senderDone:
	case <-time.After(30 * time.Second):
		t.Fatal("sender timeout")
	}
	select {
	case receiverErr = <-receiverDone:
	case <-time.After(30 * time.Second):
		t.Fatal("receiver timeout")
	}
	return senderErr, receiverErr
}

func TestInProcessVerifiedTransfer(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	payload := make([]byte, 256*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err)
	sum := sha256.Sum256(payload)
	md := transfer.NewMetadata("payload.bin", int64(len(payload))).WithChecksum(hex.EncodeToString(sum[:]))

	senderErr, receiverErr := runTransferCase(t, md, payload)
	require.NoError(t, senderErr)
	require.NoError(t, receiverErr)

	got, err := os.ReadFile(filepath.Join(dir, "payload.bin"))
	require.NoError(t, err)
	require.Equal(t, payload, got)
	_, err = os.Stat(filepath.Join(dir, "payload.bin"+partSuffix))
	require.True(t, os.IsNotExist(err))
}

func TestInProcessChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	payload := []byte("verified transfer payload")
	md := transfer.NewMetadata("payload.bin", int64(len(payload))).WithChecksum("0011223344556677889900112233445566778899001122334455667788990011")

	senderErr, receiverErr := runTransferCase(t, md, payload)
	require.Error(t, senderErr)
	require.ErrorContains(t, receiverErr, "checksum mismatch")
	_, err := os.Stat(filepath.Join(dir, "payload.bin"))
	require.True(t, os.IsNotExist(err))
}

func TestInProcessTruncatedPayload(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	payload := []byte("verified transfer payload")
	sum := sha256.Sum256(payload)
	md := transfer.NewMetadata("payload.bin", int64(len(payload))).WithChecksum(hex.EncodeToString(sum[:]))

	senderErr, receiverErr := runTransferCase(t, md, payload[:len(payload)-4])
	require.Error(t, senderErr)
	require.ErrorContains(t, senderErr, "metadata declares")
	require.ErrorContains(t, receiverErr, "before transfer completed")
	_, err := os.Stat(filepath.Join(dir, "payload.bin"))
	require.True(t, os.IsNotExist(err))
}

func TestInProcessOversizedPayload(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	payload := []byte("verified transfer payload")
	sum := sha256.Sum256(payload)
	md := transfer.NewMetadata("payload.bin", 4).WithChecksum(hex.EncodeToString(sum[:]))

	senderErr, receiverErr := runTransferCase(t, md, payload)
	require.Error(t, senderErr)
	require.ErrorContains(t, receiverErr, "more data than expected size")
}

func TestInProcessEmptyFile(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	sum := sha256.Sum256(nil)
	md := transfer.NewMetadata("empty.bin", 0).WithChecksum(hex.EncodeToString(sum[:]))

	senderErr, receiverErr := runTransferCase(t, md, nil)
	require.NoError(t, senderErr)
	require.NoError(t, receiverErr)
	got, err := os.ReadFile(filepath.Join(dir, "empty.bin"))
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestInProcessLegacyPeerRejected(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	AutoAccept = true
	AcceptTimeout = time.Second

	sender := newTestPeer(t)
	receiver := newTestPeer(t)
	receiverSignal := gatherTestSignal(t, receiver)
	senderSignal := gatherTestSignal(t, sender)
	startConcurrent(t, receiver, senderSignal, webrtc.ICERoleControlled, sender, receiverSignal, webrtc.ICERoleControlling)

	receiverDone := make(chan error, 1)
	channel, receiverChannel := connectTestDataChannel(t, sender, receiver)
	FileTransferHandler(receiverChannel, func(err error) { receiverDone <- err })
	require.NoError(t, channel.Send([]byte("raw legacy data without metadata")))

	select {
	case err := <-receiverDone:
		require.ErrorContains(t, err, "transfer metadata")
	case <-time.After(30 * time.Second):
		t.Fatal("receiver timeout")
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	return func() { require.NoError(t, os.Chdir(orig)) }
}
