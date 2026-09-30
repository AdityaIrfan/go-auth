package repositories

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"kda-auth-service/internal/core/domain"
	"regexp"
	"testing"
	"time"
)

func TestPGEventDateRange(t *testing.T) {
	owner := uuid.New()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	for _, tc := range []struct {
		name       string
		start, end time.Time
	}{{"no bounds", time.Time{}, time.Time{}}, {"lower bound", start, time.Time{}}, {"upper bound", time.Time{}, end}, {"both bounds", start, end}} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGorm(t)
			repo := NewPGEventRepository(db)
			sql := `SELECT * FROM "events" WHERE user_id = $1`
			if !tc.start.IsZero() {
				sql += ` AND start_time >= $2`
			}
			if !tc.end.IsZero() {
				if tc.start.IsZero() {
					sql += ` AND end_time <= $2`
				} else {
					sql += ` AND end_time <= $3`
				}
			}
			sql += ` ORDER BY start_time ASC`
			q := mock.ExpectQuery(regexp.QuoteMeta(sql))
			switch {
			case tc.start.IsZero() && tc.end.IsZero():
				q.WithArgs(owner)
			case tc.start.IsZero():
				q.WithArgs(owner, tc.end)
			case tc.end.IsZero():
				q.WithArgs(owner, tc.start)
			default:
				q.WithArgs(owner, tc.start, tc.end)
			}
			q.WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "start_time", "end_time"}).AddRow(uuid.New(), owner, "meeting", start, end))
			events, err := repo.GetByDateRange(context.Background(), owner, tc.start, tc.end)
			if err != nil || len(events) != 1 || events[0].UserID != owner || events[0].Title != "meeting" {
				t.Fatalf("events=%#v err=%v", events, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPGEventLookupOwnership(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name string
		err  error
	}{{"found", nil}, {"not found", gorm.ErrRecordNotFound}, {"database failure", errors.New("db")}} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGorm(t)
			repo := NewPGEventRepository(db)
			q := mock.ExpectQuery(`SELECT .* FROM "events" WHERE id = \$1 AND user_id = \$2.*LIMIT \$3`).WithArgs(id, owner, 1)
			if tc.err != nil {
				q.WillReturnError(tc.err)
			} else {
				q.WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(id, owner))
			}
			event, err := repo.GetByID(context.Background(), id, owner)
			if !errors.Is(err, tc.err) {
				t.Fatalf("err=%v", err)
			}
			if tc.err == nil {
				if event == nil || event.ID != id || event.UserID != owner {
					t.Fatalf("event=%#v", event)
				}
			} else if event != nil {
				t.Fatal("unexpected event")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPGEventWrites(t *testing.T) {
	owner, id := uuid.New(), uuid.New()
	now := time.Now()
	for _, operation := range []string{"create", "update", "delete"} {
		for _, fail := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " success", true: " failure"}[fail], func(t *testing.T) {
				db, mock := newMockGorm(t)
				repo := NewPGEventRepository(db)
				want := errors.New("write failed")
				event := &domain.Event{ID: id, UserID: owner, Title: "meeting", Description: "notes", StartTime: now, EndTime: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}
				mock.ExpectBegin()
				var q *sqlmock.ExpectedExec
				switch operation {
				case "create":
					q = mock.ExpectExec(`INSERT INTO "events"`).WithArgs(id, owner, "meeting", "notes", event.StartTime, event.EndTime, now, now)
				case "update":
					q = mock.ExpectExec(`UPDATE "events" SET .* WHERE "id" = \$8`).WithArgs(owner, "meeting", "notes", event.StartTime, event.EndTime, now, sqlmock.AnyArg(), id)
				case "delete":
					q = mock.ExpectExec(`DELETE FROM "events" WHERE id = \$1 AND user_id = \$2`).WithArgs(id, owner)
				}
				if fail {
					q.WillReturnError(want)
					mock.ExpectRollback()
				} else {
					q.WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
				var err error
				switch operation {
				case "create":
					err = repo.Create(context.Background(), event)
				case "update":
					err = repo.Update(context.Background(), event)
				case "delete":
					err = repo.Delete(context.Background(), id, owner)
				}
				if fail {
					if !errors.Is(err, want) {
						t.Fatalf("err=%v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestPGEventListEmptyAndFailure(t *testing.T) {
	for _, byRange := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			db, mock := newMockGorm(t)
			repo := NewPGEventRepository(db)
			owner := uuid.New()
			want := errors.New("query failed")
			q := mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "events" WHERE user_id = $1 ORDER BY start_time ASC`)).WithArgs(owner)
			if fail {
				q.WillReturnError(want)
			} else {
				q.WillReturnRows(sqlmock.NewRows([]string{"id"}))
			}
			var events []domain.Event
			var err error
			if byRange {
				events, err = repo.GetByDateRange(context.Background(), owner, time.Time{}, time.Time{})
			} else {
				events, err = repo.GetByUserID(context.Background(), owner)
			}
			if fail {
				if !errors.Is(err, want) {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil || events == nil || len(events) != 0 {
				t.Fatalf("events=%#v err=%v", events, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
