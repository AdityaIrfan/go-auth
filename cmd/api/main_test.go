package main

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestCustomValidator(t *testing.T) {
	type request struct {
		Email string `validate:"required,email"`
	}
	custom := &CustomValidator{validator: validator.New()}
	if err := custom.Validate(request{Email: "user@example.com"}); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if err := custom.Validate(request{Email: "invalid"}); err == nil {
		t.Fatal("invalid request accepted")
	}
}
