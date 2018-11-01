package seriald

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"

	"github.com/kballard/go-shellquote"
	"github.com/moisespsena/go-default-logger"
)

var (
	log         = defaultlogger.NewLogger("seriald")
	ErrNotFound = errors.New("not found")
)

type Server struct {
	Clients     sync.Map
	Connections sync.Map
	Starter
	Closable
	Listener net.Listener
}

func NewServer(addr ...string) (s *Server, err error) {
	s = &Server{}
	if len(addr) > 0 && addr[0] != "" {
		if s.Listener, err = NewListener(addr[0]); err != nil {
			return nil, err
		}
		log.Info("Listening on", addr[0])
	}
	s.SetStarter(s.Serve)
	s.SetCloser(func(old func() error) error {
		return s.Listener.Close()
	})
	return
}

func (s *Server) AddClient(client *Client) {
	s.Clients.Store(client.SerialPort.Path, client)
	client.AfterClose(func() {
		s.Clients.Delete(client.SerialPort.Path)
	})
}

func (s *Server) MustServe(addr string) (err error) {
	var l net.Listener
	if l, err = NewListener(addr); err != nil {
		return
	}
	defer l.Close()
	fmt.Println("Listening on " + addr)
	return s.ServeListener(l)
}

func (s *Server) Serve() error {
	return s.ServeListener(s.Listener)
}

func (s *Server) ServeListener(l net.Listener) error {
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("Error accepting: %v", err.Error())
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("%s: %v\n%v", conn.RemoteAddr(), string(debug.Stack()), r)
		}
		conn.Close()
	}()
	s.HandleConnection(&NetConn{conn, defaultlogger.NewLogger(conn.RemoteAddr().String())})
}

func (s *Server) OpenPort(conn Conn, pth, brs string) (err error) {
	if pth, err = FindPort(pth); err != nil {
		if err == ErrNotFound {
			return err
		}
		return fmt.Errorf("Find port failed: %v", err)
	}
	if _, ok := s.Clients.Load(pth); ok {
		return fmt.Errorf("SerialPort %q is OpenPort.", pth)
	}
	var br int
	if br, err = strconv.Atoi(brs); err != nil {
		return fmt.Errorf("Parse BaudRate error: %v", err)
	}

	port, err := NewPort(pth, br)
	if err != nil {
		return fmt.Errorf("Open SerialPort %q with %s Baud Rate failed: %v", pth, brs, err)
	}
	defer port.Close()

	client := NewClient(conn.String(), port, conn, conn)
	s.AddClient(client)
	if _, err = s.WriteValue(conn, pth); err != nil {
		panic(err)
	}
	client.Copy()
	return nil
}

func (s *Server) WriteMessage(w io.Writer, m *ResponseMessage) (n int, err error) {
	if n, err := WriteMessage(w, m); err == nil {
		var n2 int
		n2, err = w.Write([]byte("\n"))
		n += n2
	}
	return
}

func (s *Server) WriteError(w io.Writer, msg interface{}, args ...interface{}) (n int, err error) {
	var e string
	switch et := msg.(type) {
	case string:
		e = et
	case error:
		e = et.Error()
	}
	if e == "" {
		return
	}
	if len(args) > 0 {
		e = fmt.Sprintf(e, args...)
	}
	return s.WriteMessage(w, &ResponseMessage{Error: e})
}

func (s *Server) WriteValue(w io.Writer, value interface{}, args ...interface{}) (n int, err error) {
	m := &ResponseMessage{}
	switch et := value.(type) {
	case string:
		if len(args) > 0 {
			m.Value = fmt.Sprintf(et, args...)
		} else {
			m.Value = et
		}
	default:
		m.Value = et
	}

	return s.WriteMessage(w, m)
}

// Handles incoming requests.
func (s *Server) HandleConnection(conn Conn) {
	l := conn.Logger()
	l.Debug("connected")
	s.Connections.Store(conn.String(), conn)
	defer func() {
		s.Connections.Delete(conn.String())
	}()

	e := func(err interface{}, args ...interface{}) {
		if _, err := s.WriteError(conn, err, args...); err != nil {
			panic(err)
		}
	}

	m := func(value interface{}, args ...interface{}) {
		if _, err := s.WriteValue(conn, value, args...); err != nil {
			panic(err)
		}
	}

Main:
	for {
		line, err := readLine(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			l.Errorf("Read command: %v", err)
			return
		}
		l.Debug("cli:", line)
		args, err := shellquote.Split(line)
		if err != nil {
			e("Parse args: %v", err)
			continue
		}

		switch args[0] {
		case "open":
			var err error
			if len(args) != 3 {
				err = errors.New("Invalid args count. Usage: `open PATH BOUD_RATE`")
			} else {
				err = s.OpenPort(conn, args[1], args[2])
			}
			if err != nil {
				l.Error(err)
				e(err)
			}
			return
		case "info":
			if len(args) != 2 {
				e("Invalid args count. Usage: `info PATH`")
				continue Main
			}

			if ci, ok := s.Clients.Load(args[1]); !ok {
				e("SerialPort %q is not open.\n", args[1])
			} else {
				m(ci.(*Client).Map())
			}
		case "close":
			if len(args) != 2 {
				e("Invalid args count. Usage: `close PATH`")
				continue Main
			}

			if ci, ok := s.Clients.Load(args[1]); !ok {
				e("SerialPort %q is not open.\n", args[1])
			} else {
				c := ci.(*Client)
				c.AfterClose(func() {
					m("closed")
				})
				c.Close()
			}
		case "find":
			if len(args) != 2 {
				e("Invalid args count. Usage: `find PATH`")
				continue Main
			}

			if pth, err := FindPort(args[1]); err != nil {
				e("%v", err)
			} else {
				m(pth)
			}
			continue Main
		case "exists":
			if len(args) != 2 {
				e("Invalid args count. Usage: `exists PATH`")
				continue Main
			}

			if _, err := FindPort(args[1]); err != nil {
				if err == ErrNotFound {
					m(false)
				} else {
					e(err)
				}
			} else {
				m(true)
			}
		case "ls":
			if len(args) != 1 {
				e("Invalid args count. Usage: `close PATH`")
				continue Main
			}

			var items []*Client
			s.Clients.Range(func(key, value interface{}) bool {
				items = append(items, value.(*Client))
				return true
			})

			sort.Slice(items, func(i, j int) bool {
				return items[i].SerialPort.Path < items[j].SerialPort.Path
			})

			result := make([]interface{}, len(items))

			for i, c := range items {
				result[i] = c.Map()
				items[i] = nil
			}

			m(result)
		case "exit":
			return
		default:
			e("Invalid command.")
		}
	}
}
