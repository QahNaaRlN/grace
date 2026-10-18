//go:build unix

package pdfextract

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessGroupCancellation(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "survived")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `(sleep 0.3; printf survived > "$1") & wait`, "synthetic", marker)
	if _, e := runBounded(ctx, cmd); e == nil {
		t.Fatal("cancelled process returned success")
	}
	time.Sleep(400 * time.Millisecond)
	if _, e := os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("child escaped process-group cancellation", e)
	}
}
