package polling

import (
	"encoding/base64"
	"errors"
	"io"
	"sync/atomic"
	"time"

	"github.com/googollee/go-socket.io/engineio/frame"
)

var (
	ErrPingTimeout          = errors.New("ping timeout")
	ErrSeparatorInTextFrame = errors.New("should not write 0x1e to text frames.")
	ErrNonCloseFrame        = errors.New("has a non-closed frame writer")
)

type encoder struct {
	pingTimeout       time.Duration
	lastPing          time.Time
	writer            *bufWriter
	hasFramesChan     chan struct{}
	closed            chan struct{}
	hasNonClosedFrame int32
}

func newEncoder(pingTimouet time.Duration, closed chan struct{}, buf []byte) *encoder {
	return &encoder{
		pingTimeout:   pingTimouet,
		lastPing:      time.Now(),
		writer:        newBufWriter(buf),
		hasFramesChan: make(chan struct{}, 1),
		closed:        closed,
	}
}

func (e *encoder) NextFrame(ft frame.Type) (io.WriteCloser, error) {
	select {
	case <-e.closed:
		return nil, io.EOF
	default:
	}

	if !atomic.CompareAndSwapInt32(&e.hasNonClosedFrame, 0, 1) {
		return nil, ErrNonCloseFrame
	}

	ret := &frameWriter{
		writer:            e.writer,
		hasFramesChan:     e.hasFramesChan,
		hasNonClosedFrame: &e.hasNonClosedFrame,
	}

	if ft == frame.Binary {
		if err := ret.writer.WriteByte(binaryPrefix); err != nil {
			return nil, err
		}
		ret.base64 = base64.NewEncoder(base64.StdEncoding, ret.writer)
	}

	return ret, nil
}

func (e *encoder) WriteFramesTo(w io.Writer) error {
	pingTimeout := e.pingTimeout - time.Since(e.lastPing)
	select {
	case <-e.hasFramesChan:
	case <-time.After(pingTimeout):
		e.lastPing = time.Now()
		return ErrPingTimeout
	case <-e.closed:
		return io.EOF
	}

	if _, err := e.writer.WriteFinishedFrames(w); err != nil {
		return err
	}

	return nil
}

func (e *encoder) WaitFrameClose() {
	if atomic.LoadInt32(&e.hasNonClosedFrame) == 0 {
		return
	}

	<-e.hasFramesChan
	select {
	case e.hasFramesChan <- struct{}{}:
	default: // if it already has frames, the chan is full and continues.
	}
}

type frameWriter struct {
	writer            *bufWriter
	base64            io.WriteCloser
	hasFramesChan     chan struct{}
	hasNonClosedFrame *int32
}

func (w *frameWriter) Write(b []byte) (int, error) {
	if w.base64 != nil {
		return w.base64.Write(b)
	}

	for _, by := range b {
		if by == separator {
			return 0, ErrSeparatorInTextFrame
		}
	}
	return w.writer.Write(b)
}

func (w *frameWriter) WriteByte(b byte) error {
	return w.writer.WriteByte(b)
}

func (w *frameWriter) Close() error {
	if w.base64 != nil {
		w.base64.Close()
	}
	if err := w.writer.WriteByte(separator); err != nil {
		return err
	}

	atomic.StoreInt32(w.hasNonClosedFrame, 0)

	select {
	case w.hasFramesChan <- struct{}{}:
	default: // if it already has frames, the chan is full and continues.
	}

	return nil
}