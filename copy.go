package seriald

import (
	"fmt"
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/op/go-logging"
)

type Copier struct {
	Starter
	copyFunc func() error
}

func NewCopier(copy func() error) *Copier {
	c := &Copier{copyFunc: copy}
	c.SetStarter(func() error {
		return c.Copy()
	})
	return c
}

func (c *Copier) SetCopy(f func(prev func() error) error) {
	if prev := c.copyFunc; prev == nil {
		c.copyFunc = func() error {
			return f(func() error { return nil })
		}
	} else {
		c.copyFunc = func() error {
			return f(prev)
		}
	}
}

func (c *Copier) Copy() error {
	return c.copyFunc()
}

type CopyStream struct {
	*Copier
	Src ReadCloseStringer
	Dst WriteCloseStringer
	Log *logging.Logger
	Closable

	startAt *time.Time
	written int
}

func NewStreamCopier(src ReadCloseStringer, dst WriteCloseStringer, log ...*logging.Logger) *CopyStream {
	s := &CopyStream{Src: src, Dst: dst}
	s.Copier = NewCopier(s.copy)
	s.CloseNotifier(func() {
		if m, ok := s.Src.(MustNotifyCloser); ok {
			m.NotifyClose()
		}
		if m, ok := s.Dst.(MustNotifyCloser); ok {
			m.NotifyClose()
		}
	})
	s.SetCloser(func(old func() error) error {
		return Err(old()).
			Err(s.Src.Close(), "SRC").
			Err(s.Dst.Close(), "DST").
			GetError()
	})
	if len(log) > 0 && log[0] != nil {
		s.Log = log[0]
	}
	return s
}

func (c *CopyStream) Name() string {
	return c.Src.String() + " -> " + c.Dst.String()
}

func (c *CopyStream) String() (s string) {
	s = c.Name()

	if c.startAt != nil {
		s += " uptime='" + humanize.Time(*c.startAt) + "' end_at='" + fmt.Sprint(c.closedAt) + "'"
	}
	if c.written != 0 {
		s += " written=" + fmt.Sprint(c.written)
	}
	return
}

func (s *CopyStream) copy() error {
	defer s.Close()
	if _, err := io.Copy(&WriteCounter{s.Dst, &s.written}, s.Src); err != nil {
		if !s.IsCloseNotified() {
			if s.Log != nil {
				s.Log.Errorf("done with error: %v", err)
			} else {
				return err
			}
		}
	} else if s.Log != nil {
		s.Log.Debug("Done")
	}
	return nil
}
