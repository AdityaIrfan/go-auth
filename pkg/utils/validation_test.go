package utils

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestFormatValidationError(t *testing.T) {
	type input struct {
		Required string `validate:"required"`
		Email    string `validate:"email"`
		Min      string `validate:"min=3"`
		Max      string `validate:"max=2"`
		Number   string `validate:"numeric"`
	}
	err := validator.New().Struct(input{Email: "bad", Min: "x", Max: "long", Number: "abc"})
	got := FormatValidationError(err)
	want := map[string]string{
		"required": "This field is required",
		"email":    "Invalid email format",
		"min":      "Must be at least 3 characters long",
		"max":      "Must be at most 2 characters long",
		"number":   "Failed on 'numeric' validation",
	}
	for field, message := range want {
		if got[field] != message {
			t.Errorf("%s=%q, want %q", field, got[field], message)
		}
	}
}

func TestFormatValidationErrorFallback(t *testing.T) {
	got := FormatValidationError(errors.New("cannot decode"))
	if got["request"] != "cannot decode" {
		t.Fatalf("got=%#v", got)
	}
}
