package sharearr

import (
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	// "github.com/jmoiron/sqlx"
)

const (
	jwtSessionKey = "jwt"
	refreshSessionKey = "refresh"
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
	secretKeyBase []byte
	userService   UserService
}

func NewAuthHandler(secretKeyBase []byte, userService *UserService) AuthHandler {
	return AuthHandler{secretKeyBase: secretKeyBase, userService: *userService}
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
	tokenString, err := token.SignedString(a.secretKeyBase)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	jwtSession := sessions.DefaultMany(c, jwtSessionKey)
	jwtSession.Set("jwt", tokenString)
	jwtSession.Save()
}

func (a *AuthHandler) Session() gin.HandlerFunc {
	jwtStore := cookie.NewStore(a.secretKeyBase)
	options := sessions.Options{HttpOnly: true, Secure: false, SameSite: http.SameSiteLaxMode, Path: "/"}
	jwtStore.Options(options)
	jwtSessionStore := sessions.SessionStore{Name: jwtSessionKey, Store: jwtStore}

	refreshStore := cookie.NewStore(a.secretKeyBase)
	options = sessions.Options{HttpOnly: true, Secure: false, SameSite: http.SameSiteStrictMode, Path: "/auth/refresh"}
	refreshStore.Options(options)
	refreshSessionStore := sessions.SessionStore{Name: refreshSessionKey, Store: refreshStore}

	return sessions.SessionsManyStores([]sessions.SessionStore{jwtSessionStore, refreshSessionStore})
}
