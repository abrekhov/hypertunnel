package datachannel

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abrekhov/hypertunnel/pkg/archive"
	"github.com/abrekhov/hypertunnel/pkg/transfer"
	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

const partSuffix = ".htpart"

const (
	finalAckCloseTimeout = 30 * time.Second
	failAckCloseTimeout  = 10 * time.Second
)

type receiveSession struct {
	mu              sync.Mutex
	finished        bool
	finalized       bool
	ackedFinal      bool
	closed          bool
	closedCh        chan struct{}
	md              *transfer.Metadata
	targetPath      string
	isArchive       bool
	fd              *os.File
	buf             *bytes.Buffer
	cw              *transfer.ChecksumWriter
	received        int64
	progress        *transfer.Progress
	progressStop    chan struct{}
	progressStopped bool
	channel         *webrtc.DataChannel
	done            func(error)
}

// FileTransferHandler receives a verified transfer on the channel and calls done with the final result.
func FileTransferHandler(channel *webrtc.DataChannel, done func(error)) {
	s := &receiveSession{channel: channel, done: done, closedCh: make(chan struct{})}
	channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		s.mu.Lock()
		if s.finished || s.finalized {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		if s.md == nil {
			s.begin(msg.Data)
			return
		}
		s.data(msg.Data)
	})
	channel.OnClose(func() {
		s.markClosed()
		s.finish(fmt.Errorf("connection closed before transfer completed"))
	})
}

func (s *receiveSession) begin(meta []byte) {
	if !transfer.IsMetadataMessage(meta) {
		s.fail("peer did not send transfer metadata; run the same HyperTunnel version on both peers")
		return
	}
	md, err := transfer.UnwrapMetadata(meta)
	if err != nil {
		s.fail(fmt.Sprintf("invalid transfer metadata: %v", err))
		return
	}
	if err := md.Validate(); err != nil {
		s.fail(fmt.Sprintf("invalid transfer metadata: %v", err))
		return
	}
	if md.Checksum == "" {
		s.fail("transfer metadata is missing a checksum")
		return
	}
	target := md.SafeFilename()
	s.isArchive = md.IsArchive || strings.HasSuffix(target, ".tar.gz")
	if s.isArchive {
		target = strings.TrimSuffix(target, ".tar.gz")
	}
	if err := s.confirm(target); err != nil {
		s.fail(err.Error())
		return
	}

	s.mu.Lock()
	s.md = md
	s.targetPath = target
	s.progress = transfer.NewProgress(md.Size)
	s.progressStop = make(chan struct{})
	if s.isArchive {
		s.buf = &bytes.Buffer{}
		s.cw = transfer.NewChecksumWriter(s.buf)
	} else {
		fd, err := os.Create(target + partSuffix) // #nosec G304 - target is validated by Metadata.SafeFilename
		if err != nil {
			s.mu.Unlock()
			s.fail(fmt.Sprintf("create file: %v", err))
			return
		}
		s.fd = fd
		s.cw = transfer.NewChecksumWriter(fd)
	}
	s.mu.Unlock()

	go renderReceiveProgress(s.progress, s.progressStop)

	if err := s.channel.Send([]byte(transfer.AckOK)); err != nil {
		s.fail(fmt.Sprintf("send acknowledgement: %v", err))
		return
	}
	if md.Size == 0 {
		s.complete()
	}
}

func (s *receiveSession) confirm(target string) error {
	_, err := os.Stat(target)
	targetExists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check destination %s: %w", target, err)
	}
	kind := "file"
	if s.isArchive {
		kind = "directory"
	}
	if !AutoAccept {
		accept := askForConfirmationWithTimeout(fmt.Sprintf("Receive %s %s?", kind, target), os.Stdin, AcceptTimeout)
		if !accept {
			return fmt.Errorf("transfer declined")
		}
	}
	if targetExists {
		if !AutoAccept {
			overwrite := askForConfirmationWithTimeout(fmt.Sprintf("%s %s exists. Overwrite?", kind, target), os.Stdin, AcceptTimeout)
			if !overwrite {
				return fmt.Errorf("overwrite declined")
			}
		}
		log.Infof("Overwriting existing %s %s", kind, target)
	}
	return nil
}

func (s *receiveSession) data(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	s.mu.Lock()
	if s.finished || s.finalized {
		s.mu.Unlock()
		return
	}
	if s.received+int64(len(chunk)) > s.md.Size {
		s.mu.Unlock()
		s.fail("received more data than expected size")
		return
	}
	if _, err := s.cw.Write(chunk); err != nil {
		s.mu.Unlock()
		s.fail(fmt.Sprintf("write data: %v", err))
		return
	}
	s.received += int64(len(chunk))
	s.mu.Unlock()
	s.progress.Update(int64(len(chunk)))
	if s.received == s.md.Size {
		s.complete()
	}
}

func (s *receiveSession) complete() {
	s.mu.Lock()
	if s.finished || s.finalized {
		s.mu.Unlock()
		return
	}
	if s.received != s.md.Size {
		s.mu.Unlock()
		s.fail(fmt.Sprintf("received %d of %d bytes", s.received, s.md.Size))
		return
	}
	actual := s.cw.SumHex()
	fd, target, isArchive, buf := s.fd, s.targetPath, s.isArchive, s.buf
	s.mu.Unlock()

	if actual != s.md.Checksum {
		s.fail("checksum mismatch")
		return
	}

	if isArchive {
		if err := os.MkdirAll(target, 0750); err != nil {
			s.fail(fmt.Sprintf("create directory: %v", err))
			return
		}
		if err := archive.ExtractTarGz(bytes.NewReader(buf.Bytes()), target, archive.DefaultOptions()); err != nil {
			s.fail(fmt.Sprintf("extract archive: %v", err))
			return
		}
	} else {
		if err := fd.Close(); err != nil {
			s.fail(fmt.Sprintf("close file: %v", err))
			return
		}
		if err := os.Rename(target+partSuffix, target); err != nil {
			s.fail(fmt.Sprintf("publish file: %v", err))
			return
		}
		s.mu.Lock()
		s.fd = nil
		s.mu.Unlock()
	}

	s.stopProgress()
	fmt.Println("Checksum verified")
	printReceiveSummary(s.label(), s.progress)
	s.mu.Lock()
	s.finished = true
	s.ackedFinal = true
	s.mu.Unlock()
	go func() {
		if err := s.channel.Send([]byte(transfer.AckOK)); err != nil {
			s.finalize(fmt.Errorf("send completion acknowledgement: %v", err))
			return
		}
		if !s.awaitClose(finalAckCloseTimeout) {
			log.Warningln("sender did not close the connection after final acknowledgement")
		}
		s.finalize(nil)
	}()
}

func (s *receiveSession) fail(reason string) {
	s.mu.Lock()
	if s.finished || s.finalized {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.ackedFinal = true
	s.mu.Unlock()
	s.stopProgress()
	go func() {
		if err := s.channel.Send(transfer.AckFail(reason)); err != nil {
			log.Debugf("send failure acknowledgement: %v", err)
		}
		s.awaitClose(failAckCloseTimeout)
		s.finalize(fmt.Errorf("%s", reason))
	}()
}

func (s *receiveSession) finish(err error) {
	s.mu.Lock()
	if s.finished || s.finalized {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.mu.Unlock()
	s.finalize(err)
}

func (s *receiveSession) finalize(err error) {
	s.mu.Lock()
	if s.finalized {
		s.mu.Unlock()
		return
	}
	s.finalized = true
	fd, target := s.fd, s.targetPath
	s.fd = nil
	s.mu.Unlock()

	s.stopProgress()
	if fd != nil {
		if cerr := fd.Close(); cerr != nil {
			log.Debugf("close partial file: %v", cerr)
		}
		if err != nil {
			if rerr := os.Remove(target + partSuffix); rerr != nil && !os.IsNotExist(rerr) {
				log.Debugf("remove partial file: %v", rerr)
			}
		}
	}
	if err != nil {
		log.Errorf("%v", err)
		if cerr := s.channel.Close(); cerr != nil {
			log.Debugf("close data channel: %v", cerr)
		}
	}
	s.done(err)
}

func (s *receiveSession) markClosed() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.closedCh)
	}
	s.mu.Unlock()
}

func (s *receiveSession) awaitClose(timeout time.Duration) bool {
	select {
	case <-s.closedCh:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *receiveSession) stopProgress() {
	s.mu.Lock()
	if s.progressStop != nil && !s.progressStopped {
		s.progressStopped = true
		close(s.progressStop)
	}
	s.mu.Unlock()
}

func (s *receiveSession) label() string {
	if s.isArchive {
		return s.targetPath + ".tar.gz"
	}
	return s.targetPath
}

func renderReceiveProgress(progress *transfer.Progress, stop <-chan struct{}) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			metrics := progress.Metrics()
			fmt.Printf("\r%s\n", transfer.FormatProgressLine("Receiving", metrics))
			return
		case <-ticker.C:
			metrics := progress.Metrics()
			fmt.Printf("\r%s", transfer.FormatProgressLine("Receiving", metrics))
		}
	}
}

func printReceiveSummary(name string, progress *transfer.Progress) {
	metrics := progress.Metrics()
	elapsed := progress.Elapsed()
	avgSpeed := 0.0
	if elapsed.Seconds() > 0 {
		avgSpeed = float64(metrics.TransferredBytes) / elapsed.Seconds()
	}
	fmt.Println()
	fmt.Println("Receive complete")
	fmt.Printf("File: %s (%s)\n", name, transfer.FormatSize(metrics.TransferredBytes))
	fmt.Printf("Time: %s, Avg: %s\n", transfer.FormatDuration(elapsed), transfer.FormatSpeed(avgSpeed))
}

// AutoAccept controls whether incoming transfers are accepted without prompting.
var AutoAccept bool

// AcceptTimeout controls how long the accept prompt waits before auto-accepting.
var AcceptTimeout time.Duration

func askForConfirmation(s string, in io.Reader) bool {
	return askForConfirmationWithTimeout(s, in, 0)
}

func askForConfirmationWithTimeout(s string, in io.Reader, timeout time.Duration) bool {
	tries := 3
	reader := bufio.NewReader(in)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [Y/n]: ", s)

		res, err := readLineWithTimeout(reader, timeout)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("No confirmation input; declining.")
				return false
			}
			log.Fatal(err)
		}

		if len(res) < 2 {
			return true
		}

		trimmed := strings.ToLower(strings.TrimSpace(res))
		if trimmed == "y" || trimmed == "yes" {
			return true
		}
		if trimmed == "n" || trimmed == "no" {
			return false
		}
	}

	return false
}

func readLineWithTimeout(reader *bufio.Reader, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		return reader.ReadString('\n')
	}
	result := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		line, err := reader.ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}
		result <- line
	}()

	select {
	case line := <-result:
		return line, nil
	case err := <-errCh:
		return "", err
	case <-time.After(timeout):
		fmt.Printf("\nNo response within %s, auto-accepting.\n", timeout.Round(time.Second))
		return "\n", nil
	}
}
