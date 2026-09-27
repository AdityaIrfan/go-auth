package docs

// File ini adalah satu-satunya sumber anotasi Swagger. Isinya hanya kontrak
// dokumentasi dan tidak menjalankan business logic aplikasi.

// @title KDA Auth Service API
// @version 1.0
// @description API autentikasi email/password dan Google OAuth2. Semua contoh response mengikuti envelope runtime aplikasi.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Gunakan format: Bearer {access_token}
func API() {}

type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret123" minLength:"6"`
	Name     string `json:"name" example:"Jane Doe"`
}

type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret123"`
}

type TokenData struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"86400"`
}

type RegisterSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"201"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"registered successfully"`
}

type LoginSuccessResponse struct {
	StatusCode int       `json:"status_code" example:"200"`
	Status     string    `json:"status" example:"success"`
	Message    string    `json:"message" example:"logged in successfully"`
	Data       TokenData `json:"data"`
}

type LogoutSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"logged out successfully"`
}

type ValidationErrorDetails struct {
	Email    string `json:"email,omitempty" example:"Invalid email format"`
	Password string `json:"password,omitempty" example:"This field is required"`
	Name     string `json:"name,omitempty" example:"This field is required"`
}

type ValidationErrorResponse struct {
	StatusCode int                    `json:"status_code" example:"400"`
	Status     string                 `json:"status" example:"failed"`
	Message    string                 `json:"message" example:"invalid request body"`
	Errors     ValidationErrorDetails `json:"errors,omitempty"`
}

type RegisterErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"register failed"`
}

type LoginErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"login failed"`
}

type LogoutErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"logout failed"`
}

type JWTErrorResponse struct {
	Error string `json:"error" example:"invalid or expired token"`
}

type HealthResponse struct {
	Message string `json:"message" example:"I'm healthy"`
}

type ReadyResponse struct {
	Message string `json:"message" example:"I'm ready"`
}

type ReadinessErrorResponse struct {
	Postgres string `json:"postgres" enums:"up,down" example:"down"`
	Redis    string `json:"redis" enums:"up,down" example:"up"`
}

// Register
// @Summary Register dengan email dan password
// @Description Membuat user provider email. Kegagalan service tidak membocorkan detail internal dan dikembalikan sebagai `register failed`.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration payload"
// @Success 201 {object} RegisterSuccessResponse "User berhasil dibuat"
// @Failure 400 {object} ValidationErrorResponse "JSON atau input tidak valid"
// @Failure 401 {object} RegisterErrorResponse "Registrasi gagal"
// @Router /api/v1/auth/register [post]
func Register() {}

// Login
// @Summary Login dengan email dan password
// @Description Memverifikasi user provider email dan mengembalikan JWT aplikasi yang disimpan sebagai session Redis.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login payload"
// @Success 200 {object} LoginSuccessResponse "Login berhasil"
// @Failure 400 {object} ValidationErrorResponse "JSON atau input tidak valid"
// @Failure 401 {object} LoginErrorResponse "Email/password salah atau login gagal"
// @Router /api/v1/auth/login [post]
func Login() {}

// GoogleLogin
// @Summary Mulai login Google OAuth2
// @Description Buka endpoint ini langsung melalui browser. Backend membuat URL authorization Google dengan scope openid, email, dan profile.
// @Tags Authentication
// @Success 307 "Temporary redirect ke halaman login/consent Google"
// @Header 307 {string} Location "Google authorization URL"
// @Router /api/v1/auth/google/login [get]
func GoogleLogin() {}

// GoogleCallback
// @Summary Callback Google OAuth2
// @Description Google mengirim authorization code ke endpoint ini. Backend menukar code menjadi ID token, memverifikasi identitas Google, lalu menerbitkan JWT aplikasi.
// @Tags Authentication
// @Produce json
// @Param code query string true "Authorization code dari Google"
// @Success 200 {object} LoginSuccessResponse "Login Google berhasil"
// @Failure 401 {object} LoginErrorResponse "Exchange code, ID token, atau login Google gagal"
// @Router /api/v1/auth/google/callback [get]
func GoogleCallback() {}

// Logout
// @Summary Logout dan revoke access token
// @Description JWT tidak valid ditolak middleware dengan field `error`; kegagalan revocation dikembalikan dengan envelope `logout failed`.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutSuccessResponse "Logout berhasil"
// @Failure 401 {object} LogoutErrorResponse "Token invalid/expired atau revocation gagal"
// @Router /api/v1/auth/logout [post]
func Logout() {}

// Health
// @Summary Liveness check
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func Health() {}

// Ready
// @Summary Readiness check PostgreSQL dan Redis
// @Tags System
// @Produce json
// @Success 200 {object} ReadyResponse
// @Failure 503 {object} ReadinessErrorResponse
// @Router /ready [get]
func Ready() {}
