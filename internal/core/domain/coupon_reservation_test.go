package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCouponReservation_RequiresHolder(t *testing.T) {
	_, err := NewCouponReservation(NewCouponReservationInput{
		OrgId: "o", CouponId: "c", ExpiresAt: time.Now().Add(time.Hour),
	})
	require.Error(t, err, "a reservation with no order and no session is invalid")
}

func TestCouponReservation_IsLive(t *testing.T) {
	r := CouponReservation{ExpiresAt: time.Now().Add(time.Hour)}
	assert.True(t, r.IsLive(time.Now()))
	expired := CouponReservation{ExpiresAt: time.Now().Add(-time.Hour)}
	assert.False(t, expired.IsLive(time.Now()))
}
