package guard

import "testing"

func TestDenied(t *testing.T) {
	tests := []struct {
		command string
		denied  bool
	}{
		{"gh pr merge 12 --squash", true},
		{"cd x && gh pr merge --auto", true},
		{"gh pr ready", true},
		{"gh pr ready 12", true},
		{"git push -f origin x", true},
		{"git push --force origin x", true},
		{"git push origin x --force", true},
		{"git push origin +x", true},
		{"gh pr edit 5 --add-reviewer a", true},
		{"gh pr create --draft --reviewer bob", true},
		{"gh api repos/o/r/pulls/5/requested_reviewers -f reviewers[]=a", true},
		{"gh pr create --draft --title x\n--body y", false},
		{"gh pr create --draft", false},
		{"git push --force-with-lease origin x", false},
		{"git push -u origin issue-7", false},
		{"gh pr view 12", false},
		{"gh pr list --state all", false},
		{"git status", false},
		{"yarn test --force", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := Denied(Defaults, tt.command); got != tt.denied {
			t.Errorf("Denied(%q) = %v, want %v", tt.command, got, tt.denied)
		}
	}
	if Denied([]string{"("}, "anything") {
		t.Error("a broken pattern matched")
	}
	if Denied(nil, "gh pr merge") {
		t.Error("no patterns still denied")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv(Env, Encode([]string{"rm -rf"}))
	if got := FromEnv(); len(got) != 1 || got[0] != "rm -rf" {
		t.Errorf("FromEnv = %v", got)
	}
	t.Setenv(Env, "not json")
	if got := FromEnv(); len(got) != len(Defaults) {
		t.Errorf("bad value gave %v", got)
	}
	t.Setenv(Env, "[]")
	if got := FromEnv(); len(got) != 0 {
		t.Errorf("empty list gave %v", got)
	}
}
