package browser

import (
	"context"
	"encoding/json"
)

// OpenTab opens address in a new tab from the session's page, as a click on a link with target _blank does.
func (b *Browsers) OpenTab(ctx context.Context, id, address string) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	quoted, _ := json.Marshal(address)
	_, err = t.call(ctx, "Runtime.evaluate", map[string]any{"expression": "window.open(" + string(quoted) + ", '_blank')", "userGesture": true})
	return err
}
