package jeveval

import (
	"encoding/json"
	"fmt"
	"os"
)

func WriteReport(path string, report Report) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(report)
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("encode report: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close report: %w", closeErr)
	}
	return nil
}

func Summary(report Report) string {
	ru := report.Metrics["ru"]
	en := report.Metrics["en"]
	return fmt.Sprintf(
		"dataset=%s questions=%s requested_model=%s actual_models=%v\n"+
			"ru: total=%d completed=%d tool_accuracy=%.3f provider_errors=%d composition_errors=%d\n"+
			"en: total=%d completed=%d tool_accuracy=%.3f provider_errors=%d composition_errors=%d",
		report.DatasetVersion, report.QuestionSetVersion, report.RequestedModel, report.ActualModels,
		ru.Total, ru.Completed, ru.ToolAccuracy, ru.ProviderErrors, ru.CompositionErrors,
		en.Total, en.Completed, en.ToolAccuracy, en.ProviderErrors, en.CompositionErrors,
	)
}
