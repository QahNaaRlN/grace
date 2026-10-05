# Grace import review

Source: synthetic-workbook. Captured: 2026-10-06T00:00:00+05:00.

No financial facts were published. Candidates require individual confirmation.

## Domains

| Domain | Rows |
| --- | ---: |
| transaction | 6 |

timezone_override: Confirmed operator timezone overrides source metadata; raw dates are retained.

## transaction

| Source row | Kind | Date | Amount minor | Issues |
| --- | --- | --- | ---: | --- |
| Операции!5 | debt_payment | 2026-08-15 | 550000 | date_precision, unresolved_debt_target |
| Операции!6 | debt_payment | 2026-08-15 | 550000 | date_precision, unresolved_debt_target |
| Операции!7 | purchase | 2026-08-25 | 14497 | date_precision |
| Операции!305 | debt_payment | 2026-08-15 | 550000 | date_precision, outside_formula_scope, unresolved_debt_target, duplicate_candidate |
| Операции!306 | debt_payment | 2026-08-15 | 550000 | date_precision, outside_formula_scope, unresolved_debt_target, duplicate_candidate |
| Операции!307 | purchase | 2026-08-27 | 14497 | date_precision, outside_formula_scope, date_conflict, duplicate_candidate |

| Source row | Named amount | Minor units |
| --- | --- | ---: |

## Reconciliation candidates

| Source row | Candidate row | Rule | Matched | Different source columns | Decision |
| --- | --- | --- | --- | --- | --- |
| Операции!305 | Операции!5 | date_kind_amount | kind, amountMinor, localDate, description | O | needs_confirmation |
| Операции!305 | Операции!6 | date_kind_amount | kind, amountMinor, localDate | C, K, O | needs_confirmation |
| Операции!306 | Операции!5 | date_kind_amount | kind, amountMinor, localDate | C, K, O | needs_confirmation |
| Операции!306 | Операции!6 | date_kind_amount | kind, amountMinor, localDate, description | O | needs_confirmation |
| Операции!307 | Операции!7 | description_kind_amount_date_conflict | kind, amountMinor, description | A, O | needs_confirmation |

## Issues

| Coordinate | Severity | Code | Action |
| --- | --- | --- | --- |
| Операции!A5 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!K5 | warning | unresolved_debt_target | Debt target and principal/interest allocation require evidence. |
| Операции!A6 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!K6 | warning | unresolved_debt_target | Debt target and principal/interest allocation require evidence. |
| Операции!A7 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!A305 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!A305 | warning | outside_formula_scope | Row lies outside the current summary formula range; review individually. |
| Операции!K305 | warning | unresolved_debt_target | Debt target and principal/interest allocation require evidence. |
| Операции!A305 | warning | duplicate_candidate | Possible overlap requires individual confirmation; records are not merged. |
| Операции!A306 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!A306 | warning | outside_formula_scope | Row lies outside the current summary formula range; review individually. |
| Операции!K306 | warning | unresolved_debt_target | Debt target and principal/interest allocation require evidence. |
| Операции!A306 | warning | duplicate_candidate | Possible overlap requires individual confirmation; records are not merged. |
| Операции!A307 | info | date_precision | Only a calendar date is known; no event time is invented. |
| Операции!A307 | warning | outside_formula_scope | Row lies outside the current summary formula range; review individually. |
| Операции!A307 | warning | date_conflict | A candidate has the same description/kind/amount but another date; source evidence is required. |
| Операции!A307 | warning | duplicate_candidate | Possible overlap requires individual confirmation; records are not merged. |
