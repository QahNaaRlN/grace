package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type readinessMetadata struct {
	get, post int
	fault     string
}

func (f *readinessMetadata) Do(r *http.Request) (*http.Response, error) {
	if r.Method != "GET" {
		f.post++
		return nil, errors.New("inference forbidden in readiness")
	}
	f.get++
	params := []string{"max_tokens", "structured_outputs", "response_format"}
	ep := map[string]any{"model_id": "fixture/plain", "tag": "fixture/eu", "provider_name": "Fixture", "status": 0, "context_length": 50000, "max_completion_tokens": 16000, "supported_parameters": params, "pricing": map[string]any{"prompt": "0.0000001", "completion": "0.0000001", "image": "0"}}
	var body any
	switch r.URL.Path {
	case "/api/v1/models":
		mods := []string{"text", "image"}
		if f.fault == "image" {
			mods = []string{"text"}
		}
		body = map[string]any{"data": []any{map[string]any{"id": "fixture/plain", "canonical_slug": "fixture/plain-v1", "architecture": map[string]any{"input_modalities": mods, "output_modalities": []string{"text"}}, "supported_parameters": params}}}
	case "/api/v1/endpoints/zdr":
		eps := []any{ep}
		if f.fault == "zdr" {
			eps = nil
		}
		body = map[string]any{"data": eps}
	default:
		if !strings.HasSuffix(r.URL.Path, "/endpoints") {
			return nil, errors.New("unexpected metadata path")
		}
		body = map[string]any{"data": map[string]any{"endpoints": []any{ep}}}
	}
	b, _ := json.Marshal(body)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
}
func TestGroupedReadinessUsesProductProfileWithoutInference(t *testing.T) {
	for _, tc := range []struct {
		name, fault         string
		strict, expired, ok bool
	}{
		{name: "plain_absent_request_cap", ok: true},
		{name: "strict_still_rejects_missing_request", strict: true},
		{name: "missing_image_modality", fault: "image"},
		{name: "no_zdr", fault: "zdr"},
		{name: "expired_qualification", expired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			until := time.Now().Add(time.Hour)
			if tc.expired {
				until = time.Now().Add(-time.Hour)
			}
			profile, mode, n := assistant.PlainDocumentReading, assistant.PlainRequestFeeRouterCeilingOnly, 5
			if tc.strict {
				profile, mode, n = assistant.DocumentReading, "", 4
			}
			m := assistant.GroupMembership{RequestFeeEvidence: mode, Policy: assistant.ModelPolicy{Model: "fixture/plain", Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, RequestFeeMicros: 1000, ImageFeeMicros: 1, Vision: true, ImageTokenUpperBound: 1000}, Canonical: "fixture/plain-v1", MaxImages: n, MaxImageSide: 2048, Region: "eu", Qualification: "SPEC-0058 accepted offline fixture", QualifiedUntil: until}
			raw, _ := json.Marshal([]assistant.ModelGroup{{Revision: "s58-readiness-v1", Purpose: "statement", Profile: profile, Members: []assistant.GroupMembership{m}}})
			f := &readinessMetadata{fault: tc.fault}
			out, err := groupedReadiness(context.Background(), string(raw), "offline", f)
			if (err == nil) != tc.ok {
				t.Fatal("readiness", err)
			}
			if f.post != 0 {
				t.Fatal("readiness sent inference")
			}
			if tc.ok && (f.get != 3 || out["profile"] != assistant.PlainDocumentReading || out["maximumMicros"] != int64(5605)) {
				t.Fatal("wrong readiness", out, f.get)
			}
		})
	}
}
