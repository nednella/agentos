package sessions

import (
	"fmt"
	"strings"
	"testing"
)

func prJSON(state string, draft bool, rollup string, comments, reviews int) string {
	repeat := func(n int) string {
		return "[" + strings.TrimSuffix(strings.Repeat(`{"id":"x"},`, n), ",") + "]"
	}
	return fmt.Sprintf(`[{"number":12,"url":"https://github.com/acme/widgets/pull/12","state":%q,"isDraft":%v,"statusCheckRollup":%s,"comments":%s,"reviews":%s,"updatedAt":"2026-10-01T12:00:00Z"}]`,
		state, draft, rollup, repeat(comments), repeat(reviews))
}

func TestParsePR(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		state, check string
		comments     int
	}{
		{"draft", prJSON("OPEN", true, "[]", 0, 0), "draft", "none", 0},
		{"open", prJSON("OPEN", false, "[]", 1, 2), "open", "none", 3},
		{"merged", prJSON("MERGED", false, "[]", 0, 0), "merged", "none", 0},
		{"closed", prJSON("CLOSED", false, "[]", 0, 0), "closed", "none", 0},
		{"passing", prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"SUCCESS"},{"status":"COMPLETED","conclusion":"SKIPPED"},{"state":"SUCCESS"}]`, 0, 0), "open", "passing", 0},
		{"pending", prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"SUCCESS"},{"status":"IN_PROGRESS","conclusion":""}]`, 0, 0), "open", "pending", 0},
		{"failing wins", prJSON("OPEN", false, `[{"status":"IN_PROGRESS"},{"status":"COMPLETED","conclusion":"FAILURE"},{"status":"COMPLETED","conclusion":"SUCCESS"}]`, 0, 0), "open", "failing", 0},
		{"status context error", prJSON("OPEN", false, `[{"state":"ERROR"}]`, 0, 0), "open", "failing", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := parsePR([]byte(tt.data))
			if err != nil || pr == nil {
				t.Fatalf("parsePR = %v, %v", pr, err)
			}
			if pr.State != tt.state || pr.Checks != tt.check || pr.Comments != tt.comments || pr.Number != 12 || pr.UpdatedAt == 0 {
				t.Errorf("pr = %+v", pr)
			}
		})
	}
	if pr, err := parsePR([]byte("[]")); pr != nil || err != nil {
		t.Errorf("no PR gave %v, %v", pr, err)
	}
	if _, err := parsePR([]byte("nope")); err == nil {
		t.Error("bad json accepted")
	}
}

func TestParseWorktrees(t *testing.T) {
	w := parseWorktrees("worktree /p\nHEAD abc\nbranch refs/heads/main\n\nworktree /p/trees/issue-7\nHEAD def\nbranch refs/heads/issue-7\n\nworktree /p/detached\nHEAD 123\ndetached\n")
	if w.main != "/p" || w.byBranch["issue-7"] != "/p/trees/issue-7" || w.byBranch["main"] != "/p" || len(w.byBranch) != 2 {
		t.Errorf("worktrees = %+v", w)
	}
}
