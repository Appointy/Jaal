package internal

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error represents the error returned by server in response
type Error struct {
	Message    string     `json:"message"`
	Extensions *extension `json:"extensions"`
	Paths      []string   `json:"paths"`
}

type extension struct {
	Code string `json:"code"`
}

func (e *Error) Error() string {
	return e.Message
}

//NestErrorPaths is used to nest paths along with the error
func NestErrorPaths(e error, path string) error {
	err := ConvertError(e)

	newError := &Error{
		Paths: []string{path},
		Extensions: &extension{
			Code: err.Extensions.Code,
		},
		Message: err.Message,
	}
	newError.Paths = append(newError.Paths, err.Paths...)

	return newError
}

// ConvertError converts any error to internal.Error
func ConvertError(e error) *Error {
	err, ok := (e).(*Error)
	if !ok {
		codeErr, statusError := status.FromError(err)
		if statusError {
			return &Error{
				Paths: []string{},
				Extensions: &extension{
					Code: codeErr.Code().String(),
				},
				Message: codeErr.Message(),
			}
		}

		return &Error{
			Paths: []string{},
			Extensions: &extension{
				Code: codes.Unknown.String(),
			},
			Message: e.Error(),
		}
	}

	return err
}
