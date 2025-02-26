package gen

import "errors"

var (
	ErrFunctionNotFound        = errors.New("required/linked function not found")
	ErrHandlerIllegalSignature = errors.New("handler function has an illegal signature")
	ErrMultiplePackages        = errors.New("multiple packages in one directory")
	ErrInvalidNumberBits       = errors.New("invalid number bits")
)
