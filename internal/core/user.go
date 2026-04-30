package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type userCtxKey struct{}

// ContextWithUser returns a context with the user set.
func ContextWithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

// UserFromContext returns the user from context, if set.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userCtxKey{}).(User)
	return u, ok
}

// userIDFromContext returns the user ID from context, defaulting to 1 (local).
func userIDFromContext(ctx context.Context) int64 {
	if u, ok := UserFromContext(ctx); ok {
		return u.ID
	}
	return 1
}

// User represents a registered user.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// UserService manages users in the registry database.
type UserService struct {
	db *sql.DB
}

// NewUserService creates a new UserService.
func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// Create creates a new user with a hashed password.
func (s *UserService) Create(ctx context.Context, username, password string) (User, error) {
	if username == "" {
		return User{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if password == "" {
		return User{}, fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hashing password: %w", err)
	}

	result, err := s.db.ExecContext(ctx, "INSERT INTO users (username, password_hash) VALUES (?, ?)", username, string(hash))
	if err != nil {
		return User{}, fmt.Errorf("creating user: %w", err)
	}

	id, _ := result.LastInsertId()
	return User{ID: id, Username: username, CreatedAt: time.Now()}, nil
}

// List returns all users (excluding the default 'local' user).
func (s *UserService) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, username, created_at FROM users WHERE username != 'local' ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Authenticate checks username/password and returns the user if valid.
func (s *UserService) Authenticate(ctx context.Context, username, password string) (User, bool) {
	var u User
	var hash string
	err := s.db.QueryRowContext(ctx, "SELECT id, username, password_hash, created_at FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &hash, &u.CreatedAt)
	if err != nil {
		return User{}, false
	}
	if hash == "" {
		return User{}, false
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, false
	}
	return u, true
}

// HasUsers returns true if any users with passwords exist (auth is enabled).
func (s *UserService) HasUsers(ctx context.Context) bool {
	var count int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE password_hash != ''").Scan(&count)
	return count > 0
}

// GetByID returns a user by ID.
func (s *UserService) GetByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, "SELECT id, username, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("user %d: %w", id, ErrNotFound)
		}
		return User{}, err
	}
	return u, nil
}
