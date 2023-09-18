package client

import (
	"errors"
	"fmt"
	"runtime/debug"
)

func NewWrappedError(msg string, err error) error {
	if err == nil {
		return errors.New(msg)
	}

	return fmt.Errorf(msg+" %w", err)
}

func RecoverHandler(desc string, handler func(err, stack string)) {
	if err := recover(); err != nil {
		s := string(debug.Stack())
		e := fmt.Sprintf("%v", err)
		if handler != nil {
			handler(e, s)
		}
		fmt.Println(desc, e, s)
	}
}
