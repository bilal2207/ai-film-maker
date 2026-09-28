package migrations

import (
	"testing"
)

func TestGetMigrations(t *testing.T) {
	list, err := GetMigrations()
	if err != nil {
		t.Fatalf("expected no error reading embedded migrations, got %v", err)
	}

	if len(list) == 0 {
		t.Fatalf("expected at least one embedded migration, got 0")
	}

	if list[0].Version != "000001" {
		t.Errorf("expected first migration version to be '000001', got '%s'", list[0].Version)
	}

	if list[0].SQL == "" {
		t.Errorf("expected non-empty SQL content for migration")
	}
}
