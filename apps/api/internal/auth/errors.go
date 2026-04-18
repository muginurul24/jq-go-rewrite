package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("inactive user")
	ErrDuplicateUser      = errors.New("duplicate user")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrMFARequired        = errors.New("mfa required")
	ErrMFAInvalidCode     = errors.New("invalid mfa code")
	ErrMFAAlreadyEnabled  = errors.New("mfa already enabled")
	ErrMFANotEnabled      = errors.New("mfa not enabled")
	ErrMFASetupNotPending = errors.New("mfa setup not pending")
)
