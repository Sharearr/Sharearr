package sharearr

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ctxUserKey struct{}

var ctxKeyUser ctxUserKey

func userFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(*User)
	return u, ok
}

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID        int64
	Username  string
	Email     string
	APIKey    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) InsertIfEmpty(ctx context.Context, u *User) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, email, api_key)
		 SELECT ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM users)`,
		u.Username, u.Email, u.APIKey,
	)
	if err != nil {
		return false, fmt.Errorf("insert user: %w", err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*User, error) {
	u := &User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, email, api_key, created_at, updated_at
		 FROM users WHERE api_key = ?`, apiKey,
	).Scan(&u.ID, &u.Username, &u.Email, &u.APIKey, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by api key: %w", err)
	}
	return u, nil
}

type UserService struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

func NewUserServiceFromDB(db *sql.DB) *UserService {
	return NewUserService(NewUserRepository(db))
}

func (s *UserService) Provision(ctx context.Context) error {
	email := os.Getenv("SHAREARR_EMAIL")
	if email == "" {
		return nil
	}

	username := os.Getenv("SHAREARR_USERNAME")
	if username == "" {
		username, _, _ = strings.Cut(email, "@")
	}

	apiKey := os.Getenv("SHAREARR_API_KEY")
	if apiKey == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate api key: %w", err)
		}
		apiKey = fmt.Sprintf("%x", b)
	}

	u := &User{Username: username, Email: email, APIKey: apiKey}
	inserted, err := s.repo.InsertIfEmpty(ctx, u)
	if err != nil {
		return err
	}
	if inserted {
		log.Printf("user %q api_key=%s", u.Username, u.APIKey)
	}
	return nil
}

func (s *UserService) GetByAPIKey(ctx context.Context, apiKey string) (*User, error) {
	u, err := s.repo.GetByAPIKey(ctx, apiKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func Auth(db *sql.DB) gin.HandlerFunc {
	service := NewUserServiceFromDB(db)
	return func(c *gin.Context) {
		apiKey := c.Param("apikey")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}
		if apiKey == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		u, err := service.GetByAPIKey(c.Request.Context(), apiKey)
		if errors.Is(err, ErrUserNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(c.Request.Context(), ctxKeyUser, u)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
