package repositories

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockGorm(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}
	return db, mock
}

func TestPGUserRepositoryCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockGorm(t)
		repo := NewPGUserRepository(db)
		user := &domain.User{ID: uuid.New(), Email: "user@example.com", Password: "hash", Name: "Jane", Provider: "email"}

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at", "id"}).AddRow(time.Now(), time.Now(), user.ID))
		mock.ExpectCommit()

		if err := repo.Create(context.Background(), user); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		db, mock := newMockGorm(t)
		repo := NewPGUserRepository(db)
		want := errors.New("insert failed")
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).WillReturnError(want)
		mock.ExpectRollback()
		if err := repo.Create(context.Background(), &domain.User{ID: uuid.New()}); !errors.Is(err, want) {
			t.Fatalf("Create() error = %v, want %v", err, want)
		}
	})
}

func TestPGUserRepositoryFindByEmail(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, mock := newMockGorm(t)
		repo := NewPGUserRepository(db)
		userID := uuid.New()
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "provider", "google_id", "created_at", "updated_at", "deleted_at"}).
			AddRow(userID, "user@example.com", "hash", "Jane", "email", nil, now, now, nil)
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*ORDER BY.*LIMIT \$2`).
			WithArgs("user@example.com", 1).
			WillReturnRows(rows)

		got, err := repo.FindByEmail(context.Background(), "user@example.com")
		if err != nil || got.ID != userID || got.Name != "Jane" {
			t.Fatalf("FindByEmail() = %#v, %v", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet SQL expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGorm(t)
		repo := NewPGUserRepository(db)
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*ORDER BY.*LIMIT \$2`).
			WithArgs("missing@example.com", 1).
			WillReturnError(gorm.ErrRecordNotFound)
		got, err := repo.FindByEmail(context.Background(), "missing@example.com")
		if got != nil || !errors.Is(err, ports.ErrUserNotFound) {
			t.Fatalf("FindByEmail() = %#v, %v", got, err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		db, mock := newMockGorm(t)
		repo := NewPGUserRepository(db)
		want := errors.New("query failed")
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = \$1.*ORDER BY.*LIMIT \$2`).
			WithArgs("user@example.com", 1).
			WillReturnError(want)
		got, err := repo.FindByEmail(context.Background(), "user@example.com")
		if got != nil || !errors.Is(err, want) {
			t.Fatalf("FindByEmail() = %#v, %v", got, err)
		}
	})
}
