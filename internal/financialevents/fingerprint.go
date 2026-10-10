package financialevents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
)

// LedgerFingerprintInTx is internal confirmation evidence, never model context.
// At personal-ledger scale a canonical snapshot avoids a lossy count/max key.
func LedgerFingerprintInTx(ctx context.Context, tx pgx.Tx) (string, error) {
	var payload []byte
	err := tx.QueryRow(ctx, `SELECT jsonb_build_object(
 'accounts',(SELECT COALESCE(jsonb_agg(to_jsonb(a) ORDER BY a.id),'[]'::jsonb) FROM accounts a),
 'events',(SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.id),'[]'::jsonb) FROM financial_events e),
 'postings',(SELECT COALESCE(jsonb_agg(to_jsonb(p) ORDER BY p.id),'[]'::jsonb) FROM postings p),
 'receipts',(SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY r.id),'[]'::jsonb) FROM receipts r))`).Scan(&payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
