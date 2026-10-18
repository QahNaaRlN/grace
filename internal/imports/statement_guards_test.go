package imports

import (
	"testing"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func TestStatementPostingEvidenceMatchesEachTransferSide(t *testing.T) {
	posts := []financialevents.Posting{{AccountID: "from", Direction: "debit", Amount: 1000, Currency: "RUB"}, {AccountID: "to", Direction: "credit", Amount: 1000, Currency: "RUB"}}
	debit := statementdata.Row{Direction: "debit", AccountAmount: &statementdata.Money{AmountMinor: 1000, Currency: "RUB"}}
	credit := statementdata.Row{Direction: "credit", AccountAmount: &statementdata.Money{AmountMinor: 1000, Currency: "RUB"}}
	if !statementPostingsMatch("from", debit, posts) || !statementPostingsMatch("to", credit, posts) {
		t.Fatal("second transfer side was a new fact")
	}
	if statementPostingsMatch("from", credit, posts) || statementPostingsMatch("wrong", debit, posts) {
		t.Fatal("wrong account/direction")
	}
	debit.AccountAmount.AmountMinor++
	if statementPostingsMatch("from", debit, posts) {
		t.Fatal("wrong amount")
	}
	debit.AccountAmount.AmountMinor--
	debit.AccountAmount.Currency = "USD"
	if statementPostingsMatch("from", debit, posts) {
		t.Fatal("wrong currency")
	}
}

func TestStatementCreditRequiresOwnerMeaningAndBorrowingIsRetained(t *testing.T) {
	source := statementRecord(Statement{ID: "synthetic", AccountID: "owned-debit"}, statementdata.Row{Ordinal: 1, Direction: "credit", AccountAmount: &statementdata.Money{AmountMinor: 1000, Currency: "RUB"}, Description: "Пополнение: Кубышка, заёмные средства"})
	if source.Normalized.Kind != "" || source.Normalized.Funding == nil || source.Normalized.Funding.Kind != "borrowed" {
		t.Fatal("credit was automatically classified as income")
	}
}
