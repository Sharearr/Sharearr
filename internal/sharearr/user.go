package sharearr

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type ctxUserKey struct{}

var ctxKeyUser ctxUserKey

func userFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(*User)
	return u, ok
}

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	APIKey    string    `db:"api_key"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
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
	err := r.db.GetContext(ctx, u,
		`SELECT id, username, email, api_key, created_at, updated_at
		 FROM users WHERE api_key = ?`, apiKey,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by api key: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := r.db.GetContext(ctx, u,
		`SELECT id, username, email, api_key, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

type UserService struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

func NewUserServiceFromDB(db *sqlx.DB) *UserService {
	return NewUserService(NewUserRepository(db))
}

func (s *UserService) Init(ctx context.Context, cfg UserConfig) error {
	if cfg.Email == "" {
		return nil
	}

	if cfg.Username == "" {
		cfg.Username, _, _ = strings.Cut(cfg.Email, "@")
	}

	if cfg.APIKey == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate api key: %w", err)
		}
		cfg.APIKey = fmt.Sprintf("%x", b)
	}

	u := &User{Username: cfg.Username, Email: cfg.Email, APIKey: cfg.APIKey}
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

func (s *UserService) GetByUsername(ctx context.Context, username string) (*User, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func Auth(db *sqlx.DB) gin.HandlerFunc {
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
