package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ---- fakes specific to CustomerService ----

// fakeCustomerRepoFull extends the lookup-only fakeCustomerRepo (defined in
// order_test.go) with the create/update/cohort surface CustomerService needs.
type fakeCustomerRepoFull struct {
	port.CustomerRepository
	byEmail       domain.Customer
	byEmailErr    error
	byId          domain.Customer
	byIdErr       error
	created       []domain.Customer
	updated       []domain.Customer
	addedToCohort []string // cohortId values
	listResult    []domain.Customer
	listErr       error
}

func (r *fakeCustomerRepoFull) FindByEmail(_ context.Context, _, _ string) (domain.Customer, error) {
	return r.byEmail, r.byEmailErr
}

func (r *fakeCustomerRepoFull) FindById(_ context.Context, _, _ string) (domain.Customer, error) {
	if r.byIdErr != nil {
		return domain.Customer{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeCustomerRepoFull) Create(_ context.Context, c domain.Customer) (domain.Customer, error) {
	r.created = append(r.created, c)
	return c, nil
}

func (r *fakeCustomerRepoFull) Update(_ context.Context, c domain.Customer) (domain.Customer, error) {
	r.updated = append(r.updated, c)
	return c, nil
}

func (r *fakeCustomerRepoFull) AddToCohort(_ context.Context, _, _, cohortId, _ string) (domain.Customer, error) {
	r.addedToCohort = append(r.addedToCohort, cohortId)
	return domain.Customer{}, nil
}

func (r *fakeCustomerRepoFull) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.Customer, int, error) {
	return r.listResult, len(r.listResult), r.listErr
}

// fakePaymentMethodRepoFull extends fakePaymentMethodRepo (order_test.go) with
// the find/update surface CustomerService exercises.
type fakePaymentMethodRepoFull struct {
	port.PaymentMethodRepository
	byId     domain.PaymentMethod
	byIdErr  error
	created  []domain.PaymentMethod
	updated  []domain.PaymentMethod
	expiring []domain.PaymentMethod
}

func (r *fakePaymentMethodRepoFull) FindById(_ context.Context, _, _ string) (domain.PaymentMethod, error) {
	if r.byIdErr != nil {
		return domain.PaymentMethod{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakePaymentMethodRepoFull) Create(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	r.created = append(r.created, pm)
	return pm, nil
}

func (r *fakePaymentMethodRepoFull) Update(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	r.updated = append(r.updated, pm)
	return pm, nil
}

func (r *fakePaymentMethodRepoFull) FindExpiringPaymentMethods(_ context.Context, _ time.Time) ([]domain.PaymentMethod, error) {
	return r.expiring, nil
}

// noopScheduler swallows the cron registration the customer/report constructors
// perform; it never invokes the task, so no goroutines spawn.
type noopScheduler struct{}

func (noopScheduler) ScheduleTask(string, func()) error { return nil }

func newCustomerService(cust port.CustomerRepository, pm port.PaymentMethodRepository, ps port.PubSub) *CustomerService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	svc, err := NewCustomerService(cust, pm, ps, silentLogger{}, noopScheduler{})
	if err != nil {
		panic(err) // test-only: a constructor failure here is a test bug, not a runtime path
	}
	return svc
}

func addressFor() domain.Address {
	return domain.Address{FirstName: "Ada", Line1: "1 Main St"}
}

func TestCustomerService_Create(t *testing.T) {
	t.Run("new email creates and publishes", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{}
		ps := &recordingPubSub{}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, ps)

		got, err := svc.Create(context.Background(), "org_1", CreateCustomerInput{Email: "a@b.com", FirstName: "Ada"})

		require.NoError(t, err)
		assert.Equal(t, "org_1", got.OrgId)
		assert.NotEmpty(t, got.Id)
		require.Len(t, repo.created, 1)
		assert.True(t, ps.hasTopic(port.TopicCustomerCreated))
	})

	t.Run("existing email is rejected without persistence", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byEmail: domain.Customer{Id: "cus_existing"}}
		ps := &recordingPubSub{}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, ps)

		_, err := svc.Create(context.Background(), "org_1", CreateCustomerInput{Email: "a@b.com"})

		require.Error(t, err)
		assert.Empty(t, repo.created)
		assert.False(t, ps.hasTopic(port.TopicCustomerCreated))
	})

	t.Run("lookup error is surfaced", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byEmailErr: errors.New("db down")}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, nil)

		_, err := svc.Create(context.Background(), "org_1", CreateCustomerInput{Email: "a@b.com"})

		require.Error(t, err)
		assert.Empty(t, repo.created)
	})
}

func TestCustomerService_CreatePaymentMethod(t *testing.T) {
	t.Run("uses input billing address, creates and publishes", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1"}}
		pm := &fakePaymentMethodRepoFull{}
		ps := &recordingPubSub{}
		svc := newCustomerService(repo, pm, ps)

		got, err := svc.CreatePaymentMethod(context.Background(), "org_1", CreatePaymentMethodInput{
			CustomerId: "cus_1", Name: "Visa", Token: "tok_1", BillingAddress: addressFor(),
		})

		require.NoError(t, err)
		assert.Equal(t, domain.PaymentMethodStatusActive, got.Status)
		require.Len(t, pm.created, 1)
		assert.Equal(t, "tok_1", pm.created[0].Token)
		assert.True(t, ps.hasTopic(port.TopicPaymentMethodCreated))
		assert.Empty(t, repo.updated, "no default update when IsDefault is false")
	})

	t.Run("falls back to the customer's billing address", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1", BillingAddress: addressFor()}}
		pm := &fakePaymentMethodRepoFull{}
		svc := newCustomerService(repo, pm, nil)

		got, err := svc.CreatePaymentMethod(context.Background(), "org_1", CreatePaymentMethodInput{
			CustomerId: "cus_1", Token: "tok_1",
		})

		require.NoError(t, err)
		assert.Equal(t, "Ada", got.BillingAddress.FirstName)
	})

	t.Run("rejects when no billing address can be resolved", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1"}}
		pm := &fakePaymentMethodRepoFull{}
		svc := newCustomerService(repo, pm, nil)

		_, err := svc.CreatePaymentMethod(context.Background(), "org_1", CreatePaymentMethodInput{CustomerId: "cus_1", Token: "tok_1"})

		require.Error(t, err)
		assert.Empty(t, pm.created)
	})

	t.Run("customer not found is rejected", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byIdErr: errors.New("nope")}
		pm := &fakePaymentMethodRepoFull{}
		svc := newCustomerService(repo, pm, nil)

		_, err := svc.CreatePaymentMethod(context.Background(), "org_1", CreatePaymentMethodInput{CustomerId: "cus_x", Token: "tok_1", BillingAddress: addressFor()})

		require.Error(t, err)
		assert.Empty(t, pm.created)
	})

	t.Run("IsDefault updates the customer's default payment method", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1", BillingAddress: addressFor()}}
		pm := &fakePaymentMethodRepoFull{}
		svc := newCustomerService(repo, pm, nil)

		got, err := svc.CreatePaymentMethod(context.Background(), "org_1", CreatePaymentMethodInput{
			CustomerId: "cus_1", Token: "tok_1", IsDefault: true,
		})

		require.NoError(t, err)
		require.Len(t, repo.updated, 1)
		assert.Equal(t, got.Id, repo.updated[0].DefaultPaymentMethodId)
	})
}

func TestCustomerService_UpdatePaymentMethod(t *testing.T) {
	t.Run("updates token and publishes", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1"}}
		pm := &fakePaymentMethodRepoFull{byId: domain.PaymentMethod{Id: "pm_1", Token: "old"}}
		ps := &recordingPubSub{}
		svc := newCustomerService(repo, pm, ps)

		got, err := svc.UpdatePaymentMethod(context.Background(), "org_1", UpdatePaymentMethodInput{
			CustomerId: "cus_1", PaymentMethodId: "pm_1", Token: "new",
		})

		require.NoError(t, err)
		assert.Equal(t, "new", got.Token)
		require.Len(t, pm.updated, 1)
		assert.True(t, ps.hasTopic(port.TopicPaymentMethodUpdated))
	})

	t.Run("payment method not found is rejected", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byId: domain.Customer{Id: "cus_1"}}
		pm := &fakePaymentMethodRepoFull{byIdErr: errors.New("missing")}
		svc := newCustomerService(repo, pm, nil)

		_, err := svc.UpdatePaymentMethod(context.Background(), "org_1", UpdatePaymentMethodInput{CustomerId: "cus_1", PaymentMethodId: "pm_x"})

		require.Error(t, err)
		assert.Empty(t, pm.updated)
	})
}

func TestCustomerService_GetAndList(t *testing.T) {
	t.Run("get surfaces not-found", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{byIdErr: errors.New("missing")}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, nil)

		_, err := svc.Get(context.Background(), "org_1", "cus_x")
		require.Error(t, err)
	})

	t.Run("list returns the repo result with a total", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{listResult: []domain.Customer{{Id: "cus_1"}, {Id: "cus_2"}}}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, nil)

		got, total, err := svc.List(context.Background(), "org_1", domain.Pagination{})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, 2, total)
	})
}

func TestCustomerService_HandleOrderEvent(t *testing.T) {
	t.Run("order.completed adds the customer to the signup_date cohort", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, nil)

		envelope := port.PubSubPayload{Data: domain.Order{OrgId: "org_1", CustomerId: "cus_1"}}
		raw, err := json.Marshal(envelope)
		require.NoError(t, err)

		svc.HandleOrderEvent(port.TopicOrderCompleted, raw)

		require.Len(t, repo.addedToCohort, 1)
		assert.Equal(t, "signup_date", repo.addedToCohort[0])
	})

	t.Run("malformed envelope is dropped without a cohort write", func(t *testing.T) {
		repo := &fakeCustomerRepoFull{}
		svc := newCustomerService(repo, &fakePaymentMethodRepoFull{}, nil)

		svc.HandleOrderEvent(port.TopicOrderCompleted, []byte("not json"))

		assert.Empty(t, repo.addedToCohort)
	})
}
