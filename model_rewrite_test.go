package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveUpstreamModel_StripsWorkBuddyPrefix(t *testing.T) {
	t.Parallel()
	got := resolveUpstreamModel("WorkBuddy/hy3", "WorkBuddy/hy3", nil)
	if got != "hy3" {
		t.Fatalf("got %q, want hy3", got)
	}
}

func TestResolveUpstreamModel_PrefersHostResolvedModel(t *testing.T) {
	t.Parallel()
	// Host already applied alias; payload still has client-facing id.
	got := resolveUpstreamModel("hy3-preview-agent", "WorkBuddy/hy3", nil)
	if got != "hy3-preview-agent" {
		t.Fatalf("got %q, want hy3-preview-agent", got)
	}
}

func TestResolveUpstreamModel_CredentialPrefixAndAlias(t *testing.T) {
	t.Parallel()
	sa := &storedAuth{
		Prefix: "wb",
		ModelAliases: []modelAlias{
			{Name: "hy3-preview-agent", Alias: "hy3"},
		},
	}
	got := resolveUpstreamModel("", "wb/hy3", sa)
	if got != "hy3-preview-agent" {
		t.Fatalf("got %q, want hy3-preview-agent", got)
	}
}

func TestRewriteModelForUpstream_RewritesPayload(t *testing.T) {
	t.Parallel()
	in := []byte(`{"model":"WorkBuddy/hy3","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	out := rewriteModelForUpstream(in, "hy3", nil)
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["model"] != "hy3" {
		t.Fatalf("model=%v, want hy3", obj["model"])
	}
	// Preserve other fields.
	if _, ok := obj["messages"]; !ok {
		t.Fatal("messages dropped")
	}
}

func TestRewriteModelForUpstream_WithAlias(t *testing.T) {
	t.Parallel()
	sa := &storedAuth{
		ModelAliases: []modelAlias{
			{Name: "hy3-preview-agent", Alias: "hy3"},
		},
	}
	in := []byte(`{"model":"WorkBuddy/hy3","stream":true}`)
	// Host left client id (same-format path / no alias channel).
	out := rewriteModelForUpstream(in, "WorkBuddy/hy3", sa)
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["model"] != "hy3-preview-agent" {
		t.Fatalf("model=%v, want hy3-preview-agent", obj["model"])
	}
}

func TestForceMaxThinking_AfterPrefixedModel(t *testing.T) {
	t.Parallel()
	obj := map[string]any{"model": "WorkBuddy/hy3"}
	if !forceMaxThinking(obj) {
		t.Fatal("expected forceMaxThinking to apply")
	}
	if obj["reasoning_effort"] != "high" {
		t.Fatalf("effort=%v", obj["reasoning_effort"])
	}
}

func TestEnsureModelAllowed_UsesBareAndAlias(t *testing.T) {
	t.Parallel()
	sa := &storedAuth{
		Prefix:         "WorkBuddy",
		ExcludedModels: []string{"hy3-preview-agent"},
		ModelAliases: []modelAlias{
			{Name: "hy3-preview-agent", Alias: "hy3"},
		},
	}
	body := []byte(`{"model":"WorkBuddy/hy3"}`)
	// After rewrite body would be hy3-preview-agent; ensure still works on client form.
	err := ensureModelAllowed(body, sa)
	if err == nil || !strings.Contains(err.Error(), "model_excluded") {
		t.Fatalf("err=%v", err)
	}
}
