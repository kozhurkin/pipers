package pipers

import "errors"

type Errors []error

func (errs Errors) Join() error {
	return errors.Join(errs...)
}
