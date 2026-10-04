package main

import (
	"context"
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
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jev-eval:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: jev-eval <validate|run> [flags]")
	}
	switch arguments[0] {
	case "validate":
		return validateCommand(arguments[1:])
	case "run":
		return runCommand(arguments[1:])
	default:
		return fmt.Errorf("unknown command %q; expected validate or run", arguments[0])
	}
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
	dataset, err := jeveval.LoadDataset(*datasetPath)
	if err != nil {
		return err
	}
	provider, err := jeveval.NewTypeSafeProvider(apiKey, *endpoint, &http.Client{Timeout: *timeout})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
