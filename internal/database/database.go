package database

import (
	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openPostgres dipisahkan sebagai variabel agar koneksi dapat diganti saat pengujian.
var openPostgres = func(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// Connect membuka koneksi PostgreSQL untuk runtime aplikasi.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	return openPostgres(cfg.DSN())
}

// OpenSQLite membuka SQLite in-memory yang dipakai oleh unit test mandiri.
func OpenSQLite() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err == nil {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(1)
	}
	return db, err
}

// Migrate menjalankan AutoMigrate seluruh model.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

type seedUser struct {
	username string
	email    string
	password string
	role     string
}

// Seed menyuntikkan akun bawaan: admin (dari .env) dan member/member.
func Seed(db *gorm.DB, cfg *config.Config) error {
	users := []seedUser{
		{cfg.AdminUsername, "admin@delcom.org", cfg.AdminPassword, "admin"},
		{"member", "member@delcom.org", "member", "member"},
	}
	for _, s := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		// Buat bila belum ada; bila sudah ada, pastikan password & role sesuai .env.
		user := User{Username: s.username}
		if err := db.Where("username = ?", s.username).
			Attrs(User{Email: s.email}).
			Assign(User{Password: string(hash), Role: s.role}).
			FirstOrCreate(&user).Error; err != nil {
			return err
		}
	}
	return nil
}	