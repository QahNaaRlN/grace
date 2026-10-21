package statementqualification

import (
	"testing"
	"time"
)

func TestGroundingProfileInvalidatesPreviousQualification(t *testing.T) {
	current := validReportFixture()
	if ValidateReport(current, time.Now()) != nil {
		t.Fatal("current fixture rejected")
	}
	current.Profile.Scorer = "statement-fixed-corpus.v2.record-descriptions"
	current.Digest = reportDigest(current)
	if ValidateReport(current, time.Now()) == nil {
		t.Fatal("pre-grounding qualification accepted")
	}
	screening := screeningReportFixture()
	if ValidateScreeningReport(screening, time.Now()) != nil {
		t.Fatal("current screening fixture rejected")
	}
	screening.Profile.Scorer = "statement-screening.v1.19-20-23"
	screening.Digest = reportDigest(screening)
	if ValidateScreeningReport(screening, time.Now()) == nil {
		t.Fatal("historical screening qualifies current rules")
	}
}
