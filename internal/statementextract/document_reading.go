package statementextract

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

const DocumentPromptVersion = "statement.document-reading.0056.v3"
const DocumentPrompt = DirectPrompt + ` Every non-null field and row anchor requires 1..8 verbatim page/quote/box evidence entries. If no printed quote supports a field, return null, never a value with empty evidence; do not infer original transaction amount/currency from account fields. Use fractional normalized boxes [left,top,right,bottom] in 0..1 relative to full page dimensions, with left<right and top<bottom. Never pixels or 0..1000. Geometry example only: [0.1,0.2,0.3,0.4]; use the actual enclosing printed region. Boxes are model-proposed, not OCR proof. Do not omit operations; at most 200, explicit issues. No tools.`

type DocumentPolicy struct {
	InputMicrosPerMillion  int64           `json:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64           `json:"outputMicrosPerMillion"`
	Model                  string          `json:"model"`
	Provider               string          `json:"provider"`
	ProviderName           string          `json:"providerName"`
	Profile                string          `json:"profile"`
	GroupRevision          string          `json:"groupRevision"`
	ProofSHA256            string          `json:"proofSHA256"`
	MaximumMicros          int64           `json:"maximumMicros"`
	Snapshot               json.RawMessage `json:"snapshot"`
}

func (p DocumentPolicy) Valid() bool {
	return p.Model != "" && p.Provider != "" && (p.Profile == "document_reading" || p.Profile == "plain_document_reading") && p.GroupRevision != "" && len(p.ProofSHA256) == 64 && p.MaximumMicros > 0 && p.MaximumMicros <= 10000000 && json.Valid(p.Snapshot)
}
func (p DocumentPolicy) Policy() Policy {
	return Policy{Model: p.Model, Provider: p.Provider, ProviderName: p.ProviderName, InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion, Document: &p}
}

type DocumentPlan struct {
	Profile        string              `json:"profile"`
	SourceSHA256   string              `json:"sourceSHA256"`
	TextSHA256     string              `json:"textSHA256"`
	Pages          []documentdata.Page `json:"pages"`
	SnapshotSHA256 string              `json:"snapshotSHA256"`
}
type DocumentReasoner interface {
	CurrentDocumentPolicy(context.Context, int) (DocumentPolicy, error)
	PrepareDocument(context.Context, DocumentPolicy, documentdata.Document) error
	CompleteDocument(context.Context, DocumentPolicy, documentdata.Document) (statementdata.VisualReading, *Billing, error)
}

func (s *Service) SetDocumentReasoner(r DocumentReasoner) { s.document = r }
func documentPlanValid(p Plan) bool {
	plain := p.Policy.Document != nil && p.Policy.Document.Profile == "plain_document_reading"
	profile, prompt, planner, schema, maxPages := "document_reading", DocumentPromptVersion, "document-pages.0056.v1", Hash(statementdata.VisualSchema()), 4
	if plain {
		profile, prompt, planner, schema, maxPages = "plain_document_reading", PlainPromptVersion, PlainPlannerVersion, Hash(PlainPromptVersion), 5
	}
	expectedLimits := PlanLimits{MaxInputBytes: 12 << 20, InputTokens: 30000, OutputTokens: 16000, MaxSpans: 0, CallTimeoutSeconds: 180, AttemptTimeoutSeconds: 210}
	if p.PlannerVersion != planner || p.AliasVersion != "none.direct-reading.v1" || p.Limits != expectedLimits || len(p.Atoms) != 0 || len(p.Aliases) != 0 || p.Document == nil || p.Policy.Document == nil {
		return false
	}
	policy := p.Policy.Document
	if !policy.Valid() || p.Policy.Model != policy.Model || p.Policy.Provider != policy.Provider || p.Policy.ProviderName != policy.ProviderName || p.Policy.InputMicrosPerMillion != policy.InputMicrosPerMillion || p.Policy.OutputMicrosPerMillion != policy.OutputMicrosPerMillion || policy.Profile != profile {
		return false
	}
	if p.PromptVersion != prompt || p.SchemaHash != schema || p.RedactionVersion != "full-pages.explicit-consent.v1" || len(p.Document.Pages) < 1 || len(p.Document.Pages) > maxPages || len(p.Chunks) != 1 || p.MaximumMicros != policy.MaximumMicros || p.Document.SnapshotSHA256 != Hash(policy.Snapshot) || p.Document.Profile != profile || p.Document.SourceSHA256 != p.DocumentHash {
		return false
	}
	if plain && p.Document.TextSHA256 != documentdata.Hash(nil) {
		return false
	}
	for i, page := range p.Document.Pages {
		if page.Number != i+1 || len(page.SHA256) != 64 || len(page.Data) != 0 || page.Width < 1 || page.Height < 1 || page.Width > 2048 || page.Height > 2048 {
			return false
		}
	}
	return true
}
func (s *Service) previewDocument(ctx context.Context, id string, in PreviewInput) (Preview, error) {
	var out Preview
	if s.document == nil {
		return out, fault("ai_disabled", 503)
	}
	source, e := s.imports.Statement(ctx, id)
	if e != nil {
		return out, e
	}
	if source.Rendered.Validate() != nil || source.DeletedAt != nil {
		return out, fault("document_limit", 422)
	}
	if len(in.VisibleSpanIDs) > 0 {
		return out, fault("full_page_consent_required", 422)
	}
	plain := source.ReadingMode == "plain_review"
	policy, e := s.readingPolicy(ctx, len(source.Rendered.Pages), plain)
	if e != nil {
		return out, e
	}
	if !policy.Valid() {
		return out, fault("price_unknown", 503)
	}
	if e = s.prepareReading(ctx, policy, source.Rendered, plain); e != nil {
		return out, e
	}
	plan := Plan{PlannerVersion: "document-pages.0056.v1", AliasVersion: "none.direct-reading.v1", Atoms: [][]string{}, Aliases: map[string]string{}, Limits: PlanLimits{MaxInputBytes: 12 << 20, InputTokens: 30000, OutputTokens: 16000, MaxSpans: 0, CallTimeoutSeconds: 180, AttemptTimeoutSeconds: 210}, DocumentHash: source.SHA256, Policy: policy.Policy(), PromptVersion: DocumentPromptVersion, SchemaHash: Hash(json.RawMessage(statementdata.VisualSchema())), RedactionVersion: "full-pages.explicit-consent.v1", MaximumMicros: policy.MaximumMicros, Chunks: []Chunk{{Index: 1, Context: []RedactedSpan{}, Spans: []RedactedSpan{}}}, Document: &DocumentPlan{Profile: policy.Profile, SourceSHA256: source.SHA256, TextSHA256: documentdata.Hash([]byte(source.Rendered.Text)), Pages: source.Rendered.Manifest().Pages, SnapshotSHA256: Hash(policy.Snapshot)}}
	if plain {
		plan.PromptVersion = PlainPromptVersion
		plan.SchemaHash = Hash(PlainPromptVersion)
		plan.PlannerVersion = PlainPlannerVersion
		plan.Document.TextSHA256 = documentdata.Hash(nil)
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	if e = sourceCurrent(ctx, tx, id, in.ExpectedRevision); e != nil {
		return out, e
	}
	out = Preview{StatementID: id, SourceRevision: in.ExpectedRevision, Hash: Hash(plan), Plan: plan}
	encoded, _ := json.Marshal(plan)
	e = tx.QueryRow(ctx, "INSERT INTO statement_extraction_previews(statement_id,source_revision,state_hash,plan_hash,plan,maximum_micros,call_count,policy,prompt_version,schema_hash,redaction_version) VALUES($1,$2,$3,$4,$5,$6,1,$7,$8,$9,$10) RETURNING id::text,expires_at", id, in.ExpectedRevision, plan.DocumentHash, out.Hash, encoded, plan.MaximumMicros, mustPolicyJSON(plan.Policy), plan.PromptVersion, plan.SchemaHash, plan.RedactionVersion).Scan(&out.ID, &out.ExpiresAt)
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}
func visualDirect(v statementdata.VisualReading) DirectReading {
	quotes := func(qs []statementdata.ImageQuote) []PageQuote {
		out := []PageQuote{}
		for _, q := range qs {
			out = append(out, PageQuote{Page: q.Page, Quote: q.Quote})
		}
		return out
	}
	fields := func(m map[string]*statementdata.VisualField) map[string]*DirectField {
		out := map[string]*DirectField{}
		for k, f := range m {
			if f == nil {
				out[k] = nil
			} else {
				out[k] = &DirectField{Value: f.Value, Evidence: quotes(f.Evidence)}
			}
		}
		return out
	}
	out := DirectReading{Header: fields(v.Header), Rows: []DirectRow{}}
	for _, r := range v.Rows {
		out.Rows = append(out.Rows, DirectRow{Anchor: quotes(r.Anchor), Fields: fields(r.Fields), Issues: r.Issues})
	}
	return out
}
func BindVisualReading(d statementdata.Document, v statementdata.VisualReading) (DirectBinding, error) {
	if e := v.Validate(len(d.Pages)); e != nil {
		return DirectBinding{}, e
	}
	return bindDirectReading(d, visualDirect(v), 200)
}
