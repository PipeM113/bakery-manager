// Package repository keeps the data of the business that the quotation PDF prints (name, contact,
// transfer data). The RUT and the bank account number are not stored: they come from the environment.
package repository

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountTypes are the kinds of bank account a customer can transfer to.
var AccountTypes = []string{"Cuenta Corriente", "Cuenta Vista", "Cuenta RUT", "Cuenta de Ahorro"}

// ValidationError is a profile the owner can fix.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Profile struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Instagram   string `json:"instagram"`
	Bank        string `json:"bank"`
	AccountType string `json:"account_type"`
	Holder      string `json:"holder"`
	Email       string `json:"email"`
}

// Missing names what keeps the PDF from being made: the empty fields of the profile (by their JSON
// name) and then the environment variables that are not set.
func (p Profile) Missing(env config.Business) []string {
	missing := []string{}
	for _, f := range []struct{ key, value string }{
		{"name", p.Name}, {"phone", p.Phone}, {"instagram", p.Instagram}, {"bank", p.Bank},
		{"account_type", p.AccountType}, {"holder", p.Holder}, {"email", p.Email},
	} {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.key)
		}
	}
	return append(missing, env.Missing()...)
}

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Get(ctx context.Context) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `SELECT name, phone, instagram, bank, account_type, holder, email FROM business_profile`).
		Scan(&p.Name, &p.Phone, &p.Instagram, &p.Bank, &p.AccountType, &p.Holder, &p.Email)
	if err != nil {
		return p, fmt.Errorf("read business profile: %w", err)
	}
	return p, nil
}

// Save checks and stores the profile (all of it, every field required).
func (r *Repository) Save(ctx context.Context, in Profile, userID string) (Profile, error) {
	p, err := Clean(in)
	if err != nil {
		return Profile{}, err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE business_profile
		SET name = $1, phone = $2, instagram = $3, bank = $4, account_type = $5, holder = $6, email = $7,
		    updated_at = NOW(), updated_by = NULLIF($8, '')::uuid`,
		p.Name, p.Phone, p.Instagram, p.Bank, p.AccountType, p.Holder, p.Email, userID)
	if err != nil {
		return Profile{}, fmt.Errorf("save business profile: %w", err)
	}
	return p, nil
}

var instagramRE = regexp.MustCompile(`^@?[A-Za-z0-9._]{1,30}$`)

// Clean trims and checks every field, and returns the profile as it will be stored.
func Clean(in Profile) (Profile, error) {
	p := Profile{
		Name: strings.TrimSpace(in.Name), Phone: strings.TrimSpace(in.Phone), Instagram: strings.TrimSpace(in.Instagram),
		Bank: strings.TrimSpace(in.Bank), AccountType: strings.TrimSpace(in.AccountType),
		Holder: strings.TrimSpace(in.Holder), Email: strings.ToLower(strings.TrimSpace(in.Email)),
	}
	text := func(label, v string, max int) error {
		switch {
		case v == "":
			return &ValidationError{label + " es obligatorio"}
		case len([]rune(v)) > max:
			return &ValidationError{fmt.Sprintf("%s no puede superar %d caracteres", label, max)}
		}
		return nil
	}
	for _, c := range []struct {
		label, v string
		max      int
	}{{"el nombre del negocio", p.Name, 100}, {"el banco", p.Bank, 60}, {"el titular de la cuenta", p.Holder, 100}} {
		if err := text(c.label, c.v, c.max); err != nil {
			return Profile{}, err
		}
	}

	if err := text("el teléfono", p.Phone, 30); err != nil {
		return Profile{}, err
	}
	digits := 0
	for _, r := range p.Phone {
		switch {
		case unicode.IsDigit(r):
			digits++
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return Profile{}, &ValidationError{"el teléfono solo puede llevar números, espacios, +, - y paréntesis"}
		}
	}
	if digits < 8 {
		return Profile{}, &ValidationError{"el teléfono debe tener al menos 8 dígitos"}
	}

	if !instagramRE.MatchString(p.Instagram) {
		return Profile{}, &ValidationError{"el Instagram debe ser un usuario sin espacios, como @mitienda"}
	}
	p.Instagram = "@" + strings.TrimPrefix(p.Instagram, "@")

	found := false
	for _, t := range AccountTypes {
		if p.AccountType == t {
			found = true
		}
	}
	if !found {
		return Profile{}, &ValidationError{"el tipo de cuenta debe ser " + strings.Join(AccountTypes, ", ")}
	}

	if err := text("el correo", p.Email, 120); err != nil {
		return Profile{}, err
	}
	addr, err := mail.ParseAddress(p.Email)
	if err != nil || addr.Address != p.Email || !strings.Contains(p.Email[strings.LastIndex(p.Email, "@")+1:], ".") {
		return Profile{}, &ValidationError{"el correo no es válido"}
	}
	return p, nil
}
