package input

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type blockingReader struct {
	started chan struct{}
	chunks  chan []byte
	reads   int32
}

func newBlockingReader() *blockingReader {
	return &blockingReader{
		started: make(chan struct{}, 16),
		chunks:  make(chan []byte, 16),
	}
}

func (r *blockingReader) Read(p []byte) (int, error) {
	atomic.AddInt32(&r.reads, 1)
	select {
	case r.started <- struct{}{}:
	default:
	}
	chunk, ok := <-r.chunks
	if !ok {
		return 0, io.EOF
	}
	copy(p, chunk)
	return len(chunk), nil
}

func (r *blockingReader) ReadCount() int {
	return int(atomic.LoadInt32(&r.reads))
}

func TestParser(t *testing.T) {
	data := []byte("a\r\x1b[A\x1b[3~")
	p := NewParser(bytes.NewReader(data))

	ev, _ := p.NextEvent()
	if ev.Key != KeyRune || ev.Rune != 'a' {
		t.Errorf("expected 'a', got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyEnter {
		t.Errorf("expected Enter, got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyUp {
		t.Errorf("expected Up, got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyDelete {
		t.Errorf("expected Delete, got %v", ev)
	}
}

func TestParser_CtrlDelete(t *testing.T) {
	data := []byte("\x1b[3;5~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlDelete {
		t.Errorf("expected KeyCtrlDelete, got %v", ev.Key)
	}
}

func TestParserCloseUnblocksNextEvent(t *testing.T) {
	p := NewParser(bytes.NewReader(nil))
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
	_, err := p.NextEvent()
	if err == nil {
		t.Fatal("expected EOF after parser close")
	}
}

func TestParserOnlyReadsOnDemand(t *testing.T) {
	r := newBlockingReader()
	p := NewParser(r)

	select {
	case <-r.started:
		t.Fatal("parser started reading before NextEvent")
	case <-time.After(30 * time.Millisecond):
	}

	resultCh := make(chan InputEvent, 1)
	errCh := make(chan error, 1)
	go func() {
		ev, err := p.NextEvent()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- ev
	}()

	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("parser did not request input")
	}

	r.chunks <- []byte("a")

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case ev := <-resultCh:
		if ev.Key != KeyRune || ev.Rune != 'a' {
			t.Fatalf("expected rune a, got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for parser result")
	}

	if got := r.ReadCount(); got != 1 {
		t.Fatalf("expected one read after first event, got %d", got)
	}

	select {
	case <-r.started:
		t.Fatal("parser prefetched input without a second request")
	case <-time.After(30 * time.Millisecond):
	}

	go func() {
		ev, err := p.NextEvent()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- ev
	}()

	select {
	case <-r.started:
	case <-time.After(time.Second):
		t.Fatal("parser did not request second input")
	}

	r.chunks <- []byte("b")

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error on second event: %v", err)
	case ev := <-resultCh:
		if ev.Key != KeyRune || ev.Rune != 'b' {
			t.Fatalf("expected rune b, got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second parser result")
	}

	if got := r.ReadCount(); got != 2 {
		t.Fatalf("expected two reads after two events, got %d", got)
	}

	close(r.chunks)
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
