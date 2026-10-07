package analytics

import (
	"context"
	"time"
)

func historyDay(t time.Time) string {
	zone, _ := time.LoadLocation("Asia/Yekaterinburg")
	return t.In(zone).Format("2006-01-02")
}
func partialHistoryDay(t time.Time) bool {
	zone, _ := time.LoadLocation("Asia/Yekaterinburg")
	t = t.In(zone)
	return t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0
}
func checkPeriodPrecision(ctx context.Context, db periodQuerier, from, to time.Time) error {
	if !partialHistoryDay(from) && !partialHistoryDay(to) {
		return nil
	}
	rows, err := db.Query(ctx, `SELECT 1 FROM financial_events WHERE
	(occurred_date=$1::date AND $2) OR (occurred_date=$3::date AND $4) LIMIT 1`, historyDay(from), partialHistoryDay(from), historyDay(to), partialHistoryDay(to))
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return ErrTimePrecision
	}
	return rows.Err()
}
