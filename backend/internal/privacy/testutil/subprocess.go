// Package testutil runs irreversible privacy-policy tests in isolated processes,
// so upstream tests of the reusable components retain their normal behavior.
package testutil

import (
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/privacy"
)

func Run(t *testing.T, test func(*testing.T)) {
	t.Helper()
	const childKey = "SUB2API_PRIVACY_TEST_CHILD"
	if os.Getenv(childKey) == t.Name() {
		privacy.Enforce()
		test(t)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$", "-test.v")
	cmd.Env = append(os.Environ(), childKey+"="+t.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("privacy regression failed: %v\n%s", err, output)
	}
}
