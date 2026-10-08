package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    string
	Name  string
	Email string
	Role  string
}

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret []byte
}

// NewAuthService takes the JWT secret explicitly: it is validated once at startup, so
// the service never reads the environment while handling a request.
func NewAuthService(db *pgxpool.Pool, jwtSecret string) *AuthService {
	return &AuthService{db: db, jwtSecret: []byte(jwtSecret)}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	var user User
	var hashedPassword string

	row := s.db.QueryRow(ctx,
		"SELECT id, name, email, role, password FROM users WHERE email = $1",
		email,
	)
	if err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &hashedPassword); err != nil {
		return "", fmt.Errorf("credenciales inválidas")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return "", fmt.Errorf("credenciales inválidas")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Name,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error generando token")
	}

	return signed, nil
}
