package db

import (
	"testing"
)

func TestGetSettings_StrictModelAssignment(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepo(database)

	settingsJSON := `{
		"rtkEnabled": true,
		"providerStrategies": {
			"openai": {
				"proxyPoolId": "pool-1",
				"rotateStrategy": "round-robin",
				"strictModelAssignment": true
			},
			"anthropic": {
				"proxyPoolId": "pool-2",
				"rotateStrategy": "none",
				"strictModelAssignment": false
			},
			"gemini": {
				"proxyPoolId": "pool-3"
			}
		}
	}`

	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY,
		data TEXT NOT NULL
	);`); err != nil {
		t.Fatalf("failed to create settings table: %v", err)
	}

	_, err := database.Exec(`INSERT INTO settings (id, data) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`, settingsJSON)
	if err != nil {
		t.Fatalf("failed to insert settings: %v", err)
	}

	settings, err := repo.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}

	if settings == nil {
		t.Fatal("expected non-nil settings")
	}

	stratOpenAI, ok := settings.ProviderStrategies["openai"]
	if !ok {
		t.Fatal("expected provider strategy for openai")
	}
	if !stratOpenAI.StrictModelAssignment {
		t.Errorf("expected StrictModelAssignment true for openai, got false")
	}
	if stratOpenAI.ProxyPoolID != "pool-1" {
		t.Errorf("expected ProxyPoolID pool-1, got %s", stratOpenAI.ProxyPoolID)
	}

	stratAnthropic, ok := settings.ProviderStrategies["anthropic"]
	if !ok {
		t.Fatal("expected provider strategy for anthropic")
	}
	if stratAnthropic.StrictModelAssignment {
		t.Errorf("expected StrictModelAssignment false for anthropic, got true")
	}

	stratGemini, ok := settings.ProviderStrategies["gemini"]
	if !ok {
		t.Fatal("expected provider strategy for gemini")
	}
	if stratGemini.StrictModelAssignment {
		t.Errorf("expected StrictModelAssignment false (default) for gemini, got true")
	}
}
