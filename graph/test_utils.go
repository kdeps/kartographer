package graph

import (
	"bytes"
	"io"
	"os"
)

// PipeCreator interface for testable pipe creation
type PipeCreator interface {
	CreatePipe() (*os.File, *os.File, error)
}

// OsPipeCreator implements PipeCreator using os.Pipe
type OsPipeCreator struct{}

func (o *OsPipeCreator) CreatePipe() (*os.File, *os.File, error) {
	return os.Pipe()
}

// captureOutput captures the output of a function for testing
func captureOutput(f func()) string {
	r, w := createPipe()
	old := redirectStdout(w)
	outC := startReader(r)
	f()
	w.Close()
	os.Stdout = old
	return <-outC
}

func createPipe() (*os.File, *os.File) {
	creator := &OsPipeCreator{}
	return createPipeWithCreator(creator)
}

func createPipeWithCreator(creator PipeCreator) (*os.File, *os.File) {
	r, w, err := creator.CreatePipe()
	if err != nil {
		panic(err)
	}
	return r, w
}

func redirectStdout(w *os.File) *os.File {
	old := os.Stdout
	os.Stdout = w
	return old
}

func startReader(r *os.File) chan string {
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()
	return outC
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
