package post

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"app/internal/database"

	"gorm.io/gorm"
)

// ErrNotFound dikembalikan saat artikel/user tidak ditemukan.
var ErrNotFound = errors.New("post not found")

// ErrForbidden dikembalikan saat pengguna mencoba mengakses/mengubah artikel milik orang lain.
var ErrForbidden = errors.New("forbidden: article belongs to another user")

// Categories adalah daftar kategori artikel yang valid.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify mengubah judul menjadi slug URL (huruf kecil, dipisah tanda hubung).
func Slugify(title string) string {
	s := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if s == "" {
		return "artikel"
	}
	return s
}

// ReadingTime menghitung estimasi menit baca (kata / 200 wpm, minimal 1).
func ReadingTime(content string) int {
	minutes := int(math.Ceil(float64(len(strings.Fields(content))) / 200.0))
	if minutes < 1 {
		return 1
	}
	return minutes
}

func normalizeCategory(category string) string {
	for _, c := range Categories {
		if c == category {
			return c
		}
	}
	return "Umum"
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

// Service berisi logika bisnis artikel.
type Service struct{ repo *Repository }

// NewService membuat Service baru.
func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) uniqueSlug(title string, excludeID uint) (string, error) {
	base := Slugify(title)
	slug := base
	for i := 2; ; i++ {
		taken, err := s.repo.SlugTaken(slug, excludeID)
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func applyPublish(p *database.Post, publish bool) {
	if !publish {
		p.IsPublished = false
		p.PublishedAt = nil
		return
	}
	if p.PublishedAt == nil {
		now := time.Now()
		p.PublishedAt = &now
	}
	p.IsPublished = true
}

// Create membuat artikel baru lengkap dengan slug unik.
func (s *Service) Create(userID uint, req PostRequest) (*database.Post, error) {
	slug, err := s.uniqueSlug(req.Title, 0)
	if err != nil {
		return nil, err
	}
	p := &database.Post{
		UserID: userID, Title: req.Title, Slug: slug,
		Category: normalizeCategory(req.Category), Excerpt: req.Excerpt,
		Content: req.Content, CoverImagePath: req.CoverImagePath,
	}
	applyPublish(p, req.IsPublished != nil && *req.IsPublished)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return s.repo.FindByID(p.ID)
}

// owned mengambil artikel milik user; artikel milik orang lain mengembalikan ErrForbidden.
func (s *Service) owned(userID, id uint) (*database.Post, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if p.UserID != userID {
		return nil, ErrForbidden
	}
	return p, nil
}

// CheckOwner memastikan artikel ada (404) dan milik user (403) tanpa mengubah data.
func (s *Service) CheckOwner(userID, id uint) error {
	_, err := s.owned(userID, id)
	return err
}

// Get mengambil artikel milik penulis berdasarkan ID.
func (s *Service) Get(userID, id uint) (*database.Post, error) { return s.owned(userID, id) }

// ListByAuthor mengambil seluruh artikel milik penulis.
func (s *Service) ListByAuthor(userID uint) ([]database.Post, error) { return s.repo.ListByUser(userID) }

// Update memperbarui artikel milik penulis.
func (s *Service) Update(userID, id uint, req PostRequest) (*database.Post, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	if req.Title != p.Title {
		slug, err := s.uniqueSlug(req.Title, p.ID)
		if err != nil {
			return nil, err
		}
		p.Slug = slug
	}
	p.Title = req.Title
	p.Category = normalizeCategory(req.Category)
	p.Excerpt = req.Excerpt
	p.Content = req.Content
	p.CoverImagePath = req.CoverImagePath
	if req.IsPublished != nil {
		applyPublish(p, *req.IsPublished)
	}
	return p, s.repo.Save(p)
}

// TogglePublish mengubah status Draf <-> Terbit beserta published_at.
func (s *Service) TogglePublish(userID, id uint) (*database.Post, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	applyPublish(p, !p.IsPublished)
	return p, s.repo.Save(p)
}

// Delete menghapus artikel milik penulis.
func (s *Service) Delete(userID, id uint) error {
	p, err := s.owned(userID, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(p)
}

// Stats menghitung ringkasan 3 kartu statistik penulis.
func (s *Service) Stats(userID uint) (*Stats, error) {
	total, published, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}
	draft := total - published
	return &Stats{
		TotalArticles: total,
		Total:         total,
		TotalSelesai:  published,
		TotalTerbit:   published,
		BelumSelesai:  draft,
		TotalDraf:     draft,
	}, nil
}

// PublicList mengambil artikel terbit dengan pencarian & filter kategori.
func (s *Service) PublicList(q, category string) ([]database.Post, error) {
	return s.repo.ListPublic(strings.TrimSpace(q), category)
}

// PublicDetail mengambil artikel terbit berdasarkan slug.
func (s *Service) PublicDetail(slug string) (*database.Post, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	return p, mapNotFound(err)
}

// AdminUsers mengambil seluruh user beserta total artikelnya.
func (s *Service) AdminUsers() ([]UserSummary, error) {
	return s.repo.UserSummaries()
}

// AdminPosts mengambil seluruh artikel dari semua penulis.
func (s *Service) AdminPosts() ([]database.Post, error) { return s.repo.ListAll() }

// AdminUserPosts mengambil seluruh artikel milik user tertentu; user tanpa artikel / tidak ada mengembalikan daftar kosong.
func (s *Service) AdminUserPosts(userID uint) ([]database.Post, error) {
	return s.repo.ListByUser(userID)
}