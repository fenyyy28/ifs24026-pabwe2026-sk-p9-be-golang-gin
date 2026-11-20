package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

var seq int

// failFrom membuat operasi database ke-n dan seterusnya gagal.
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
	case "delete":
		_ = db.Callback().Delete().Before("gorm:delete").Register(name, fn)
	}
}

func newDB(t *testing.T) *gorm.DB {
	db, err := database.OpenSQLite()
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	for _, u := range []database.User{
		{Username: "alice", Email: "alice@mail.com", Password: "x", Role: "member"},
		{Username: "bob", Email: "bob@mail.com", Password: "x", Role: "member"},
		{Username: "root", Email: "root@mail.com", Password: "x", Role: "admin"},
	} {
		u := u
		require.NoError(t, db.Create(&u).Error)
	}
	return db
}

func newService(t *testing.T) (*Service, *gorm.DB) {
	db := newDB(t)
	return NewService(NewRepository(db)), db
}

func boolPtr(b bool) *bool { return &b }

func TestSlugifyAndReadingTime(t *testing.T) {
	assert.Equal(t, "halo-dunia-2026", Slugify("  Halo, Dunia! 2026 "))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))
	assert.Equal(t, 3, ReadingTime(strings.Repeat("kata ", 450)))
}

func TestNormalizeCategory(t *testing.T) {
	assert.Equal(t, "Teknologi", normalizeCategory("Teknologi"))
	assert.Equal(t, "Umum", normalizeCategory("Aneh"))
	assert.Equal(t, "Umum", normalizeCategory(""))
}

func TestCreateAndUniqueSlug(t *testing.T) {
	svc, _ := newService(t)
	p1, err := svc.Create(1, PostRequest{Title: "Belajar Go", Content: "isi", Category: "Teknologi"})
	require.NoError(t, err)
	assert.Equal(t, "belajar-go", p1.Slug)
	assert.False(t, p1.IsPublished)
	assert.Nil(t, p1.PublishedAt)
	assert.Equal(t, "alice", p1.User.Username)

	p2, err := svc.Create(2, PostRequest{Title: "Belajar Go", Content: "isi", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	assert.Equal(t, "belajar-go-2", p2.Slug)
	assert.True(t, p2.IsPublished)
	assert.NotNil(t, p2.PublishedAt)
	assert.Equal(t, "Umum", p2.Category)

	p3, err := svc.Create(2, PostRequest{Title: "Belajar Go", Content: "isi"})
	require.NoError(t, err)
	assert.Equal(t, "belajar-go-3", p3.Slug)
}

func TestCreateErrors(t *testing.T) {
	svc, db := newService(t)
	failFrom(db, "create", 1)
	_, err := svc.Create(1, PostRequest{Title: "A", Content: "b"})
	assert.Error(t, err)

	svc2, db2 := newService(t)
	failFrom(db2, "query", 1)
	_, err = svc2.Create(1, PostRequest{Title: "A", Content: "b"})
	assert.Error(t, err)
}

func TestOwnedGetListUpdate(t *testing.T) {
	svc, db := newService(t)
	p, err := svc.Create(1, PostRequest{Title: "Judul Awal", Content: "isi"})
	require.NoError(t, err)

	got, err := svc.Get(1, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Judul Awal", got.Title)
	_, err = svc.Get(2, p.ID) // milik orang lain -> 403
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.Get(1, 999)
	assert.ErrorIs(t, err, ErrNotFound)

	list, err := svc.ListByAuthor(1)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	up, err := svc.Update(1, p.ID, PostRequest{Title: "Judul Baru", Content: "isi baru", Category: "Bisnis", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	assert.Equal(t, "judul-baru", up.Slug)
	assert.True(t, up.IsPublished)
	// judul sama + publish lagi: PublishedAt tidak berubah
	up2, err := svc.Update(1, p.ID, PostRequest{Title: "Judul Baru", Content: "isi baru", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	assert.Equal(t, up.Slug, up2.Slug)
	// tanpa is_published: status tidak berubah
	up3, err := svc.Update(1, p.ID, PostRequest{Title: "Judul Baru", Content: "x"})
	require.NoError(t, err)
	assert.True(t, up3.IsPublished)
	// unpublish lewat update
	up4, err := svc.Update(1, p.ID, PostRequest{Title: "Judul Baru", Content: "x", IsPublished: boolPtr(false)})
	require.NoError(t, err)
	assert.False(t, up4.IsPublished)
	assert.Nil(t, up4.PublishedAt)

	_, err = svc.Update(2, p.ID, PostRequest{Title: "X", Content: "y"})
	assert.ErrorIs(t, err, ErrForbidden)

	failFrom(db, "query", 3) // FindByID (2 query) sukses, SlugTaken gagal
	_, err = svc.Update(1, p.ID, PostRequest{Title: "Ganti Judul", Content: "y"})
	assert.Error(t, err)
}

func TestToggleDeleteStats(t *testing.T) {
	svc, db := newService(t)
	p, err := svc.Create(1, PostRequest{Title: "Toggle", Content: "isi"})
	require.NoError(t, err)
	_, err = svc.Create(1, PostRequest{Title: "Draf Lain", Content: "isi"})
	require.NoError(t, err)

	tp, err := svc.TogglePublish(1, p.ID)
	require.NoError(t, err)
	assert.True(t, tp.IsPublished)
	assert.NotNil(t, tp.PublishedAt)

	st, err := svc.Stats(1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), st.Total)
	assert.Equal(t, int64(1), st.TotalTerbit)
	assert.Equal(t, int64(1), st.TotalDraf)
	assert.Equal(t, int64(1), st.BelumSelesai)

	tp, err = svc.TogglePublish(1, p.ID)
	require.NoError(t, err)
	assert.False(t, tp.IsPublished)
	assert.Nil(t, tp.PublishedAt)
	_, err = svc.TogglePublish(2, p.ID)
	assert.ErrorIs(t, err, ErrForbidden)

	assert.ErrorIs(t, svc.Delete(2, p.ID), ErrForbidden)
	require.NoError(t, svc.Delete(1, p.ID))
	_, err = svc.Get(1, p.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	failFrom(db, "query", 1)
	_, err = svc.Stats(1)
	assert.Error(t, err)
}

func TestStatsSecondCountFails(t *testing.T) {
	svc, db := newService(t)
	failFrom(db, "query", 2)
	_, err := svc.Stats(1)
	assert.Error(t, err)
}

func TestDeleteError(t *testing.T) {
	svc, db := newService(t)
	p, err := svc.Create(1, PostRequest{Title: "Hapus", Content: "isi"})
	require.NoError(t, err)
	failFrom(db, "delete", 1)
	assert.Error(t, svc.Delete(1, p.ID))
}

func TestPublicQueries(t *testing.T) {
	svc, _ := newService(t)
	_, err := svc.Create(1, PostRequest{Title: "Kecerdasan Buatan", Content: "membahas AI modern", Category: "Teknologi", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	_, err = svc.Create(2, PostRequest{Title: "Resep Nasi Goreng", Content: "bumbu dapur", Category: "Gaya Hidup", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	_, err = svc.Create(2, PostRequest{Title: "Draf Rahasia", Content: "belum terbit"})
	require.NoError(t, err)

	all, err := svc.PublicList("", "")
	require.NoError(t, err)
	assert.Len(t, all, 2)
	semua, err := svc.PublicList("", "Semua")
	require.NoError(t, err)
	assert.Len(t, semua, 2)

	byQ, err := svc.PublicList("  KECERDASAN ", "")
	require.NoError(t, err)
	require.Len(t, byQ, 1)
	assert.Equal(t, "Kecerdasan Buatan", byQ[0].Title)

	byCat, err := svc.PublicList("", "Gaya Hidup")
	require.NoError(t, err)
	assert.Len(t, byCat, 1)

	d, err := svc.PublicDetail("resep-nasi-goreng")
	require.NoError(t, err)
	assert.Equal(t, "bob", d.User.Username)
	_, err = svc.PublicDetail("draf-rahasia")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAdminQueries(t *testing.T) {
	svc, _ := newService(t)
	_, err := svc.Create(1, PostRequest{Title: "A1", Content: "x", IsPublished: boolPtr(true)})
	require.NoError(t, err)
	_, err = svc.Create(1, PostRequest{Title: "A2", Content: "x"})
	require.NoError(t, err)

	users, err := svc.AdminUsers()
	require.NoError(t, err)
	require.Len(t, users, 3)
	assert.Equal(t, int64(2), users[0].TotalPosts)
	assert.Equal(t, int64(1), users[0].TotalPublished)
	assert.Equal(t, int64(0), users[1].TotalPosts)

	posts, err := svc.AdminPosts()
	require.NoError(t, err)
	assert.Len(t, posts, 2)

	ups, err := svc.AdminUserPosts(1)
	require.NoError(t, err)
	assert.Len(t, ups, 2)
	ups, err = svc.AdminUserPosts(999) // user tidak ada -> daftar kosong, bukan error
	require.NoError(t, err)
	assert.Len(t, ups, 0)
}

func TestAdminUserPostsDBError(t *testing.T) {
	svc, db := newService(t)
	failFrom(db, "query", 1)
	_, err := svc.AdminUserPosts(1)
	assert.Error(t, err)
}

func TestToResponsesEmpty(t *testing.T) {
	assert.NotNil(t, ToResponses(nil))
	assert.Len(t, ToResponses(nil), 0)
}

// ---------------- handler ----------------

type env struct {
	r  *gin.Engine
	db *gorm.DB
}

func newEnv(t *testing.T) *env {
	gin.SetMode(gin.TestMode)
	svc, db := newService(t)
	h := NewHandler(svc)
	r := gin.New()
	// simulasi Auth: user id & role dari header
	r.Use(func(c *gin.Context) {
		var id uint
		fmt.Sscanf(c.GetHeader("X-User"), "%d", &id)
		c.Set("user_id", id)
		c.Next()
	})
	r.GET("/public/categories", h.Categories)
	r.GET("/public/posts", h.PublicList)
	r.GET("/public/posts/:slug", h.PublicDetail)
	r.GET("/posts", h.List)
	r.POST("/posts", h.Create)
	r.GET("/posts/stats", h.Stats)
	r.GET("/posts/:id", h.Get)
	r.PUT("/posts/:id", h.Update)
	r.PATCH("/posts/:id/toggle-publish", h.TogglePublish)
	r.DELETE("/posts/:id", h.Delete)
	r.GET("/admin/users", h.AdminUsers)
	r.GET("/admin/posts", h.AdminPosts)
	r.GET("/admin/users/:id/posts", h.AdminUserPosts)
	_ = middleware.UserID
	return &env{r: r, db: db}
}

func (e *env) do(method, path string, user int, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User", fmt.Sprint(user))
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestHandlerHappyPath(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 200, e.do("GET", "/public/categories", 0, nil).Code)

	body := map[string]interface{}{"title": "Artikel Satu", "content": "isi artikel", "category": "Opini", "is_published": true}
	w := e.do("POST", "/posts", 1, body)
	require.Equal(t, 201, w.Code)

	assert.Equal(t, 200, e.do("GET", "/posts", 1, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/posts/stats", 1, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/posts/1", 1, nil).Code)
	assert.Equal(t, 200, e.do("PUT", "/posts/1", 1, map[string]interface{}{"title": "Artikel Satu", "content": "baru"}).Code)
	assert.Equal(t, 200, e.do("PATCH", "/posts/1/toggle-publish", 1, nil).Code)
	assert.Equal(t, 200, e.do("PATCH", "/posts/1/toggle-publish", 1, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/public/posts?q=artikel&category=Opini", 0, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/public/posts/artikel-satu", 0, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/admin/users", 3, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/admin/posts", 3, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/admin/users/1/posts", 3, nil).Code)
	assert.Equal(t, 200, e.do("DELETE", "/posts/1", 1, nil).Code)
}

func TestHandlerClientErrors(t *testing.T) {
	e := newEnv(t)
	for _, c := range []struct{ m, p string }{
		{"GET", "/posts/abc"}, {"PUT", "/posts/abc"}, {"PATCH", "/posts/abc/toggle-publish"},
		{"DELETE", "/posts/abc"}, {"GET", "/admin/users/abc/posts"},
	} {
		assert.Equal(t, 400, e.do(c.m, c.p, 1, `{"title":"a","content":"b"}`).Code, c.m+" "+c.p)
	}
	assert.Equal(t, 400, e.do("POST", "/posts", 1, `{}`).Code)
	assert.Equal(t, 400, e.do("PUT", "/posts/1", 1, `{}`).Code)

	assert.Equal(t, 404, e.do("GET", "/posts/99", 1, nil).Code)
	assert.Equal(t, 404, e.do("PUT", "/posts/99", 1, `{"title":"a","content":"b"}`).Code)
	assert.Equal(t, 404, e.do("PATCH", "/posts/99/toggle-publish", 1, nil).Code)
	assert.Equal(t, 404, e.do("DELETE", "/posts/99", 1, nil).Code)
	assert.Equal(t, 404, e.do("GET", "/public/posts/tidak-ada", 0, nil).Code)
	assert.Equal(t, 200, e.do("GET", "/admin/users/99/posts", 3, nil).Code)
}

func TestHandlerServerErrors(t *testing.T) {
	e := newEnv(t)
	failFrom(e.db, "query", 1)
	for _, c := range []struct{ m, p string }{
		{"GET", "/public/posts"}, {"GET", "/public/posts/x"}, {"GET", "/posts"}, {"GET", "/posts/stats"},
		{"GET", "/posts/1"}, {"GET", "/admin/users"}, {"GET", "/admin/posts"}, {"GET", "/admin/users/1/posts"},
	} {
		assert.Equal(t, 500, e.do(c.m, c.p, 1, nil).Code, c.m+" "+c.p)
	}
	assert.Equal(t, 500, e.do("POST", "/posts", 1, `{"title":"a","content":"b"}`).Code)
}