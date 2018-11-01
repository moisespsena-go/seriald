package seriald

import (
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/op/go-logging"
)

type WriteCounter struct {
	io.Writer
	count *int
}

func NewWriteCounter(w io.Writer) *WriteCounter {
	c := 0
	return &WriteCounter{w, &c}
}

func (wc *WriteCounter) Write(d []byte) (n int, err error) {
	n, err = wc.Writer.Write(d)
	*wc.count += n
	return
}

func (wc *WriteCounter) Count() int {
	return *wc.count
}

type CopyStreams struct {
	*Copier
	Closable

	Copiers []*CopyStream
	Log     *logging.Logger

	Done chan bool
}

func NewStreamCopiers(copiers ...*CopyStream) *CopyStreams {
	s := &CopyStreams{Copiers: copiers}
	s.Copier = NewCopier(s.copy)
	s.CloseNotifier(func() {
		for _, c := range s.Copiers {
			c.NotifyClose()
		}
	})
	s.SetCloser(func(old func() error) (err error) {
		errs := Err(old())
		for i, c := range s.Copiers {
			errs = errs.Append(c.Close(), "Closer #%d failed", i)
		}
		return errs.GetError()
	})
	return s
}

func (s *CopyStreams) Append(c ...*CopyStream) {
	s.Copiers = append(s.Copiers, c...)
}

func (s *CopyStreams) copy() error {
	defer s.Close()
	var (
		qnt  = len(s.Copiers)
		done = make(chan int)
	)

	s.startAt = time.Now()

	for i, c := range s.Copiers {
		if c.Log == nil {
			c.Log = s.Log
		}
		func(i int) {
			c.Closer(func() error {
				done <- i
				return nil
			})
		}(i)
		c.Start()
	}

	for i := 0; i < qnt; i++ {
		<-done
	}

	if s.Log != nil {
		s.Log.Debugf("Closed after %s.\n", humanize.Time(s.startAt))
	}

	if s.Done != nil {
		s.Done <- true
	}

	return nil
}
