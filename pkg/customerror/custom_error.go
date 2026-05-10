package customerror

import (
	"fmt"
)

const (
	InvalidRequestError   = "INVALID_REQUEST_ERROR"
	NotFoundError         = "NOT_FOUND_ERROR"
	RequestExecutionError = "REQUEST_EXECUTION_ERROR"
	InternalError         = "INTERNAL_ERROR"
	UnknownError          = "UNKNOWN_ERROR"
)

type CustomError struct {
	ComponentName string
	Msg           string
	err           error
}

func ThrowNew(componentName string, msg string, err error) *CustomError {
	return &CustomError{
		ComponentName: componentName,
		Msg:           msg,
		err:           err,
	}
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("%s(): %s: %v", e.ComponentName, e.Msg, e.err)
}

func (e *CustomError) Unwrap() error {
	return e.err
}

func (e *CustomError) Is(tgt error) bool {
	tgtConverted, ok := tgt.(*CustomError)

	if ok && tgtConverted.Msg == e.Msg {
		return true
	}
	return false
}

func ToWrap(componentName string, err error) error {
	return fmt.Errorf("[%s] - throw error: %w", componentName, err)
}

func ExtractInfo(err error) (code string, rootCauseMsg string) {
	if customErr, ok := err.(*CustomError); ok {
		code = customErr.Msg
		if wrappedErr := customErr.Unwrap(); wrappedErr != nil {
			rootCauseMsg = wrappedErr.Error()
		}
	}
	return
}
