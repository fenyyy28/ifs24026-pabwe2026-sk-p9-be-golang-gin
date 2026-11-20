package ai

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fakes ----------

type fakeSearch struct {
	res []Source
	err error
}

func (f fakeSearch) Search(ctx context.Context, q string) ([]Source, error) { return f.res, f.err }

type fakeChat struct {
	out string
	err error
	got string
}

func (f *fakeChat) Chat(ctx context.Context, system, user string) (string, error) {
	f.got = user
	return f.out, f.err
}

const goodJSON = "```json\n" + `{"title":"Judul AI","category":"Teknologi","excerpt":"Ringkas","content":"# Isi"}` + "\n```"

// ---------- service ----------

func TestGenerateWithSearch(t *testing.T) {
	chat := &fakeChat{out: goodJSON}
	svc := NewService(fakeSearch{res: []Source{{Title: "T", URL: "http://u", Snippet: "S"}}}, chat)
	res, err := svc.Generate(context.Background(), GenerateRequest{Topic: "Go", Category: "Teknologi", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.Equal(t, "Judul AI", res.Title)
	assert.Equal(t, "Teknologi", res.Category)
	assert.Len(t, res.Sources, 1)
	assert.Contains(t, chat.got, "[1] T (http://u): S")
}

func TestGenerateSearchFailureOrDisabled(t *testing.T) {
	chat := &fakeChat{out: `{"title":"A","category":"Aneh","excerpt":"e","content":"c"}`}
	svc := NewService(fakeSearch{err: errors.New("down")}, chat)
	res, err := svc.Generate(context.Background(), GenerateRequest{Topic: "Go", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.Equal(t, "Umum", res.Category)
	assert.NotNil(t, res.Sources)
	assert.Len(t, res.Sources, 0)
	assert.NotContains(t, chat.got, "hasil penelusuran")

	res, err = svc.Generate(context.Background(), GenerateRequest{Topic: "Go"})
	require.NoError(t, err)
	assert.Len(t, res.Sources, 0)
}

func TestGenerateErrors(t *testing.T) {
	svc := NewService(fakeSearch{}, &fakeChat{err: errors.New("boom")})
	_, err := svc.Generate(context.Background(), GenerateRequest{Topic: "x"})
	assert.Error(t, err)

	svc = NewService(fakeSearch{}, &fakeChat{out: "bukan json"})
	_, err = svc.Generate(context.Background(), GenerateRequest{Topic: "x"})
	assert.ErrorContains(t, err, "invalid AI response")
}

func TestHelpers(t *testing.T) {
	assert.Equal(t, `{"a":1}`, cleanJSON("```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, cleanJSON("  {\"a\":1} "))
	assert.Equal(t, "Bisnis", normalizeCategory("Bisnis"))
	assert.Equal(t, "Umum", normalizeCategory("x"))
}

// ---------- search client ----------

func TestSearchClient(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		switch r.URL.Path {
		case "/top":
			w.Write([]byte(`{"results":[{"title":"A","url":"u","snippet":"s"}]}`))
		case "/nested":
			w.Write([]byte(`{"data":{"results":[{"title":"1"},{"title":"2"},{"title":"3"},{"title":"4"},{"title":"5"},{"title":"6"}]}}`))
		case "/bad":
			w.Write([]byte(`not json`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	res, err := NewSearchClient(srv.URL+"/top").Search(ctx, "kata kunci")
	require.NoError(t, err)
	assert.Equal(t, "kata kunci", gotQuery)
	assert.Equal(t, "A", res[0].Title)

	res, err = NewSearchClient(srv.URL+"/nested").Search(ctx, "x")
	require.NoError(t, err)
	assert.Len(t, res, 5)

	_, err = NewSearchClient(srv.URL+"/bad").Search(ctx, "x")
	assert.Error(t, err)
	_, err = NewSearchClient(srv.URL+"/error").Search(ctx, "x")
	assert.ErrorContains(t, err, "status 500")
	_, err = NewSearchClient("://bad-url").Search(ctx, "x")
	assert.Error(t, err)

	srv.Close()
	_, err = NewSearchClient(srv.URL+"/top").Search(ctx, "x")
	assert.Error(t, err)
}

// ---------- openai client ----------

func TestOpenAIClient(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		gotBody = buf.String()
		switch r.URL.Path {
		case "/ok/chat/completions":
			w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"halo"}}]}`))
		case "/empty/chat/completions":
			w.Write([]byte(`{"choices":[]}`))
		case "/bad/chat/completions":
			w.Write([]byte(`oops`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	out, err := NewOpenAIClient(srv.URL+"/ok", "kunci", "model-x").Chat(ctx, "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, "halo", out)
	assert.Equal(t, "Bearer kunci", gotAuth)
	assert.True(t, strings.Contains(gotBody, `"model":"model-x"`))

	_, err = NewOpenAIClient(srv.URL+"/empty", "k", "m").Chat(ctx, "s", "u")
	assert.ErrorContains(t, err, "no choices")
	_, err = NewOpenAIClient(srv.URL+"/bad", "k", "m").Chat(ctx, "s", "u")
	assert.Error(t, err)
	_, err = NewOpenAIClient(srv.URL+"/denied", "k", "m").Chat(ctx, "s", "u")
	assert.ErrorContains(t, err, "status 401")
	_, err = NewOpenAIClient("://bad-url", "k", "m").Chat(ctx, "s", "u")
	assert.Error(t, err)

	srv.Close()
	_, err = NewOpenAIClient(srv.URL+"/ok", "k", "m").Chat(ctx, "s", "u")
	assert.Error(t, err)
}

// ---------- handler ----------

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/ok", NewHandler(NewService(fakeSearch{}, &fakeChat{out: goodJSON})).Generate)
	r.POST("/err", NewHandler(NewService(fakeSearch{}, &fakeChat{err: errors.New("x")})).Generate)

	post := func(path, body string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w.Code
	}
	assert.Equal(t, 200, post("/ok", `{"topic":"Go","include_web_search":true}`))
	assert.Equal(t, 400, post("/ok", `{}`))
	assert.Equal(t, 502, post("/err", `{"topic":"Go"}`))
}
