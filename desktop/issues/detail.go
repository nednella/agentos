package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/internal/util"
)

// IssueDetail is what the issue view adds to a row: the body and the comments.
type IssueDetail struct {
	Number   int            `json:"number"`
	BodyHTML string         `json:"bodyHTML"`
	Comments []IssueComment `json:"comments"`
}

// IssueComment is one comment on an issue.
type IssueComment struct {
	Author    string `json:"author"`
	CreatedAt int64  `json:"createdAt"`
	BodyHTML  string `json:"bodyHTML"`
}

// renderedHTML asks the GitHub API for Markdown rendered the way github.com shows it, which GitHub
// also sanitizes; the app needs no Markdown renderer of its own.
const renderedHTML = "Accept: application/vnd.github.html+json"

// Detail reads the issue's body and comments, rendered, through the gh CLI.
func (i *Issues) Detail(ctx context.Context, dir string, number int) (IssueDetail, error) {
	path := fmt.Sprintf("repos/{owner}/{repo}/issues/%d", number)
	var issue struct {
		BodyHTML string `json:"body_html"`
		Comments int    `json:"comments"`
	}
	if err := i.api(ctx, dir, path, &issue); err != nil {
		return IssueDetail{}, fmt.Errorf("reading issue #%d: %w", number, err)
	}
	detail := IssueDetail{Number: number, BodyHTML: issue.BodyHTML, Comments: []IssueComment{}}
	if issue.Comments == 0 {
		return detail, nil
	}
	var comments []struct {
		User      ghLogin   `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		BodyHTML  string    `json:"body_html"`
	}
	if err := i.api(ctx, dir, path+"/comments?per_page=100", &comments); err != nil {
		return IssueDetail{}, fmt.Errorf("reading the comments of issue #%d: %w", number, err)
	}
	for _, c := range comments {
		detail.Comments = append(detail.Comments, IssueComment{Author: c.User.Login, CreatedAt: util.Millis(c.CreatedAt), BodyHTML: c.BodyHTML})
	}
	return detail, nil
}

func (i *Issues) api(ctx context.Context, dir, path string, into any) error {
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := i.run(ctx, dir, "gh", "api", path, "-H", renderedHTML)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, into); err != nil {
		return fmt.Errorf("reading gh api: %w", err)
	}
	return nil
}
