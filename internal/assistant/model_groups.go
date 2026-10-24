package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

const TextReasoning = "text_reasoning"
const DocumentReading = "document_reading"

// GroupMembership is operator configuration, never model or upload input.
// Qualification attests task suitability; the catalog only attests capabilities.
type GroupMembership struct {
	Policy         ModelPolicy `json:"policy"`
	Canonical      string      `json:"canonical"`
	Priority       int         `json:"priority"`
	MaxImages      int         `json:"maxImages"`
	MaxImageSide   int         `json:"maxImageSide"`
	Region         string      `json:"region"`
	Qualification  string      `json:"qualification"`
	QualifiedUntil time.Time   `json:"qualifiedUntil"`
}
type ModelGroup struct {
	Revision string            `json:"revision"`
	Purpose  string            `json:"purpose"`
	Profile  string            `json:"profile"`
	Members  []GroupMembership `json:"members"`
}
type ModelSelection struct {
	Profile       string `json:"profile"`
	Purpose       string `json:"purpose"`
	GroupRevision string `json:"groupRevision"`
	Canonical     string `json:"canonical"`
	ProofSHA256   string `json:"proofSHA256"`
	MaxImages     int    `json:"maxImages"`
	MaxImageSide  int    `json:"maxImageSide"`
}
type ModelGroups struct {
	mu        sync.Mutex
	groups    []ModelGroup
	client    HTTPDoer
	key       string
	base      string
	checked   map[string][]ModelPolicy
	attempted map[string]time.Time
	now       func() time.Time
}

func NewModelGroups(raw, key string, client HTTPDoer) (*ModelGroups, error) {
	var groups []ModelGroup
	if len(raw) > 64<<10 || json.Unmarshal([]byte(raw), &groups) != nil || len(groups) < 1 || len(groups) > 4 {
		return nil, errors.New("no_eligible_model")
	}
	seen := map[string]bool{}
	for _, g := range groups {
		if seen[g.Purpose] || g.Revision == "" || len(g.Revision) > 100 || len(g.Members) < 1 || len(g.Members) > 8 || (g.Purpose != "analysis" && g.Purpose != "receipt" && g.Purpose != "statement") || (g.Profile != TextReasoning && g.Profile != DocumentReading) || (g.Purpose != "analysis" && g.Profile != DocumentReading) {
			return nil, errors.New("no_eligible_model")
		}
		seen[g.Purpose] = true
		ids := map[string]bool{}
		for _, m := range g.Members {
			p := m.Policy
			p.VerifiedAt = time.Now()
			id := p.Model + "/" + p.Provider
			if ids[id] || !concreteModelID(p.Model) || !concreteModelID(m.Canonical) || !p.Valid(time.Now()) || strings.Count(m.Canonical, "/") != 1 || strings.ContainsAny(m.Canonical, ": ?#\r\n") || m.Qualification == "" || m.QualifiedUntil.IsZero() || (m.Region != "" && !strings.HasSuffix(p.Provider, "/"+m.Region)) || p.Selection != nil || (g.Profile == DocumentReading && (!p.Vision || m.MaxImages < 1 || m.MaxImages > 4 || m.MaxImageSide < 16 || m.MaxImageSide > 2048)) {
				return nil, errors.New("capability_unverified")
			}
			ids[id] = true
		}
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &ModelGroups{groups: groups, client: client, key: key, base: "https://openrouter.ai/api/v1/", checked: map[string][]ModelPolicy{}, attempted: map[string]time.Time{}, now: time.Now}, nil
}

// Legacy policy migration is explicit and bounded to 30 days from its original
// operator verification. It never requalifies a model on restart.
func LegacyModelGroups(analysis, receipt string) string {
	groups := []ModelGroup{}
	for _, v := range []struct{ purpose, profile, raw string }{{"analysis", TextReasoning, analysis}, {"receipt", DocumentReading, receipt}} {
		var p ModelPolicy
		if json.Unmarshal([]byte(v.raw), &p) != nil || !approvedPolicyShape(p) {
			continue
		}
		images, side := 0, 0
		if v.profile == DocumentReading {
			if !p.Vision {
				continue
			}
			images = 1
			side = 2048
		}
		groups = append(groups, ModelGroup{Revision: "legacy-spec0026-v1", Purpose: v.purpose, Profile: v.profile, Members: []GroupMembership{{Policy: p, Canonical: p.Model, MaxImages: images, MaxImageSide: side, Qualification: "SPEC-0026: existing operator-approved policy", QualifiedUntil: p.VerifiedAt.AddDate(0, 0, 30)}}})
	}
	b, _ := json.Marshal(groups)
	return string(b)
}
func (g *ModelGroups) Resolve(ctx context.Context, purpose string, images int) (ModelPolicy, error) {
	if g == nil {
		return ModelPolicy{}, errors.New("no_eligible_model")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	var group *ModelGroup
	for i := range g.groups {
		if g.groups[i].Purpose == purpose {
			group = &g.groups[i]
			break
		}
	}
	if group == nil || images < 0 || images > 4 {
		return ModelPolicy{}, errors.New("no_eligible_model")
	}
	policies := g.checked[purpose]
	fresh := len(policies) > 0
	for _, p := range policies {
		fresh = fresh && p.Valid(now)
	}
	if !fresh {
		if last := g.attempted[purpose]; !last.IsZero() && now.Sub(last) < time.Minute {
			return ModelPolicy{}, errors.New("capability_unverified")
		}
		g.attempted[purpose] = now
		var e error
		policies, e = g.refresh(ctx, *group, now)
		if e != nil {
			return ModelPolicy{}, e
		}
		g.checked[purpose] = policies
	}
	for _, p := range policies {
		qualified := false
		for _, m := range group.Members {
			if m.Policy.Model == p.Model && m.Policy.Provider == p.Provider && now.Before(m.QualifiedUntil) {
				qualified = true
			}
		}
		if qualified && p.Selection != nil && p.Selection.MaxImages >= images {
			selection := *p.Selection
			p.Selection = &selection
			return p, nil
		}
	}
	return ModelPolicy{}, errors.New("no_eligible_model")
}
func (g *ModelGroups) check(p ModelPolicy) bool {
	if g == nil || p.Selection == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, group := range g.groups {
		if group.Purpose != p.Selection.Purpose {
			continue
		}
		for _, m := range group.Members {
			if m.Policy.Model == p.Model && m.Policy.Provider == p.Provider && !g.now().Before(m.QualifiedUntil) {
				return false
			}
		}
	}
	for _, known := range g.checked[p.Selection.Purpose] {
		if known.Valid(g.now()) && aiHash(known) == aiHash(p) {
			return true
		}
	}
	return false
}
func (g *ModelGroups) metadata(ctx context.Context, path string, out any) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, g.base+path, nil)
	if e != nil {
		return "", errors.New("capability_unverified")
	}
	if g.key != "" {
		req.Header.Set("Authorization", "Bearer "+g.key)
	}
	res, e := g.client.Do(req)
	if e != nil {
		return "", errors.New("capability_unverified")
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if e != nil || res.StatusCode != 200 || len(raw) > 8<<20 || json.Unmarshal(raw, out) != nil {
		return "", errors.New("capability_unverified")
	}
	return aiHash(json.RawMessage(raw)), nil
}

type capabilityModel struct {
	ID           string `json:"id"`
	Canonical    string `json:"canonical_slug"`
	Architecture struct {
		Input  []string `json:"input_modalities"`
		Output []string `json:"output_modalities"`
	} `json:"architecture"`
	Parameters []string `json:"supported_parameters"`
}
type capabilityEndpoint struct {
	Model      string   `json:"model_id"`
	Tag        string   `json:"tag"`
	Name       string   `json:"provider_name"`
	Status     *int     `json:"status"`
	Context    int      `json:"context_length"`
	Input      *int     `json:"max_prompt_tokens"`
	Output     int      `json:"max_completion_tokens"`
	Parameters []string `json:"supported_parameters"`
	Pricing    struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
		Image      string `json:"image"`
		Request    string `json:"request"`
	} `json:"pricing"`
}

func parametersFit(parameters []string) bool {
	return slices.Contains(parameters, "structured_outputs") && slices.Contains(parameters, "response_format") && slices.Contains(parameters, "max_tokens")
}
func nonnegativePriceWithin(raw string, cap int64, perToken bool) bool {
	if !decimalCost.MatchString(raw) {
		return false
	}
	n, ok := new(big.Rat).SetString(raw)
	if !ok || n.Sign() < 0 {
		return false
	}
	denominator := int64(1000000)
	if perToken {
		denominator = 1000000000000
	}
	return n.Cmp(big.NewRat(cap, denominator)) <= 0
}
func (g *ModelGroups) refresh(ctx context.Context, group ModelGroup, now time.Time) ([]ModelPolicy, error) {
	var catalog struct {
		Data []capabilityModel `json:"data"`
	}
	ch, e := g.metadata(ctx, "models", &catalog)
	if e != nil {
		return nil, e
	}
	var zdr struct {
		Data []capabilityEndpoint `json:"data"`
	}
	zh, e := g.metadata(ctx, "endpoints/zdr", &zdr)
	if e != nil {
		return nil, e
	}
	members := slices.Clone(group.Members)
	slices.SortFunc(members, func(a, b GroupMembership) int {
		if a.Priority != b.Priority {
			if a.Priority < b.Priority {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Policy.Model+"/"+a.Policy.Provider, b.Policy.Model+"/"+b.Policy.Provider)
	})
	policies := []ModelPolicy{}
	for _, m := range members {
		if !now.Before(m.QualifiedUntil) {
			continue
		}
		p := m.Policy
		matches := 0
		for _, model := range catalog.Data {
			if model.ID == p.Model && model.Canonical == m.Canonical && slices.Contains(model.Architecture.Input, "text") && slices.Contains(model.Architecture.Output, "text") && parametersFit(model.Parameters) && (group.Profile != DocumentReading || slices.Contains(model.Architecture.Input, "image")) {
				matches++
			}
		}
		if matches != 1 {
			continue
		}
		matches = 0
		for _, ep := range zdr.Data {
			if (ep.Model == p.Model || ep.Model == m.Canonical) && ep.Tag == p.Provider && ep.Status != nil && *ep.Status == 0 {
				matches++
			}
		}
		if matches != 1 {
			continue
		}
		var endpoints struct {
			Data struct {
				Endpoints []capabilityEndpoint `json:"endpoints"`
			} `json:"data"`
		}
		path := "models/" + url.PathEscape(strings.Split(m.Canonical, "/")[0]) + "/" + url.PathEscape(strings.Split(m.Canonical, "/")[1]) + "/endpoints"
		eh, err := g.metadata(ctx, path, &endpoints)
		if err != nil {
			continue
		}
		input, output := 10000, 2000
		if group.Profile == DocumentReading {
			input, output = 30000, 16000
		}
		matches = 0
		for _, ep := range endpoints.Data.Endpoints {
			if (ep.Model != p.Model && ep.Model != m.Canonical) || ep.Tag != p.Provider || ep.Name != p.DisplayProvider() || ep.Status == nil || *ep.Status != 0 || ep.Context < input+output || ep.Output < output || (ep.Input != nil && *ep.Input < input) || !parametersFit(ep.Parameters) || !nonnegativePriceWithin(ep.Pricing.Prompt, p.InputMicrosPerMillion, true) || !nonnegativePriceWithin(ep.Pricing.Completion, p.OutputMicrosPerMillion, true) {
				continue
			}
			request, image := ep.Pricing.Request, ep.Pricing.Image
			// Missing flat-price information is not evidence of a zero fee.
			if request == "" || image == "" || !nonnegativePriceWithin(request, p.RequestFeeMicros, false) || !nonnegativePriceWithin(image, p.ImageFeeMicros, false) {
				continue
			}
			matches++
		}
		if matches != 1 {
			continue
		}
		p.VerifiedAt = now
		p.Selection = &ModelSelection{Profile: group.Profile, Purpose: group.Purpose, GroupRevision: group.Revision, Canonical: m.Canonical, ProofSHA256: aiHash([]string{ch, zh, eh, m.Qualification}), MaxImages: m.MaxImages, MaxImageSide: m.MaxImageSide}
		policies = append(policies, p)
	}
	if len(policies) == 0 {
		return nil, errors.New("no_eligible_model")
	}
	return policies, nil
}

// Routing aliases cannot be consented as one concrete model.
func concreteModelID(id string) bool {
	if strings.Count(id, "/") != 1 || strings.HasPrefix(id, "openrouter/") || strings.ContainsAny(id, ":* ?#\r\n") {
		return false
	}
	for _, part := range strings.Split(id, "/") {
		if part == "" || part == "latest" || part == "auto" || strings.HasSuffix(part, "-latest") {
			return false
		}
	}
	return true
}
