package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secret = "test-secret"

func init() { gin.SetMode(gin.TestMode) }

func makeToken(t *testing.T, key string, method jwt.SigningMethod, exp time.Duration) string {
	claims := Claims{UserID: 7, Username: "u", Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp))}}
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(key))
	assert.NoError(t, err)
	return s
}

func do(r *gin.Engine, method, path, authHeader string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := do(r, http.MethodGet, "/x", "")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = do(r, http.MethodOptions, "/x", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuth(t *testing.T) {
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/me", func(c *gin.Context) {
		c.JSON(200, gin.H{"id": UserID(c), "role": c.GetString("role"), "username": c.GetString("username")})
	})

	assert.Equal(t, 401, do(r, "GET", "/me", "").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Basic abc").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer garbage").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+makeToken(t, "other", jwt.SigningMethodHS256, time.Hour)).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+makeToken(t, secret, jwt.SigningMethodHS256, -time.Hour)).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+makeToken(t, secret, jwt.SigningMethodHS512, time.Hour)).Code)

	w := do(r, "GET", "/me", "bearer "+makeToken(t, secret, jwt.SigningMethodHS256, time.Hour))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"id":7`)
}

func TestUserIDWithoutAuth(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, uint(0), UserID(c))
}

func TestRequireRole(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", c.GetHeader("X-Role")); c.Next() })
	r.GET("/admin", RequireRole("admin", "superadmin"), func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("X-Role", "admin")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("X-Role", "member")
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestRateLimiter(t *testing.T) {
	orig := now
	defer func() { now = orig }()
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now = func() time.Time { return current }

	r := gin.New()
	r.Use(RateLimiter(2, time.Minute))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
	assert.Equal(t, 429, do(r, "GET", "/x", "").Code)

	current = current.Add(2 * time.Minute) // jendela baru
	assert.Equal(t, 200, do(r, "GET", "/x", "").Code)
}
