package response

import (
	"net/http"

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

func EchoResponse(c echo.Context, response Response) error {
	return c.JSON(response.StatusCode, response)
}

func SuccessResponse(statusCode int, message string, data any) Response {
	return Response{
		StatusCode: statusCode,
		Status:     StatusSuccess,
		Message:    message,
		Errors:     nil,
		Data:       data,
	}
}

func ErrorResponse(statusCode int, message string, errors map[string]string) Response {
	return Response{
		StatusCode: statusCode,
		Status:     StatusFailed,
		Message:    message,
		Data:       nil,
	}
}

func EchoResponseInvalidRequestBody(c echo.Context, errors map[string]string) error {
	return EchoResponse(c, ErrorResponse(http.StatusBadRequest, "invalid request body", errors))
}

func TokenExpiryResponse() Response {
	return ErrorResponse(http.StatusUnauthorized, "token expired", nil)
}
