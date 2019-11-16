package jerrors

import (
	"strings"

	"google.golang.org/grpc/status"
)

// Error represents the error returned by server in response
type Error struct {
	Message    string     `json:"message"`
	Extensions *Extension `json:"extensions"`
	Paths      []string   `json:"paths"`
}

// Extension contains extra fields in the error
type Extension struct {
	Code string `json:"code"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

//NestErrorPaths is used to nest paths along with the error
func NestErrorPaths(e error, path string) error {
	err := ConvertError(e)

	newError := &Error{
		Paths: []string{path},
		Extensions: &Extension{
			Code: err.Extensions.Code,
		},
		Message: err.Message,
	}
	newError.Paths = append(newError.Paths, err.Paths...)

	return newError
}

// ConvertError converts any error to jerrors.Error
func ConvertError(e error) *Error {
	err, ok := (e).(*Error)
	if !ok {
		codeErr := status.Convert(e)

		return &Error{
			Paths: []string{},
			Extensions: &Extension{
				Code: codeErr.Code().String(),
			},
			Message: codeErr.Message(),
		}
	}

	return err
}

type MultiError struct {
	Errors []*Error
}

func (e *MultiError) Error() string {
	var s strings.Builder

	for _, e := range e.Errors {
		s.WriteString(e.Error())
		s.WriteString("\n")
	}

	return s.String()
}
