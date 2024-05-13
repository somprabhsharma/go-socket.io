package packet

import (
	"io"

	"github.com/somprabhsharma/go-socket.io/engineio/frame"
)

type fakeConnWriter struct {
	Frames []Frame
}

func NewFakeConnWriter() *fakeConnWriter {
	return &fakeConnWriter{}
}

func (w *fakeConnWriter) NextWriter(fType frame.Type) (io.WriteCloser, error) {
	return newFakeFrame(w, fType), nil
}
