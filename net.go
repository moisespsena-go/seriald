package seriald

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
)

func NewListener(addr string) (l net.Listener, err error) {
	if strings.HasPrefix(addr, "unix:") {
		sockFile := addr[5:]
		if _, err := os.Stat(sockFile); err == nil {
			if err = syscall.Unlink(sockFile); err != nil {
				return nil, fmt.Errorf("Unlink old sock file: %v", err.Error())
			}
		}
		defer func() {
			if _, err := os.Stat(sockFile); err == nil {
				syscall.Unlink(sockFile)
			}
		}()
		l, err = net.Listen("unix", sockFile)
	} else {
		l, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("Error listening: %v", err.Error())
	}
	return
}
