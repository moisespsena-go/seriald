package seriald

import (
	"fmt"
	"time"

	"github.com/moisespsena/go-default-logger"
	"github.com/op/go-logging"

	"github.com/dustin/go-humanize"
)

type Client struct {
	*CopyStreams
	Name        string
	PortPath    string
	SerialPort  *Port
	ConnectedAt time.Time
}

func (c *Client) In() *CopyStream {
	return c.Copiers[0]
}

func (c *Client) Out() *CopyStream {
	return c.Copiers[1]
}

func (c *Client) Map() map[string]interface{} {
	port := map[string]interface{}{
		"Path": c.SerialPort.Path,
	}

	in, out := c.In(), c.Out()
	if in.written > 0 {
		port["Output"] = in.written
	}
	if out.written > 0 {
		port["Input"] = out.written
	}

	m := map[string]interface{}{
		"Name": c.Name,
		"SerialPort": port,
	}

	if !c.startAt.IsZero() {
		m["StartAt"] = c.startAt
	}

	if !c.ConnectedAt.IsZero() {
		m["ConnectedAt"] = c.ConnectedAt
	}
	return m
}

func (c *Client) String() (s string) {
	s += c.SerialPort.Path
	if c.Name != "" {
		s += "@" + c.Name
	}
	if !c.startAt.IsZero() {
		s += " uptime='" + humanize.Time(c.startAt) + "' started_at='" + fmt.Sprint(c.startAt) + "'"
	}

	s += "  " + c.Copiers[0].String()
	s += "  " + c.Copiers[1].String()

	return
}

func (c *Client) Monitor() {
	for !c.IsCloseNotified() && !c.IsClosed() {
		if state := c.SerialPort.State(); state != PORT_OK {
			c.Log.Warningf("port %s.", state)
			c.Close()
			return
		}
		<-time.After(time.Second)
	}
}

func (c *Client) copy() {
	go c.Monitor()
}

func NewClient(name string, port *Port, in ReadCloseStringer, out WriteCloseStringer, Log ...*logging.Logger) *Client {
	c := &Client{Name: name, SerialPort: port}
	c.CopyStreams = NewStreamCopiers(NewStreamCopier(in, port), NewStreamCopier(port, out))
	if len(Log) == 0 || Log[0] == nil {
		Log = []*logging.Logger{defaultlogger.NewLogger(c.Name + "@" + port.Path)}
	}
	for _, c := range c.CopyStreams.Copiers {
		l := *Log[0]
		l.Module += "{" + c.Name() + "}"
		c.Log = &l
	}
	c.CopyStreams.Log = Log[0]
	c.SetCopy(func(prev func() error) error {
		go c.Monitor()
		return prev()
	})
	return c
}
