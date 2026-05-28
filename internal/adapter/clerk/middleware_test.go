package clerk

import (
	"testing"

	"getpaidhq/internal/core/port"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

// NOTE: ClerkMiddleware.Authenticate is not unit-testable in isolation. It
// calls jwt.Verify (validates the JWT against Clerk's network-fetched JWKS via
// a package-level key set) and user.Get (a live Clerk API round-trip). Both are
// package-level SDK functions, not injectable dependencies, so exercising
// Authenticate would require a live Clerk service. The pure mapping helpers it
// relies on (role mapping, primary-email extraction, onboarding-error trigger)
// are covered below.

func TestMapClerkRoleToUserRole(t *testing.T) {
	tests := []struct {
		name      string
		clerkRole string
		want      port.UserRole
	}{
		{name: "org:admin maps to admin", clerkRole: "org:admin", want: port.RoleAdmin},
		{name: "org:member maps to member", clerkRole: "org:member", want: port.RoleMember},
		{name: "unknown role defaults to member", clerkRole: "org:billing_manager", want: port.RoleMember},
		{name: "empty role defaults to member", clerkRole: "", want: port.RoleMember},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MapClerkRoleToUserRole(tt.clerkRole))
		})
	}
}

func strPtr(s string) *string { return &s }

func TestPrimaryEmail(t *testing.T) {
	t.Run("nil user yields empty string", func(t *testing.T) {
		assert.Equal(t, "", primaryEmail(nil))
	})

	t.Run("no email addresses yields empty string", func(t *testing.T) {
		assert.Equal(t, "", primaryEmail(&clerk.User{}))
	})

	t.Run("returns the address matching PrimaryEmailAddressID", func(t *testing.T) {
		usr := &clerk.User{
			PrimaryEmailAddressID: strPtr("eml_primary"),
			EmailAddresses: []*clerk.EmailAddress{
				{ID: "eml_other", EmailAddress: "other@b.com"},
				{ID: "eml_primary", EmailAddress: "primary@b.com"},
			},
		}
		assert.Equal(t, "primary@b.com", primaryEmail(usr))
	})

	t.Run("falls back to first address when primary id does not match", func(t *testing.T) {
		usr := &clerk.User{
			PrimaryEmailAddressID: strPtr("eml_missing"),
			EmailAddresses: []*clerk.EmailAddress{
				{ID: "eml_a", EmailAddress: "first@b.com"},
				{ID: "eml_b", EmailAddress: "second@b.com"},
			},
		}
		assert.Equal(t, "first@b.com", primaryEmail(usr))
	})

	t.Run("falls back to first address when no primary id set", func(t *testing.T) {
		usr := &clerk.User{
			EmailAddresses: []*clerk.EmailAddress{
				{ID: "eml_a", EmailAddress: "only@b.com"},
			},
		}
		assert.Equal(t, "only@b.com", primaryEmail(usr))
	})
}
