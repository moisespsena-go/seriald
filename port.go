package seriald

import (
	"fmt"
	"os"

	"github.com/gobwas/glob"
	"go.bug.st/serial.v1"
)

type PortState string

const (
	PORT_CHANGED PortState = "changed"
	PORT_REMOVED PortState = "removed"
	PORT_ERROR   PortState = "error"
	PORT_OK      PortState = ""
)

type Port struct {
	serial.Port
	Path string
	info os.FileInfo

	readCount, writeCount uint64
}

func NewPort(path string, boudRate int) (port *Port, err error) {
	var (
		info os.FileInfo
		p    serial.Port
	)
	if info, err = os.Stat(path); err != nil {
		return
	}

	if p, err = serial.Open(path, &serial.Mode{
		BaudRate: boudRate,
	}); err != nil {
		return
	}

	return &Port{Port: p, Path: path, info: info}, nil
}

func (p *Port) String() string {
	return p.Path
}

func (p *Port) Write(d []byte) (n int, err error) {
	n, err = p.Port.Write(d)
	p.writeCount += uint64(n)
	return
}

func (p *Port) Read(d []byte) (n int, err error) {
	n, err = p.Port.Read(d)
	p.readCount += uint64(n)
	return
}

func (p *Port) Info() os.FileInfo {
	return p.info
}

func (p *Port) State() PortState {
	if info, err := os.Stat(p.Path); err != nil {
		if os.IsNotExist(err) {
			return PORT_REMOVED
		}
		return PORT_ERROR
	} else if os.SameFile(p.info, info) {
		return PORT_OK
	}
	return PORT_CHANGED
}

func FindPort(path string) (pth string, err error) {
	if path[0] == '~' {
		path = path[1:]
		var g glob.Glob
		g, err = glob.Compile(path)
		if err != nil {
			err = fmt.Errorf("Path expression error: %v", err)
			return
		}
		var ports []string
		if ports, err = serial.GetPortsList(); err != nil {
			err = fmt.Errorf("Get ports list failed: %v", err)
			return
		}
		for _, pth = range ports {
			if g.Match(pth) {
				return
			}
		}
		return "", ErrNotFound
	}
	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = ErrNotFound
		}
		return "", err
	}
	return path, nil
}
