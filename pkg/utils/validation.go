package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatValidationError(err error) map[string]string {
	var ve validator.ValidationErrors
	
	if errors.As(err, &ve) {
		errs := make(map[string]string)
		
		for _, fe := range ve {
			field := strings.ToLower(fe.Field())
			
			switch fe.Tag() {
			case "required":
				errs[field] = "This field is required"
			case "email":
				errs[field] = "Invalid email format"
			case "min":
				errs[field] = fmt.Sprintf("Must be at least %s characters long", fe.Param())
			case "max":
				errs[field] = fmt.Sprintf("Must be at most %s characters long", fe.Param())
			default:
				errs[field] = fmt.Sprintf("Failed on '%s' validation", fe.Tag())
			}
		}
		return errs
	}

	return map[string]string{"request": err.Error()}
}