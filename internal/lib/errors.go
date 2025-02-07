package lib

type ServiceErrorType string

const (
	ErrTypeBadRequest = "bad_request"
	ErrTypeInternal   = "internal"
	ErrTypeNotFound   = "not_found"
	ErrTypeConflict   = "conflict"
)

// NewServiceError creates a new ServiceError
func NewServiceError(errType ServiceErrorType, err error) ServiceError {
	return ServiceError{
		Type: errType,
		Err:  err,
	}
}

type ServiceError struct {
	Type ServiceErrorType
	Err  error
}

func (e ServiceError) Error() string {
	return e.Err.Error()
}
