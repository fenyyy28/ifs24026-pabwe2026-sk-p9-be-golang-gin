package auth

import (
	"errors"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	// ErrDuplicate dikembalikan saat username/email sudah terdaftar.
	ErrDuplicate = errors.New("username or email already registered")
	// ErrInvalidCredentials dikembalikan saat login gagal.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrNotFound dikembalikan saat user tidak ditemukan.
	ErrNotFound = errors.New("user not found")
)

// Service berisi logika bisnis autentikasi.
type Service struct {
	repo *Repository
	key  interface{}
	cost int
}

// NewService membuat Service dengan secret JWT.
func NewService(repo *Repository, secret string) *Service {
	return &Service{repo: repo, key: []byte(secret), cost: bcrypt.DefaultCost}
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

// Register mendaftarkan akun baru dengan role member.
func (s *Service) Register(req RegisterRequest) (*database.User, error) {
	exists, err := s.repo.Exists(req.Username, req.Email, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicate
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cost)
	if err != nil {
		return nil, err
	}
	u := &database.User{Username: req.Username, Email: req.Email, Password: string(hash), Role: "member"}
	return u, s.repo.Create(u)
}

// Login memverifikasi kredensial lalu menerbitkan token JWT.
func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	u, err := s.repo.FindByLogin(req.Username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	token, err := s.token(u)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: ToUserResponse(u)}, nil
}

func (s *Service) token(u *database.User) (string, error) {
	claims := middleware.Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.key)
}

// Profile mengambil data user berdasarkan ID.
func (s *Service) Profile(id uint) (*database.User, error) {
	u, err := s.repo.FindByID(id)
	return u, mapNotFound(err)
}

// UpdateProfile memperbarui username, email, dan/atau password.
func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*database.User, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	if req.Username != "" {
		u.Username = req.Username
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	exists, err := s.repo.Exists(u.Username, u.Email, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicate
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cost)
		if err != nil {
			return nil, err
		}
		u.Password = string(hash)
	}
	return u, s.repo.Save(u)
}
