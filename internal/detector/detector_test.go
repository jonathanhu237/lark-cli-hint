package detector

import (
	"strings"
	"testing"
)

func TestDetectFeishuScopeError(t *testing.T) {
	scenario, ok := Detect("LarkApiError: missing required scope: docx:document:read")
	if !ok {
		t.Fatal("expected detection")
	}
	if scenario.ID != FeishuAPIScopeError {
		t.Fatalf("scenario ID = %q", scenario.ID)
	}
}

func TestDetectFeishuPermissionDeniedWithContext(t *testing.T) {
	scenario, ok := Detect("LarkApiError: permission denied")
	if !ok {
		t.Fatal("expected Feishu-context permission denied detection")
	}
	if scenario.ID != FeishuAPIScopeError {
		t.Fatalf("scenario ID = %q", scenario.ID)
	}
}

func TestIgnoreUnrelatedFailure(t *testing.T) {
	if _, ok := Detect("TypeError: cannot read property of undefined"); ok {
		t.Fatal("unexpected detection")
	}
}

func TestIgnoreGenericUnauthorized(t *testing.T) {
	if _, ok := Detect("HTTP 401 Unauthorized"); ok {
		t.Fatal("unexpected detection for generic unauthorized failure")
	}
}

func TestIgnoreGenericPermissionDenied(t *testing.T) {
	if _, ok := Detect("ssh: Permission denied (publickey)."); ok {
		t.Fatal("unexpected detection for generic permission denied failure")
	}
}

func TestIgnorePermissionDeniedWithIncidentalLarkSubstring(t *testing.T) {
	if _, ok := Detect("skylark: Permission denied"); ok {
		t.Fatal("unexpected detection for incidental lark substring")
	}
}

func TestIgnorePermissionDeniedWithIncidentalScopePrefixSubstring(t *testing.T) {
	if _, ok := Detect("vim: Permission denied"); ok {
		t.Fatal("unexpected detection for incidental im: substring")
	}
}

func TestIgnorePermissionDeniedForGenericPackageScope(t *testing.T) {
	if _, ok := Detect("npm ERR! Permission denied for package scope @acme/foo"); ok {
		t.Fatal("unexpected detection for generic package scope permission failure")
	}
}

func TestIgnorePermissionDeniedForPackageNamedLark(t *testing.T) {
	if _, ok := Detect("npm ERR! Permission denied for package lark"); ok {
		t.Fatal("unexpected detection for generic package named lark")
	}
}

func TestIgnoreGenericOAuthScopeErrors(t *testing.T) {
	for _, output := range []string{
		"GitHub OAuthError: missing required scope: repo",
		"GitHub OAuthError: scope not granted for repo",
	} {
		if _, ok := Detect(output); ok {
			t.Fatalf("unexpected detection for generic OAuth scope failure: %q", output)
		}
	}
}

func TestIgnoreScopeTokenSubstrings(t *testing.T) {
	for _, output := range []string{
		"tool error: notdocx:document:read",
		"tool error: docx:document:readme",
		"tool error: docx:document:read-extra",
		"missing required scope: notdocx:document:read",
		"missing required scope: docx:document:readme",
		"missing required scope: docx:document:read-extra",
	} {
		if _, ok := Detect(output); ok {
			t.Fatalf("unexpected detection for scope token substring: %q", output)
		}
	}
}

func TestDetectGenericScopePhraseWithFeishuContext(t *testing.T) {
	if _, ok := Detect("Feishu OpenAPI error: missing required scope: drive:file:read"); !ok {
		t.Fatal("expected Feishu-context scope failure detection")
	}
}

func TestSignalBufferCapturesMiddleSignal(t *testing.T) {
	buffer := NewSignalBuffer(120)
	_, _ = buffer.Write([]byte(strings.Repeat("x", 1000)))
	_, _ = buffer.Write([]byte("LarkApiError: missing required scope: docx:document:read"))
	_, _ = buffer.Write([]byte(strings.Repeat("y", 1000)))
	if _, ok := Detect(buffer.String()); !ok {
		t.Fatalf("signal buffer did not retain Feishu signal: %q", buffer.String())
	}
}

func TestSignalBufferBoundsStoredHits(t *testing.T) {
	buffer := NewSignalBuffer(180)
	for i := 0; i < 20; i++ {
		_, _ = buffer.Write([]byte(strings.Repeat("x", i+1)))
		_, _ = buffer.Write([]byte("LarkApiError: missing required scope: docx:document:read\n"))
	}
	if len(buffer.String()) > 180 {
		t.Fatalf("signal buffer stored unbounded hits, len=%d value=%q", len(buffer.String()), buffer.String())
	}
}
