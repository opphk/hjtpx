package migration

import (
	"testing"
	"time"
)

func TestIndexMigrationTableName(t *testing.T) {
	m := IndexMigration{}
	tableName := m.TableName()
	if tableName != "index_migrations" {
		t.Errorf("Expected table name 'index_migrations', got '%s'", tableName)
	}
}

func TestGetDefaultIndexes(t *testing.T) {
	indexes := GetDefaultIndexes()

	if len(indexes) == 0 {
		t.Fatal("Expected default indexes to be non-empty")
	}

	seen := make(map[string]bool)
	for _, idx := range indexes {
		if seen[idx.IndexName] {
			t.Errorf("Duplicate index name: %s", idx.IndexName)
		}
		seen[idx.IndexName] = true

		if idx.IndexName == "" {
			t.Error("Index name should not be empty")
		}

		if idx.TargetTable == "" {
			t.Error("Table name should not be empty")
		}

		if idx.Columns == "" {
			t.Error("Columns should not be empty")
		}

		if idx.Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", idx.Status)
		}
	}

	expectedIndexes := []string{
		"idx_users_email",
		"idx_applications_app_key",
		"idx_verifications_session",
		"idx_captcha_session_id",
		"idx_blacklist_value_type",
	}

	for _, expected := range expectedIndexes {
		found := false
		for _, idx := range indexes {
			if idx.IndexName == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected index %s not found in default indexes", expected)
		}
	}
}

func TestIndexMigrationFields(t *testing.T) {
	now := time.Now()
	m := IndexMigration{
		ID:          1,
		IndexName:   "idx_test",
		TargetTable: "test_table",
		Columns:     "col1, col2",
		IsUnique:    true,
		IsPartial:   false,
		WhereClause: "",
		Description: "Test index",
		CreatedAt:   now,
		AppliedAt:   &now,
		Status:      "applied",
	}

	if m.IndexName != "idx_test" {
		t.Errorf("Expected IndexName 'idx_test', got '%s'", m.IndexName)
	}

	if m.TargetTable != "test_table" {
		t.Errorf("Expected TargetTable 'test_table', got '%s'", m.TargetTable)
	}

	if m.Columns != "col1, col2" {
		t.Errorf("Expected Columns 'col1, col2', got '%s'", m.Columns)
	}

	if !m.IsUnique {
		t.Error("Expected IsUnique to be true")
	}

	if m.Status != "applied" {
		t.Errorf("Expected Status 'applied', got '%s'", m.Status)
	}
}

func TestPartialIndexWhereClause(t *testing.T) {
	indexes := GetDefaultIndexes()

	partialIndexes := 0
	for _, idx := range indexes {
		if idx.IsPartial {
			partialIndexes++
			if idx.WhereClause == "" {
				t.Errorf("Partial index %s should have WhereClause", idx.IndexName)
			}
		}
	}

	if partialIndexes == 0 {
		t.Error("Expected at least one partial index")
	}
}

func TestUniqueIndexes(t *testing.T) {
	indexes := GetDefaultIndexes()

	uniqueIndexes := 0
	for _, idx := range indexes {
		if idx.IsUnique {
			uniqueIndexes++
		}
	}

	if uniqueIndexes == 0 {
		t.Error("Expected at least one unique index")
	}
}

func TestIndexCoverage(t *testing.T) {
	indexes := GetDefaultIndexes()

	tables := make(map[string]bool)
	for _, idx := range indexes {
		tables[idx.TargetTable] = true
	}

	expectedTables := []string{
		"users",
		"applications",
		"verifications",
		"blacklist",
		"verification_logs",
		"captcha_sessions",
		"risk_logs",
		"admin_login_logs",
		"configs",
	}

	for _, expected := range expectedTables {
		if !tables[expected] {
			t.Errorf("Expected table %s to have indexes", expected)
		}
	}
}

func TestIndexMigrationStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"pending", true},
		{"applied", true},
		{"failed", true},
		{"rolled_back", true},
		{"missing", true},
		{"", true},
		{"invalid", true},
	}

	for _, test := range tests {
		m := IndexMigration{Status: test.status}
		if m.Status != test.status {
			t.Errorf("Expected Status '%s', got '%s'", test.status, m.Status)
		}
	}
}

func TestBuildCreateIndexSQL(t *testing.T) {
	manager := &IndexMigrationManager{}

	tests := []struct {
		name     string
		migration IndexMigration
		expected string
	}{
		{
			name: "Simple index",
			migration: IndexMigration{
				IndexName: "idx_test",
				TargetTable: "test_table",
				Columns:   "col1",
				IsUnique:  false,
			},
			expected: "CREATE INDEX CONCURRENTLY idx_test ON test_table (col1)",
		},
		{
			name: "Unique index",
			migration: IndexMigration{
				IndexName: "idx_unique_test",
				TargetTable: "test_table",
				Columns:   "col1",
				IsUnique:  true,
			},
			expected: "CREATE UNIQUE INDEX CONCURRENTLY idx_unique_test ON test_table (col1)",
		},
		{
			name: "Composite index",
			migration: IndexMigration{
				IndexName: "idx_composite",
				TargetTable: "test_table",
				Columns:   "col1, col2",
				IsUnique:  false,
			},
			expected: "CREATE INDEX CONCURRENTLY idx_composite ON test_table (col1, col2)",
		},
		{
			name: "Partial index",
			migration: IndexMigration{
				IndexName:   "idx_partial",
				TargetTable:   "test_table",
				Columns:     "col1",
				IsUnique:    false,
				IsPartial:   true,
				WhereClause: "status = 'active'",
			},
			expected: "CREATE INDEX CONCURRENTLY idx_partial ON test_table (col1) WHERE status = 'active'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := manager.buildCreateIndexSQL(&test.migration)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}
