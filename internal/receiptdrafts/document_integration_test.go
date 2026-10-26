//go:build integration

package receiptdrafts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type s56ReceiptGateway struct {
	posts          int
	unknown        bool
	receiptCeiling bool
	wire           string
}

func (g *s56ReceiptGateway) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "openrouter.ai" {
		return nil, fmt.Errorf("unexpected host")
	}
	params := []string{"structured_outputs", "response_format", "max_tokens"}
	endpoint := map[string]any{"model_id": "fixture/receipt", "tag": "fixture/eu", "provider_name": "Fixture", "status": 0, "context_length": 100000, "max_completion_tokens": 16000, "supported_parameters": params, "pricing": map[string]any{"prompt": "0.0000001", "completion": "0.0000001", "image": "0", "request": "0"}}
	if g.receiptCeiling {
		delete(endpoint["pricing"].(map[string]any), "request")
	}
	var body any
	switch req.URL.Path {
	case "/api/v1/models":
		body = map[string]any{"data": []any{map[string]any{"id": "fixture/receipt", "canonical_slug": "fixture/receipt-v1", "architecture": map[string]any{"input_modalities": []string{"text", "image"}, "output_modalities": []string{"text"}}, "supported_parameters": params}}}
	case "/api/v1/endpoints/zdr":
		body = map[string]any{"data": []any{endpoint}}
	case "/api/v1/models/fixture/receipt-v1/endpoints":
		body = map[string]any{"data": map[string]any{"endpoints": []any{endpoint}}}
	case "/api/v1/chat/completions":
		if req.Method != "POST" {
			return nil, fmt.Errorf("expected POST")
		}
		g.posts++
		raw, _ := io.ReadAll(req.Body)
		g.wire = string(raw)
		fields := map[string]any{"seller": "SYNTHETIC SHOP", "issuedAt": nil, "issuedDate": "2026-10-25", "currency": "RUB", "total": "100.00", "fiscalDocumentNumber": nil, "items": []any{map[string]any{"name": "Milk", "quantity": "2", "unitPrice": "50.00", "lineTotal": "100.00", "discount": nil, "brand": nil, "package": nil, "broadGroup": nil}}, "sourceText": "SYNTHETIC SHOP\n25.10.2026\nMilk 2 x 50.00 100.00\nTOTAL RUB 100.00"}
		answer, _ := json.Marshal(map[string]any{"complete": true, "receiptCount": 1, "issues": []string{}, "receipt": fields})
		usage := map[string]any{"prompt_tokens": 500, "completion_tokens": 500, "cost": 0.0001}
		if g.unknown {
			delete(usage, "cost")
		}
		body = map[string]any{"id": fmt.Sprintf("fixture-receipt-%d", g.posts), "model": "fixture/receipt-v1", "provider": "Fixture", "usage": usage, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(answer)}}}, "openrouter_metadata": map[string]any{"requested": "fixture/receipt", "strategy": "direct", "attempt": 1, "is_byok": false, "pipeline": []any{}, "endpoints": map[string]any{"available": []any{map[string]any{"model": "fixture/receipt-v1", "provider": "Fixture", "selected": true}}}}}
	default:
		return nil, fmt.Errorf("unexpected API path")
	}
	raw, _ := json.Marshal(body)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(raw))}, nil
}
func s56ReceiptImage(t *testing.T) ([]byte, []byte) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 640, 300))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	d := font.Drawer{Dst: img, Src: image.NewUniform(color.Black), Face: basicfont.Face7x13}
	for i, line := range []string{"SYNTHETIC SHOP", "25.10.2026", "Milk  2 x 50.00  100.00", "TOTAL RUB 100.00"} {
		d.Dot = fixed.P(24, 40+i*45)
		d.DrawString(line)
	}
	var pngBytes, jpegBytes bytes.Buffer
	if e := png.Encode(&pngBytes, img); e != nil {
		t.Fatal(e)
	}
	if e := jpeg.Encode(&jpegBytes, img, &jpeg.Options{Quality: 95}); e != nil {
		t.Fatal(e)
	}
	return pngBytes.Bytes(), jpegBytes.Bytes()
}

// Test-only in-memory raster PDF. It contains the same printed receipt as the
// photo control; production never constructs or parses a PDF in the API.
func s56ReceiptPDF(jpegBytes []byte) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	content := []byte("q\n640 0 0 300 0 0 cm\n/Im0 Do\nQ\n")
	objects := [][]byte{[]byte("<< /Type /Catalog /Pages 2 0 R >>"), []byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"), []byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 640 300] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>"), append(append([]byte(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 640 /Height 300 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", len(jpegBytes))), jpegBytes...), []byte("\nendstream")...), append(append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(content))), content...), []byte("endstream")...)}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n", i+1)
		b.Write(obj)
		b.WriteString("\nendobj\n")
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}
func TestS56ReceiptDocumentLifecycle(t *testing.T) {
	if os.Getenv("TEST_S56_DOCUMENTS") != "true" {
		t.Skip("disposable database and helper required")
	}
	pool, e := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	if _, e = pool.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=1000000"); e != nil {
		t.Fatal(e)
	}
	helper, e := pdfextract.NewClient(os.Getenv("TEST_PDF_EXTRACTOR_SOCKET"))
	if e != nil {
		t.Fatal(e)
	}
	photo, jpegBytes := s56ReceiptImage(t)
	for _, kind := range []string{"image", "pdf", "image_receipt_ceiling"} {
		t.Run(kind, func(t *testing.T) {
			var owner, other, account string
			for _, dest := range []*string{&owner, &other} {
				if e = pool.QueryRow(context.Background(), "INSERT INTO users(display_name,role) VALUES('S56 receipt fixture','member') RETURNING id::text").Scan(dest); e != nil {
					t.Fatal(e)
				}
			}
			ctx := access.WithPrincipal(context.Background(), access.Principal{ID: owner, Role: "member"})
			tenant := database.ForUsers(pool)
			if e = tenant.QueryRow(ctx, "INSERT INTO accounts(name,type,currency,opening_balance_minor) VALUES('S56 receipt account','debit','RUB',100000) RETURNING id::text").Scan(&account); e != nil {
				t.Fatal(e)
			}
			member := assistant.GroupMembership{Policy: assistant.ModelPolicy{Model: "fixture/receipt", Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, ImageTokenUpperBound: 1000, Vision: true}, Canonical: "fixture/receipt-v1", MaxImages: 4, MaxImageSide: 2048, Region: "eu", Qualification: "public synthetic receipt control", QualifiedUntil: time.Now().Add(time.Hour)}
			if kind == "image_receipt_ceiling" {
				member.RequestFeeEvidence = assistant.ReceiptRequestFeeRouterCeilingOnly
				member.Policy.RequestFeeMicros = 1000
				member.Policy.ImageFeeMicros = 1
				member.MaxImages = 1
			}
			configuration, _ := json.Marshal([]assistant.ModelGroup{{Revision: "s56-receipt-fixture", Profile: assistant.DocumentReading, Purpose: "receipt", Members: []assistant.GroupMembership{member}}})
			gateway := &s56ReceiptGateway{receiptCeiling: kind == "image_receipt_ceiling"}
			groups, e := assistant.NewModelGroups(string(configuration), "offline-receipt-key", gateway)
			if e != nil {
				t.Fatal(e)
			}
			provider, e := assistant.NewGroupedOpenRouterProvider("offline-receipt-key", groups, gateway)
			if e != nil {
				t.Fatal(e)
			}
			consents := assistant.NewConversationRepository(pool)
			policy, e := groups.Resolve(ctx, "receipt", 1)
			if e != nil {
				t.Fatal(e)
			}
			if e = consents.Consent(ctx, "receipt", assistant.ConsentHash("receipt", policy), true); e != nil {
				t.Fatal(e)
			}
			verifier := assistant.NewPolicyVerifier()
			verifier.Groups = groups
			extractor := &assistant.ReceiptExtractor{Verifier: verifier, Policy: policy, Provider: provider, Meter: assistant.NewUsageRepository(pool), Consents: consents}
			store, e := receiptdrafts.NewPhotoStore(t.TempDir())
			if e != nil {
				t.Fatal(e)
			}
			handler := receiptdrafts.NewHandler(pool, store, extractor)
			handler.SetPDFRenderer(helper)
			router := chi.NewRouter()
			handler.Routes(router)
			invoke := func(callCtx context.Context, method, path string, body io.Reader, contentType, key string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, path, body).WithContext(callCtx)
				if contentType != "" {
					r.Header.Set("Content-Type", contentType)
				}
				if key != "" {
					r.Header.Set("Idempotency-Key", key)
				}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				return w
			}
			raw := photo
			filename := "receipt.png"
			field, path := "photo", "/"
			if kind == "pdf" {
				raw = s56ReceiptPDF(jpegBytes)
				filename = "receipt.pdf"
				field, path = "file", "/documents"
			}
			var upload bytes.Buffer
			form := multipart.NewWriter(&upload)
			file, _ := form.CreateFormFile(field, filename)
			_, _ = file.Write(raw)
			_ = form.Close()
			response := invoke(ctx, "POST", path, &upload, form.FormDataContentType(), "")
			if response.Code != 201 {
				t.Fatal(response.Code, response.Body.String())
			}
			var draft receiptdrafts.Draft
			if json.Unmarshal(response.Body.Bytes(), &draft) != nil {
				t.Fatal("bad upload")
			}
			// A stored draft remains readable as metadata when storage is unavailable;
			// its page route must fail privately instead of dereferencing a nil store.
			noStore := receiptdrafts.NewHandler(pool, nil, nil)
			noStoreRouter := chi.NewRouter()
			noStore.Routes(noStoreRouter)
			unavailable := httptest.NewRecorder()
			noStoreRouter.ServeHTTP(unavailable, httptest.NewRequest("GET", "/"+draft.ID+"/pages/1", nil).WithContext(ctx))
			if unavailable.Code != 503 || unavailable.Header().Get("Cache-Control") != "private, no-store" || gateway.posts != 0 {
				t.Fatal("unconfigured storage leaked or called provider", unavailable.Code)
			}
			foreign := access.WithPrincipal(context.Background(), access.Principal{ID: other, Role: "member"})
			if w := invoke(foreign, "GET", "/"+draft.ID+"/pages/1", nil, "", ""); w.Code == 200 {
				t.Fatal("foreign source exposed")
			}
			page := invoke(ctx, "GET", "/"+draft.ID+"/pages/1", nil, "", "")
			if page.Code != 200 || !strings.HasPrefix(page.Header().Get("Content-Type"), "image/") {
				t.Fatal("source preview missing", page.Code, page.Body.String())
			}
			response = invoke(ctx, "POST", "/"+draft.ID+"/reading-previews", nil, "", "")
			if response.Code != 200 {
				t.Fatal(response.Code, response.Body.String())
			}
			var plan receiptdrafts.ReadingPlan
			_ = json.Unmarshal(response.Body.Bytes(), &plan)
			if plan.MaximumMicros <= 0 || len(plan.Pages) != 1 || gateway.posts != 0 {
				t.Fatal("preview called provider or omitted bounds")
			}
			input := receiptdrafts.ReadingInput{Revision: draft.Revision, PreviewHash: plan.Hash, Consent: true}
			body, _ := json.Marshal(input)
			no := input
			no.Consent = false
			bad, _ := json.Marshal(no)
			response = invoke(ctx, "POST", "/"+draft.ID+"/extract", bytes.NewReader(bad), "application/json", "read")
			if response.Code == 200 || gateway.posts != 0 {
				t.Fatal("missing consent called")
			}
			response = invoke(ctx, "POST", "/"+draft.ID+"/extract", bytes.NewReader(body), "application/json", "read")
			if response.Code != 200 {
				t.Fatal(response.Code, response.Body.String())
			}
			if json.Unmarshal(response.Body.Bytes(), &draft) != nil || draft.Status != "extracted" || draft.Fields.Total == nil || *draft.Fields.Total != "100.00" || gateway.posts != 1 {
				t.Fatal("reading missing", draft, gateway.posts)
			}
			response = invoke(ctx, "POST", "/"+draft.ID+"/extract", bytes.NewReader(body), "application/json", "read")
			if response.Code != 200 || gateway.posts != 1 {
				t.Fatal("replay charged", response.Code)
			}
			if !strings.Contains(gateway.wire, "data:image/") || strings.Contains(gateway.wire, "file-parser") || strings.Contains(gateway.wire, "%PDF-") {
				t.Fatal("raw PDF or hidden OCR sent")
			}
			var postings int
			_ = tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&postings)
			if postings != 0 {
				t.Fatal("recognition wrote ledger")
			}
			fields := draft.Fields
			fields.AccountID = account
			fields.Action = "create"
			category := "20000000-0000-0000-0000-000000000001"
			fields.CategoryID = &category
			repo := receiptdrafts.NewRepository(pool)
			draft, e = repo.Edit(ctx, draft.ID, receiptdrafts.Edit{Revision: draft.Revision, Fields: fields})
			if e != nil {
				t.Fatal(e)
			}
			preview, e := repo.Preview(ctx, draft.ID, draft.Revision)
			if e != nil {
				t.Fatal(e)
			}
			confirmed, e := repo.Confirm(ctx, draft.ID, "confirm", receiptdrafts.Confirmation{PreviewID: preview.ID, Hash: preview.Hash})
			if e != nil {
				t.Fatal(e)
			}
			again, e := repo.Confirm(ctx, draft.ID, "confirm", receiptdrafts.Confirmation{PreviewID: preview.ID, Hash: preview.Hash})
			if e != nil || again.ReceiptID == nil || confirmed.ReceiptID == nil || *again.ReceiptID != *confirmed.ReceiptID {
				t.Fatal("confirmation replay", e)
			}
			var delta int64
			if e = tenant.QueryRow(ctx, "SELECT count(*),COALESCE(sum(CASE WHEN direction='debit' THEN -amount_minor ELSE amount_minor END),0) FROM postings WHERE account_id=$1", account).Scan(&postings, &delta); e != nil || postings != 1 || delta != -10000 {
				t.Fatal("wrong confirmed purchase", postings, delta, e)
			}
			if e = repo.DeletePhoto(ctx, confirmed, store); e != nil {
				t.Fatal(e)
			}
			if w := invoke(ctx, "GET", "/"+draft.ID+"/pages/1", nil, "", ""); w.Code == 200 {
				t.Fatal("deleted receipt image visible")
			}
			if kind == "image_receipt_ceiling" {
				var reserved int64
				if e = tenant.QueryRow(ctx, "SELECT reserved_micros FROM ai_usage WHERE purpose='receipt' ORDER BY created_at DESC LIMIT 1").Scan(&reserved); e != nil || reserved != plan.MaximumMicros || reserved <= 1001 {
					t.Fatal("receipt ceiling not reserved", reserved, plan.MaximumMicros, e)
				}
				var wire map[string]any
				if e = json.Unmarshal([]byte(gateway.wire), &wire); e != nil {
					t.Fatal(e)
				}
				if wire["provider"].(map[string]any)["max_price"].(map[string]any)["request"] != 0.001 {
					t.Fatal("receipt request ceiling not sent")
				}
			}
			var bill string
			if e = tenant.QueryRow(ctx, "SELECT actual_micros::text FROM ai_usage WHERE purpose='receipt' ORDER BY created_at DESC LIMIT 1").Scan(&bill); e != nil || bill != strconv.Itoa(100) {
				t.Fatal("billing not settled", bill, e)
			}
		})
	}
}
