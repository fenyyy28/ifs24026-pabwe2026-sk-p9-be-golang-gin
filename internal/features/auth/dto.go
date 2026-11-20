package auth

import "app/internal/database"

// RegisterRequest adalah payload registrasi akun baru.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest adalah payload login (username atau email).
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest adalah payload pembaruan profil (semua kolom opsional).
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
}

// UserResponse adalah representasi publik data pengguna.
type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// AuthResponse dikembalikan setelah login berhasil.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ToUserResponse mengubah model User menjadi UserResponse.
func ToUserResponse(u *database.User) UserResponse {
	return UserResponse{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role}
}
