package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
)

// Sprint 13, rules C1 and C2: failed logins are limited per e-mail (the counter lives in the
// database, because every Vercel instance is a different process), and the password can be changed
// from the app. The clock of these tests stands at noon of 2026-03-31 (Santiago).

const noonMarch31 = "2026-03-31T16:00:00Z"

func loginAs(api http.Handler, email, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec
}

func failN(t *testing.T, api *timeAPI, email string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		if rec := loginAs(api.h, email, "wrong-password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("failed attempt %d: status = %d, want 401: %s", i, rec.Code, rec.Body)
		}
	}
}

func TestC1_TheSixthAttemptIsLockedEvenWithTheRightPassword(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	failN(t, api, testEmail, 5)

	for _, pw := range []string{"wrong-password", testPassword} {
		rec := loginAs(api.h, testEmail, pw)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("after 5 failures: status = %d, want 429: %s", rec.Code, rec.Body)
		}
		secs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		if err != nil || secs < 1 || secs > 900 {
			t.Errorf("Retry-After = %q, want whole seconds between 1 and 900", rec.Header().Get("Retry-After"))
		}
		if !strings.Contains(strings.ToLower(rec.Body.String()), "intentos") {
			t.Errorf("the message should say there were too many attempts: %s", rec.Body)
		}
	}
}

func TestC1_TheLockLastsFifteenMinutes(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	failN(t, api, testEmail, 5)

	api.setAt("2026-03-31T16:14:59Z")
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusTooManyRequests {
		t.Errorf("at 14:59 after the failures: status = %d, want 429", rec.Code)
	}
	api.setAt("2026-03-31T16:15:01Z")
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusOK {
		t.Errorf("after 15 minutes: status = %d, want 200: %s", rec.Code, rec.Body)
	}
}

func TestC1_ASuccessfulLoginClearsTheCounter(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	failN(t, api, testEmail, 4)
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusOK {
		t.Fatalf("login after 4 failures: status = %d, want 200", rec.Code)
	}
	failN(t, api, testEmail, 4) // would be the 5th..8th without the reset
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusOK {
		t.Errorf("login after 4 more failures: status = %d, want 200 (the counter was cleared)", rec.Code)
	}
}

func TestC1_OtherEmailsAreNotAffected(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	secondUser(t, api) // helper@example.test
	failN(t, api, testEmail, 5)
	if rec := loginAs(api.h, "helper@example.test", testPassword); rec.Code != http.StatusOK {
		t.Errorf("another account: status = %d, want 200", rec.Code)
	}
}

func TestC1_TheEmailIsComparedWithoutCase(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	failN(t, api, strings.ToUpper(testEmail), 3)
	failN(t, api, testEmail, 2)
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429: upper and lower case are the same e-mail", rec.Code)
	}
}

func TestC1_AnUnknownEmailLooksTheSameAndIsLimitedToo(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	unknown := loginAs(api.h, "nadie@example.test", "wrong-password")
	known := loginAs(api.h, testEmail, "wrong-password")
	if unknown.Code != http.StatusUnauthorized || unknown.Body.String() != known.Body.String() {
		t.Errorf("unknown e-mail answers %d %q, a wrong password %d %q: they must be identical", unknown.Code, unknown.Body, known.Code, known.Body)
	}
	failN(t, api, "nadie@example.test", 4) // 5 in all
	if rec := loginAs(api.h, "nadie@example.test", "wrong-password"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("unknown e-mail after 5 failures: status = %d, want 429 (otherwise the answer tells which e-mails exist)", rec.Code)
	}
}

func TestC1_TheCounterLivesInTheDatabaseNotInTheProcess(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	failN(t, api, testEmail, 5)

	now := mustParse(t, noonMarch31)
	otherInstance := app.NewWithClock(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4}, api.db, nil, func() time.Time { return now })
	if rec := loginAs(otherInstance, testEmail, testPassword); rec.Code != http.StatusTooManyRequests {
		t.Errorf("a second instance of the app: status = %d, want 429", rec.Code)
	}
	var n int
	if err := api.db.QueryRow(context.Background(), `select count(*) from login_attempts`).Scan(&n); err != nil || n != 5 {
		t.Errorf("login_attempts rows = %d (%v), want 5", n, err)
	}
}

// C2: changing the password.
func changePassword(api *client, current, next string) (int, []byte) {
	return api.do("PUT", "/auth/password", map[string]string{"current_password": current, "new_password": next})
}

const newPassword = "una-clave-nueva-123"

func TestC2_ChangingThePassword(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	if status, raw := changePassword(api.client, testPassword, newPassword); status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", status, raw)
	}
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusUnauthorized {
		t.Errorf("the old password still works: status = %d", rec.Code)
	}
	if rec := loginAs(api.h, testEmail, newPassword); rec.Code != http.StatusOK {
		t.Errorf("the new password does not work: status = %d: %s", rec.Code, rec.Body)
	}
	var stored string
	if err := api.db.QueryRow(context.Background(), `select password from users where email = $1`, testEmail).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$2") || strings.Contains(stored, newPassword) {
		t.Errorf("the password is not stored as a bcrypt hash: %.10q", stored)
	}
}

func TestC2_OnlyTheOwnPasswordChanges(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	secondUser(t, api)
	if status, _ := changePassword(api.client, testPassword, newPassword); status != http.StatusNoContent {
		t.Fatalf("status = %d", status)
	}
	if rec := loginAs(api.h, "helper@example.test", testPassword); rec.Code != http.StatusOK {
		t.Errorf("the other account's password changed too: status = %d", rec.Code)
	}
}

func TestC2_RulesOfTheNewPassword(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	cases := map[string]struct{ current, next string }{
		"wrong current password": {"not-my-password", newPassword},
		"empty current password": {"", newPassword},
		"empty new password":     {testPassword, ""},
		"9 characters":           {testPassword, "123456789"},
		"more than 72 bytes":     {testPassword, strings.Repeat("a", 73)},
		"same as the current":    {testPassword, testPassword},
	}
	for label, c := range cases {
		if status, raw := changePassword(api.client, c.current, c.next); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusOK {
		t.Errorf("a refused change altered the password: status = %d", rec.Code)
	}
	// 10 characters and 72 bytes are both fine.
	if status, raw := changePassword(api.client, testPassword, strings.Repeat("b", 72)); status != http.StatusNoContent {
		t.Errorf("72 bytes: status = %d, want 204: %s", status, raw)
	}
}

func TestC2_RequiresLogin(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	anon := &client{t: t, h: api.h}
	if status, _ := changePassword(anon, testPassword, newPassword); status != http.StatusUnauthorized {
		t.Errorf("without a token: status = %d, want 401", status)
	}
}

func TestC2_WrongCurrentPasswordsCountAsFailedAttempts(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	for i := 0; i < 5; i++ {
		if status, _ := changePassword(api.client, "not-my-password", newPassword); status != http.StatusBadRequest {
			t.Fatalf("attempt %d: status = %d, want 400", i+1, status)
		}
	}
	if status, _ := changePassword(api.client, testPassword, newPassword); status != http.StatusTooManyRequests {
		t.Errorf("a 6th change: status = %d, want 429", status)
	}
	if rec := loginAs(api.h, testEmail, testPassword); rec.Code != http.StatusTooManyRequests {
		t.Errorf("login after guessing the current password 5 times: status = %d, want 429", rec.Code)
	}
}
