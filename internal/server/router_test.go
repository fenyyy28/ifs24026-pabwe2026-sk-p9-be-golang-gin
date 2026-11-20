package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T, openai, search string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{AppKey: "router-secret", AdminUsername: "admin", AdminPassword: "admin",
		OpenAIEndpoint: openai, OpenAIKey: "k", OpenAIModel: "m", SearchEngineEndpoint: search}
	db, err := database.OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, database.Seed(db, cfg))
	return NewRouter(cfg, db)
}

func call(r *gin.Engine, method, path, token string, body interface{}, headers ...string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func login(t *testing.T, r *gin.Engine, user, pass string) string {
	w := call(r, "POST", "/api/auth/login", "", map[string]string{"username": user, "password": pass})
	require.Equal(t, 200, w.Code)
	var out struct {
		Data struct{ Token string } `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out.Data.Token
}

func TestNoRouteAndDocs(t *testing.T) {
	r := setup(t, "", "")
	w := call(r, "GET", "/tidak-ada", "", nil)
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "Endpoint not found")

	w = call(r, "GET", "/api.json", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"http"`)
	w = call(r, "GET", "/api.json", "", nil, "X-Forwarded-Proto", "https")
	assert.Contains(t, w.Body.String(), `"https"`)

	w = call(r, "GET", "/swagger/index.html", "", nil)
	assert.Equal(t, 200, w.Code)
}

func TestFullBlogFlow(t *testing.T) {
	ai := http.NewServeMux()
	ai.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"AI Judul\",\"category\":\"Teknologi\",\"excerpt\":\"e\",\"content\":\"c\"}"}}]}`))
	})
	ai.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"title":"Sumber","url":"http://s","snippet":"x"}]}`))
	})
	srv := httptest.NewServer(ai)
	defer srv.Close()

	r := setup(t, srv.URL+"/v1", srv.URL+"/search")

	// publik
	assert.Equal(t, 200, call(r, "GET", "/api/public/categories", "", nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/public/posts", "", nil).Code)
	assert.Equal(t, 404, call(r, "GET", "/api/public/posts/belum-ada", "", nil).Code)

	// registrasi + login penulis
	w := call(r, "POST", "/api/auth/register", "", map[string]string{"username": "penulis", "email": "p@mail.com", "password": "rahasia1"})
	require.Equal(t, 201, w.Code)
	token := login(t, r, "penulis", "rahasia1")

	assert.Equal(t, 401, call(r, "GET", "/api/posts", "", nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/auth/profile", token, nil).Code)
	assert.Equal(t, 200, call(r, "POST", "/api/auth/logout", token, nil).Code)

	w = call(r, "POST", "/api/posts", token, map[string]interface{}{"title": "Halo Dunia", "content": "isi artikel", "is_published": true})
	require.Equal(t, 201, w.Code)
	assert.Equal(t, 200, call(r, "GET", "/api/posts", token, nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/posts/stats", token, nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/posts/1", token, nil).Code)
	assert.Equal(t, 200, call(r, "PATCH", "/api/posts/1/toggle-publish", token, nil).Code)
	assert.Equal(t, 200, call(r, "PATCH", "/api/posts/1/toggle-publish", token, nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/public/posts/halo-dunia", "", nil).Code)

	w = call(r, "POST", "/api/posts/ai-generate", token, map[string]interface{}{"topic": "Go", "include_web_search": true})
	assert.Equal(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "AI Judul"))

	// admin
	assert.Equal(t, 403, call(r, "GET", "/api/admin/users", token, nil).Code)
	adminToken := login(t, r, "admin", "admin")
	assert.Equal(t, 200, call(r, "GET", "/api/admin/users", adminToken, nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/admin/posts", adminToken, nil).Code)
	assert.Equal(t, 200, call(r, "GET", "/api/admin/users/3/posts", adminToken, nil).Code)

	assert.Equal(t, 200, call(r, "PUT", "/api/posts/1", token, map[string]interface{}{"title": "Halo Dunia", "content": "baru"}).Code)
	assert.Equal(t, 200, call(r, "PUT", "/api/auth/profile", token, map[string]string{"username": "penulis2"}).Code)
	assert.Equal(t, 200, call(r, "DELETE", "/api/posts/1", token, nil).Code)
}
