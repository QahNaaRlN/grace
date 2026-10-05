package auth

import "errors"

// failure carries only a controlled stage, never an underlying provider error.
type failure string

func (f failure) Error() string { return ErrDenied.Error() }
func (f failure) Unwrap() error { return ErrDenied }

func failureStage(err error) string {
	var f failure
	if errors.As(err, &f) {
		switch f {
		case "provider_timeout", "token_exchange", "id_token_missing", "token_expired", "issuer_mismatch", "audience_mismatch", "signature_verification", "token_verification", "subject_missing", "claims_invalid", "nonce_missing", "nonce_mismatch":
			return string(f)
		}
	}
	return "provider_failure"
}
