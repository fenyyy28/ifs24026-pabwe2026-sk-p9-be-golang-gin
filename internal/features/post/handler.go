package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler adalah controller endpoint /api/public/*, /api/posts/*, dan /api/admin/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler baru.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) fail(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		response.Fail(c, http.StatusForbidden, err.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, "Internal server error")
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "Invalid ID")
		return 0, false
	}
	return uint(id), true
}

func bindPost(c *gin.Context) (PostRequest, bool) {
	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return req, false
	}
	return req, true
}

// Categories godoc: GET /api/public/categories
func (h *Handler) Categories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Categories retrieved", Categories)
}

// PublicList godoc: GET /api/public/posts?q=&category=
func (h *Handler) PublicList(c *gin.Context) {
	posts, err := h.svc.PublicList(c.Query("q"), c.Query("category"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved", ToPublicResponses(posts))
}

// PublicDetail godoc: GET /api/public/posts/:slug
func (h *Handler) PublicDetail(c *gin.Context) {
	p, err := h.svc.PublicDetail(c.Param("slug"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", ToResponse(p))
}

// List godoc: GET /api/posts
func (h *Handler) List(c *gin.Context) {
	posts, err := h.svc.ListByAuthor(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved", ToResponses(posts))
}

// Create godoc: POST /api/posts
func (h *Handler) Create(c *gin.Context) {
	req, ok := bindPost(c)
	if !ok {
		return
	}
	p, err := h.svc.Create(middleware.UserID(c), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Post created", ToResponse(p))
}

// Stats godoc: GET /api/posts/stats
func (h *Handler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Stats retrieved", stats)
}

// Get godoc: GET /api/posts/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", ToResponse(p))
}

// Update godoc: PUT /api/posts/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	req, ok := bindPost(c)
	if !ok {
		return
	}
	p, err := h.svc.Update(middleware.UserID(c), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post updated", ToResponse(p))
}

// TogglePublish godoc: PATCH /api/posts/:id/toggle-publish
func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.TogglePublish(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Publication status updated", ToResponse(p))
}

// Delete godoc: DELETE /api/posts/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(middleware.UserID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Post deleted", nil)
}

// AdminUsers godoc: GET /api/admin/users
func (h *Handler) AdminUsers(c *gin.Context) {
	users, err := h.svc.AdminUsers()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Users retrieved", users)
}

// AdminPosts godoc: GET /api/admin/posts
func (h *Handler) AdminPosts(c *gin.Context) {
	posts, err := h.svc.AdminPosts()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved", ToResponses(posts))
}

// AdminUserPosts godoc: GET /api/admin/users/:id/posts
func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.svc.AdminUserPosts(id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Posts retrieved", ToResponses(posts))
}
