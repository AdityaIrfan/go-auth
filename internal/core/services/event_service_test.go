package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/pkg/response"
	"testing"
	"time"
)

type fakeEventRepository struct {
	create func(context.Context, *domain.Event) error
	list   func(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.Event, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (*domain.Event, error)
	update func(context.Context, *domain.Event) error
	delete func(context.Context, uuid.UUID, uuid.UUID) error
}

func (f *fakeEventRepository) Create(c context.Context, e *domain.Event) error { return f.create(c, e) }
func (f *fakeEventRepository) GetByUserID(context.Context, uuid.UUID) ([]domain.Event, error) {
	panic("unexpected unfiltered lookup")
}
func (f *fakeEventRepository) GetByDateRange(c context.Context, u uuid.UUID, a, b time.Time) ([]domain.Event, error) {
	return f.list(c, u, a, b)
}
func (f *fakeEventRepository) GetByID(c context.Context, e, u uuid.UUID) (*domain.Event, error) {
	return f.get(c, e, u)
}
func (f *fakeEventRepository) Update(c context.Context, e *domain.Event) error { return f.update(c, e) }
func (f *fakeEventRepository) Delete(c context.Context, e, u uuid.UUID) error {
	return f.delete(c, e, u)
}

func assertServiceResponse(t *testing.T, r response.Response, code int, message string) {
	t.Helper()
	status := response.StatusSuccess
	if code >= 400 {
		status = response.StatusFailed
	}
	if r.StatusCode != code || r.Message != message || r.Status != status {
		t.Fatalf("response=%#v, want %d %q", r, code, message)
	}
}

func TestCreateEvent(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	owner := uuid.New()
	for _, tc := range []struct {
		name     string
		duration time.Duration
		repoErr  error
		code     int
		message  string
	}{
		{"success", time.Hour, nil, 201, "create event success"},
		{"equal times", 0, nil, 400, "end_time must be after start_time"},
		{"end before start", -time.Hour, nil, 400, "end_time must be after start_time"},
		{"repository error", time.Hour, errors.New("db"), 422, "create event failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			repo := &fakeEventRepository{create: func(_ context.Context, e *domain.Event) error {
				called = true
				if e.ID == uuid.Nil || e.UserID != owner || e.Title != "meeting" || e.Description != "notes" || !e.StartTime.Equal(start) || !e.EndTime.Equal(start.Add(tc.duration)) || e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
					t.Fatalf("event=%#v", e)
				}
				return tc.repoErr
			}}
			r := NewEventService(repo).CreateEvent(context.Background(), owner, domain.CreateEventReq{Title: "meeting", Description: "notes", StartTime: start, EndTime: start.Add(tc.duration)})
			assertServiceResponse(t, r, tc.code, tc.message)
			if called != (tc.duration > 0) {
				t.Fatalf("repository called=%v", called)
			}
			if tc.code == 201 {
				if e, ok := r.Data.(*domain.Event); !ok || e.UserID != owner {
					t.Fatalf("data=%#v", r.Data)
				}
			} else if r.Data != nil {
				t.Fatal("error exposes data")
			}
		})
	}
}

func TestListEvents(t *testing.T) {
	owner := uuid.New()
	start := time.Now()
	end := start.Add(time.Hour)
	for _, fail := range []bool{false, true} {
		repo := &fakeEventRepository{list: func(_ context.Context, u uuid.UUID, a, b time.Time) ([]domain.Event, error) {
			if u != owner || !a.Equal(start) || !b.Equal(end) {
				t.Fatal("incorrect owner/date filter")
			}
			if fail {
				return nil, errors.New("db")
			}
			return []domain.Event{{UserID: owner, Title: "meeting"}}, nil
		}}
		r := NewEventService(repo).ListEvents(context.Background(), owner, domain.ListEventReq{StartTime: start, EndTime: end})
		if fail {
			assertServiceResponse(t, r, 422, "list events failed")
		} else {
			assertServiceResponse(t, r, 200, "list events success")
			if events, ok := r.Data.([]domain.Event); !ok || len(events) != 1 || events[0].UserID != owner {
				t.Fatalf("data=%#v", r.Data)
			}
		}
	}
}

func TestUpdateEvent(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	start := time.Now()
	for _, tc := range []struct {
		name               string
		delta              time.Duration
		lookupErr, saveErr error
		code               int
		message            string
	}{
		{"success", time.Hour, nil, nil, 200, "update event success"},
		{"equal times accepted", 0, nil, nil, 200, "update event success"},
		{"end before start", -time.Hour, nil, nil, 400, "end_time must be after start_time"},
		{"lookup error", time.Hour, errors.New("db"), nil, 404, "event not found"},
		{"save error", time.Hour, nil, errors.New("db"), 422, "update event failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saved := false
			created := start.Add(-time.Hour)
			repo := &fakeEventRepository{
				get: func(_ context.Context, e, u uuid.UUID) (*domain.Event, error) {
					if e != id || u != owner {
						t.Fatal("incorrect ownership lookup")
					}
					return &domain.Event{ID: id, UserID: owner, CreatedAt: created}, tc.lookupErr
				},
				update: func(_ context.Context, e *domain.Event) error {
					saved = true
					if e.ID != id || e.UserID != owner || e.Title != "updated" || e.Description != "notes" || !e.StartTime.Equal(start) || !e.EndTime.Equal(start.Add(tc.delta)) || !e.CreatedAt.Equal(created) || e.UpdatedAt.IsZero() {
						t.Fatalf("event=%#v", e)
					}
					return tc.saveErr
				},
			}
			r := NewEventService(repo).UpdateEvent(context.Background(), id, owner, domain.UpdateEventReq{Title: "updated", Description: "notes", StartTime: start, EndTime: start.Add(tc.delta)})
			assertServiceResponse(t, r, tc.code, tc.message)
			if saved != (tc.lookupErr == nil && tc.delta >= 0) {
				t.Fatalf("save called=%v", saved)
			}
		})
	}
}

func TestDeleteEvent(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name                 string
		lookupErr, deleteErr error
		code                 int
		message              string
	}{
		{"success", nil, nil, 200, "delete event success"}, {"lookup failure", errors.New("missing"), nil, 404, "event not found"}, {"delete failure", nil, errors.New("db"), 422, "delete event failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deleted := false
			repo := &fakeEventRepository{get: func(_ context.Context, e, u uuid.UUID) (*domain.Event, error) {
				if e != id || u != owner {
					t.Fatal("incorrect lookup")
				}
				return &domain.Event{ID: id}, tc.lookupErr
			}, delete: func(_ context.Context, e, u uuid.UUID) error {
				deleted = true
				if e != id || u != owner {
					t.Fatal("incorrect delete scope")
				}
				return tc.deleteErr
			}}
			r := NewEventService(repo).DeleteEvent(context.Background(), id, owner)
			assertServiceResponse(t, r, tc.code, tc.message)
			if deleted != (tc.lookupErr == nil) || r.Data != nil {
				t.Fatalf("deleted=%v response=%#v", deleted, r)
			}
		})
	}
}
