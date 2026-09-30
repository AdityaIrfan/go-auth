package handlers

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"kda-auth-service/internal/core/domain"
	jwtPkg "kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"
	"net/http"
	"testing"
	"time"
)

type fakeEventService struct {
	create func(context.Context, uuid.UUID, domain.CreateEventReq) response.Response
	list   func(context.Context, uuid.UUID, domain.ListEventReq) response.Response
	update func(context.Context, uuid.UUID, uuid.UUID, domain.UpdateEventReq) response.Response
	delete func(context.Context, uuid.UUID, uuid.UUID) response.Response
}

func (f *fakeEventService) CreateEvent(c context.Context, u uuid.UUID, r domain.CreateEventReq) response.Response {
	return f.create(c, u, r)
}
func (f *fakeEventService) ListEvents(c context.Context, u uuid.UUID, r domain.ListEventReq) response.Response {
	return f.list(c, u, r)
}
func (f *fakeEventService) UpdateEvent(c context.Context, e, u uuid.UUID, r domain.UpdateEventReq) response.Response {
	return f.update(c, e, u, r)
}
func (f *fakeEventService) DeleteEvent(c context.Context, e, u uuid.UUID) response.Response {
	return f.delete(c, e, u)
}

func TestEventHandlerInputRejection(t *testing.T) {
	h := NewEventHandler(&fakeEventService{})
	for _, tc := range []struct {
		name, method, id, body, message string
		handler                         func(echo.Context) error
	}{
		{"create malformed", http.MethodPost, "", "{", "invalid request body", h.Create},
		{"create missing fields", http.MethodPost, "", "{}", "invalid request body", h.Create},
		{"create invalid time", http.MethodPost, "", `{"title":"x","start_time":"bad"}`, "invalid request body", h.Create},
		{"update invalid id", http.MethodPut, "bad", "{}", "invalid event id", h.Update},
		{"update malformed", http.MethodPut, uuid.NewString(), "{", "invalid request body", h.Update},
		{"update missing fields", http.MethodPut, uuid.NewString(), "{}", "invalid request body", h.Update},
		{"delete invalid id", http.MethodDelete, "bad", "", "invalid event id", h.Delete},
		{"list malformed", http.MethodGet, "", "{", "invalid request body", h.List},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newHandlerContext(tc.method, "/", tc.body)
			c.SetParamNames("id")
			c.SetParamValues(tc.id)
			if err := tc.handler(c); err != nil {
				t.Fatal(err)
			}
			body := responseBody(t, rec)
			if rec.Code != 400 || body["status_code"] != float64(400) || body["status"] != "failed" || body["message"] != tc.message || body["errors"] != nil {
				t.Fatalf("response=%s", rec.Body.String())
			}
		})
	}
}

func TestEventHandlersForwardClaimsAndServiceResponse(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	start, _ := time.Parse(time.RFC3339, "2026-10-01T09:00:00Z")
	end := start.Add(time.Hour)
	payload := `{"title":"meeting","description":"notes","start_time":"2026-10-01T09:00:00Z","end_time":"2026-10-01T10:00:00Z","user_id":"00000000-0000-0000-0000-000000000000"}`
	for _, name := range []string{"create", "list", "update", "delete"} {
		for _, code := range []int{200, 422} {
			t.Run(name+http.StatusText(code), func(t *testing.T) {
				called := false
				result := response.SuccessResponse(code, "service result", map[string]string{"source": "service"})
				if code == 422 {
					result = response.ErrorResponse(code, "service failure", nil)
				}
				check := func(u uuid.UUID) {
					called = true
					if u != owner {
						t.Fatal("owner must come from JWT")
					}
				}
				checkEvent := func(title, description string, a, b time.Time) {
					if title != "meeting" || description != "notes" || !a.Equal(start) || !b.Equal(end) {
						t.Fatal("incorrect payload")
					}
				}
				h := NewEventHandler(&fakeEventService{
					create: func(_ context.Context, u uuid.UUID, r domain.CreateEventReq) response.Response {
						check(u)
						checkEvent(r.Title, r.Description, r.StartTime, r.EndTime)
						return result
					},
					list: func(_ context.Context, u uuid.UUID, r domain.ListEventReq) response.Response {
						check(u)
						if !r.StartTime.Equal(start) || !r.EndTime.Equal(end) {
							t.Fatal("incorrect filter")
						}
						return result
					},
					update: func(_ context.Context, e, u uuid.UUID, r domain.UpdateEventReq) response.Response {
						check(u)
						if e != id {
							t.Fatal("incorrect event id")
						}
						checkEvent(r.Title, r.Description, r.StartTime, r.EndTime)
						return result
					},
					delete: func(_ context.Context, e, u uuid.UUID) response.Response {
						check(u)
						if e != id {
							t.Fatal("incorrect event id")
						}
						return result
					},
				})
				method := http.MethodPost
				if name == "list" {
					method = http.MethodGet
				}
				c, rec := newHandlerContext(method, "/", payload)
				c.SetParamNames("id")
				c.SetParamValues(id.String())
				c.Set("token", &jwt.Token{Claims: &jwtPkg.CustomClaims{UserID: owner}})
				handlers := map[string]func(echo.Context) error{"create": h.Create, "list": h.List, "update": h.Update, "delete": h.Delete}
				if err := handlers[name](c); err != nil {
					t.Fatal(err)
				}
				body := responseBody(t, rec)
				if !called || rec.Code != code || body["message"] != result.Message || body["status"] != string(result.Status) {
					t.Fatalf("response=%s", rec.Body.String())
				}
				if code == 200 && body["data"].(map[string]interface{})["source"] != "service" {
					t.Fatal("missing service data")
				}
			})
		}
	}
}

func TestListEventsIgnoresQueryAndDoesNotValidateRequiredFields(t *testing.T) {
	owner := uuid.New()
	called := false
	h := NewEventHandler(&fakeEventService{list: func(_ context.Context, u uuid.UUID, r domain.ListEventReq) response.Response {
		called = true
		if u != owner || !r.StartTime.IsZero() || !r.EndTime.IsZero() {
			t.Fatalf("unexpected binding: %#v", r)
		}
		return response.SuccessResponse(200, "list events success", []domain.Event{})
	}})
	c, rec := newHandlerContext(http.MethodGet, "/api/v1/events?start_time=invalid&end_time=invalid", "")
	c.Set("token", &jwt.Token{Claims: &jwtPkg.CustomClaims{UserID: owner}})
	if err := h.List(c); err != nil || !called || rec.Code != 200 {
		t.Fatalf("called=%v code=%d err=%v", called, rec.Code, err)
	}
}
