package core

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidInput    = errors.New("invalid input")
	ErrProjectNotFound = errors.New("project not found")
	ErrNoProject       = errors.New("no project found in current directory (run 'klt init' first)")
)
