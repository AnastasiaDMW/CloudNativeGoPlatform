package middleware

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"go-postgres/models"

	"github.com/gorilla/mux"

	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func setupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	oldDB := db
	db = mockDB

	t.Cleanup(func() {
		db = oldDB
		_ = mockDB.Close()
	})

	return mock
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestReadyzDatabaseUnavailable(t *testing.T) {
	oldDB := db
	db = nil
	t.Cleanup(func() {
		db = oldDB
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}

func TestReadyzDatabaseReady(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	oldDB := db
	db = mockDB
	t.Cleanup(func() {
		db = oldDB
		_ = mockDB.Close()
	})

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}

func TestReadyzDatabasePingFailure(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	oldDB := db
	db = mockDB
	t.Cleanup(func() {
		db = oldDB
		_ = mockDB.Close()
	})

	mock.ExpectPing().WillReturnError(errors.New("database unavailable"))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet SQL expectations: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("Anna", "Berlin", int64(25)).
			WillReturnRows(sqlmock.NewRows([]string{"userid"}).AddRow(1))

		body := strings.NewReader(`{"name":"Anna","city":"Berlin","age":25}`)
		req := httptest.NewRequest(http.MethodPost, "/api/newuser", body)
		rec := httptest.NewRecorder()

		CreateUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var result response
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}
		if result.ID != 1 || result.Message != "User created successfully" {
			t.Errorf("unexpected response: %+v", result)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/newuser", strings.NewReader(`{invalid}`))
		rec := httptest.NewRecorder()

		CreateUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("Anna", "Berlin", int64(25)).
			WillReturnError(errors.New("insert failed"))

		body := strings.NewReader(`{"name":"Anna","city":"Berlin","age":25}`)
		req := httptest.NewRequest(http.MethodPost, "/api/newuser", body)
		rec := httptest.NewRecorder()

		CreateUser(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`SELECT userid, name, city, age`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"userid", "name", "city", "age"}).
				AddRow(1, "Anna", "Berlin", 25))

		req := httptest.NewRequest(http.MethodGet, "/api/user/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		GetUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var user models.User
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}
		if user.ID != 1 || user.Name != "Anna" || user.City != "Berlin" || user.Age != 25 {
			t.Errorf("unexpected user: %+v", user)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/abc", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		rec := httptest.NewRecorder()

		GetUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`SELECT userid, name, city, age`).
			WithArgs(int64(99)).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodGet, "/api/user/99", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99"})
		rec := httptest.NewRecorder()

		GetUser(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`SELECT userid, name, city, age`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("select failed"))

		req := httptest.NewRequest(http.MethodGet, "/api/user/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		GetUser(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestGetAllUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`SELECT userid, name, city, age`).
			WillReturnRows(sqlmock.NewRows([]string{"userid", "name", "city", "age"}).
				AddRow(1, "Anna", "Berlin", 25).
				AddRow(2, "Maria", "Paris", 30))

		req := httptest.NewRequest(http.MethodGet, "/api/user", nil)
		rec := httptest.NewRecorder()

		GetAllUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var users []models.User
		if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d", len(users))
		}
		if users[0].Name != "Anna" || users[1].Name != "Maria" {
			t.Errorf("unexpected users: %+v", users)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectQuery(`SELECT userid, name, city, age`).
			WillReturnError(errors.New("select failed"))

		req := httptest.NewRequest(http.MethodGet, "/api/user", nil)
		rec := httptest.NewRecorder()

		GetAllUser(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs(int64(1), "Anna", "Berlin", int64(25)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		body := strings.NewReader(`{"name":"Anna","city":"Berlin","age":25}`)
		req := httptest.NewRequest(http.MethodPut, "/api/user/1", body)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		UpdateUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/user/abc", strings.NewReader(`{}`))
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		rec := httptest.NewRecorder()

		UpdateUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/user/1", strings.NewReader(`{invalid}`))
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		UpdateUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs(int64(99), "Anna", "Berlin", int64(25)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		body := strings.NewReader(`{"name":"Anna","city":"Berlin","age":25}`)
		req := httptest.NewRequest(http.MethodPut, "/api/user/99", body)
		req = mux.SetURLVars(req, map[string]string{"id": "99"})
		rec := httptest.NewRecorder()

		UpdateUser(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs(int64(1), "Anna", "Berlin", int64(25)).
			WillReturnError(errors.New("update failed"))

		body := strings.NewReader(`{"name":"Anna","city":"Berlin","age":25}`)
		req := httptest.NewRequest(http.MethodPut, "/api/user/1", body)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		UpdateUser(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := httptest.NewRequest(http.MethodDelete, "/api/user/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		DeleteUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/abc", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		rec := httptest.NewRecorder()

		DeleteUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(99)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		req := httptest.NewRequest(http.MethodDelete, "/api/user/99", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99"})
		rec := httptest.NewRecorder()

		DeleteUser(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		mock := setupMockDB(t)
		mock.ExpectExec(`DELETE FROM users`).
			WithArgs(int64(1)).
			WillReturnError(errors.New("delete failed"))

		req := httptest.NewRequest(http.MethodDelete, "/api/user/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})
		rec := httptest.NewRecorder()

		DeleteUser(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}
