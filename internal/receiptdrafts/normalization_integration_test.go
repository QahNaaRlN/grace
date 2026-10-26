//go:build integration

package receiptdrafts

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestS56ReceiptNormalizationOriginalRetentionAndFreeConfirmation(t *testing.T) {
	if os.Getenv("TEST_S56_DOCUMENTS") != "true" {
		t.Skip("disposable database required")
	}
	bg := context.Background()
	pool, err := pgxpool.New(bg, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, other, account string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic R receipt owner','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	foreign := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	tenant := database.ForUsers(pool)
	if err = tenant.QueryRow(ctx, "INSERT INTO accounts(name,type,currency) VALUES('Synthetic R account','cash','RUB') RETURNING id::text").Scan(&account); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(pool)
	d, created, err := repo.Create(ctx, strings.Repeat("a", 48), Hash("synthetic R receipt"))
	if err != nil || !created {
		t.Fatal(err)
	}
	d, started, err := repo.BeginExtraction(ctx, d.ID, "offline-original")
	if err != nil || !started {
		t.Fatal(err)
	}
	raw := fixtureFields("")
	raw.IssuedAt = ptr("07:08:09")
	raw.Items = append(raw.Items, Item{Name: ptr("Complimentary"), Quantity: ptr("2"), UnitPrice: ptr("0.00"), LineTotal: ptr("0.00")})
	out := Extracted{Fields: raw, SourceText: "Synthetic original\n07:08:09\nComplimentary 2 x 0.00 = 0.00", Model: "fixture", Provider: "fixture", CostMicros: "0"}
	original := string(marshal(out))
	d, err = repo.FinishExtraction(ctx, d.ID, "offline-original", &out)
	if err != nil || d.Fields.IssuedAt != nil || len(d.Fields.Items) != 2 || string(marshal(d.Extractions[0])) != original || string(marshal(out)) != original {
		t.Fatal("new candidate or original corrupted", err)
	}
	var stored Fields
	var storedJSON []byte
	if err = tenant.QueryRow(ctx, "SELECT fields FROM receipt_drafts WHERE id::text=$1", d.ID).Scan(&storedJSON); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(storedJSON, &stored); err != nil || stored.IssuedAt != nil {
		t.Fatal("new candidate not stored normalized", err)
	}
	// Model the persisted pre-R state, without rewriting its immutable extraction.
	if _, err = pool.Exec(bg, "UPDATE receipt_drafts SET fields=$2 WHERE id::text=$1", d.ID, marshal(raw)); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		t.Helper()
		var s string
		if err = tenant.QueryRow(ctx, "SELECT jsonb_build_array(fields,extractions,revision,status,xmin::text)::text FROM receipt_drafts WHERE id::text=$1", d.ID).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := snapshot()
	handler := NewHandler(pool, nil, nil)
	router := chi.NewRouter()
	handler.Routes(router)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/"+d.ID, nil).WithContext(ctx))
	if w.Code != 200 {
		t.Fatal("GET failed", w.Code)
	}
	var projected Draft
	if err = json.Unmarshal(w.Body.Bytes(), &projected); err != nil || projected.Fields.IssuedAt != nil || projected.Revision != d.Revision || string(marshal(projected.Extractions[0])) != original {
		t.Fatal("legacy candidate not projected safely", err)
	}
	if snapshot() != before {
		t.Fatal("GET wrote SQL/revision")
	}
	if _, err = repo.Get(foreign, d.ID); err == nil {
		t.Fatal("foreign draft exposed")
	}
	if _, err = repo.Confirm(ctx, d.ID, "missing", Confirmation{}); err == nil {
		t.Fatal("missing confirmation accepted")
	}
	// An owner-saved clock is not silently normalized.
	manual, err := repo.Edit(ctx, d.ID, Edit{Revision: d.Revision, Fields: raw})
	if err != nil {
		t.Fatal(err)
	}
	manual, err = repo.Get(ctx, d.ID)
	if err != nil || manual.Fields.IssuedAt == nil {
		t.Fatal("manual edit projected", err)
	}
	if _, err = repo.Preview(ctx, d.ID, manual.Revision); err == nil {
		t.Fatal("invalid manual clock accepted")
	}
	fields := projected.Fields
	fields.AccountID = account
	fields.Action = "create"
	fields.CategoryID = ptr("20000000-0000-0000-0000-000000000001")
	d, err = repo.Edit(ctx, d.ID, Edit{Revision: manual.Revision, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if string(marshal(d.Extractions[0])) != original {
		t.Fatal("edit changed original")
	}
	preview, err := repo.Preview(ctx, d.ID, d.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Confirm(foreign, d.ID, "foreign", Confirmation{preview.ID, preview.Hash}); err == nil {
		t.Fatal("foreign confirmation accepted")
	}
	d, err = repo.Edit(ctx, d.ID, Edit{Revision: d.Revision, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Confirm(ctx, d.ID, "stale", Confirmation{preview.ID, preview.Hash}); err == nil {
		t.Fatal("stale confirmation accepted")
	}
	preview, err = repo.Preview(ctx, d.ID, d.Revision)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&count); err != nil || count != 0 {
		t.Fatal("preview wrote finance", count, err)
	}
	confirmed, err := repo.Confirm(ctx, d.ID, "confirm-r", Confirmation{preview.ID, preview.Hash})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := repo.Confirm(ctx, d.ID, "confirm-r", Confirmation{preview.ID, preview.Hash})
	if err != nil || confirmed.ReceiptID == nil || replay.ReceiptID == nil || *confirmed.ReceiptID != *replay.ReceiptID {
		t.Fatal("confirmation replay", err)
	}
	var debit int64
	if err = tenant.QueryRow(ctx, "SELECT count(*),COALESCE(sum(amount_minor),0) FROM postings").Scan(&count, &debit); err != nil || count != 1 || debit != 18000 {
		t.Fatal("wrong purchase postings", count, debit, err)
	}
	var freeQuantity int64
	if err = tenant.QueryRow(ctx, "SELECT quantity_milli FROM receipt_items WHERE receipt_id::text=$1 AND unit_price_minor=0 AND line_total_minor=0", *confirmed.ReceiptID).Scan(&freeQuantity); err != nil || freeQuantity != 2000 {
		t.Fatal("free row lost", err)
	}
	final, err := repo.Get(ctx, d.ID)
	if err != nil || string(marshal(final.Extractions[0])) != original {
		t.Fatal("confirmation changed original", err)
	}
	down, err := os.ReadFile("../../migrations/000032_receipt_free_items.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(bg, string(down)); err == nil || !strings.Contains(err.Error(), "Retain free receipt items") {
		t.Fatal("unsafe migration downgrade", err)
	}
	if err = tenant.QueryRow(ctx, "SELECT quantity_milli FROM receipt_items WHERE receipt_id::text=$1 AND unit_price_minor=0 AND line_total_minor=0", *confirmed.ReceiptID).Scan(&freeQuantity); err != nil || freeQuantity != 2000 {
		t.Fatal("downgrade changed retained data", err)
	}
}
