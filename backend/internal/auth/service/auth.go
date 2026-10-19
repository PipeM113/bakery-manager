package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	// MaxFailures failed attempts of one e-mail inside Window lock it (docs/acceso.md).
	MaxFailures = 5
	Window      = 15 * time.Minute

	// MinPasswordChars and MaxPasswordBytes bound a new password; bcrypt reads at most 72 bytes.
	MinPasswordChars = 10
	MaxPasswordBytes = 72
)

type User struct {
	ID    string
	Name  string
	Email string
	Role  string
}

// LockedError means too many failed attempts: nothing is checked until RetryAfter has passed.
type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string { return "demasiados intentos fallidos" }

// ErrInvalidCredentials is the one answer for a wrong password and for an e-mail that does not exist.
var ErrInvalidCredentials = errors.New("credenciales inválidas")

// ErrWrongCurrentPassword is a password change with a bad current password.
var ErrWrongCurrentPassword = errors.New("la contraseña actual no es correcta")

// ErrUnknownUser is a token of a user that no longer exists.
var ErrUnknownUser = errors.New("usuario no encontrado")

// ValidationError is a new password the person can fix.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret []byte
	clk       clock.Clock
}

// NewAuthService takes the JWT secret explicitly: it is validated once at startup, so
// the service never reads the environment while handling a request.
func NewAuthService(db *pgxpool.Pool, jwtSecret string, clk clock.Clock) *AuthService {
	return &AuthService{db: db, jwtSecret: []byte(jwtSecret), clk: clk}
}

// dummyHash is compared when the e-mail does not exist, so that answering takes about as long as
// for a real one.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

func normalize(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// checkLock returns a LockedError when the e-mail already failed MaxFailures times in the window.
// The failures of a locked e-mail are not recorded, so the lock ends when the oldest one ages out.
func (s *AuthService) checkLock(ctx context.Context, email string) error {
	now := s.clk()
	var count int
	var oldest *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT count(*), min(attempted_at) FROM login_attempts
		WHERE email = $1 AND attempted_at > $2`, email, now.Add(-Window)).Scan(&count, &oldest)
	if err != nil {
		return fmt.Errorf("check login attempts: %w", err)
	}
	if count >= MaxFailures && oldest != nil {
		retry := oldest.Add(Window).Sub(now)
		if retry < time.Second {
			retry = time.Second
		}
		return &LockedError{RetryAfter: retry}
	}
	return nil
}

func (s *AuthService) recordFailure(ctx context.Context, email string) error {
	now := s.clk()
	if _, err := s.db.Exec(ctx, `DELETE FROM login_attempts WHERE attempted_at <= $1`, now.Add(-Window)); err != nil {
		return fmt.Errorf("clean login attempts: %w", err)
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO login_attempts (email, attempted_at) VALUES ($1, $2)`, email, now); err != nil {
		return fmt.Errorf("record login attempt: %w", err)
	}
	return nil
}

func (s *AuthService) clearFailures(ctx context.Context, email string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM login_attempts WHERE email = $1`, email)
	return err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	key := normalize(email)
	if err := s.checkLock(ctx, key); err != nil {
		return "", err
	}

	var user User
	var hashedPassword string
	err := s.db.QueryRow(ctx,
		"SELECT id, name, email, role, password FROM users WHERE email = $1", email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &hashedPassword)
	known := err == nil
	if !known {
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("read user: %w", err)
		}
		hashedPassword = string(dummyHash)
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil || !known {
		if err := s.recordFailure(ctx, key); err != nil {
			return "", err
		}
		return "", ErrInvalidCredentials
	}
	if err := s.clearFailures(ctx, key); err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Name,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(), // real time: the JWT library checks it against the real clock
	})
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error generando token")
	}
	return signed, nil
}

// ChangePassword replaces the password of the user after checking the current one. A wrong current
// password counts as a failed login of that e-mail; open sessions stay valid until their token expires.
func (s *AuthService) ChangePassword(ctx context.Context, userID, current, next string) error {
	var email, hash string
	err := s.db.QueryRow(ctx, `SELECT email, password FROM users WHERE id = $1`, userID).Scan(&email, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUnknownUser
	}
	if err != nil {
		return fmt.Errorf("read user: %w", err)
	}
	key := normalize(email)
	if err := s.checkLock(ctx, key); err != nil {
		return err
	}

	switch {
	case current == "":
		return &ValidationError{"escribe tu contraseña actual"}
	case utf8.RuneCountInString(next) < MinPasswordChars:
		return &ValidationError{fmt.Sprintf("la contraseña nueva debe tener al menos %d caracteres", MinPasswordChars)}
	case len(next) > MaxPasswordBytes:
		return &ValidationError{fmt.Sprintf("la contraseña nueva no puede pasar de %d bytes", MaxPasswordBytes)}
	case next == current:
		return &ValidationError{"la contraseña nueva debe ser distinta de la actual"}
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		if err := s.recordFailure(ctx, key); err != nil {
			return err
		}
		return ErrWrongCurrentPassword
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := s.db.Exec(ctx, `UPDATE users SET password = $1 WHERE id = $2`, string(newHash), userID); err != nil {
		return fmt.Errorf("save password: %w", err)
	}
	return s.clearFailures(ctx, key)
}
