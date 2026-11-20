package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/debuglog"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload JWT aplikasi.
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// CORS memberi akses lintas-domain bagi frontend React.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth memverifikasi Bearer Token JWT lalu menyuntikkan data user ke Gin context.
func Auth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.AbortFail(c, http.StatusUnauthorized, "Missing or invalid authorization header")
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", strings.TrimSpace(claims.Role))
		c.Next()
	}
}

// UserID mengambil ID user dari context (diisi oleh Auth).
func UserID(c *gin.Context) uint {
	v, _ := c.Get("user_id")
	id, _ := v.(uint)
	return id
}

// RequireRole memastikan pengguna memiliki salah satu role yang diizinkan.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := strings.ToLower(strings.TrimSpace(c.GetString("role")))
		for _, r := range roles {
			if strings.ToLower(strings.TrimSpace(r)) == role {
				c.Next()
				return
			}
		}
		debuglog.Addf("DENIED %s %s user=%q role=%q need=%v", c.Request.Method, c.Request.URL.Path, c.GetString("username"), role, roles)
		response.AbortFail(c, http.StatusForbidden, "Forbidden: insufficient role")
	}
}

type bucket struct {
	count   int
	resetAt time.Time
}

var now = time.Now

// RateLimiter membatasi jumlah request per IP dalam satu jendela waktu.
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		b, ok := buckets[ip]
		if !ok || now().After(b.resetAt) {
			b = &bucket{resetAt: now().Add(window)}
			buckets[ip] = b
		}
		b.count++
		exceeded := b.count > limit
		mu.Unlock()
		if exceeded {
			response.AbortFail(c, http.StatusTooManyRequests, "Too many requests")
			return
		}
		c.Next()
	}
}
