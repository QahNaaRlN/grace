package statementqualification

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"time"
)

const S117ComparisonAuthorization = "s117-comparison-20261023"
const S117ObservedComparisonVersion = "statement.comparison.observed.s117.v1"

func S117ComparisonPreflight(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner) (ComparisonReport, error) {
	r, _, err := comparisonPrepareFor(ctx, samples, reasoners, s117Comparison)
	return r, err
}
func RunS117Comparison(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter) (ObservedComparisonReport, error) {
	return runObservedComparisonFor(ctx, samples, reasoners, meter, s117Comparison)
}
func ValidateS117Comparison(e ObservedComparisonReport, now time.Time) error {
	return validateObservedComparisonFor(e, now, s117Comparison)
}
func DecodeS117ComparisonArtifact(r io.Reader) (ComparisonReport, *ObservedComparisonReport, error) {
	return decodeComparisonArtifactFor(r, s117Comparison, false)
}
func EncodeS117Comparison(w io.Writer, e ObservedComparisonReport) error {
	return encodeObservedComparisonFor(w, e, s117Comparison)
}
func S117ComparisonMetricsFor(e ObservedComparisonReport, now time.Time) ([]ComparisonMetrics, error) {
	if err := ValidateS117Comparison(e, now); err != nil {
		return nil, err
	}
	return comparisonMetricsFor(e.Report, now, s117Comparison)
}
