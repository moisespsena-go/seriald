package seriald

import (
	"fmt"
	"strings"
)

type Errors []string

func (e Errors) Error() string {
	return strings.Join(e, " ~~~ ")
}

func (e Errors) GetError() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

func (errs Errors) Err(err error, msgf ...interface{}) Errors {
	return errs.Append(err, msgf...)
}

func (errs Errors) Append(err error, msgf ...interface{}) Errors {
	if err != nil {
		if len(msgf) > 0 && msgf[0] != "" {
			errs = append(errs, fmt.Sprintf(msgf[0].(string), msgf[1:]...)+": "+err.Error())
		} else {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func Err(err error, msgf ...interface{}) (e Errors) {
	return e.Err(err, msgf...)
}
