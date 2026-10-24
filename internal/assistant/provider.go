package assistant

import "net/http"

type ProviderErrorCode string

const (
	ProviderAuthentication ProviderErrorCode = "authentication"
	ProviderRequest        ProviderErrorCode = "request"
	ProviderRateLimit      ProviderErrorCode = "rate_limit"
	ProviderOverload       ProviderErrorCode = "overload"
	ProviderProtocol       ProviderErrorCode = "protocol"
	ProviderTimeout        ProviderErrorCode = "timeout"
	ProviderTransport      ProviderErrorCode = "transport"
)

type ProviderError struct {
	Code ProviderErrorCode
	Err  error
}

func (err *ProviderError) Error() string { return string(err.Code) }
func (err *ProviderError) Unwrap() error { return err.Err }

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Legacy offline classification DTO; not a provider implementation.
type choiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}
