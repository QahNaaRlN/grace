package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/QahNaaRlN/grace/internal/jeveval"
	"github.com/QahNaaRlN/grace/internal/routereval"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jev-eval:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: jev-eval <validate|run|analyze> [flags]")
	}
	switch arguments[0] {
	case "validate":
		return validateCommand(arguments[1:])
	case "run":
		return runCommand(arguments[1:])
	case "analyze":
		return analyzeCommand(arguments[1:])
	default:
		return fmt.Errorf("unknown command %q; expected validate, run, or analyze", arguments[0])
	}
}

func analyzeCommand(arguments []string) error {
	flags := flag.NewFlagSet("analyze", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	datasetPath := flags.String("dataset", "", "path to the versioned JSON dataset")
	inputPath := flags.String("input-report", "", "path to a raw or previously analyzed JSON report")
	reportPath := flags.String("report", "", "absolute path outside the repository for the recalculated JSON report")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *datasetPath == "" || *inputPath == "" || *reportPath == "" {
		return errors.New("--dataset, --input-report, and --report are required")
	}
	if err := validateReportPath(*reportPath); err != nil {
		return err
	}
	version, err := readDatasetVersion(*datasetPath)
	if err != nil {
		return err
	}
	if version == routereval.DatasetVersion {
		dataset, err := routereval.LoadDataset(*datasetPath)
		if err != nil {
			return err
		}
		report, err := routereval.LoadReport(*inputPath)
		if err != nil {
			return err
		}
		report, err = routereval.Reanalyze(dataset, report)
		if err != nil {
			return err
		}
		if err := routereval.WriteReport(*reportPath, report); err != nil {
			return err
		}
		fmt.Println(routereval.Summary(report))
		return nil
	}
	dataset, err := jeveval.LoadDataset(*datasetPath)
	if err != nil {
		return err
	}
	report, err := jeveval.LoadReport(*inputPath)
	if err != nil {
		return err
	}
	report, err = jeveval.Reanalyze(dataset, report, time.Now)
	if err != nil {
		return err
	}
	if err := jeveval.WriteReport(*reportPath, report); err != nil {
		return err
	}
	fmt.Println(jeveval.Summary(report))
	return nil
}

func validateCommand(arguments []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	datasetPath := flags.String("dataset", "", "path to the versioned JSON dataset")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *datasetPath == "" {
		return errors.New("--dataset is required")
	}
	version, err := readDatasetVersion(*datasetPath)
	if err != nil {
		return err
	}
	if version == routereval.DatasetVersion {
		dataset, err := routereval.LoadDataset(*datasetPath)
		if err != nil {
			return err
		}
		fmt.Printf("validated dataset=%s cases=%d\n", dataset.Version, len(dataset.Cases))
		return nil
	}
	dataset, err := jeveval.LoadDataset(*datasetPath)
	if err != nil {
		return err
	}
	fmt.Printf("validated dataset=%s cases=%d\n", dataset.Version, len(dataset.Cases))
	return nil
}

func runCommand(arguments []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	datasetPath := flags.String("dataset", "", "path to the versioned JSON dataset")
	model := flags.String("model", "", "explicit versioned Jev model id")
	reportPath := flags.String("report", "", "absolute path outside the repository for the JSON report")
	endpoint := flags.String("endpoint", jeveval.DefaultTypeSafeEndpoint, "TypeSafe System One endpoint")
	timeout := flags.Duration("timeout", 30*time.Second, "timeout for one provider request")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *datasetPath == "" || *model == "" || *reportPath == "" {
		return errors.New("--dataset, --model, and --report are required")
	}
	if err := jeveval.ValidateModelID(*model); err != nil {
		return err
	}
	if *timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if err := validateReportPath(*reportPath); err != nil {
		return err
	}
	apiKey := os.Getenv("TYPESAFE_API_KEY")
	if strings.TrimSpace(apiKey) == "" {
		return errors.New("TYPESAFE_API_KEY is required for live run")
	}
	version, err := readDatasetVersion(*datasetPath)
	if err != nil {
		return err
	}
	provider, err := jeveval.NewTypeSafeProvider(apiKey, *endpoint, &http.Client{Timeout: *timeout})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if version == routereval.DatasetVersion {
		dataset, err := routereval.LoadDataset(*datasetPath)
		if err != nil {
			return err
		}
		report, err := routereval.Evaluate(ctx, dataset, *model, provider, time.Now)
		if err != nil {
			return err
		}
		if err := routereval.WriteReport(*reportPath, report); err != nil {
			return err
		}
		fmt.Println(routereval.Summary(report))
		if report.MetricsByLanguage["ru"].Completed+report.MetricsByLanguage["en"].Completed == 0 {
			return errors.New("report contains no completed evaluations")
		}
		if !report.GatePassed {
			return errors.New("production routing gate failed")
		}
		return nil
	}
	dataset, err := jeveval.LoadDataset(*datasetPath)
	if err != nil {
		return err
	}
	report, err := jeveval.Evaluate(ctx, dataset, *model, provider, time.Now)
	if err != nil {
		return err
	}
	if err := jeveval.WriteReport(*reportPath, report); err != nil {
		return err
	}
	fmt.Println(jeveval.Summary(report))
	if report.Metrics["ru"].Completed+report.Metrics["en"].Completed == 0 {
		return errors.New("report contains no completed evaluations")
	}
	return nil
}

func readDatasetVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read dataset version: %w", err)
	}
	var header struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return "", fmt.Errorf("decode dataset version: %w", err)
	}
	return header.Version, nil
}

func validateReportPath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("--report must be an absolute path")
	}
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return err
	}
	cleanPath := filepath.Clean(path)
	relative, err := filepath.Rel(moduleRoot, cleanPath)
	if err != nil {
		return fmt.Errorf("compare report path: %w", err)
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("--report must be outside the repository")
	}
	return nil
}

func findModuleRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("cannot find repository root containing go.mod")
		}
		directory = parent
	}
}
