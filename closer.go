package seriald

import (
	"io"
	"time"
)

type Closer interface {
	io.Closer
	Closer(f ...func())
	SetCloser(f func(old func() error) error)
	Done() chan error
}

type MustNotifyCloser interface {
	NotifyClose()
}

type NotifyCloser interface {
	Closer
	MustNotifyCloser
	CloseNotifier(f ...func())
	SetCloseNotifier(f func(old func()))
}

type Closable struct {
	closers      []func() error
	closeFunc    func() error
	afterClosers []func()

	closeNotifyFunc func()
	notifiers       []func()

	notified bool
	closedAt time.Time
	done     chan error
}

func (c *Closable) IsClosed() bool {
	return !c.closedAt.IsZero()
}

func (s *Closable) IsCloseNotified() bool {
	return s.notified
}

func (c *Closable) Closer(f ...func() error) {
	c.closers = append(c.closers, f...)
}

func (c *Closable) AfterClose(f ...func()) {
	c.afterClosers = append(c.afterClosers, f...)
}

func (c *Closable) SetCloser(f func(old func() error) error) {
	old := c.closeFunc
	if old == nil {
		old = func() error { return nil }
	}
	c.closeFunc = func() error {
		return f(old)
	}
}

func (c *Closable) SetCloseNotifier(f func(old func())) {
	old := c.closeNotifyFunc
	if old == nil {
		old = func() {}
	}
	c.closeNotifyFunc = func() {
		f(old)
	}
}

func (c *Closable) CloseNotifier(f ...func()) {
	c.notifiers = append(c.notifiers, f...)
}

func (c *Closable) Done() chan error {
	if c.done == nil {
		c.done = make(chan error)
	}
	return c.done
}

func (s *Closable) MustClose() (err error) {
	if s.IsClosed() {
		return nil
	}

	s.closedAt = time.Now()

	if s.closeFunc != nil {
		err = s.closeFunc()
	}

	var errs Errors

	for i, f := range s.closers {
		errs.Append(f(), "Closer #%d", i)
	}

	s.closers = nil

	for _, f := range s.afterClosers {
		f()
	}

	return
}

func (s *Closable) NotifyClose() {
	if s.notified {
		return
	}
	s.notified = true

	if s.closeNotifyFunc != nil {
		s.closeNotifyFunc()
	}

	for _, f := range s.notifiers {
		f()
	}
}

func (s *Closable) Close() (err error) {
	if s.IsClosed() {
		return nil
	}
	s.NotifyClose()
	err = s.MustClose()

	if s.done != nil {
		s.done <- err
	}
	return
}
