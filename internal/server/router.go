package server

import (
	"net/http"
	"time"

	"app/docs"
	"app/internal/config"
	"app/internal/debuglog"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// NewRouter mendaftarkan seluruh rute API publik, penulis, admin, serta Swagger UI.
func NewRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})

	// Swagger UI & spesifikasi OpenAPI dinamis
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))
	r.GET("/api.json", func(c *gin.Context) {
		scheme := c.GetHeader("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
		c.Data(http.StatusOK, "application/json", docs.GetSwaggerJSON(c.Request.Host, scheme))
	})

	// DIAGNOSIS SEMENTARA (hapus setelah masalah admin selesai)
	r.GET("/api/_debug/log", func(c *gin.Context) {
		if c.Query("k") != "feny-debug-2026" {
			response.Fail(c, http.StatusNotFound, "Endpoint not found")
			return
		}
		response.Success(c, http.StatusOK, "debug", debuglog.Entries())
	})

	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.AppKey))
	postHandler := post.NewHandler(post.NewService(post.NewRepository(db)))
	aiHandler := ai.NewHandler(ai.NewService(
		ai.NewSearchClient(cfg.SearchEngineEndpoint),
		ai.NewOpenAIClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel),
	))

	api := r.Group("/api", middleware.RateLimiter(600, time.Minute))

	// Publik
	api.GET("/public/categories", postHandler.Categories)
	api.GET("/public/posts", postHandler.PublicList)
	api.GET("/public/posts/:slug", postHandler.PublicDetail)

	// Auth
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	secured := api.Group("", middleware.Auth(cfg.AppKey))
	secured.POST("/auth/logout", authHandler.Logout)
	secured.GET("/auth/profile", authHandler.GetProfile)
	secured.PUT("/auth/profile", authHandler.UpdateProfile)

	// Penulis
	secured.GET("/posts", postHandler.List)
	secured.POST("/posts", postHandler.Create)
	secured.GET("/posts/stats", postHandler.Stats)
	secured.POST("/posts/ai-generate", aiHandler.Generate)
	secured.GET("/posts/:id", postHandler.Get)
	secured.PUT("/posts/:id", postHandler.Update)
	secured.PATCH("/posts/:id/toggle-publish", postHandler.TogglePublish)
	secured.DELETE("/posts/:id", postHandler.Delete)

	// Admin
	admin := secured.Group("/admin", middleware.RequireRole("admin"))
	admin.GET("/users", postHandler.AdminUsers)
	admin.GET("/posts", postHandler.AdminPosts)
	admin.GET("/users/:id/posts", postHandler.AdminUserPosts)

	return r
}