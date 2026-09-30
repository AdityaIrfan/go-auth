package repositories

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"kda-auth-service/internal/core/ports"
	"testing"
)

func TestPGUserFindByID(t *testing.T) {
	failure := errors.New("db")
	for _, tc := range []struct {
		name           string
		dbErr, wantErr error
	}{{"found", nil, nil}, {"not found", gorm.ErrRecordNotFound, ports.ErrUserNotFound}, {"database error", failure, failure}} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGorm(t)
			repo := NewPGUserRepository(db)
			id := uuid.New()
			q := mock.ExpectQuery(`SELECT .* FROM "users" WHERE id = \$1.*"users"\."deleted_at" IS NULL.*LIMIT \$2`).WithArgs(id, 1)
			if tc.dbErr != nil {
				q.WillReturnError(tc.dbErr)
			} else {
				q.WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(id, "user@example.com"))
			}
			user, err := repo.FindByID(context.Background(), id)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v", err)
			}
			if tc.wantErr == nil {
				if user == nil || user.ID != id || user.Email != "user@example.com" {
					t.Fatalf("user=%#v", user)
				}
			} else if user != nil {
				t.Fatal("unexpected user")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
