package datachannel

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/pion/webrtc/v3"
)

const ackTimeout = 90 * time.Second

// SendFile performs the metadata/acknowledgement protocol: it sends metadata, streams the reader,
// and completes only after the receiver verifies size and checksum.
func SendFile(channel *webrtc.DataChannel, md *transfer.Metadata, r io.Reader, done func(error)) {
	meta, err := md.WrapForTransfer()
	if err != nil {
		done(fmt.Errorf("encode transfer metadata: %w", err))
		return
	}
	acks := make(chan error, 2)
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if transfer.IsAckMessage(msg.Data) {
			acks <- transfer.AckError(msg.Data)
		}
	})
	if err := channel.Send(meta); err != nil {
		done(fmt.Errorf("send metadata: %w", err))
		return
	}
	if err := waitAck(acks); err != nil {
		done(err)
		return
	}
	chunk := make([]byte, 65534)
	var sent int64
	for {
		n, readErr := r.Read(chunk)
		if n > 0 {
			sent += int64(n)
			if err := channel.Send(chunk[:n]); err != nil {
				done(fmt.Errorf("send data: %w", err))
				return
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			done(fmt.Errorf("read source: %w", readErr))
			return
		}
	}
	if sent != md.Size {
		_ = channel.Close()
		done(fmt.Errorf("source contains %d bytes but metadata declares %d", sent, md.Size))
		return
	}
	if err := waitAck(acks); err != nil {
		done(err)
		return
	}
	if err := channel.Close(); err != nil {
		done(fmt.Errorf("close data channel: %w", err))
		return
	}
	done(nil)
}

func waitAck(acks <-chan error) error {
	select {
	case err := <-acks:
		return err
	case <-time.After(ackTimeout):
		return errors.New("timed out waiting for receiver acknowledgement")
	}
}
