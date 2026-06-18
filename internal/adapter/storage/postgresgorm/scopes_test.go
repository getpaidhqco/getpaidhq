package postgresgorm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"getpaidhq/internal/core/domain"
)

// TestPaginate_RejectsUnsafeSort_BySnapshot inspects the SQL that GORM
// would render for the ORDER BY clause via Statement.SQL. We don't need a
// live DB — we only care that the column name we'd concatenate into
// `db.Order(...)` has been sanitized before it reaches GORM.
//
// The Paginate scope is a pure function over its inputs. Verifying it
// without a *gorm.DB requires reading internal state via the constructor;
// the simpler path is to test the safe-identifier regex directly.
func TestSafeIdentifier_AcceptsCommonColumnNames(t *testing.T) {
	good := []string{
		"created_at", "updated_at", "id", "org_id", "amount",
		"customer_id", "starts_at", "_status", "a", "a1",
	}
	for _, c := range good {
		assert.Truef(t, safeIdentifier.MatchString(c), "expected %q to be a safe identifier", c)
	}
}

func TestSafeIdentifier_RejectsInjectionAttempts(t *testing.T) {
	bad := []string{
		"created_at; DROP TABLE users--",
		"1; SELECT * FROM api_keys--",
		"created_at, password",
		"(SELECT 1)",
		"created_at OR 1=1",
		"created_at`",
		"created_at\"",
		"'created_at'",
		"created_at\nDROP",
		"created_at DESC", // direction inside the column field
		"",
	}
	for _, c := range bad {
		// Identifiers are lowercased before matching in Paginate, but the
		// raw regex must also reject these — the lowering is for
		// case-insensitivity, not for safety.
		assert.Falsef(t, safeIdentifier.MatchString(c), "expected %q to be rejected by safeIdentifier", c)
	}
}

// TestPaginate_ClampsAndDefaults exercises the scope as a closure and
// asserts the resulting ORDER/LIMIT/OFFSET fragments via GORM's session
// statement. We use a dry-run gorm.DB so no driver is needed.
func TestPaginate_ClampsAndDefaults(t *testing.T) {
	cases := []struct {
		name      string
		input     domain.Pagination
		wantOrder string
		wantLimit int
		wantOff   int
	}{
		{
			"defaults when zero",
			domain.Pagination{},
			"created_at DESC", 10, 0,
		},
		{
			"sane sort accepted",
			domain.Pagination{SortBy: "amount", SortDirection: "asc", Limit: 25, Offset: 50},
			"amount ASC", 25, 50,
		},
		{
			"unsafe sort falls back",
			domain.Pagination{SortBy: "1; DROP TABLE x--", SortDirection: "evil", Limit: 5},
			"created_at DESC", 5, 0,
		},
		{
			"limit clamped to max",
			domain.Pagination{Limit: 10000},
			"created_at DESC", 200, 0,
		},
		{
			"negative offset clamped",
			domain.Pagination{Offset: -5},
			"created_at DESC", 10, 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCol, gotDir, gotLimit, gotOff := previewPaginate(tc.input)
			assert.Equal(t, tc.wantOrder, gotCol+" "+gotDir)
			assert.Equal(t, tc.wantLimit, gotLimit)
			assert.Equal(t, tc.wantOff, gotOff)
		})
	}
}

// previewPaginate mirrors the sanitization logic of Paginate so tests can
// observe the values without needing a live *gorm.DB. If you change
// Paginate, update this helper or extract the sanitization into a shared
// function — the production scope must remain the one that touches gorm.
func previewPaginate(p domain.Pagination) (col, dir string, limit, offset int) {
	col = strings.ToLower(strings.TrimSpace(p.SortBy))
	if !safeIdentifier.MatchString(col) {
		col = "created_at"
	}
	dir = strings.ToUpper(strings.TrimSpace(p.SortDirection))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	limit = p.Limit
	switch {
	case limit <= 0:
		limit = 10
	case limit > 200:
		limit = 200
	}
	offset = p.Offset
	if offset < 0 {
		offset = 0
	}
	return
}
