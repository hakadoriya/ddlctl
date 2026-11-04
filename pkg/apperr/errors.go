package apperr

import (
	"errors"
	"fmt"
)

var (
	ErrNotSupported                       = errors.New("not supported")
	ErrCanceled                           = errors.New("canceled")
	ErrDialectIsEmpty                     = errors.New("dialect is empty")
	ErrDDLTagGoAnnotationNotFoundInSource = errors.New("go-ddl-tag annotation not found in source")
	ErrTwoArgumentsRequired               = errors.New("two arguments required")
	ErrBothArgumentsIsDSN                 = errors.New("both arguments is dsn")
	ErrBothArgumentsAreNotDSNOrSQLFile    = errors.New("both arguments are not dsn or sql file")
	ErrUnexpectedColumnOrdering           = errors.New("unexpected column ordering")
)

//nolint:gochecknoglobals
var Errorf = fmt.Errorf
