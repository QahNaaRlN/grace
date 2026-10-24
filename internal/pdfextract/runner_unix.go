//go:build unix

package pdfextract

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type Poppler struct{}

func (Poppler) Verify(ctx context.Context) error {
	for _, binary := range []string{Binary, "/usr/bin/pdfinfo", "/usr/bin/pdftoppm"} {
		cmd := exec.CommandContext(ctx, binary, "-v")
		cmd.Env = []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}
		output, e := cmd.CombinedOutput()
		name := strings.TrimPrefix(binary, "/usr/bin/")
		if e != nil || !strings.HasPrefix(string(output), name+" version 25.04.0\n") {
			return errors.New("unsupported_poppler_version")
		}
	}
	return nil
}
func (Poppler) Run(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, Binary, "-bbox-layout", path, "-")
	return runBounded(ctx, cmd)
}
func runBounded(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	cmd.Env = []string{"LANG=C", "LC_ALL=C", "PATH=/usr/bin:/bin"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Kill the process group, not just its parent, when the request disappears.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		e := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if e == syscall.ESRCH {
			return nil
		}
		return e
	}
	cmd.WaitDelay = time.Second
	out := &limitedBuffer{limit: statementdata.MaxXMLBytes}
	cmd.Stdout = out
	// Neither raw bank text nor parser diagnostics enter logs or the HTTP error.
	cmd.Stderr = nil
	if e := cmd.Run(); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
