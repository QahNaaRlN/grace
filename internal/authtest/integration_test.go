//go:build integration

package authtest

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/httpserver"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/jackc/pgx/v5"
)

type provider struct{}

func (provider) AuthorizationURL(f auth.Flow) string {
	return "https://test.invalid/auth?state=" + f.State
}
func (provider) Exchange(_ context.Context, code string, _ auth.Flow) (auth.Identity, error) {
	return auth.Identity{Issuer: "https://test.invalid", Subject: code, Name: code}, nil
}
func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func login(t *testing.T, h *auth.Handler, invitation, subject string) (*http.Cookie, bool) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"invitation": invitation})
	r := httptest.NewRequest("POST", "https://grace.test/api/auth/login", strings.NewReader(string(body)))
	r.Header.Set("Origin", "https://grace.test")
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != 200 {
		t.Fatalf("start login: %d %s", w.Code, w.Body.String())
	}
	var result struct {
		URL string `json:"authorizationUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(result.URL)
	var flowCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "__Host-grace_login" {
			flowCookie = c
		}
	}
	if flowCookie == nil {
		t.Fatal("flow cookie missing")
	}
	r = httptest.NewRequest("GET", "https://grace.test/api/auth/callback?state="+u.Query().Get("state")+"&code="+subject, nil)
	r.AddCookie(flowCookie)
	w = httptest.NewRecorder()
	h.Callback(w, r)
	for _, c := range w.Result().Cookies() {
		if c.Name == "__Host-grace_session" && c.Value != "" {
			return c, true
		}
	}
	return nil, false
}

func TestAuthenticationAndFinancialIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := auth.NewRepository(pool)
	h := auth.NewHandler(store, provider{}, "telegram", "https://grace.test")
	expiredFlow := auth.Flow{State: token(), Browser: token(), Verifier: token(), Nonce: token(), Expires: time.Now().Add(-time.Minute)}
	if err := store.SaveFlow(ctx, expiredFlow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeFlow(ctx, expiredFlow.State, expiredFlow.Browser); !errors.Is(err, auth.ErrDenied) {
		t.Fatal("expired login flow accepted", err)
	}
	expiredInvite, err := store.CreateInvitation(ctx, access.BootstrapOwnerID, "Expired invitation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE auth_invitations SET expires_at=now()-interval '1 second' WHERE user_id=(SELECT id FROM users WHERE display_name='Expired invitation')`); err != nil {
		t.Fatal(err)
	}
	if _, ok := login(t, h, expiredInvite.Token, "expired-invite-subject"); ok {
		t.Fatal("expired invitation accepted")
	}
	ownerInvite, err := store.OwnerInvitation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ownerCookie, ok := login(t, h, ownerInvite.Token, "owner")
	if !ok {
		t.Fatal("owner invitation failed")
	}
	if _, ok = login(t, h, ownerInvite.Token, "intruder"); ok {
		t.Fatal("invitation replay accepted")
	}
	if _, ok = login(t, h, "", "unknown"); ok {
		t.Fatal("open registration accepted")
	}
	ownerCookie2, ok := login(t, h, "", "owner")
	if !ok {
		t.Fatal("existing identity login failed")
	}
	memberInvite, err := store.CreateInvitation(ctx, access.BootstrapOwnerID, "Member")
	if err != nil {
		t.Fatal(err)
	}
	memberCookie, ok := login(t, h, memberInvite.Token, "member")
	if !ok {
		t.Fatal("member invitation failed")
	}
	ownerSession, err := store.FindSession(ctx, ownerCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	memberSession, err := store.FindSession(ctx, memberCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	ownerCtx := access.WithPrincipal(ctx, ownerSession.User.Principal())
	memberCtx := access.WithPrincipal(ctx, memberSession.User.Principal())
	accountRepo := accounts.NewRepository(pool)
	eventRepo := financialevents.NewRepository(pool)
	receiptRepo := receipts.NewRepository(pool)
	obligationRepo := obligations.NewRepository(pool)
	analyticsRepo := analytics.NewRepository(pool)
	if _, err = accountRepo.List(ctx); !errors.Is(err, database.ErrMissingPrincipal) {
		t.Fatal("missing principal allowed", err)
	}
	ownerAccounts, err := accountRepo.List(ownerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerAccounts) != 1 || ownerAccounts[0].BalanceMinor != 12500000 {
		t.Fatalf("migration changed baseline: %#v", ownerAccounts)
	}
	memberAccounts, err := accountRepo.List(memberCtx)
	if err != nil || len(memberAccounts) != 0 {
		t.Fatal("member sees old accounts", err)
	}
	memberAccount, err := accountRepo.Create(memberCtx, accounts.CreateRequest{Name: "Member cash", Type: "cash", Currency: "RUB", OpeningBalanceMinor: 10000})
	if err != nil {
		t.Fatal(err)
	}
	ownerID := ownerAccounts[0].ID
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	command := financialevents.Command{Kind: "income_received", AccountID: ownerID, AmountMinor: 1000, OccurredAt: now, IdempotencyKey: "same-key", RequestHash: strings.Repeat("a", 64)}
	ownerEvent, _, err := eventRepo.Create(ownerCtx, command)
	if err != nil {
		t.Fatal(err)
	}
	command.AccountID = memberAccount.ID
	memberEvent, created, err := eventRepo.Create(memberCtx, command)
	if err != nil || !created || memberEvent.ID == ownerEvent.ID {
		t.Fatal("idempotency not isolated", err)
	}
	command.Kind = "internal_transfer"
	command.AccountID = memberAccount.ID
	command.DestinationAccountID = ownerID
	command.IdempotencyKey = "cross-transfer"
	if _, _, err = eventRepo.Create(memberCtx, command); !errors.Is(err, financialevents.ErrInvalid) {
		t.Fatal("cross-owner transfer accepted", err)
	}
	command.Kind = "purchase"
	command.AccountID = ownerID
	command.DestinationAccountID = ""
	command.IdempotencyKey = "purchase"
	categoryID := "20000000-0000-0000-0000-000000000001"
	command.CategoryID = &categoryID
	purchase, _, err := eventRepo.Create(ownerCtx, command)
	if err != nil {
		t.Fatal(err)
	}
	receiptCommand := receipts.Command{PurchaseEventID: purchase.ID, Seller: "Store", IssuedAt: now, TotalMinor: 1000, IdempotencyKey: "receipt", RequestHash: strings.Repeat("b", 64)}
	if err = json.Unmarshal([]byte(`{"Items":[{"name":"Item","quantityMilli":1000,"unitPriceMinor":1000,"lineTotalMinor":1000}]}`), &receiptCommand); err != nil {
		t.Fatal(err)
	}
	ownerReceipt, _, err := receiptRepo.Create(ownerCtx, receiptCommand)
	if err != nil {
		t.Fatal(err)
	}
	obligationCommand := obligations.CreateCommand{Name: "Owner loan", Kind: "loan", Currency: "RUB", TotalMinor: 500, IdempotencyKey: "loan", RequestHash: strings.Repeat("c", 64), AsOf: "2026-10-05"}
	if err = json.Unmarshal([]byte(`{"Payments":[{"dueOn":"2026-10-10","amountMinor":500}]}`), &obligationCommand); err != nil {
		t.Fatal(err)
	}
	ownerObligation, _, err := obligationRepo.Create(ownerCtx, obligationCommand)
	if err != nil {
		t.Fatal(err)
	}
	if rows, err := receiptRepo.List(memberCtx, 100); err != nil || len(rows) != 0 {
		t.Fatal("member sees receipt", err)
	}
	if rows, err := obligationRepo.List(memberCtx, "2026-10-05", 100); err != nil || len(rows) != 0 {
		t.Fatal("member sees obligation", err)
	}
	if _, err := receiptRepo.Get(memberCtx, ownerReceipt.ID); err == nil {
		t.Fatal("member fetched owner's receipt by ID")
	}
	if _, err := obligationRepo.Get(memberCtx, ownerObligation.ID, "2026-10-05"); err == nil {
		t.Fatal("member fetched owner's obligation by ID")
	}
	if _, _, err := eventRepo.Create(memberCtx, financialevents.Command{Kind: "refund", OriginalEventID: purchase.ID, AmountMinor: 1, OccurredAt: now, IdempotencyKey: "foreign-refund", RequestHash: strings.Repeat("f", 64)}); err == nil {
		t.Fatal("member refunded owner's event")
	}
	if _, _, err := obligationRepo.CreateSettlement(memberCtx, obligations.SettlementCommand{ObligationID: ownerObligation.ID, PaymentID: ownerObligation.Payments[0].ID, FinancialEventID: memberEvent.ID, IdempotencyKey: "foreign-payment", RequestHash: strings.Repeat("e", 64)}); err == nil {
		t.Fatal("member settled owner's scheduled payment")
	}
	from, to := now.Add(-time.Hour), now.Add(time.Hour)
	summary, err := analyticsRepo.PeriodSummary(memberCtx, from, to)
	if err != nil || len(summary.Currencies) != 1 || summary.Currencies[0].IncomeMinor != 1000 || summary.Currencies[0].PurchasesMinor != 0 {
		t.Fatalf("analytics leaked: %#v %v", summary, err)
	}
	cash, err := analyticsRepo.CashPosition(memberCtx, "2026-10-05", "2026-10-20")
	if err != nil || len(cash.Currencies) != 1 || cash.Currencies[0].UpcomingObligationsMinor != 0 {
		t.Fatal("cash position leaked", err)
	}
	executor := assistant.NewExecutor(accountRepo, analyticsRepo, obligationRepo, receiptRepo)
	for _, call := range []struct{ name, args string }{{"get_accounts", `{}`}, {"get_period_summary", `{"from":"2026-10-05T11:00:00Z","to":"2026-10-05T13:00:00Z"}`}, {"get_cash_position", `{"asOf":"2026-10-05","through":"2026-10-20"}`}, {"get_obligations", `{"asOf":"2026-10-05"}`}, {"get_receipts", `{}`}, {"get_period_since_income", `{"asOf":"2026-10-05T13:00:00Z"}`}} {
		result, err := executor.Call(memberCtx, call.name, json.RawMessage(call.args))
		if err != nil {
			t.Fatal(call.name, err)
		}
		data, _ := json.Marshal(result)
		if strings.Contains(string(data), ownerID) || strings.Contains(string(data), purchase.ID) || strings.Contains(string(data), "Owner loan") {
			t.Fatalf("tool leaked owner: %s", call.name)
		}
	}
	// Even a direct INSERT under the financial role cannot link another user's account.
	tx, err := database.ForUsers(pool).BeginTx(memberCtx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(memberCtx, `INSERT INTO postings(financial_event_id,account_id,direction,amount_minor,currency) VALUES($1,$2,'credit',1,'RUB')`, memberEvent.ID, ownerID)
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("cross-owner FK missing")
	}
	// Reuse of a single pooled connection must not carry ownership to a new caller.
	for i := 0; i < 10; i++ {
		a, err := accountRepo.List(ownerCtx)
		if err != nil || len(a) != 1 {
			t.Fatal("pooled owner scope leaked", err)
		}
		a, err = accountRepo.List(memberCtx)
		if err != nil || len(a) != 1 || a[0].ID != memberAccount.ID {
			t.Fatal("pooled member scope leaked", err)
		}
	}
	server := httpserver.New(config.Config{AuthMode: "telegram", PublicURL: "https://grace.test", TelegramClientID: "test", TelegramClientSecret: "test"}, pool)
	request := func(method, path, body string, cookie *http.Cookie, csrf bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://grace.test"+path, strings.NewReader(body))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		r.Header.Set("Origin", "https://grace.test")
		if csrf && cookie != nil {
			me := httptest.NewRecorder()
			q := httptest.NewRequest("GET", "https://grace.test/api/auth/me", nil)
			q.AddCookie(cookie)
			server.Handler.ServeHTTP(me, q)
			var session struct {
				CSRF string `json:"csrfToken"`
			}
			_ = json.Unmarshal(me.Body.Bytes(), &session)
			r.Header.Set("X-CSRF-Token", session.CSRF)
		}
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/v1/accounts", "", nil, false); w.Code != 401 {
		t.Fatal("anonymous API accessible", w.Code)
	}
	if w := request("POST", "/api/auth/logout", "", memberCookie, false); w.Code != 403 {
		t.Fatal("CSRF bypass", w.Code)
	}
	if w := request("POST", "/api/auth/invitations", `{"name":"Forbidden"}`, memberCookie, true); w.Code != 403 {
		t.Fatal("member invited user", w.Code)
	}
	if w := request("POST", "/api/auth/logout", "", ownerCookie, true); w.Code != 204 {
		t.Fatal("logout failed", w.Code)
	}
	if _, err = store.FindSession(ctx, ownerCookie.Value); !errors.Is(err, auth.ErrDenied) {
		t.Fatal("logout session alive")
	}
	if _, err = store.FindSession(ctx, ownerCookie2.Value); err != nil {
		t.Fatal("logout revoked unrelated session")
	}
	if w := request("POST", "/api/auth/revoke-sessions", "", ownerCookie2, true); w.Code != 204 {
		t.Fatal("revoke failed", w.Code)
	}
	ownerCookie, ok = login(t, h, "", "owner")
	if !ok {
		t.Fatal("owner login after revoke failed")
	}
	if w := request("POST", "/api/auth/users/"+memberSession.User.ID+"/disable", "", ownerCookie, true); w.Code != 204 {
		t.Fatal("disable failed", w.Code, w.Body.String())
	}
	if _, err = store.FindSession(ctx, memberCookie.Value); !errors.Is(err, auth.ErrDenied) {
		t.Fatal("disabled session alive")
	}
	if _, ok = login(t, h, "", "member"); ok {
		t.Fatal("disabled user authenticated")
	}
	if _, err = pool.Exec(ctx, `UPDATE auth_sessions SET expires_at=now()-interval '1 second' WHERE user_id=$1`, access.BootstrapOwnerID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/api/v1/accounts", "", ownerCookie, false); w.Code != 401 {
		t.Fatal("expired session alive")
	}
}
