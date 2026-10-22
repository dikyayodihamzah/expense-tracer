package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

const claudeCLITimeout = 120 * time.Second

// claudeCLIProvider shells out to the local `claude` CLI, so extraction is billed
// to a Claude subscription instead of an API key.
type claudeCLIProvider struct{}

func newClaudeCLI() (Provider, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}
	return &claudeCLIProvider{}, nil
}

func (claudeCLIProvider) ExtractExpenses(ctx context.Context, imageData []byte) ([]*model.Expense, error) {
	// A fresh empty dir is both the CLI's cwd (no project settings picked up)
	// and the only place its Read tool needs to reach.
	dir, err := os.MkdirTemp("", "expense-claude-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	img := filepath.Join(dir, "receipt.jpg")
	if err := os.WriteFile(img, imageData, 0o600); err != nil {
		return nil, fmt.Errorf("write temp image: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, claudeCLITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p",
		"Read the image at "+img+" and extract the expenses.\n"+prompt,
		"--output-format", "json", "--allowedTools", "Read")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Auth and rate-limit failures exit 1 but still print a JSON envelope on stdout.
		if _, perr := parseCLIOutput(out); perr != nil && len(out) > 0 {
			return nil, fmt.Errorf("claude CLI: %w: %v", err, perr)
		}
		return nil, fmt.Errorf("claude CLI: %w", err)
	}
	return parseCLIOutput(out)
}

// parseCLIOutput unwraps the `claude -p --output-format json` envelope.
func parseCLIOutput(out []byte) ([]*model.Expense, error) {
	var env struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("parse claude CLI envelope: %w", err)
	}
	if env.IsError {
		return nil, fmt.Errorf("claude CLI reported error: %s", env.Result)
	}
	return ParseJSON([]byte(env.Result))
}
