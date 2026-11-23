package middleware

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-postgres/models"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type response struct {
	ID      int64  `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

var db *sql.DB

func InitDatabase() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Info(".env file not found, using environment variables")
	}

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		log.Fatal("POSTGRES_URL is not set")
	}

	var err error

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Unable to open PostgreSQL connection: %v", err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		log.Fatalf("Unable to ping PostgreSQL: %v", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	slog.Info("Successfully connected to PostgreSQL")

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		userid BIGSERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		city VARCHAR(255),
		age BIGINT
	)
	`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		_ = db.Close()
		log.Fatalf("Unable to create users table: %v", err)
	}

	slog.Info("Users table is ready")
}

func CloseDatabase() error {
	if db == nil {
		return nil
	}

	return db.Close()
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		slog.Warn("Invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	insertID, err := insertUser(user)
	if err != nil {
		slog.Error("Unable to insert user", "error", err)
		http.Error(w, "Unable to create user", http.StatusInternalServerError)
		return
	}

	res := response{
		ID:      insertID,
		Message: "User created successfully",
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("Unable to encode response", "error", err)
		return
	}

	slog.Info("User created successfully", "user_id", insertID)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err != nil {
		slog.Warn("Invalid user ID", "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := getUser(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("User not found", "user_id", id)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		slog.Error("Unable to get user", "user_id", id, "error", err)
		http.Error(w, "Unable to get user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error("Unable to encode response", "user_id", id, "error", err)
		return
	}

	slog.Info("User retrieved successfully", "user_id", id)
}

func GetAllUser(w http.ResponseWriter, r *http.Request) {
	users, err := getAllUsers()
	if err != nil {
		slog.Error("Unable to get all users", "error", err)
		http.Error(w, "Unable to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(users); err != nil {
		slog.Error("Unable to encode response", "error", err)
		return
	}

	slog.Info("Users retrieved successfully", "count", len(users))
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err != nil {
		slog.Warn("Invalid user ID", "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User

	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		slog.Warn("Invalid request body", "user_id", id, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedRows, err := updateUser(id, user)
	if err != nil {
		slog.Error("Unable to update user", "user_id", id, "error", err)
		http.Error(w, "Unable to update user", http.StatusInternalServerError)
		return
	}

	if updatedRows == 0 {
		slog.Warn("User not found", "user_id", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	msg := fmt.Sprintf(
		"User updated successfully. Total rows/record affected %v",
		updatedRows,
	)

	res := response{
		ID:      id,
		Message: msg,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("Unable to encode response", "user_id", id, "error", err)
		return
	}

	slog.Info("User updated successfully", "user_id", id, "rows_affected", updatedRows)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err != nil {
		slog.Warn("Invalid user ID", "error", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	deletedRows, err := deleteUser(id)
	if err != nil {
		slog.Error("Unable to delete user", "user_id", id, "error", err)
		http.Error(w, "Unable to delete user", http.StatusInternalServerError)
		return
	}

	if deletedRows == 0 {
		slog.Warn("User not found", "user_id", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	msg := fmt.Sprintf(
		"User deleted successfully. Total rows/record affected %v",
		deletedRows,
	)

	res := response{
		ID:      id,
		Message: msg,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("Unable to encode response", "user_id", id, "error", err)
		return
	}

	slog.Info("User deleted successfully", "user_id", id, "rows_affected", deletedRows)
}

func insertUser(user models.User) (int64, error) {
	if db == nil {
		return 0, errors.New("database is not initialized")
	}

	sqlStatement := `
		INSERT INTO users (name, city, age)
		VALUES ($1, $2, $3)
		RETURNING userid
	`

	var id int64

	err := db.QueryRow(
		sqlStatement,
		user.Name,
		user.City,
		user.Age,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("unable to execute insert query: %w", err)
	}

	slog.Info("User inserted", "user_id", id)

	return id, nil
}

func getUser(id int64) (models.User, error) {
	if db == nil {
		return models.User{}, errors.New("database is not initialized")
	}

	var user models.User

	sqlStatement := `
		SELECT userid, name, city, age
		FROM users
		WHERE userid = $1
	`

	err := db.QueryRow(sqlStatement, id).Scan(
		&user.ID,
		&user.Name,
		&user.City,
		&user.Age,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, sql.ErrNoRows
		}

		return models.User{}, fmt.Errorf("unable to scan user: %w", err)
	}

	return user, nil
}

func getAllUsers() ([]models.User, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}

	sqlStatement := `
		SELECT userid, name, city, age
		FROM users
	`

	rows, err := db.Query(sqlStatement)
	if err != nil {
		return nil, fmt.Errorf("unable to execute select query: %w", err)
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var user models.User

		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.City,
			&user.Age,
		)

		if err != nil {
			return nil, fmt.Errorf("unable to scan user: %w", err)
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating users: %w", err)
	}

	return users, nil
}

func updateUser(id int64, user models.User) (int64, error) {
	if db == nil {
		return 0, errors.New("database is not initialized")
	}

	sqlStatement := `
		UPDATE users
		SET name = $2, city = $3, age = $4
		WHERE userid = $1
	`

	res, err := db.Exec(
		sqlStatement,
		id,
		user.Name,
		user.City,
		user.Age,
	)

	if err != nil {
		return 0, fmt.Errorf("unable to execute update query: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error while checking affected rows: %w", err)
	}

	slog.Info("Rows updated", "user_id", id, "rows_affected", rowsAffected)

	return rowsAffected, nil
}

func deleteUser(id int64) (int64, error) {
	if db == nil {
		return 0, errors.New("database is not initialized")
	}

	sqlStatement := `
		DELETE FROM users
		WHERE userid = $1
	`

	res, err := db.Exec(sqlStatement, id)
	if err != nil {
		return 0, fmt.Errorf("unable to execute delete query: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error while checking affected rows: %w", err)
	}

	slog.Info("Rows deleted", "user_id", id, "rows_affected", rowsAffected)

	return rowsAffected, nil
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func Readyz(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := db.Ping(); err != nil {
		slog.Error("Database readiness check failed", "error", err)
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
