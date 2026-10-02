package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGuardVerdict(t *testing.T) {
	tests := []struct {
		name  string
		input string
		deny  bool
	}{
		{"merge", `{"tool_name":"Bash","tool_input":{"command":"gh pr merge 12 --squash"}}`, true},
		{"force push", `{"tool_input":{"command":"git push -f origin x"}}`, true},
		{"lease", `{"tool_input":{"command":"git push --force-with-lease"}}`, false},
		{"draft pr", `{"tool_input":{"command":"gh pr create --draft"}}`, false},
		{"not json", `oops`, false},
		{"no command", `{"tool_input":{}}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := guardVerdict(strings.NewReader(tt.input))
			if !tt.deny {
				if out != "" {
					t.Fatalf("printed %q, want nothing", out)
				}
				return
			}
			var v struct {
				HookSpecificOutput struct{ HookEventName, PermissionDecision, PermissionDecisionReason string }
			}
			if err := json.Unmarshal([]byte(out), &v); err != nil {
				t.Fatalf("output %q: %v", out, err)
			}
			if h := v.HookSpecificOutput; h.HookEventName != "PreToolUse" || h.PermissionDecision != "deny" || !strings.HasPrefix(h.PermissionDecisionReason, "agentos:") {
				t.Errorf("output = %+v", h)
			}
		})
	}
}
