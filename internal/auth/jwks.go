package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// rs256KeyTransport keeps unrelated, unsupported curves out of the JOSE
// decoder. It does not change the accepted algorithm or verify any signature.
type rs256KeyTransport struct {
	base http.RoundTripper
	url  string
}

func (t rs256KeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || req.URL.String() != t.url || resp.StatusCode != http.StatusOK {
		return resp, err
	}
	defer resp.Body.Close()
	const maxSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil || len(body) > maxSize {
		return nil, errors.New("invalid JWKS response")
	}
	var set struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if json.Unmarshal(body, &set) != nil {
		return nil, errors.New("invalid JWKS response")
	}
	selected := make([]json.RawMessage, 0, len(set.Keys))
	for _, raw := range set.Keys {
		var metadata struct {
			Algorithm string   `json:"alg"`
			Type      string   `json:"kty"`
			Use       string   `json:"use"`
			KeyOps    []string `json:"key_ops"`
		}
		if json.Unmarshal(raw, &metadata) != nil {
			return nil, errors.New("invalid JWKS key")
		}
		verify := metadata.KeyOps == nil
		for _, op := range metadata.KeyOps {
			if op == "verify" {
				verify = true
			}
		}
		if metadata.Algorithm == "RS256" && metadata.Type == "RSA" && (metadata.Use == "" || metadata.Use == "sig") && verify {
			selected = append(selected, raw)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no RS256 verification keys")
	}
	set.Keys = selected
	body, err = json.Marshal(set)
	if err != nil {
		return nil, errors.New("invalid JWKS response")
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Del("Content-Encoding")
	return resp, nil
}
