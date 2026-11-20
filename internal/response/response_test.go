package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func run(t *testing.T, fn func(c *gin.Context)) (int, Body) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	fn(c)
	var body Body
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body
}

func TestSuccess(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { Success(c, 201, "ok", gin.H{"a": 1}) })
	assert.Equal(t, 201, code)
	assert.Equal(t, "success", body.Status)
	assert.Equal(t, "ok", body.Message)
	assert.NotNil(t, body.Data)
}

func TestFailAndError(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { Fail(c, 400, "bad") })
	assert.Equal(t, 400, code)
	assert.Equal(t, "fail", body.Status)
	assert.Nil(t, body.Data)

	code, body = run(t, func(c *gin.Context) { Error(c, 500, "boom") })
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", body.Status)
}

func TestAbort(t *testing.T) {
	code, body := run(t, func(c *gin.Context) { AbortFail(c, 401, "no") })
	assert.Equal(t, 401, code)
	assert.Equal(t, "fail", body.Status)

	code, body = run(t, func(c *gin.Context) { AbortError(c, 503, "down") })
	assert.Equal(t, 503, code)
	assert.Equal(t, "error", body.Status)
}
