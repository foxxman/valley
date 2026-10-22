package auth

import "regexp"

const minPasswordLen = 8

var loginPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

func ValidateLogin(login string) error {
	if !loginPattern.MatchString(login) {
		return ErrInvalidLogin
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < minPasswordLen {
		return ErrInvalidPassword
	}
	return nil
}
