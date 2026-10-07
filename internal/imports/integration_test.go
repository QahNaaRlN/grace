//go:build integration

package imports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/importreview"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const accountID = "10000000-0000-0000-0000-000000000001"
const categoryID = "20000000-0000-0000-0000-000000000001"

func cell(s string) importreview.Cell {
	return importreview.Cell{UserEnteredValue: &importreview.Value{StringValue: &s}}
}
func snapshot(source, kind string, overlap bool) []byte {
	values := make([]importreview.Cell, 15)
	values[0] = cell("2026-10-06")
	values[1] = cell(kind)
	values[2] = cell("Synthetic purchase " + source)
	values[3] = cell("Food")
	values[5] = cell("10,00")
	values[6] = cell("Собственные средства")
	values[7] = cell("Test card")
	values[10] = cell("Test merchant")
	values[14] = cell(source + "-row-1")
	ranges := []importreview.Range{{StartRow: 5, StartColumn: 1, Rows: []importreview.Row{{Values: values}}}}
	if overlap {
		copyValues := append([]importreview.Cell(nil), values...)
		copyValues[14] = cell("additional-row")
		ranges = append(ranges, importreview.Range{StartRow: 305, StartColumn: 1, Rows: []importreview.Row{{Values: copyValues}}})
	}
	s := importreview.Snapshot{SchemaVersion: importreview.SnapshotVersion, SourceID: source, CapturedAt: "2026-10-06T00:00:00+05:00", Locale: "ru_RU", DeclaredTimeZone: "Asia/Yekaterinburg", ConfirmedTimeZone: "Asia/Yekaterinburg", Sheets: []importreview.Sheet{{SheetID: 1, Title: "Операции", RowCount: 400, ColumnCount: 15, Ranges: ranges}}}
	raw, _ := json.Marshal(s)
	return raw
}
func createDecision(kind string) Decision {
	category := categoryID
	return Decision{Action: "create", Reason: "Verified synthetic source", Fields: &Fields{Kind: kind, Currency: "RUB", AmountMinor: 1000, AccountID: accountID, CategoryID: &category, OccurredAt: "2026-10-06T12:00:00+05:00", TimeReason: "Synthetic explicit time", Merchant: "Test merchant", Description: "Synthetic purchase"}}
}
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if asFault(err).Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}
func setup(t *testing.T) (context.Context, *pgxpool.Pool, *Repository) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := access.WithPrincipal(context.Background(), access.Principal{ID: access.BootstrapOwnerID, Role: "admin"})
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return ctx, p, NewRepository(p)
}
func uploadOne(t *testing.T, ctx context.Context, r *Repository, source, kind string, overlap bool) (Batch, []Record) {
	t.Helper()
	b, err := r.Upload(ctx, snapshot(source, kind, overlap), source)
	if err != nil {
		t.Fatal(err)
	}
	page, err := r.Records(ctx, b.ID, "", "", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	return b, page.Items
}
func TestPersistentImportLifecycle(t *testing.T) {
	ctx, p, r := setup(t)
	count := func() int {
		var n int
		if err := p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	b, rows := uploadOne(t, ctx, r, "lifecycle", "Расход", true)
	if len(rows) != 2 {
		t.Fatal("lost source rows")
	}
	replay, err := r.Upload(ctx, snapshot("lifecycle", "Расход", true), "upload-retry")
	if err != nil || replay.ID != b.ID {
		t.Fatal("duplicate batch", err)
	}
	base, extra := rows[0], rows[1]
	if base.Source.Row != 5 {
		base, extra = extra, base
	}
	b, err = r.Edit(ctx, b.ID, base.ID, "edit-base", Edit{b.Revision, createDecision("purchase")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Edit(ctx, b.ID, extra.ID, "stale", Edit{b.Revision - 1, Decision{Action: "exclude", Reason: "review"}})
	requireCode(t, err, "revision_conflict")
	blocked, err := r.Preview(ctx, b.ID, "blocked", Selection{b.Revision, []string{base.ID}})
	if err != nil || len(blocked.Blockers) == 0 {
		t.Fatal("unreviewed candidate published", err)
	}
	b, err = r.Edit(ctx, b.ID, extra.ID, "edit-extra", Edit{b.Revision, Decision{Action: "duplicate_of", CanonicalRecordID: base.ID, Reason: "Same source fact"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Edit(ctx, b.ID, base.ID, "cycle", Edit{b.Revision, Decision{Action: "duplicate_of", CanonicalRecordID: extra.ID, Reason: "cycle"}})
	requireCode(t, err, "invalid_duplicate")
	detail, err := NewRepository(p).Record(ctx, b.ID, base.ID)
	if err != nil || len(detail.History) != 1 || detail.Source.Raw["F"].UserEnteredValue == nil {
		t.Fatal("reload lost evidence", err)
	}
	preview, err := r.Preview(ctx, b.ID, "preview", Selection{b.Revision, []string{base.ID, extra.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Blockers) > 0 {
		t.Fatalf("blockers %v", preview.Blockers)
	}
	if preview.NewEvents != 1 || preview.Links != 1 || len(preview.Changes) != 1 || preview.Changes[0].DeltaMinor != -1000 {
		t.Fatal("incorrect preview")
	}
	if count() != before {
		t.Fatal("review changed ledger")
	}
	first, err := r.Confirm(ctx, b.ID, "confirm", Confirmation{preview.ID, preview.Hash})
	if err != nil {
		t.Fatal(err)
	}
	r.now = func() time.Time { return time.Now().Add(time.Hour) }
	again, err := r.Confirm(ctx, b.ID, "confirm", Confirmation{preview.ID, preview.Hash})
	r.now = time.Now
	if err != nil || again.ID != first.ID || count() != before+1 {
		t.Fatal("retry duplicated ledger", err)
	}
	changed := Confirmation{preview.ID, strings.Repeat("0", 64)}
	_, err = r.Confirm(ctx, b.ID, "confirm", changed)
	requireCode(t, err, "idempotency_conflict")
	_, err = r.Confirm(ctx, b.ID, "new-key", Confirmation{preview.ID, preview.Hash})
	requireCode(t, err, "revision_conflict")
	_, err = r.Edit(ctx, b.ID, base.ID, "edit-published", Edit{b.Revision + 1, Decision{Action: "unresolved"}})
	requireCode(t, err, "already_published")
	saved, err := r.Receipt(ctx, b.ID, first.ID)
	if err != nil || len(saved.Outcomes) != 2 || *saved.Outcomes[0].EventID != *saved.Outcomes[1].EventID {
		t.Fatal("provenance missing", err)
	}
	done, err := r.Batch(ctx, b.ID)
	if err != nil || done.Status != "completed" {
		t.Fatal("batch not completed", err)
	}
	var eventSource string
	if err = p.QueryRow(ctx, "SELECT source::text FROM financial_events WHERE id=$1", *first.Outcomes[0].EventID).Scan(&eventSource); err != nil || eventSource != "import" {
		t.Fatal("wrong source", err)
	}
	// A new hash retains source matching; owner explicitly links rather than importing twice.
	raw := append(snapshot("lifecycle", "Расход", false), ' ')
	nb, err := r.Upload(ctx, raw, "new-snapshot")
	if err != nil {
		t.Fatal(err)
	}
	nr, _ := r.Records(ctx, nb.ID, "", "", "", 100)
	nd, err := r.Record(ctx, nb.ID, nr.Items[0].ID)
	if err != nil || len(nd.Candidates) == 0 {
		t.Fatal("published source not detected", err)
	}
	nb, err = r.Edit(ctx, nb.ID, nd.ID, "link", Edit{nb.Revision, Decision{Action: "link_existing", EventID: *first.Outcomes[0].EventID, Reason: "Same confirmed fact"}})
	if err != nil {
		t.Fatal(err)
	}
	np, err := r.Preview(ctx, nb.ID, "link-preview", Selection{nb.Revision, []string{nd.ID}})
	if err != nil || len(np.Blockers) > 0 {
		t.Fatal("link preview", err, np.Blockers)
	}
	_, err = r.Confirm(ctx, nb.ID, "link-confirm", Confirmation{np.ID, np.Hash})
	if err != nil || count() != before+1 {
		t.Fatal("link changed balance", err)
	}
}
func TestIsolationAndHTTP(t *testing.T) {
	ctx, p, r := setup(t)
	b, rows := uploadOne(t, ctx, r, "isolation", "Расход", false)
	var otherID string
	if err := p.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('Synthetic other owner','admin') RETURNING id::text").Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	other := access.WithPrincipal(context.Background(), access.Principal{ID: otherID, Role: "admin"})
	_, err := r.Batch(other, b.ID)
	requireCode(t, err, "not_found")
	page, err := r.Batches(other, "", 100)
	if err != nil || len(page.Items) != 0 {
		t.Fatal("RLS leaked batch", err)
	}
	_, err = r.Record(other, b.ID, rows[0].ID)
	requireCode(t, err, "not_found")
	tx, err := database.ForUsers(p).BeginTx(other, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"import_batches", "import_records", "import_decisions", "import_previews", "import_confirmations", "import_provenance", "import_requests"} {
		var n int
		if err = tx.QueryRow(other, "SELECT count(*) FROM "+table).Scan(&n); err != nil || n != 0 {
			t.Fatal("tenant rows leaked", table, n, err)
		}
		var forced bool
		if err = tx.QueryRow(other, "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid=$1::regclass", table).Scan(&forced); err != nil || !forced {
			t.Fatal("RLS is not forced", table, err)
		}
	}
	_, err = tx.Exec(other, "INSERT INTO import_records(batch_id,snapshot_record_id,domain,source,raw_hash) VALUES($1,'cross-owner','transaction','{}',$2)", b.ID, strings.Repeat("0", 64))
	tx.Rollback(other)
	if err == nil {
		t.Fatal("cross-owner FK accepted")
	}
	handler := chi.NewRouter()
	NewHandler(r).Routes(handler)
	req := httptest.NewRequest(http.MethodGet, "/batches/"+b.ID, nil).WithContext(other)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatal("foreign HTTP batch leaked", w.Code)
	}
	category := categoryID
	event, _, err := financialevents.NewRepository(p).Create(ctx, financialevents.Command{Kind: "purchase", IdempotencyKey: "isolation-event", RequestHash: "isolation-event", AccountID: accountID, CategoryID: &category, AmountMinor: 33331, Description: "Synthetic isolation proof", OccurredAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	otherBatch, otherRows := uploadOne(t, other, r, "other-link", "Расход", false)
	otherBatch, err = r.Edit(other, otherBatch.ID, otherRows[0].ID, "foreign-link", Edit{otherBatch.Revision, Decision{Action: "link_existing", EventID: event.ID, Reason: "Synthetic forbidden cross-owner reference"}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := r.Preview(other, otherBatch.ID, "foreign-preview", Selection{otherBatch.Revision, []string{otherRows[0].ID}})
	if err != nil || !strings.Contains(strings.Join(preview.Blockers, ","), "event_not_found") {
		t.Fatal("foreign event linked", err, preview.Blockers)
	}
	_, err = r.Confirm(other, otherBatch.ID, "foreign-confirm", Confirmation{preview.ID, preview.Hash})
	requireCode(t, err, "preview_blocked")
}
func TestExpiredAndChangedPreview(t *testing.T) {
	ctx, p, r := setup(t)
	b, rows := uploadOne(t, ctx, r, "expiry", "Расход", false)
	d := createDecision("purchase")
	d.Fields.AmountMinor = 2000
	b, err := r.Edit(ctx, b.ID, rows[0].ID, "expiry-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := r.Preview(ctx, b.ID, "expiry-preview", Selection{b.Revision, []string{rows[0].ID}})
	if err != nil {
		t.Fatal(err)
	}
	r.now = func() time.Time { return time.Now().Add(time.Hour) }
	_, err = r.Confirm(ctx, b.ID, "expired", Confirmation{preview.ID, preview.Hash})
	requireCode(t, err, "preview_expired")
	r.now = time.Now
	// A manual writer creates a fresh matching candidate after preview.
	cmd, _ := command(d.Fields, "new-manual-candidate")
	if _, _, err = financialevents.NewRepository(p).Create(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	_, err = r.Confirm(ctx, b.ID, "changed", Confirmation{preview.ID, preview.Hash})
	requireCode(t, err, "preview_changed")
}
func TestConcurrentConfirmation(t *testing.T) {
	ctx, _, r := setup(t)
	b, rows := uploadOne(t, ctx, r, "concurrent", "Доход", false)
	d := createDecision("income_received")
	d.Fields.CategoryID = nil
	d.Fields.AmountMinor = 3456
	b, err := r.Edit(ctx, b.ID, rows[0].ID, "concurrent-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := r.Preview(ctx, b.ID, "concurrent-preview", Selection{b.Revision, []string{rows[0].ID}})
	if err != nil || len(preview.Blockers) > 0 {
		t.Fatal(err, preview.Blockers)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"concurrent-first", "concurrent-second"} {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, e := r.Confirm(ctx, b.ID, k, Confirmation{preview.ID, preview.Hash})
			results <- e
		}(key)
	}
	wg.Wait()
	close(results)
	success := 0
	conflicts := 0
	for e := range results {
		if e == nil {
			success++
		} else if asFault(e).Code == "revision_conflict" {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("concurrent duplicate", success, conflicts)
	}
}
func TestBlockedDateAndFinancialGroup(t *testing.T) {
	ctx, _, r := setup(t)
	b, rows := uploadOne(t, ctx, r, "invalid-time", "Расход", false)
	d := createDecision("purchase")
	d.Fields.TimeReason = ""
	d.Fields.AmountMinor = 7891
	b, err := r.Edit(ctx, b.ID, rows[0].ID, "date-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := r.Preview(ctx, b.ID, "date-preview", Selection{b.Revision, []string{rows[0].ID}})
	if err != nil || len(preview.Blockers) == 0 {
		t.Fatal("invented time accepted", err)
	}
	_, err = r.Confirm(ctx, b.ID, "blocked-confirm", Confirmation{preview.ID, preview.Hash})
	requireCode(t, err, "preview_blocked")
	if _, err = r.Upload(ctx, []byte(`{"schemaVersion":"unknown"}`), "bad"); asFault(err).Code != "invalid_snapshot" {
		t.Fatal("invalid snapshot accepted")
	}
	if _, err = r.Preview(ctx, b.ID, "bad-selection", Selection{b.Revision, []string{rows[0].ID, rows[0].ID}}); asFault(err).Code != "invalid_selection" {
		t.Fatal("duplicate selection")
	}
}

func TestFiveKindsAndAtomicRollback(t *testing.T) {
	ctx, p, r := setup(t)
	var destination string
	if err := p.QueryRow(ctx, "INSERT INTO accounts(name,type,currency,opening_balance_minor,owner_id) VALUES('Synthetic destination','debit','RUB',0,$1) RETURNING id::text", access.BootstrapOwnerID).Scan(&destination); err != nil {
		t.Fatal(err)
	}
	var purchaseID string
	for i, kind := range []string{"purchase", "income_received", "debt_payment", "internal_transfer", "refund"} {
		sourceKind := map[string]string{"purchase": "Расход", "income_received": "Доход", "debt_payment": "Погашение", "internal_transfer": "Перевод между своими счетами", "refund": "Возврат"}[kind]
		b, rows := uploadOne(t, ctx, r, "five-kinds-"+kind, sourceKind, false)
		d := createDecision(kind)
		d.Fields.AmountMinor = int64(19000 + i*1000)
		if kind == "income_received" || kind == "internal_transfer" {
			d.Fields.CategoryID = nil
		}
		if kind == "internal_transfer" {
			d.Fields.DestinationAccountID = destination
		}
		if kind == "refund" {
			d.Fields.OriginalEventID = purchaseID
			d.Fields.AmountMinor = 1000
			d.Fields.CategoryID = nil
		}
		b, err := r.Edit(ctx, b.ID, rows[0].ID, "five-edit-"+kind, Edit{b.Revision, d})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := r.Preview(ctx, b.ID, "five-preview-"+kind, Selection{b.Revision, []string{rows[0].ID}})
		if err != nil || len(preview.Blockers) > 0 {
			t.Fatal(kind, err, preview.Blockers)
		}
		receipt, err := r.Confirm(ctx, b.ID, "five-confirm-"+kind, Confirmation{preview.ID, preview.Hash})
		if err != nil {
			t.Fatal(kind, err)
		}
		if kind == "purchase" {
			purchaseID = *receipt.Outcomes[0].EventID
		}
		if kind == "internal_transfer" {
			if len(preview.Changes) != 2 || preview.Changes[0].DeltaMinor+preview.Changes[1].DeltaMinor != 0 {
				t.Fatal("unbalanced transfer")
			}
		}
	}
	// Inject a deterministic failure in the last financial event INSERT after a valid preview.
	b, rows := uploadOne(t, ctx, r, "atomic-last-failure", "Расход", true)
	for i, rec := range rows {
		d := createDecision("purchase")
		d.Fields.AmountMinor = int64(41001 + i)
		d.Fields.Merchant = "atomic-ok"
		if i == len(rows)-1 {
			d.Fields.Merchant = "atomic-fail"
		}
		for _, other := range rows {
			if other.ID != rec.ID {
				d.Resolutions = append(d.Resolutions, Resolution{"record:" + other.ID, "Different synthetic purchases"})
			}
		}
		var err error
		b, err = r.Edit(ctx, b.ID, rec.ID, "atomic-edit-"+rec.ID, Edit{b.Revision, d})
		if err != nil {
			t.Fatal(err)
		}
	}
	preview, err := r.Preview(ctx, b.ID, "atomic-preview", Selection{b.Revision, []string{rows[0].ID, rows[1].ID}})
	if err != nil || len(preview.Blockers) > 0 {
		t.Fatal(err, preview.Blockers)
	}
	_, err = p.Exec(ctx, `CREATE FUNCTION test_import_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.merchant='atomic-fail' THEN RAISE EXCEPTION 'Synthetic failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER test_import_failure BEFORE INSERT ON financial_events FOR EACH ROW EXECUTE FUNCTION test_import_failure()`)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Exec(ctx, "DROP TRIGGER test_import_failure ON financial_events; DROP FUNCTION test_import_failure()")
	var before int
	if err = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&before); err != nil {
		t.Fatal(err)
	}
	_, err = r.Confirm(ctx, b.ID, "atomic-confirm", Confirmation{preview.ID, preview.Hash})
	if err == nil {
		t.Fatal("synthetic insert failure ignored")
	}
	var after, n int
	p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&after)
	p.QueryRow(ctx, "SELECT count(*) FROM import_confirmations WHERE batch_id=$1", b.ID).Scan(&n)
	if before != after || n != 0 {
		t.Fatal("partial publication", before, after, n)
	}
	for _, rec := range rows {
		saved, err := r.Record(ctx, b.ID, rec.ID)
		if err != nil || saved.Published {
			t.Fatal("partial draft state", err)
		}
	}
}

func TestUnsupportedSemanticsAndSourceFilters(t *testing.T) {
	ctx, _, r := setup(t)
	for _, tc := range []struct{ name, kind, funding, blocker string }{
		{"hold", "HOLD", "Собственные средства", "unsupported_semantics"},
		{"borrowed", "Доход", "Кредитные средства", "borrowed_is_not_income"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := snapshot("blocked-"+tc.name, tc.kind, false)
			raw = []byte(strings.ReplaceAll(string(raw), "Собственные средства", tc.funding))
			b, err := r.Upload(ctx, raw, "blocked-upload-"+tc.name)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := r.Records(ctx, b.ID, "", "transaction", "needs_review", 100)
			if err != nil || len(rows.Items) != 1 {
				t.Fatal("review filter lost source", err)
			}
			d := createDecision("income_received")
			d.Fields.CategoryID = nil
			b, err = r.Edit(ctx, b.ID, rows.Items[0].ID, "blocked-edit-"+tc.name, Edit{b.Revision, d})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := r.Preview(ctx, b.ID, "blocked-preview-"+tc.name, Selection{b.Revision, []string{rows.Items[0].ID}})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(preview.Blockers, ","), tc.blocker) {
				t.Fatal("unsafe semantics accepted", preview.Blockers)
			}
			_, err = r.Confirm(ctx, b.ID, "blocked-confirm-"+tc.name, Confirmation{preview.ID, preview.Hash})
			requireCode(t, err, "preview_blocked")
		})
	}
	fixture, err := os.ReadFile("../../testdata/import-review/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Upload(ctx, fixture, "unsupported-fixture")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := r.Records(ctx, b.ID, "", "unsupported", "", 100)
	if err != nil || len(rows.Items) != 9 {
		t.Fatal("unsupported filter", len(rows.Items), err)
	}
	_, err = r.Edit(ctx, b.ID, rows.Items[0].ID, "unsupported-edit", Edit{b.Revision, createDecision("purchase")})
	requireCode(t, err, "unsupported_domain")
}

func TestImportSessionCSRFAndOrigin(t *testing.T) {
	ctx, p, r := setup(t)
	store := auth.NewRepository(p)
	invite, err := store.OwnerInvitation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 43)
	csrf := hashBytes([]byte("grace-csrf:" + token))
	_, err = store.Login(ctx, auth.Identity{Issuer: "https://synthetic.invalid", Subject: "test-owner", Name: "Synthetic"}, hashBytes([]byte(invite.Token)), token, csrf)
	if err != nil {
		t.Fatal(err)
	}
	authHandler := auth.NewHandler(store, nil, "telegram", "https://grace.test")
	router := chi.NewRouter()
	router.Use(authHandler.Require)
	NewHandler(r).Routes(router)
	for _, tc := range []struct {
		cookie, origin, csrf bool
		status               int
	}{{false, true, true, 401}, {true, false, true, 403}, {true, true, false, 403}, {true, true, true, 201}} {
		req := httptest.NewRequest(http.MethodPost, "https://grace.test/batches", strings.NewReader(string(snapshot("http-upload", "Расход", false))))
		req.Header.Set("Idempotency-Key", "http-upload")
		if tc.cookie {
			req.AddCookie(&http.Cookie{Name: "__Host-grace_session", Value: token})
		}
		if tc.origin {
			req.Header.Set("Origin", "https://grace.test")
		}
		if tc.csrf {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatal("HTTP gate", tc, w.Code, w.Body.String())
		}
	}
	// The CSV endpoint uses the same authenticated mutation boundary.
	for _, tc := range []struct {
		cookie, origin, csrf bool
		status               int
	}{{false, true, true, 401}, {true, false, true, 403}, {true, true, false, 403}, {true, true, true, 201}} {
		req := httptest.NewRequest(http.MethodPost, "https://grace.test/csv?sourceId=http-csv", strings.NewReader(string(csvFixture(',', 1))))
		req.Header.Set("Idempotency-Key", "http-csv")
		if tc.cookie {
			req.AddCookie(&http.Cookie{Name: "__Host-grace_session", Value: token})
		}
		if tc.origin {
			req.Header.Set("Origin", "https://grace.test")
		}
		if tc.csrf {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatal("CSV gate", tc, w.Code, w.Body.String())
		}
	}
}
