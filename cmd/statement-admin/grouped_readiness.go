package main

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/assistant"
)

// Readiness sends metadata only. It never constructs a provider, document,
// consent, reservation or inference request. The product resolves the same group.
func groupedReadiness(ctx context.Context, raw, key string, client assistant.HTTPDoer) (map[string]any, error) {
	groups, err := assistant.NewModelGroups(raw, key, client)
	if err != nil {
		return nil, err
	}
	p, err := groups.Resolve(ctx, "statement", 1)
	if err != nil {
		return nil, err
	}
	maximum, err := p.ReservationMicros(30000, 16000)
	if err != nil {
		return nil, err
	}
	return map[string]any{"statementAI": true, "policyVerified": true, "profile": p.Selection.Profile, "groupRevision": p.Selection.GroupRevision, "maximumMicros": maximum}, nil
}
