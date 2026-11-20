package ai

import (
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler adalah controller endpoint POST /api/posts/ai-generate.
type Handler struct{ svc *Service }

// NewHandler membuat Handler baru.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Generate godoc: POST /api/posts/ai-generate
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}
	res, err := h.svc.Generate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "AI generation failed: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Article draft generated", res)
}
