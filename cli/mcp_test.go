package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/control"
)

// mcpTools runs agentos mcp-browser with no server named, sends initialize and tools/list, and returns the tools listed.
func mcpTools(t *testing.T) []string {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" +
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"))
	root.SetArgs([]string{"mcp-browser"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("answers = %q", out.String())
	}
	var list struct {
		Result struct {
			Tools []struct{ Name string }
		}
	}
	if err := json.Unmarshal([]byte(lines[1]), &list); err != nil {
		t.Fatalf("tools/list answer %q: %v", lines[1], err)
	}
	names := make([]string, len(list.Result.Tools))
	for i, tool := range list.Result.Tools {
		names[i] = tool.Name
	}
	return names
}

func TestMCPBrowserWithoutAServerAsksTheAppForTheSessionsBrowser(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		server, _ := json.Marshal(agent.MCPServer{Command: "npx", Args: []string{"chrome-devtools-mcp"}})
		return control.Response{OK: true, Out: string(server)}
	})
	t.Setenv("AGENTOS_SESSION", "demo/a1")
	if tools := mcpTools(t); len(tools) == 0 || tools[0] != "click" {
		t.Errorf("tools with a browser = %v", tools)
	}
	if got.Cmd != "mcp-browser" || got.Session != "demo/a1" {
		t.Errorf("the app was asked %+v", got)
	}
}

func TestMCPBrowserListsNoToolsWhenTheSessionHasNoBrowser(t *testing.T) {
	fakeApp(t, func(context.Context, control.Request) control.Response {
		return control.Response{OK: true, Out: `{}`}
	})
	t.Setenv("AGENTOS_SESSION", "demo/a1")
	if tools := mcpTools(t); len(tools) != 0 {
		t.Errorf("tools without a browser = %v", tools)
	}
	// The owner's own agy has the global entry too, and must not reach the app.
	fakeApp(t, func(context.Context, control.Request) control.Response {
		t.Error("the app was asked outside a session")
		return control.Response{OK: true}
	})
	t.Setenv("AGENTOS_SESSION", "")
	if tools := mcpTools(t); len(tools) != 0 {
		t.Errorf("tools outside a session = %v", tools)
	}
}
