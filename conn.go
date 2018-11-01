package seriald

import (
	"fmt"
	"io"
	"net"

	"github.com/op/go-logging"
)

type WriteReader interface {
	io.Reader
	io.Writer
}

type WriteReadCloser interface {
	WriteReader
	io.Closer
}

type WriteReadClose struct {
	io.Reader
	io.Writer
	io.Closer
}

type Conn interface {
	WriteReadCloser
	fmt.Stringer
	Logger() *logging.Logger
}

type NamedConn struct {
	WriteReadCloser
	Name string
	Log  *logging.Logger
}

func (nc *NamedConn) String() string {
	return nc.Name
}

func (nc *NamedConn) Logger() *logging.Logger {
	return nc.Log
}

type NetConn struct {
	net.Conn
	Log *logging.Logger
}

func (nc *NetConn) String() string {
	return nc.RemoteAddr().String()
}

func (nc *NetConn) Logger() *logging.Logger {
	return nc.Log
}
