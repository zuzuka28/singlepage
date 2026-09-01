package config

import (
	"errors"
	"fmt"
)

var errInvalidEnvironment = errors.New("invalid environment configuration")

func invalid(name, message string) error {
	return fmt.Errorf("%w: %s %s", errInvalidEnvironment, name, message)
}
