package exporter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darthlynx/kafka-exporter/internal/kafka"
)

type fakeConsumer struct {
	ch chan kafka.Message
}

func (f *fakeConsumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case m := <-f.ch:
		return m, nil
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	}
}
func (f *fakeConsumer) Close() error { return nil }

type fakeProducer struct {
	mu        sync.Mutex
	written   []kafka.Message
	errOnCall int32
	calls     int32
}

func (p *fakeProducer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	atomic.AddInt32(&p.calls, 1)
	call := atomic.LoadInt32(&p.calls)

	p.mu.Lock()
	p.written = append(p.written, msgs...)
	p.mu.Unlock()

	if p.errOnCall > 0 && call == p.errOnCall {
		return errors.New("producer error")
	}
	return nil
}
func (p *fakeProducer) Close() error { return nil }

func waitForWritten(t *testing.T, p *fakeProducer, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		p.mu.Lock()
		got := len(p.written)
		p.mu.Unlock()
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for written messages, got=%d want=%d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRun_ProcessesMessagesAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// prepare consumer that will produce one message and then block
	ch := make(chan kafka.Message, 1)

	var committed int32
	msg := kafka.Message{
		Key:   []byte("k1"),
		Value: []byte("v1"),
		Time:  time.Now(),
		CommitFunc: func(ctx context.Context) error {
			atomic.StoreInt32(&committed, 1)
			return nil
		},
	}
	ch <- msg

	consumer := &fakeConsumer{ch: ch}
	producer := &fakeProducer{}

	exp := New(consumer, producer, nil)

	done := make(chan error, 1)
	go func() {
		done <- exp.Run(ctx)
	}()

	// wait until producer received the message
	waitForWritten(t, producer, 1, 2*time.Second)

	// stop the exporter
	cancel()

	// ensure Run returns
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not return after cancel")
	}

	// validate produced message
	producer.mu.Lock()
	if len(producer.written) != 1 {
		t.Fatalf("expected 1 written message, got %d", len(producer.written))
	}
	if string(producer.written[0].Key) != "k1" || string(producer.written[0].Value) != "v1" {
		t.Fatalf("unexpected message content: key=%s value=%s", string(producer.written[0].Key), string(producer.written[0].Value))
	}
	producer.mu.Unlock()

	// ensure commit was invoked
	if atomic.LoadInt32(&committed) != 1 {
		t.Fatalf("expected message commit to be called")
	}
}

func TestRun_ContinuesOnProducerError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// produce two messages
	ch := make(chan kafka.Message, 2)

	var committed int32
	m1 := kafka.Message{
		Key:   []byte("k1"),
		Value: []byte("v1"),
		Time:  time.Now(),
		CommitFunc: func(ctx context.Context) error {
			atomic.AddInt32(&committed, 1)
			return nil
		},
	}
	m2 := kafka.Message{
		Key:   []byte("k2"),
		Value: []byte("v2"),
		Time:  time.Now(),
		CommitFunc: func(ctx context.Context) error {
			atomic.AddInt32(&committed, 1)
			return nil
		},
	}

	ch <- m1
	ch <- m2

	consumer := &fakeConsumer{ch: ch}
	// fail on first call, succeed on second
	producer := &fakeProducer{errOnCall: 1}

	exp := New(consumer, producer, nil)

	done := make(chan error, 1)
	go func() {
		done <- exp.Run(ctx)
	}()

	// wait until both attempts are observed
	waitForWritten(t, producer, 2, 3*time.Second)

	// stop exporter
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Run did not return after cancel")
	}

	// verify two attempts recorded; first may have been recorded despite error
	producer.mu.Lock()
	if len(producer.written) != 2 {
		t.Fatalf("expected 2 written attempts, got %d", len(producer.written))
	}
	producer.mu.Unlock()

	// check that producer reported two calls
	if atomic.LoadInt32(&producer.calls) < 2 {
		t.Fatalf("expected producer called at least twice, got %d", atomic.LoadInt32(&producer.calls))
	}

	// commit should be called only for the successful write (second message)
	if atomic.LoadInt32(&committed) != 1 {
		t.Fatalf("expected exactly 1 commit after one successful write, got %d", atomic.LoadInt32(&committed))
	}
}
