package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/service"
)

func newOrgHandlerForTest(
	orgRepo *fakeOrgRepo,
	custRepo *fakeCustomerRepo,
	apiKeyRepo *fakeApiKeyRepo,
) *OrgHandler {
	svc := service.NewOrgService(
		orgRepo, newPubSub(), fakeAuthProvider{}, custRepo,
		&fakeSettingRepo{}, &fakeMetadataRepo{}, apiKeyRepo, silentLogger{},
	)
	return NewOrgHandler(svc, silentLogger{})
}

func TestOrgHandler_Create(t *testing.T) {
	t.Run("happy path returns the created org", func(t *testing.T) {
		orgRepo := &fakeOrgRepo{}
		custRepo := &fakeCustomerRepo{}
		apiKeyRepo := &fakeApiKeyRepo{}
		h := newOrgHandlerForTest(orgRepo, custRepo, apiKeyRepo)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/organizations", CreateOrgInput{
			Name: "Acme", Country: "US", Timezone: "America/New_York",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, orgRepo.created, 1, "org persisted")
		assert.Len(t, apiKeyRepo.created, 1, "default api key minted")
		assert.NotEmpty(t, custRepo.cohorts, "default cohort created")
	})

	t.Run("repo failure surfaces an envelope", func(t *testing.T) {
		orgRepo := &fakeOrgRepo{createErr: errors.New("dup")}
		h := newOrgHandlerForTest(orgRepo, &fakeCustomerRepo{}, &fakeApiKeyRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/organizations", CreateOrgInput{
			Name: "Acme", Country: "US", Timezone: "UTC",
		})

		// Service returns the raw repo error → "bad_request" / 400 envelope.
		assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	})
}
