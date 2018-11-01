package seriald

import (
	"fmt"
	"io"
)

type ReadCloseStringer interface {
	io.ReadCloser
	fmt.Stringer
}

type NamedReadClose struct {
	io.ReadCloser
	Name string
}

func (n *NamedReadClose) String() string {
	return n.Name
}

type WriteCloseStringer interface {
	io.WriteCloser
	fmt.Stringer
}

type NamedWriteClose struct {
	io.WriteCloser
	Name string
}

func (n *NamedWriteClose) String() string {
	return n.Name
}

func readLine(r io.Reader) (data string, err error) {
	buf := make([]byte, 1)
	var n int
	// Read the incoming connection into the buffer.
	for err == nil {
		if n, err = r.Read(buf); err == nil && n == 1 {
			if buf[0] == '\r' {
				_, err = r.Read(buf)
				break
			}
			if buf[0] == '\n' {
				break
			}
			data += string(buf)
		} else if err != nil {
			return
		} else {
			return "", io.EOF
		}
	}
	return
}
