package post

import (
	"app/internal/database"

	"gorm.io/gorm"
)

// Repository menangani query artikel via GORM.
type Repository struct{ db *gorm.DB }

// NewRepository membuat Repository baru.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create menyimpan artikel baru.
func (r *Repository) Create(p *database.Post) error { return r.db.Omit("User").Create(p).Error }

// Save memperbarui artikel.
func (r *Repository) Save(p *database.Post) error { return r.db.Omit("User").Save(p).Error }

// Delete menghapus artikel.
func (r *Repository) Delete(p *database.Post) error { return r.db.Delete(p).Error }

// FindByID mencari artikel (beserta penulis) berdasarkan ID.
func (r *Repository) FindByID(id uint) (*database.Post, error) {
	var p database.Post
	err := r.db.Preload("User").First(&p, id).Error
	return &p, err
}

// FindPublishedBySlug mencari artikel terbit berdasarkan slug.
func (r *Repository) FindPublishedBySlug(slug string) (*database.Post, error) {
	var p database.Post
	err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error
	return &p, err
}

// SlugTaken memeriksa apakah slug sudah dipakai artikel lain (selain excludeID).
func (r *Repository) SlugTaken(slug string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.Post{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&count).Error
	return count > 0, err
}

// ListByUser mengambil seluruh artikel milik seorang penulis.
func (r *Repository) ListByUser(userID uint) ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

// ListAll mengambil seluruh artikel dari semua penulis.
func (r *Repository) ListAll() ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Order("created_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

// ListPublic mengambil artikel terbit dengan pencarian kata kunci dan filter kategori.
func (r *Repository) ListPublic(q, category string) ([]database.Post, error) {
	tx := r.db.Preload("User").Where("is_published = ?", true)
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("LOWER(title) LIKE LOWER(?) OR LOWER(excerpt) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?)", like, like, like)
	}
	if category != "" && category != "Semua" {
		tx = tx.Where("category = ?", category)
	}
	var posts []database.Post
	err := tx.Order("published_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

// CountByUser menghitung total artikel dan artikel terbit milik seorang penulis.
func (r *Repository) CountByUser(userID uint) (total int64, published int64, err error) {
	if err = r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error
	return
}

// UserExists memeriksa apakah user dengan ID tertentu ada.
func (r *Repository) UserExists(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.User{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// UserSummaries mengambil seluruh user beserta total artikelnya.
func (r *Repository) UserSummaries() ([]UserSummary, error) {
	var out []UserSummary
	err := r.db.Model(&database.User{}).
		Select("users.id AS id, users.username AS username, users.email AS email, users.role AS role, "+
			"(SELECT COUNT(*) FROM posts WHERE posts.user_id = users.id) AS total_posts, "+
			"(SELECT COUNT(*) FROM posts WHERE posts.user_id = users.id AND posts.is_published = ?) AS total_published", true).
		Order("users.id").Find(&out).Error
	return out, err
}
