package statementqualification

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const S114ComparisonAuthorization = "s114-comparison-20261021"
const S114ObservedComparisonVersion = "statement.comparison.observed.s114.v1"

// The context is closed and operator-only. It never comes from JSON or a flag.
type comparisonCampaign uint8

const (
	legacyComparison comparisonCampaign = iota
	s114Comparison
)

func (c comparisonCampaign) authorization() string {
	if c == s114Comparison {
		return S114ComparisonAuthorization
	}
	return ComparisonAuthorization
}
func (c comparisonCampaign) version() string {
	if c == s114Comparison {
		return S114ObservedComparisonVersion
	}
	return ObservedComparisonVersion
}
func S114ComparisonPreflight(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner) (ComparisonReport, error) {
	r, _, err := comparisonPrepareFor(ctx, samples, reasoners, s114Comparison)
	return r, err
}
func RunS114Comparison(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter) (ObservedComparisonReport, error) {
	return runObservedComparisonFor(ctx, samples, reasoners, meter, s114Comparison)
}
func ValidateS114Comparison(e ObservedComparisonReport, now time.Time) error {
	return validateObservedComparisonFor(e, now, s114Comparison)
}
func DecodeS114ComparisonArtifact(r io.Reader) (ComparisonReport, *ObservedComparisonReport, error) {
	return decodeComparisonArtifactFor(r, s114Comparison, false)
}
func EncodeS114Comparison(w io.Writer, e ObservedComparisonReport) error {
	return encodeObservedComparisonFor(w, e, s114Comparison)
}
func S114ComparisonMetricsFor(e ObservedComparisonReport, now time.Time) ([]ComparisonMetrics, error) {
	if err := ValidateS114Comparison(e, now); err != nil {
		return nil, err
	}
	return comparisonMetricsFor(e.Report, now, s114Comparison)
}
func validateKnownObservedComparison(e ObservedComparisonReport, now time.Time) error {
	switch e.Version {
	case ObservedComparisonVersion:
		return ValidateObservedComparison(e, now)
	case S114ObservedComparisonVersion:
		return ValidateS114Comparison(e, now)
	default:
		return errors.New("invalid_observed_comparison")
	}
}
