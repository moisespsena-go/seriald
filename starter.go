package seriald

import "time"

type Starter struct {
	startAt     time.Time
	startFunc   func() error
	startErrors []func(err error)
}

func (s *Starter) SetStarter(f func() error) {
	s.startFunc = f
}
func (s *Starter) StartError(onError ...func(err error)) {
	s.startErrors = append(s.startErrors, onError...)
}

func (s *Starter) start() {
	if err := s.startFunc(); err != nil {
		for _, f := range s.startErrors {
			f(err)
		}
	}
}

func (s *Starter) Start() {
	if s.startFunc != nil {
		go s.start()
	}
}
