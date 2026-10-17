package vision

import (
	"strings"
	"testing"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

func TestParseCLIOutput(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{
			name: "success envelope",
			in:   `{"is_error":false,"result":"{\"date\":\"17/10/2026\",\"category\":\"Food\",\"description\":\"Indomaret\",\"nominal\":45200}"}`,
		},
		{
			name: "fenced result",
			in:   "{\"is_error\":false,\"result\":\"```json\\n{\\\"date\\\":\\\"17/10/2026\\\",\\\"category\\\":\\\"Food\\\",\\\"description\\\":\\\"Indomaret\\\",\\\"nominal\\\":45200}\\n```\"}",
		},
		{name: "cli error", in: `{"is_error":true,"result":"rate limited"}`, wantErr: "rate limited"},
		{name: "garbage", in: `nope`, wantErr: "envelope"},
		{name: "non-json result", in: `{"is_error":false,"result":"sorry"}`, wantErr: "parse vision JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := parseCLIOutput([]byte(tt.in))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Nominal != 45200 || e.Category != model.CategoryFood || e.Description != "Indomaret" {
				t.Errorf("unexpected expense: %+v", e)
			}
		})
	}
}
