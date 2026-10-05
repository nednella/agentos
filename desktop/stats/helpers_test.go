package stats_test

import (
	"os"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
)

func TestMain(m *testing.M) { os.Exit(apptest.Main(m)) }

var (
	eventually = apptest.Eventually
	exists     = apptest.Exists
)

func newHarness(t *testing.T) *apptest.Harness { return apptest.New(t) }
