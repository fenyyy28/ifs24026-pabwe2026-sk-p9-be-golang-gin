package auth

import (
	"app/internal/database"

	"gorm.io/gorm"
)

// Repository menangani query data user ke database.
type Repository struct{ db *gorm.DB }

// NewRepository membuat Repository baru.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create menyimpan user baru.
func (r *Repository) Create(u *database.User) error { return r.db.Create(u).Error }

// Save memperbarui user.
func (r *Repository) Save(u *database.User) error { return r.db.Save(u).Error }

// FindByID mencari user berdasarkan ID.
func (r *Repository) FindByID(id uint) (*database.User, error) {
	var u database.User
	err := r.db.First(&u, id).Error
	return &u, err
}

// FindByLogin mencari user berdasarkan username atau email.
func (r *Repository) FindByLogin(login string) (*database.User, error) {
	var u database.User
	err := r.db.Where("username = ? OR email = ?", login, login).First(&u).Error
	return &u, err
}

// Exists memeriksa apakah username/email sudah dipakai user lain (selain excludeID).
func (r *Repository) Exists(username, email string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.Model(&database.User{}).
		Where("(username = ? OR email = ?) AND id <> ?", username, email, excludeID).
		Count(&count).Error
	return count > 0, err
}
