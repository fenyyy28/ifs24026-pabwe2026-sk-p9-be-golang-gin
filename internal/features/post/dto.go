package post

import (
	"time"

	"app/internal/database"
)

// PostRequest adalah payload pembuatan/pembaruan artikel.
type PostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

// AuthorInfo adalah ringkasan penulis artikel.
type AuthorInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// PostResponse adalah representasi artikel untuk klien.
type PostResponse struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	Author         AuthorInfo `json:"author"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Stats adalah ringkasan 3 kartu statistik penulis.
type Stats struct {
	TotalArticles int64 `json:"total_articles"`
	Total         int64 `json:"total"`
	TotalSelesai  int64 `json:"total_selesai"`
	TotalTerbit   int64 `json:"total_terbit"`
	BelumSelesai  int64 `json:"belum_selesai"`
	TotalDraf     int64 `json:"total_draf"`
}

// UserSummary adalah ringkasan pengguna untuk monitoring admin.
type UserSummary struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	TotalPosts     int64  `json:"total_posts"`
	TotalArticles  int64  `json:"total_articles"`
	TotalPublished int64  `json:"total_published"`
	TotalSelesai   int64  `json:"total_selesai"`
	TotalDraf      int64  `json:"total_draf"`
	BelumSelesai   int64  `json:"belum_selesai"`
}

// PublicPostResponse adalah representasi publik tanpa data internal (tanpa content, user_id, is_published).
type PublicPostResponse struct {
	ID             uint       `json:"id"`
	Author         AuthorInfo `json:"author"`
	AuthorName     string     `json:"author_name"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	CoverImagePath string     `json:"cover_image_path"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ToPublicResponse mengubah model Post menjadi PublicPostResponse tanpa data internal.
func ToPublicResponse(p *database.Post) PublicPostResponse {
	return PublicPostResponse{
		ID:             p.ID,
		Author:         AuthorInfo{ID: p.User.ID, Username: p.User.Username},
		AuthorName:     p.User.Username,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		CoverImagePath: p.CoverImagePath,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    ReadingTime(p.Content),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToPublicResponses mengubah daftar Post menjadi daftar PublicPostResponse (selalu non-nil).
func ToPublicResponses(posts []database.Post) []PublicPostResponse {
	out := make([]PublicPostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, ToPublicResponse(&posts[i]))
	}
	return out
}

// ToResponse mengubah model Post menjadi PostResponse.
func ToResponse(p *database.Post) PostResponse {
	return PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		Author:         AuthorInfo{ID: p.User.ID, Username: p.User.Username},
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    ReadingTime(p.Content),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToResponses mengubah daftar Post menjadi daftar PostResponse (selalu non-nil).
func ToResponses(posts []database.Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, ToResponse(&posts[i]))
	}
	return out
}