package handlers

import "kda-auth-service/internal/core/domain"

// Swagger-only types mirror the actual response envelopes returned at runtime.

type RegisterSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"201"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"registered successfully"`
}

type TokenSuccessResponse struct {
	StatusCode int              `json:"status_code" example:"200"`
	Status     string           `json:"status" example:"success"`
	Message    string           `json:"message" example:"logged in successfully"`
	Data       domain.TokenResp `json:"data"`
}

type GoogleTokenSuccessResponse struct {
	StatusCode int              `json:"status_code" example:"200"`
	Status     string           `json:"status" example:"success"`
	Message    string           `json:"message" example:"logged in with google successfully"`
	Data       domain.TokenResp `json:"data"`
}

type LogoutSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"logged out successfully"`
}

type ValidationErrorResponse struct {
	StatusCode int                    `json:"status_code" example:"400"`
	Status     string                 `json:"status" example:"failed"`
	Message    string                 `json:"message" example:"invalid request body"`
	Errors     ValidationErrorDetails `json:"errors"`
}

type ValidationErrorDetails struct {
	Email    string `json:"email,omitempty" example:"Invalid email format"`
	Password string `json:"password,omitempty" example:"This field is required"`
	Name     string `json:"name,omitempty" example:"This field is required"`
	IDToken  string `json:"idtoken,omitempty" example:"This field is required"`
}

type DuplicateEmailErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"email already registered"`
}

type InvalidCredentialsErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid credentials"`
}

type InvalidGoogleTokenErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid google id_token"`
}

type JWTErrorResponse struct {
	Error string `json:"error" example:"invalid or expired token"`
}

type LogoutErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"failed to logout"`
}

type GoogleOAuthConfigErrorResponse struct {
	StatusCode int    `json:"status_code" example:"503"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"google oauth is not configured"`
}

type OAuthStateGenerationErrorResponse struct {
	StatusCode int    `json:"status_code" example:"500"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"failed to initialize google oauth"`
}

type GoogleOAuthCallbackErrorResponse struct {
	StatusCode int    `json:"status_code" example:"400"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid oauth state"`
}

type GoogleAuthorizationDeniedResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"google authorization denied"`
}

type GoogleOAuthExchangeErrorResponse struct {
	StatusCode int    `json:"status_code" example:"502"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"failed to exchange google authorization code"`
}

type HealthSuccessResponse struct {
	Message string `json:"message" example:"I'm healthy"`
}

type ReadySuccessResponse struct {
	Message string `json:"message" example:"I'm ready"`
}

type ReadinessErrorResponse struct {
	Postgres string `json:"postgres" example:"down"`
	Redis    string `json:"redis" example:"up"`
}
