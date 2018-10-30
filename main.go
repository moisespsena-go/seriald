package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"go.bug.st/serial.v1"
)

func server(addr string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + addr)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
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

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	log.Println("New client on", conn.RemoteAddr())
	defer conn.Close()
	addr, err := readLine(conn)
	log.Println("New serial connection:", addr)
	if err != nil {
		if err == io.EOF {
			return
		}
		log.Println("ERRO ao ler o DEVICE:", err)
		return
	}
	parts := strings.Split(addr, ":")
	var i int
	if i, err = strconv.Atoi(parts[1]); err != nil {
		log.Println("Parse BaudRate error:", err)
		return
	}
	mode := &serial.Mode{
		BaudRate: i,
	}
	port, err := serial.Open(parts[0], mode)
	if err != nil {
		log.Println(addr, err)
		return
	}
	defer port.Close()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(port, conn); err != nil {
			log.Println(parts[0], "STDIN done with error:", err)
			port.Close()
		} else {
			log.Println(parts[0], "STDIN done.")
		}
	}()
	go func() {
		defer wg.Done()
		if _, err := io.Copy(conn, port); err != nil {
			log.Println(parts[0], "STDOUT done with error:", err)
			port.Close()
		} else {
			log.Println(parts[0], "STDOUT done.")
		}
	}()

	wg.Wait()
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println(`Usage ` + os.Args[0] + ` HOST:PORT

Examples:

` + os.Args[0] + ` :5000
` + os.Args[0] + ` localhost:5000
` + os.Args[0] + ` 0.0.0.0:5000`)
		os.Exit(1)
		return
	}
	server(os.Args[1])
}
