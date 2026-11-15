package providers

import (
	"slices"
	"testing"
)

// Upstream 5e54377b (v0.5.119): fix(xiaomi-tokenplan) — update the model list
// to the current Token Plan catalog. mimo-v2.5-pro and mimo-v2.5 were retired
// upstream on 2026-10-21 and their ids no longer resolve; the registry also
// advertised mimo-v2-pro, mimo-v2-omni, mimo-v2-tts and the claude-native
// mimo-v2.5-pro-claude alias of the removed model. The replacement catalog is
// mimo-v2.6-pro, mimo-v2.6-flash (chat), mimo-v2.5-asr (kind stt) and the
// three TTS ids (kind tts). mimo-v2.6-pro-ultraspeed stays out: upstream
// leaves it to customized-service accounts.

func TestUpstream20261112TokenPlanCatalog(t *testing.T) {
	want := []string{
		"mimo-v2.6-pro",
		"mimo-v2.6-flash",
		"mimo-v2.5-asr",
		"mimo-v2.5-tts",
		"mimo-v2.5-tts-voiceclone",
		"mimo-v2.5-tts-voicedesign",
	}
	retired := []string{
		"mimo-v2.5-pro",
		"mimo-v2.5-pro-claude",
		"mimo-v2.5",
		"mimo-v2-pro",
		"mimo-v2-omni",
		"mimo-v2-tts",
	}
	tests := []struct {
		name     string
		provider string
	}{
		{"canonical id", "xiaomi-tokenplan"},
		{"xmtp alias", "xmtp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models := GetProviderModels(tt.provider)
			for _, id := range want {
				if !slices.Contains(models, id) {
					t.Errorf("GetProviderModels(%q) missing %q", tt.provider, id)
				}
			}
			for _, id := range retired {
				if slices.Contains(models, id) {
					t.Errorf("GetProviderModels(%q) still advertises retired id %q", tt.provider, id)
				}
			}
		})
	}
}

func TestUpstream20261112TokenPlanModelKinds(t *testing.T) {
	tests := []struct {
		model string
		kind  string
	}{
		{"mimo-v2.5-asr", "stt"},
		{"mimo-v2.5-tts", "tts"},
		{"mimo-v2.5-tts-voiceclone", "tts"},
		{"mimo-v2.5-tts-voicedesign", "tts"},
		{"mimo-v2.6-pro", ""},
		{"mimo-v2.6-flash", ""},
	}
	for _, tt := range tests {
		for _, provider := range []string{"xiaomi-tokenplan", "xmtp"} {
			if kind := GetProviderModelKind(provider, tt.model); kind != tt.kind {
				t.Errorf("GetProviderModelKind(%q, %q) = %q, want %q", provider, tt.model, kind, tt.kind)
			}
		}
	}
}

// The chat ids resolve their thinking levels from the global *mimo*v2.6*
// pattern row (deepseek format, thinking always on), captured live from
// upstream v0.5.119 via scripts/gen-thinking-levels.mjs against the new
// catalog.
func TestUpstream20261112TokenPlanThinkingLevels(t *testing.T) {
	tests := []struct {
		model string
		want  []string
	}{
		{"mimo-v2.6-pro", []string{"low", "medium", "high", "xhigh"}},
		{"mimo-v2.6-flash", []string{"low", "medium", "high", "xhigh"}},
		{"mimo-v2.5-asr", []string{"high", "max"}},
		{"mimo-v2.5-tts", []string{"high", "max"}},
	}
	for _, tt := range tests {
		for _, provider := range []string{"xiaomi-tokenplan", "xmtp"} {
			InvalidateCapabilitiesCache()
			got := GetThinkingLevels(provider, tt.model)
			if !slices.Equal(got, tt.want) {
				t.Errorf("GetThinkingLevels(%q, %q) = %v, want %v", provider, tt.model, got, tt.want)
			}
		}
	}
}

// Provider isolation: the sibling xiaomi-mimo lane keeps its own catalog; the
// Token Plan reshuffle must not touch it (§3 — no cross-provider changes).
func TestUpstream20261112XiaomiMimoUntouched(t *testing.T) {
	models := GetProviderModels("xiaomi-mimo")
	for _, id := range []string{"mimo-x-pro-preview", "mimo-v2.5-pro", "mimo-v2.5-tts"} {
		if !slices.Contains(models, id) {
			t.Errorf("GetProviderModels(xiaomi-mimo) missing %q", id)
		}
	}
	if slices.Contains(models, "mimo-v2.6-pro") {
		t.Error("GetProviderModels(xiaomi-mimo) unexpectedly contains mimo-v2.6-pro")
	}
}
