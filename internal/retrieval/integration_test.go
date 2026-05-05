package retrieval

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"lark-cue/internal/larkcli"
)

func TestRealLarkCLISmoke(t *testing.T) {
	if os.Getenv("LARK_CUE_REAL_LARKCLI") != "1" {
		t.Skip("set LARK_CUE_REAL_LARKCLI=1 to run real lark-cli smoke test")
	}
	if _, err := exec.LookPath("lark-cli"); err != nil {
		t.Skip("lark-cli not installed")
	}
	retriever := NewLarkRetriever(larkcli.New("lark-cli"))
	sources, status, err := retriever.Retrieve(context.Background(), []string{"missing required scope"})
	if err != nil {
		t.Fatalf("real retrieval failed: %v", err)
	}
	if status != StatusOK {
		t.Fatalf("status = %s, want ok", status)
	}
	if len(sources) == 0 {
		t.Fatal("expected at least one real source")
	}
}
