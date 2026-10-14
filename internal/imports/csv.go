package imports

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/importreview"
)

// csvSnapshot retains exported values as strings, never as executable formulas.
// Read blank lines as records too, so source coordinates remain meaningful.
func csvSnapshot(raw []byte, sourceID, capturedAt string) (importreview.Snapshot, error) {
	if len(raw) > importreview.MaxBytes {
		return importreview.Snapshot{}, fault("body_too_large", 413)
	}
	if strings.TrimSpace(sourceID) == "" || len(sourceID) > 500 || !utf8.Valid(raw) || bytes.ContainsRune(raw, 0) {
		return importreview.Snapshot{}, fault("invalid_csv", 400)
	}
	body := bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	// A CSV parser ignores empty physical lines. Give them a quoted empty cell;
	// do not change blank lines inside quoted multiline values.
	var preserved strings.Builder
	quoted, lineStart := false, 0
	for i := 0; i < len(body); i++ {
		if body[i] == '"' {
			quoted = !quoted
		}
		if body[i] == '\n' && !quoted {
			line := body[lineStart:i]
			if len(bytes.TrimSuffix(line, []byte{'\r'})) == 0 {
				preserved.WriteString("\"\"")
			} else {
				preserved.Write(line)
			}
			preserved.WriteByte('\n')
			lineStart = i + 1
		}
	}
	preserved.Write(body[lineStart:])
	for _, delimiter := range []rune{',', ';'} {
		reader := csv.NewReader(strings.NewReader(preserved.String()))
		reader.Comma, reader.FieldsPerRecord = delimiter, -1
		rows := []importreview.Row{}
		valid := true
		cells := 0
		for {
			values, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil || len(values) > 15 || len(rows) >= importreview.MaxRows {
				valid = false
				break
			}
			cells += len(values)
			if cells > importreview.MaxCells {
				valid = false
				break
			}
			row := importreview.Row{Values: make([]importreview.Cell, len(values))}
			for i, value := range values {
				v := value
				row.Values[i] = importreview.Cell{UserEnteredValue: &importreview.Value{StringValue: &v}, FormattedValue: value}
			}
			rows = append(rows, row)
		}
		if !valid || len(rows) < 5 || len(rows[3].Values) < 8 {
			continue
		}
		header := rows[3].Values
		if !strings.HasPrefix(strings.TrimSpace(header[0].FormattedValue), "Дата") || !strings.HasPrefix(strings.TrimSpace(header[1].FormattedValue), "Тип") || !strings.HasPrefix(strings.TrimSpace(header[5].FormattedValue), "Сумма") {
			continue
		}
		return importreview.Snapshot{SchemaVersion: importreview.SnapshotVersion, SourceID: strings.TrimSpace(sourceID), CapturedAt: capturedAt, Locale: "ru_RU", DeclaredTimeZone: "Asia/Yekaterinburg", ConfirmedTimeZone: "Asia/Yekaterinburg", Sheets: []importreview.Sheet{{SheetID: 0, Title: "Операции", RowCount: len(rows), ColumnCount: 15, Ranges: []importreview.Range{{StartRow: 1, StartColumn: 1, Rows: rows}}}}}, nil
	}
	return importreview.Snapshot{}, fault("invalid_csv", 400)
}

func (r *Repository) UploadCSV(ctx context.Context, raw []byte, sourceID, key string) (Batch, error) {
	return r.UploadCSVNamed(ctx, raw, sourceID, key, "")
}

func (r *Repository) UploadCSVNamed(ctx context.Context, raw []byte, sourceID, key, fileName string) (Batch, error) {
	snapshot, err := csvSnapshot(raw, sourceID, r.now().Format("2006-01-02T15:04:05Z07:00"))
	if err != nil {
		return Batch{}, err
	}
	report, err := importreview.Review(snapshot)
	if err != nil {
		return Batch{}, fault("review_limit", 422)
	}
	return r.uploadReport(ctx, raw, key, report, upload{"csv", fileName})
}
