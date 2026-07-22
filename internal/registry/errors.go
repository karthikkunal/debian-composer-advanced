package registry

import "errors"

var (
	ErrSourceNotFound   = errors.New("registry source not found")
	ErrDuplicateSource  = errors.New("registry source already exists")
	ErrRecipeNotFound   = errors.New("remote recipe not found")
	ErrChecksumMismatch = errors.New("recipe checksum mismatch")
	ErrInsecureURL      = errors.New("insecure registry URL")
	ErrResponseTooLarge = errors.New("registry response exceeds size limit")
	ErrNotModified      = errors.New("registry index not modified")
)
