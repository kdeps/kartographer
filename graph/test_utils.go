package graph

import (
	"bytes"
	"io"
	"os"
)

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
	r, w, err := os.Pipe()
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
