package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const secret = "secret-test"

var seq int

// failFrom membuat operasi database ke-n dan seterusnya gagal (untuk menguji jalur error).
func failFrom(db *gorm.DB, op string, n int) {
	count := 0
	seq++
	name := fmt.Sprintf("test:fail:%d", seq)
	fn := func(tx *gorm.DB) {
		count++
		if count >= n {
			_ = tx.AddError(errors.New("forced error"))
		}
	}
	switch op {
	case "query":
		_ = db.Callback().Query().Before("gorm:query").Register(name, fn)
	case "create":
		_ = db.Callback().Create().Before("gorm:create").Register(name, fn)
	case "update":
		_ = db.Callback().Update().Before("gorm:update").Register(name, fn)
	}
}

func newDB(t *testing.T) *gorm.DB {
	db, err := database.OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func newService(t *testing.T) (*Service, *gorm.DB) {
	db := newDB(t)
	return NewService(NewRepository(db), secret), db
}

func TestRegisterAndLogin(t *testing.T) {
	svc, _ := newService(t)
	u, err := svc.Register(RegisterRequest{Username: "feny", Email: "feny@mail.com", Password: "secret1"})
	require.NoError(t, err)
	assert.Equal(t, "member", u.Role)
	assert.NotEqual(t, "secret1", u.Password)

	_, err = svc.Register(RegisterRequest{Username: "feny", Email: "other@mail.com", Password: "secret1"})
	assert.ErrorIs(t, err, ErrDuplicate)

	res, err := svc.Login(LoginRequest{Username: "feny", Password: "secret1"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "feny", res.User.Username)

	res, err = svc.Login(LoginRequest{Username: "feny@mail.com", Password: "secret1"})
	require.NoError(t, err)
	assert.Equal(t, "feny", res.User.Username)

	_, err = svc.Login(LoginRequest{Username: "feny", Password: "wrong"})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = svc.Login(LoginRequest{Username: "ghost", Password: "x"})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestRegisterErrors(t *testing.T) {
	svc, db := newService(t)
	_, err := svc.Register(RegisterRequest{Username: "long", Email: "l@mail.com", Password: strings.Repeat("x", 100)})
	assert.Error(t, err)

	failFrom(db, "query", 1)
	_, err = svc.Register(RegisterRequest{Username: "a1", Email: "a@mail.com", Password: "secret1"})
	assert.Error(t, err)
}

func TestLoginErrors(t *testing.T) {
	svc, db := newService(t)
	_, err := svc.Register(RegisterRequest{Username: "feny", Email: "feny@mail.com", Password: "secret1"})
	require.NoError(t, err)

	svc.key = "bukan-byte-slice" // memaksa SignedString gagal
	_, err = svc.Login(LoginRequest{Username: "feny", Password: "secret1"})
	assert.Error(t, err)

	failFrom(db, "query", 1)
	_, err = svc.Login(LoginRequest{Username: "feny", Password: "secret1"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
}

func TestProfileAndUpdate(t *testing.T) {
	svc, _ := newService(t)
	a, err := svc.Register(RegisterRequest{Username: "alice", Email: "alice@mail.com", Password: "secret1"})
	require.NoError(t, err)
	b, err := svc.Register(RegisterRequest{Username: "bob", Email: "bob@mail.com", Password: "secret1"})
	require.NoError(t, err)

	p, err := svc.Profile(a.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice", p.Username)
	_, err = svc.Profile(999)
	assert.ErrorIs(t, err, ErrNotFound)

	up, err := svc.UpdateProfile(a.ID, UpdateProfileRequest{Username: "alice2", Email: "alice2@mail.com", Password: "newpass1"})
	require.NoError(t, err)
	assert.Equal(t, "alice2", up.Username)
	_, err = svc.Login(LoginRequest{Username: "alice2", Password: "newpass1"})
	assert.NoError(t, err)

	up, err = svc.UpdateProfile(a.ID, UpdateProfileRequest{})
	require.NoError(t, err)
	assert.Equal(t, "alice2", up.Username)

	_, err = svc.UpdateProfile(b.ID, UpdateProfileRequest{Username: "alice2"})
	assert.ErrorIs(t, err, ErrDuplicate)
	_, err = svc.UpdateProfile(999, UpdateProfileRequest{})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateProfileErrors(t *testing.T) {
	svc, db := newService(t)
	a, err := svc.Register(RegisterRequest{Username: "alice", Email: "alice@mail.com", Password: "secret1"})
	require.NoError(t, err)

	_, err = svc.UpdateProfile(a.ID, UpdateProfileRequest{Password: strings.Repeat("x", 100)})
	assert.Error(t, err)

	failFrom(db, "query", 2) // FindByID sukses, Exists gagal
	_, err = svc.UpdateProfile(a.ID, UpdateProfileRequest{})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
}

// ---------------- handler ----------------

func newRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	svc, db := newService(t)
	h := NewHandler(svc)
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	g := r.Group("/", middleware.Auth(secret))
	g.POST("/logout", h.Logout)
	g.GET("/profile", h.GetProfile)
	g.PUT("/profile", h.UpdateProfile)
	return r, db
}

func send(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginToken(t *testing.T, r *gin.Engine) string {
	w := send(r, "POST", "/login", "", map[string]string{"username": "feny", "password": "secret1"})
	require.Equal(t, 200, w.Code)
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out.Data.Token
}

func TestHandlerFlow(t *testing.T) {
	r, _ := newRouter(t)

	w := send(r, "POST", "/register", "", map[string]string{"username": "feny", "email": "feny@mail.com", "password": "secret1"})
	assert.Equal(t, 201, w.Code)
	w = send(r, "POST", "/register", "", map[string]string{"username": "feny", "email": "feny@mail.com", "password": "secret1"})
	assert.Equal(t, 409, w.Code)
	w = send(r, "POST", "/register", "", `{"username":"x"}`)
	assert.Equal(t, 400, w.Code)

	w = send(r, "POST", "/login", "", `{}`)
	assert.Equal(t, 400, w.Code)
	w = send(r, "POST", "/login", "", map[string]string{"username": "feny", "password": "bad"})
	assert.Equal(t, 401, w.Code)

	token := loginToken(t, r)
	assert.Equal(t, 200, send(r, "POST", "/logout", token, nil).Code)
	assert.Equal(t, 200, send(r, "GET", "/profile", token, nil).Code)
	assert.Equal(t, 401, send(r, "GET", "/profile", "", nil).Code)

	w = send(r, "PUT", "/profile", token, map[string]string{"username": "feny2"})
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 400, send(r, "PUT", "/profile", token, `{"email":"bukan-email"}`).Code)
}

func TestHandlerErrorMapping(t *testing.T) {
	r, db := newRouter(t)
	send(r, "POST", "/register", "", map[string]string{"username": "feny", "email": "feny@mail.com", "password": "secret1"})
	token := loginToken(t, r)

	// user dihapus -> profil tidak ditemukan (404)
	require.NoError(t, db.Exec("DELETE FROM users").Error)
	assert.Equal(t, 404, send(r, "GET", "/profile", token, nil).Code)
	assert.Equal(t, 404, send(r, "PUT", "/profile", token, map[string]string{"username": "abc"}).Code)

	// database error -> 500
	failFrom(db, "query", 1)
	assert.Equal(t, 500, send(r, "GET", "/profile", token, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, send(r, "POST", "/register", "", map[string]string{"username": "zzz", "email": "z@mail.com", "password": "secret1"}).Code)
}
