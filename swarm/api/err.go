package api

type UnsupportedError struct {
	Message string
}

func (e *UnsupportedError) Error() string {
	return "unsupported: " + e.Message
}

func NewUnsupportedError(msg string) error {
	return &UnsupportedError{Message: msg}
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return "not found: " + e.Message
}

func NewNotFoundError(msg string) error {
	return &NotFoundError{Message: msg}
}

type BadRequestError struct {
	Message string
}

func (e *BadRequestError) Error() string {
	return "bad request: " + e.Message
}

func NewBadRequestError(msg string) error {
	return &BadRequestError{Message: msg}
}

type InternalServerError struct {
	Message string
}

func (e *InternalServerError) Error() string {
	return "internal server error: " + e.Message
}

func NewInternalServerError(msg string) error {
	return &InternalServerError{Message: msg}
}

type UnauthorizedError struct {
	Message string
}

func (e *UnauthorizedError) Error() string {
	return "unauthorized: " + e.Message
}

func NewUnauthorizedError(msg string) error {
	return &UnauthorizedError{Message: msg}
}
