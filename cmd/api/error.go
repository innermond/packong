package main

import (
	"fmt"

	"github.com/pkg/errors"
)

type errid struct {
	reqid string
	err   error
}

func (e errid) wrap(cause error, txt string) error {
	var err error

	if debug {
		err = errors.Wrap(cause, txt)
	} else {
		err = errors.WithMessage(cause, txt)
	}

	e.err = err
	return e
}

func (e errid) id() string {
	return e.reqid
}

func (e errid) Error() string {
	return fmt.Sprintf("%s: %s", e.reqid, e.err.Error())
}

func (e errid) text(txt string) error {
	err := errors.WithStack(errors.New(txt))
	e.err = err
	return e

}

func (e errid) from(err error) error {
	err = errors.WithStack(err)
	e.err = err
	return e

}
