package auth

import (
	"fmt"

	fberrors "go.mewis.me/meta.go/errors"
)

type Source struct {
	Cookies     Cookies
	AppState    AppState
	Credentials *Credentials
}

type AuthSnapshot struct {
	Cookies  Cookies
	AppState AppState
	Session  Session
}

func (s Source) Validate() error {
	count := 0
	if s.Cookies != nil {
		count++
	}
	if s.AppState != nil {
		count++
	}
	if s.Credentials != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("%w: exactly one authentication source is required", fberrors.ErrInvalidInput)
	}
	if s.Cookies != nil {
		if err := s.Cookies.ValidateRegular(); err != nil {
			return fmt.Errorf("%w: %v", fberrors.ErrInvalidInput, err)
		}
		return nil
	}
	if s.AppState != nil {
		_, err := s.AppState.Cookies()
		if err != nil {
			return fmt.Errorf("%w: %v", fberrors.ErrInvalidInput, err)
		}
		return nil
	}
	return s.Credentials.Validate()
}
