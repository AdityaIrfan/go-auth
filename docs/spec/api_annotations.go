package docs

// Documentation-only source; runtime handlers and services remain unchanged.
// @title KDA Auth and Calendar Service API
// @version 1.0
// @description Authentication, refresh tokens, and user-owned calendar events. Responses mirror the current runtime, including its documented limitations.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer {access_token}
func API() {}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" format:"email" example:"user@example.com"`
	Password string `json:"password" validate:"required" minLength:"6" example:"secret123"`
	Name     string `json:"name" validate:"required" example:"Jane Doe"`
}
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" format:"email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"secret123"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"<refresh-jwt>"`
}
type TokenData struct {
	AccessToken  string `json:"access_token" example:"<access-jwt>"`
	RefreshToken string `json:"refresh_token" example:"<refresh-jwt>"`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"86400"`
}
type EventRequest struct {
	Title       string `json:"title" validate:"required" example:"Team meeting"`
	Description string `json:"description" example:"Weekly planning"`
	StartTime   string `json:"start_time" validate:"required" format:"date-time" example:"2026-10-01T09:00:00+07:00"`
	EndTime     string `json:"end_time" validate:"required" format:"date-time" example:"2026-10-01T10:00:00+07:00"`
}

// ListEventRequest is bound from a JSON body. The runtime has no query tags and does not validate this request.
type ListEventRequest struct {
	StartTime string `json:"start_time,omitempty" format:"date-time" example:"2026-10-01T00:00:00+07:00"`
	EndTime   string `json:"end_time,omitempty" format:"date-time" example:"2026-10-02T00:00:00+07:00"`
}
type Event struct {
	ID          string `json:"id" format:"uuid" example:"c27096d8-58d1-4014-9830-96f36ab04c9f"`
	UserID      string `json:"user_id" format:"uuid" example:"c8721202-510d-4a9a-b1d5-30b78c01d73b"`
	Title       string `json:"title" example:"Team meeting"`
	Description string `json:"description" example:"Weekly planning"`
	StartTime   string `json:"start_time" format:"date-time" example:"2026-10-01T09:00:00+07:00"`
	EndTime     string `json:"end_time" format:"date-time" example:"2026-10-01T10:00:00+07:00"`
	CreatedAt   string `json:"created_at" format:"date-time" example:"2026-09-30T08:00:00Z"`
	UpdatedAt   string `json:"updated_at" format:"date-time" example:"2026-09-30T08:00:00Z"`
}

type RegisterSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"201"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"register success, do login"`
}

type LoginSuccessResponse struct {
	StatusCode int       `json:"status_code" example:"200"`
	Status     string    `json:"status" example:"success"`
	Message    string    `json:"message" example:"login success"`
	Data       TokenData `json:"data"`
}

type RefreshSuccessResponse struct {
	StatusCode int       `json:"status_code" example:"200"`
	Status     string    `json:"status" example:"success"`
	Message    string    `json:"message" example:"refresh token success"`
	Data       TokenData `json:"data"`
}

type LogoutSuccessResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"logout success"`
}

type HealthResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"I'm healthy"`
}

type ReadyResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"I'm ready"`
}

type ValidationErrorResponse struct {
	StatusCode int    `json:"status_code" example:"400"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid request body"`
}

type DuplicateEmailResponse struct {
	StatusCode int    `json:"status_code" example:"400"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"email already registered"`
}

type RegisterErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"register failed"`
}

type CredentialErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid credentials"`
}

type LoginErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"login failed"`
}

type OAuthExchangeErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"login failed"`
}

type RefreshErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"refresh token failed"`
}

type LogoutErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"logout failed"`
}

type JWTErrorResponse struct {
	StatusCode int    `json:"status_code" example:"401"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid or expired token"`
}

type ReadinessErrorResponse struct {
	StatusCode int    `json:"status_code" example:"503"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"unavailable"`
}

type CreateEventResponse struct {
	StatusCode int    `json:"status_code" example:"201"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"create event success"`
	Data       Event  `json:"data"`
}

type ListEventsResponse struct {
	StatusCode int     `json:"status_code" example:"200"`
	Status     string  `json:"status" example:"success"`
	Message    string  `json:"message" example:"list events success"`
	Data       []Event `json:"data"`
}

type UpdateEventResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"update event success"`
	Data       Event  `json:"data"`
}

type DeleteEventResponse struct {
	StatusCode int    `json:"status_code" example:"200"`
	Status     string `json:"status" example:"success"`
	Message    string `json:"message" example:"delete event success"`
}

type EventTimeErrorResponse struct {
	StatusCode int    `json:"status_code" example:"400"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"end_time must be after start_time"`
}

type EventIDErrorResponse struct {
	StatusCode int    `json:"status_code" example:"400"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"invalid event id"`
}

type EventNotFoundResponse struct {
	StatusCode int    `json:"status_code" example:"404"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"event not found"`
}

type CreateEventErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"create event failed"`
}

type ListEventsErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"list events failed"`
}

type UpdateEventErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"update event failed"`
}

type DeleteEventErrorResponse struct {
	StatusCode int    `json:"status_code" example:"422"`
	Status     string `json:"status" example:"failed"`
	Message    string `json:"message" example:"delete event failed"`
}

// Register
// @Summary Register email/password
// @Description Requires valid email, password of at least six characters, and name. Duplicate email returns 400 with message email already registered.
// @Tags Authentication
// @Produce json
// @Accept json
// @Param request body RegisterRequest true "Request JSON"
// @Success 201 {object} RegisterSuccessResponse "User created"
// @Failure 400 {object} ValidationErrorResponse "Invalid input or duplicate email (message: email already registered)"
// @Failure 422 {object} RegisterErrorResponse "Lookup, password hashing, or persistence failure"
// @Router /api/v1/auth/register [post]
func Register() {}

// Login
// @Summary Login email/password
// @Description Returns access and refresh JWTs. Access token is stored in Redis. Refresh lifetime is access lifetime plus one hour.
// @Tags Authentication
// @Produce json
// @Accept json
// @Param request body LoginRequest true "Request JSON"
// @Success 200 {object} LoginSuccessResponse "Token pair"
// @Failure 400 {object} ValidationErrorResponse "Invalid JSON or required fields; errors details are omitted by the current response helper"
// @Failure 401 {object} CredentialErrorResponse "Invalid credentials or non-email provider"
// @Failure 422 {object} LoginErrorResponse "Token generation or Redis store failure"
// @Router /api/v1/auth/login [post]
func Login() {}

// GoogleLogin
// @Summary Start Google OAuth2 login
// @Description Open in a browser; redirects with openid, email, profile scopes. Generated state is currently not persisted or validated.
// @Tags Authentication
// @Produce json
// @Success 307 "Temporary redirect to Google"
// @Header 307 {string} Location "Google authorization URL"
// @Router /api/v1/auth/google/login [get]
func GoogleLogin() {}

// GoogleCallback
// @Summary Google OAuth2 callback
// @Description Exchanges authorization code for an ID token, checks Google tokeninfo, finds or creates the user by email, and issues application tokens.
// @Tags Authentication
// @Produce json
// @Param code query string true "Google authorization code"
// @Success 200 {object} LoginSuccessResponse "Token pair"
// @Failure 401 {object} OAuthExchangeErrorResponse "Code exchange failed or ID token missing"
// @Failure 422 {object} LoginErrorResponse "Tokeninfo, audience, repository, or token persistence failure"
// @Failure 500 {object} RecoveryErrorResponse "Current tokeninfo missing-claims panic, recovered by application middleware"
// @Router /api/v1/auth/google/callback [get]
func GoogleCallback() {}

// Refresh
// @Summary Refresh token pair
// @Description Public route. Validates the submitted JWT, looks up its user, and returns a new pair. No Redis validation or refresh-token rotation/revocation check is performed. Current verifier does not distinguish access tokens from refresh tokens.
// @Tags Authentication
// @Produce json
// @Accept json
// @Param request body RefreshRequest true "Request JSON"
// @Success 200 {object} RefreshSuccessResponse "Token pair"
// @Failure 400 {object} ValidationErrorResponse "Invalid JSON or required fields; errors details are omitted by the current response helper"
// @Failure 422 {object} RefreshErrorResponse "Invalid JWT, database, token generation, or Redis failure"
// @Failure 500 {object} RecoveryErrorResponse "Current missing-user panic, recovered by application middleware"
// @Router /api/v1/auth/refresh [post]
func Refresh() {}

// Logout
// @Summary Revoke access token in Redis
// @Description Deletes the supplied access token from Redis. Middleware validates JWT signature/expiry only; it does not consult Redis.
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutSuccessResponse "Logged out"
// @Failure 401 {object} JWTErrorResponse "Missing, invalid, or expired JWT"
// @Failure 422 {object} LogoutErrorResponse "Redis revoke failed"
// @Router /api/v1/auth/logout [post]
func Logout() {}

// Health
// @Summary Liveness check
// @Description Returns the standard success envelope.
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse "Healthy"
// @Router /health [get]
func Health() {}

// Ready
// @Summary Readiness check
// @Description Pings PostgreSQL and Redis. Dependency details are currently discarded by the response helper.
// @Tags System
// @Produce json
// @Success 200 {object} ReadyResponse "Ready"
// @Failure 503 {object} ReadinessErrorResponse "One or both dependencies unavailable"
// @Router /ready [get]
func Ready() {}

// CreateEvent
// @Summary Create event
// @Description User ID comes from JWT. Title and timestamps are required; end_time must be strictly after start_time. Date-time values use RFC3339.
// @Tags Events
// @Produce json
// @Accept json
// @Param request body EventRequest true "Request JSON"
// @Security BearerAuth
// @Success 201 {object} CreateEventResponse "Created"
// @Failure 400 {object} ValidationErrorResponse "Invalid input or end_time must be after start_time"
// @Failure 401 {object} JWTErrorResponse "Missing, invalid, or expired JWT"
// @Failure 422 {object} CreateEventErrorResponse "Persistence failure"
// @Router /api/v1/events [post]
func CreateEvent() {}

// ListEvents
// @Summary List user events
// @Description Optional JSON body filters start_time >= supplied start_time and end_time <= supplied end_time, ordered by start_time ASC. Query parameters are ignored because request fields have no query tags. Empty body returns all user events. GET bodies may not be supported by browser Swagger UI; use curl for filtered requests. No request validation is performed.
// @Tags Events
// @Produce json
// @Accept json
// @Param request body ListEventRequest false "Request JSON"
// @Security BearerAuth
// @Success 200 {object} ListEventsResponse "Events, empty array when none"
// @Failure 400 {object} ValidationErrorResponse "Invalid JSON or required fields; errors details are omitted by the current response helper"
// @Failure 401 {object} JWTErrorResponse "Missing, invalid, or expired JWT"
// @Failure 422 {object} ListEventsErrorResponse "Query failure"
// @Router /api/v1/events [get]
func ListEvents() {}

// UpdateEvent
// @Summary Replace event fields
// @Description Only the JWT owner can update. Title and both times are required. Unlike create, equal start_time and end_time are currently accepted. Lookup errors all return 404.
// @Tags Events
// @Produce json
// @Accept json
// @Param request body EventRequest true "Request JSON"
// @Security BearerAuth
// @Param id path string true "Event UUID" Format(uuid)
// @Success 200 {object} UpdateEventResponse "Updated"
// @Failure 400 {object} ValidationErrorResponse "Invalid event id, invalid input, or end_time must be after start_time"
// @Failure 401 {object} JWTErrorResponse "Missing, invalid, or expired JWT"
// @Failure 404 {object} EventNotFoundResponse "Not found for this user or lookup failure"
// @Failure 422 {object} UpdateEventErrorResponse "Persistence failure"
// @Router /api/v1/events/{id} [put]
func UpdateEvent() {}

// DeleteEvent
// @Summary Delete event
// @Description Only the JWT owner can delete. Lookup errors all return 404.
// @Tags Events
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event UUID" Format(uuid)
// @Success 200 {object} DeleteEventResponse "Deleted"
// @Failure 400 {object} EventIDErrorResponse "Invalid event UUID"
// @Failure 401 {object} JWTErrorResponse "Missing, invalid, or expired JWT"
// @Failure 404 {object} EventNotFoundResponse "Not found for this user or lookup failure"
// @Failure 422 {object} DeleteEventErrorResponse "Delete failure"
// @Router /api/v1/events/{id} [delete]
func DeleteEvent() {}

// RecoveryErrorResponse is emitted by Echo Recover for existing panic paths.
type RecoveryErrorResponse struct {
	Message string `json:"message" example:"Internal Server Error"`
}
