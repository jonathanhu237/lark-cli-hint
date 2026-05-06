package app

import (
	"os"
	"testing"
)

func TestFlowOpsE2ERequiresExplicitEnvironment(t *testing.T) {
	if os.Getenv("LARK_CUE_E2E") != "1" {
		t.Skip("set LARK_CUE_E2E=1 with LLM config, lark-cli profile, seeded Feishu data, and local FlowOps Airflow demo to run E2E")
	}
	for _, env := range []string{"LARK_CUE_LLM_API_KEY", "LARK_CUE_LLM_MODEL", "LARK_CUE_FEISHU_PROFILE"} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s is required for FlowOps E2E", env)
		}
	}
	t.Skip("FlowOps E2E is intentionally manual for now: run examples/flowops-airflow/scripts/seed-feishu --apply, reset examples/flowops-airflow/.demo-workspace, then execute lark-cue run against ./flowctl check billing_daily")
}
