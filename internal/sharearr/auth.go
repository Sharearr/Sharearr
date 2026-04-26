package sharearr

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/hkdf"
)

const (
	jwtCookieKey     = "jwt"
	refreshCookieKey = "refresh"

	jwtInfo            = "sharearr jwt"
	jwtSessionInfo     = "sharearr session cookie"
	refreshSessionInfo = "sharearr refresh cookie"
)

type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type LoginUser struct {
	ID       int64    `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Features []string `json:"features"`
}

type LoginResponse struct {
	User      LoginUser `json:"user"`
	ExpiresAt string    `json:"expires_at"`
}

type AuthHandler struct {
	jwtKey      []byte
	sessionKey  []byte
	refreshKey  []byte
	userService *UserService
}

func derive(info string, secretKeyBase []byte, length int) ([]byte, error) {
	r := hkdf.New(sha256.New, secretKeyBase, nil, []byte(info))
	key := make([]byte, length)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("derive %s key: %w", info, err)
	}
	return key, nil
}

func NewAuthHandler(secretKeyBase []byte, userService *UserService) (*AuthHandler, error) {
	jwtKey, err := derive(jwtInfo, secretKeyBase, 48)
	if err != nil {
		return nil, err
	}
	sessionKey, err := derive(jwtSessionInfo, secretKeyBase, 48)
	if err != nil {
		return nil, err
	}
	refreshKey, err := derive(refreshSessionInfo, secretKeyBase, 48)
	if err != nil {
		return nil, err
	}

	return &AuthHandler{
		jwtKey:      jwtKey,
		sessionKey:  sessionKey,
		refreshKey:  refreshKey,
		userService: userService,
	}, nil
}

func (a *AuthHandler) Login(c *gin.Context) {
	var lr LoginRequest
	if err := c.ShouldBind(&lr); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	_, err := a.userService.GetByUsername(c.Request.Context(), lr.Username)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	tokenString, err := token.SignedString(a.jwtKey)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	jwtSession := sessions.DefaultMany(c, jwtCookieKey)
	jwtSession.Set("jwt", tokenString)
	jwtSession.Save()
}

func (a *AuthHandler) Session() gin.HandlerFunc {
	jwtStore := cookie.NewStore(a.sessionKey)
	options := sessions.Options{HttpOnly: true, Secure: false, SameSite: http.SameSiteLaxMode, Path: "/"}
	jwtStore.Options(options)
	jwtSessionStore := sessions.SessionStore{Name: jwtCookieKey, Store: jwtStore}

	refreshStore := cookie.NewStore(a.refreshKey)
	options = sessions.Options{HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, Path: "/auth/refresh"}
	refreshStore.Options(options)
	refreshSessionStore := sessions.SessionStore{Name: refreshCookieKey, Store: refreshStore}

	return sessions.SessionsManyStores([]sessions.SessionStore{jwtSessionStore, refreshSessionStore})
}
