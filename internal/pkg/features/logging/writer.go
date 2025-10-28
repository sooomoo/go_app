package logging

import (
	"context"
	"goapp/pkg/ids"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var store Store
var writer *Writer
var writeMu sync.Mutex = sync.Mutex{}

type WriterOptions func(*Writer)

func WithInterval(interval time.Duration) WriterOptions {
	return func(w *Writer) {
		w.interval = interval
	}
}

func WithMaxCount(count int) WriterOptions {
	return func(w *Writer) {
		w.maxCount = count
	}
}

func WithBufferSize(size int) WriterOptions {
	return func(w *Writer) {
		w.bufferSize = size
	}
}

func Start(ctx context.Context, serviceName string, s Store, opts ...WriterOptions) {
	writeMu.Lock()
	defer writeMu.Unlock()

	store = s
	if store == nil {
		store = NewDBStore()
	}

	if writer != nil {
		writer.release()
	}

	writer = &Writer{
		interval:    time.Second,
		maxCount:    100,
		serviceName: serviceName,
	}
	for _, opt := range opts {
		opt(writer)
	}
	writer.init()
	go writer.listen(ctx)
}

func Stop() {
	writeMu.Lock()
	defer writeMu.Unlock()

	if writer != nil {
		writer.release()
	}
	writer = nil
}

func post(log *ServiceLog) {
	if writer == nil {
		panic("logging: writer is nil, please call Start() first")
	}
	writer.writeCh <- log
}

type Writer struct {
	interval    time.Duration
	maxCount    int
	bufferSize  int
	writeCh     chan *ServiceLog
	timer       *time.Timer
	cancel      context.CancelFunc
	serviceName string

	bufferedLogs []*ServiceLog
}

func (w *Writer) init() {
	if w.maxCount < 1 {
		w.maxCount = 1
	}
	if w.bufferSize < 1 {
		w.bufferSize = 1000
	}
	w.writeCh = make(chan *ServiceLog, w.bufferSize)
	if w.interval < time.Duration(100*time.Millisecond) {
		w.interval = time.Duration(100 * time.Millisecond)
	}
	w.timer = time.NewTimer(w.interval)
}

func (w *Writer) release() {
	close(w.writeCh)
	w.timer.Stop()
	w.cancel()
	w.timer = nil
	w.writeCh = nil
	w.cancel = nil
}

func (w *Writer) flush(ctx context.Context) {
	if len(w.bufferedLogs) == 0 {
		return
	}

	for _, log := range w.bufferedLogs {
		log.ID = ids.NewUID()
		log.Service = w.serviceName
		log.CreatedAt = time.Now()
	}

	if err := store.WriteMany(ctx, w.bufferedLogs); err != nil {
		log.Error().Err(err).Msg("logging: failed to write logs")
	}
	w.bufferedLogs = w.bufferedLogs[:0]
}

func (w *Writer) listen(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case logData := <-w.writeCh:
			w.bufferedLogs = append(w.bufferedLogs, logData)
			if len(w.bufferedLogs) >= w.maxCount {
				w.flush(ctx)
			}
		case <-w.timer.C:
			w.flush(ctx)
		}
	}
}
