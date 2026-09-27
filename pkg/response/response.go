package response

import (
	"github.com/labstack/echo/v4"
)

type ResponseStatus string

const (
	StatusSuccess ResponseStatus = "success"
	StatusFailed  ResponseStatus = "failed"
)

type Response struct {
	StatusCode int               `json:"status_code"`
	Status     ResponseStatus    `json:"status"`
	Message    string            `json:"message"`
	Data       any               `json:"data,omitempty"`
	Errors     map[string]string `json:"errors,omitempty"`
}

func SuccessResponse(c echo.Context, statusCode int, message string, data any) error {
	response := &Response{
		StatusCode: statusCode,
		Status:     StatusSuccess,
		Message:    message,
		Errors:     nil,
	}

	if data != nil {
		response.Data = data
	}

	return c.JSON(statusCode, response)
}

func ErrorResponse(c echo.Context, statusCode int, message string, errors map[string]string) error {
	response := &Response{
		StatusCode: statusCode,
		Status:     StatusFailed,
		Message:    message,
		Data:       nil,
	}

	if errors != nil {
		response.Errors = errors
	}

	return c.JSON(statusCode, response)
}
