package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/cedar"
	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// silentLogger drops everything. Mirrors the helper defined for service tests
// (those are package-private), so the HTTP tests don't get a log spam fountain.
type silentLogger struct{}

func (silentLogger) Debug(string, ...any)  {}
func (silentLogger) Info(string, ...any)   {}
func (silentLogger) Warn(string, ...any)   {}
func (silentLogger) Error(string, ...any)  {}
func (silentLogger) Fatal(string, ...any)  {}
func (silentLogger) Debugf(string, ...any) {}
func (silentLogger) Infof(string, ...any)  {}
func (silentLogger) Warnf(string, ...any)  {}
func (silentLogger) Errorf(string, ...any) {}
func (silentLogger) Panicf(string, ...any) {}
func (silentLogger) Fatalf(string, ...any) {}
func (silentLogger) Sync() error           { return nil }

// repoRootPolicyPath finds the project root policy.cedar walking up from CWD.
// Same trick the cedar package's own test uses.
func repoRootPolicyPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		candidate := filepath.Join(dir, "policy.cedar")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "reached filesystem root without finding policy.cedar")
		dir = parent
	}
}

// newRealAuthz constructs the real Cedar adapter loaded from the project's
// real policy.cedar so authz behaves exactly like production.
func newRealAuthz(t *testing.T) port.Authz {
	t.Helper()
	return cedar.NewCedarAuthz(silentLogger{}, repoRootPolicyPath(t))
}

// ---- AuthUser helpers ----

func ownerUser() port.AuthUser {
	return port.NewAuthUser("org_1", "user_1", "owner@example.com", []port.UserRole{port.RoleOwner})
}

func memberUser() port.AuthUser {
	return port.NewAuthUser("org_1", "user_2", "member@example.com", []port.UserRole{port.RoleMember})
}

func adminUser() port.AuthUser {
	return port.NewAuthUser("org_1", "user_admin", "admin@example.com", []port.UserRole{port.RoleAdmin})
}

// supportUser has no permit rule in policy.cedar; useful to drive the
// authz-denied paths without inventing fake roles.
func supportUser() port.AuthUser {
	return port.NewAuthUser("org_1", "user_sup", "sup@example.com", []port.UserRole{port.RoleSupport})
}

// ---- fakeAuthenticator ----

// fakeAuthenticator implements port.Authenticator. The user it returns can be
// switched mid-test via setUser; the zero value returns an "unauthenticated"
// error so tests that don't call setUser exercise the 401 path.
type fakeAuthenticator struct {
	mu  sync.Mutex
	u   port.AuthUser
	err error
}

func (f *fakeAuthenticator) setUser(u port.AuthUser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.u = u
	f.err = nil
}

func (f *fakeAuthenticator) setError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.u = port.AuthUser{}
	f.err = err
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, _ string) (port.AuthUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.u, f.err
	}
	if f.u.Id == "" {
		// Default to onboarding-required so /api/organizations bypass path is
		// also exercisable. Other paths still see this as an auth error.
		return port.AuthUser{}, lib.NewCustomError(lib.AuthenticationError, "no user", nil)
	}
	return f.u, nil
}

// authzStub is a static port.Authz used when a test deliberately wants to
// bypass the cedar/policy verdict (e.g. for service-error paths). The real
// adapter is preferred — this only exists for surgical injection of denial.
type authzStub struct{ allow bool }

func (s authzStub) Enforce(port.AuthUser, port.Action, string) bool { return s.allow }

// ---- HTTP plumbing helpers ----

// fixedAuthMiddleware skips Fuego's per-handler authn logic and shoves an
// AuthUser directly onto the request context. Used by tests that drive a
// single handler without the full BuildServer stack.
func fixedAuthMiddleware(user port.AuthUser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.AuthUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// testSrv bundles a fuego server with the slice of global middlewares the
// test wants run. doJSON / doRaw take this bundle so tests don't have to
// re-pass the middleware slice on every call.
type testSrv struct {
	srv *fuego.Server
	mws []func(http.Handler) http.Handler
}

// newTestServer returns a fresh fuego server with the project's error
// serializer wired in and an /api group ready to accept handler routes.
// The optional middleware list is stored on the returned testSrv so the
// driver helpers can compose it around the mux (fuego.NewServer keeps the
// middleware list internal — `s.Mux` skips it, so we re-wrap manually).
func newTestServer(mws ...func(http.Handler) http.Handler) *testSrv {
	opts := []fuego.ServerOption{
		fuego.WithErrorSerializer(ApiErrorSerializer),
		// Mirror BuildServer: pass our ApiError envelope through untouched so
		// the serializer renders the full {code,message,details}. See the note
		// on PassThroughApiError / internal/config/server.go.
		fuego.WithEngineOptions(fuego.WithErrorHandler(PassThroughApiError)),
		fuego.WithoutStartupMessages(),
	}
	s := fuego.NewServer(opts...)
	return &testSrv{srv: s, mws: mws}
}

// api returns the /api sub-server for route registration.
func (t *testSrv) api() *fuego.Server {
	return apiGroup(t.srv)
}

// apiGroup returns the /api sub-router so handlers register the same path
// prefix used in production wiring (BuildServer also creates this group).
func apiGroup(s *fuego.Server) *fuego.Server {
	return fuego.Group(s, "/api")
}

// asHandler returns the server's outer http.Handler — Mux wrapped by the
// global middlewares, exactly as Fuego does at Run() time. Tests must drive
// this and NOT s.Mux directly, otherwise the global middlewares (including
// our fixedAuthMiddleware that injects the AuthUser onto the request ctx)
// are skipped.
func asHandler(s *fuego.Server, mws []func(http.Handler) http.Handler) http.Handler {
	var h http.Handler = s.Mux
	for _, m := range mws {
		h = m(h)
	}
	return h
}

// doJSON encodes body as JSON and runs a single request through the server
// returning the recorder. body == nil means no body. The server's global
// middlewares are composed around its mux first so the AuthUser-injecting
// fixedAuthMiddleware actually runs before the route handler.
func doJSON(t *testing.T, ts *testSrv, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)
	return rec
}

// doRaw runs an arbitrary raw-body request through the server. Used for the
// webhook endpoint that takes unparsed bytes (PostStd).
func doRaw(t *testing.T, ts *testSrv, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)
	return rec
}

// decodeJSON unmarshals the recorder body into out and asserts no error.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
}

// ---- Generic fake ports ----

// recordingPubSub captures every Publish and stashes Subscribe handlers so
// tests can drive topic→handler delivery. Modeled on the service-test version
// but lives here so the HTTP package can use it.
type recordingPubSub struct {
	mu         sync.Mutex
	published  []publishedEvent
	handlers   map[string]func(topic string, data []byte)
	subErr     error
	publishErr error
}

type publishedEvent struct {
	OrgId   string
	Topic   string
	Message any
}

func newPubSub() *recordingPubSub {
	return &recordingPubSub{handlers: map[string]func(string, []byte){}}
}

func (p *recordingPubSub) Publish(orgId, topic string, message any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.publishErr != nil {
		return p.publishErr
	}
	p.published = append(p.published, publishedEvent{OrgId: orgId, Topic: topic, Message: message})
	return nil
}

func (p *recordingPubSub) Subscribe(topic string, handler func(string, []byte)) (port.PubSubSubscription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.subErr != nil {
		return nil, p.subErr
	}
	p.handlers[topic] = handler
	return noopSub{}, nil
}

func (p *recordingPubSub) Close() error { return nil }

func (p *recordingPubSub) topicsPublished() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, 0, len(p.published))
	for _, e := range p.published {
		out = append(out, e.Topic)
	}
	return out
}

type noopSub struct{}

func (noopSub) Unsubscribe() error { return nil }

// noopScheduler implements port.Scheduler without ever running tasks. Avoids
// goroutine leaks from real cron implementations.
type noopScheduler struct{}

func (noopScheduler) ScheduleTask(string, func()) error { return nil }

// noopTxManager runs fn inline. Mirrors the service-test variant — handler
// tests only care that the fn was given a chance to commit.
type noopTxManager struct{}

func (noopTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// noopCoupons / noopInvoicing satisfy service.OrderCoupons / service.OrderInvoicing
// for handler tests that don't exercise coupons or invoicing but must wire a
// real (non-nil) dependency. noopInvoicing.BuildForOrder returns port.ErrNotFound
// (order with nothing to invoice), preserving the old "no invoice" behaviour.
type noopCoupons struct{}

func (noopCoupons) Reserve(ctx context.Context, in service.ReserveInput) (domain.CouponReservation, error) {
	return domain.CouponReservation{}, nil
}

func (noopCoupons) Consume(ctx context.Context, in service.ConsumeInput) (domain.Discount, error) {
	return domain.Discount{}, nil
}

type noopInvoicing struct{}

func (noopInvoicing) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopInvoicing) SettleOrderInvoice(ctx context.Context, orgId, invoiceId string) error {
	return nil
}

// noopBillingInvoicing satisfies service.BillingInvoicing for handler tests that
// don't exercise per-cycle invoicing but must wire a real (non-nil) dependency.
// FindCurrentCycle returns port.ErrNotFound, preserving the old "no invoice" no-op.
type noopBillingInvoicing struct{}

func (noopBillingInvoicing) BuildForBillingPeriod(ctx context.Context, sub domain.Subscription) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) FindCurrentCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopBillingInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) MarkUncollectible(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) Void(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

// recordingEngine is an inert port.Engine implementation, recording the calls
// the orchestration services make through it.
type recordingEngine struct {
	mu         sync.Mutex
	started    []domain.Subscription
	updateName []string
	cancelled  []domain.Subscription
	startErr   error
	updateErr  error
	cancelErr  error
}

func (e *recordingEngine) StartWorkflow(context.Context, port.WorkflowType, any) (port.WorkflowResult, error) {
	return port.WorkflowResult{Success: true}, nil
}

func (e *recordingEngine) StartSubscriptionWorkflow(_ context.Context, sub domain.Subscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.startErr != nil {
		return e.startErr
	}
	e.started = append(e.started, sub)
	return nil
}

func (e *recordingEngine) UpdateSubscriptionWorkflow(_ context.Context, name string, _ domain.Subscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.updateErr != nil {
		return e.updateErr
	}
	e.updateName = append(e.updateName, name)
	return nil
}

func (e *recordingEngine) CancelSubscriptionWorkflow(_ context.Context, sub domain.Subscription) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancelErr != nil {
		return e.cancelErr
	}
	e.cancelled = append(e.cancelled, sub)
	return nil
}

func (e *recordingEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	return nil
}

// recordingDunningEngine satisfies port.DunningEngine.
type recordingDunningEngine struct {
	mu        sync.Mutex
	started   []port.StartDunningWorkflowInput
	signals   []string
	cancelled []domain.DunningCampaign
	startErr  error
}

func (e *recordingDunningEngine) StartDunningWorkflow(_ context.Context, in port.StartDunningWorkflowInput) (string, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.startErr != nil {
		return "", "", e.startErr
	}
	e.started = append(e.started, in)
	return "wf_1", "run_1", nil
}

func (e *recordingDunningEngine) SignalDunningWorkflow(_ context.Context, signal string, _ domain.DunningCampaign, _ any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.signals = append(e.signals, signal)
	return nil
}

func (e *recordingDunningEngine) CancelDunningWorkflow(_ context.Context, c domain.DunningCampaign) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cancelled = append(e.cancelled, c)
	return nil
}

// ---- Repository fakes ----
//
// Each fake satisfies its port via embedding so any method we forgot panics
// loudly instead of silently returning a zero value. Test files override the
// specific methods they exercise.

// fakeCustomerRepo holds the subset of CustomerRepository handlers exercise.
type fakeCustomerRepo struct {
	port.CustomerRepository
	byId          domain.Customer
	byIdErr       error
	byEmail       domain.Customer
	byEmailErr    error
	created       []domain.Customer
	listResult    []domain.Customer
	listTotal     int
	listErr       error
	createErr     error
	cohorts       []domain.Cohort
	cohortErr     error
	paymentMethod domain.PaymentMethod
	pmErr         error
}

func (r *fakeCustomerRepo) FindById(context.Context, string, string) (domain.Customer, error) {
	if r.byIdErr != nil {
		return domain.Customer{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeCustomerRepo) FindByEmail(context.Context, string, string) (domain.Customer, error) {
	if r.byEmailErr != nil {
		return domain.Customer{}, r.byEmailErr
	}
	return r.byEmail, nil
}

func (r *fakeCustomerRepo) Create(_ context.Context, c domain.Customer) (domain.Customer, error) {
	if r.createErr != nil {
		return domain.Customer{}, r.createErr
	}
	r.created = append(r.created, c)
	return c, nil
}

func (r *fakeCustomerRepo) List(context.Context, string, domain.Pagination) ([]domain.Customer, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	total := r.listTotal
	if total == 0 {
		total = len(r.listResult)
	}
	return r.listResult, total, nil
}

func (r *fakeCustomerRepo) FindByIds(_ context.Context, _ string, ids []string) ([]domain.Customer, error) {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []domain.Customer
	if want[r.byId.Id] {
		out = append(out, r.byId)
	}
	return out, nil
}

func (r *fakeCustomerRepo) FindPaymentMethodById(context.Context, string, string) (domain.PaymentMethod, error) {
	if r.pmErr != nil {
		return domain.PaymentMethod{}, r.pmErr
	}
	return r.paymentMethod, nil
}

func (r *fakeCustomerRepo) CreateCohort(_ context.Context, c domain.Cohort) (domain.Cohort, error) {
	if r.cohortErr != nil {
		return domain.Cohort{}, r.cohortErr
	}
	r.cohorts = append(r.cohorts, c)
	return c, nil
}

// fakePaymentMethodRepo satisfies port.PaymentMethodRepository.
type fakePaymentMethodRepo struct {
	port.PaymentMethodRepository
	byId    domain.PaymentMethod
	byIdErr error
	created []domain.PaymentMethod
	updated []domain.PaymentMethod

	// Capture of the last FindById args, so handler tests can prove the
	// org is taken from the authenticated user and the id from the path.
	lastFindOrg string
	lastFindId  string
}

func (r *fakePaymentMethodRepo) FindById(_ context.Context, orgId, id string) (domain.PaymentMethod, error) {
	r.lastFindOrg = orgId
	r.lastFindId = id
	if r.byIdErr != nil {
		return domain.PaymentMethod{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakePaymentMethodRepo) Create(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	r.created = append(r.created, pm)
	return pm, nil
}

func (r *fakePaymentMethodRepo) Update(_ context.Context, pm domain.PaymentMethod) (domain.PaymentMethod, error) {
	r.updated = append(r.updated, pm)
	return pm, nil
}

func (r *fakePaymentMethodRepo) FindExpiringPaymentMethods(context.Context, time.Time) ([]domain.PaymentMethod, error) {
	return nil, nil
}

// fakeCartRepo satisfies port.CartRepository.
type fakeCartRepo struct {
	port.CartRepository
	cart      domain.Cart
	findErr   error
	created   []domain.Cart
	updated   []domain.Cart
	createErr error
}

func (r *fakeCartRepo) FindById(context.Context, string, string) (domain.Cart, error) {
	if r.findErr != nil {
		return domain.Cart{}, r.findErr
	}
	return r.cart, nil
}

func (r *fakeCartRepo) Create(_ context.Context, c domain.Cart) (domain.Cart, error) {
	if r.createErr != nil {
		return domain.Cart{}, r.createErr
	}
	if c.Id == "" {
		c.Id = "cart_generated"
	}
	r.created = append(r.created, c)
	r.cart = c
	return c, nil
}

func (r *fakeCartRepo) Update(_ context.Context, c domain.Cart) (domain.Cart, error) {
	r.updated = append(r.updated, c)
	r.cart = c
	return c, nil
}

// fakeProductRepo satisfies port.ProductRepository.
type fakeProductRepo struct {
	port.ProductRepository
	byId       domain.Product
	byIdErr    error
	created    []domain.Product
	listResult []domain.Product
	listTotal  int
	listErr    error
	updated    []domain.Product
	deleted    []string
	deleteErr  error
}

func (r *fakeProductRepo) FindById(context.Context, string, string) (domain.Product, error) {
	if r.byIdErr != nil {
		return domain.Product{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeProductRepo) Create(_ context.Context, p domain.Product) (domain.Product, error) {
	r.created = append(r.created, p)
	// Single-product store: the service re-reads via FindById after create.
	r.byId = p
	return p, nil
}

func (r *fakeProductRepo) Find(_ context.Context, _ string, _ domain.Pagination, statuses []domain.ProductStatus) ([]domain.Product, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	out := r.listResult
	if len(statuses) > 0 {
		out = nil
		for _, p := range r.listResult {
			for _, st := range statuses {
				if p.Status == st {
					out = append(out, p)
					break
				}
			}
		}
	}
	total := r.listTotal
	if total == 0 {
		total = len(out)
	}
	return out, total, nil
}

func (r *fakeProductRepo) Update(_ context.Context, p domain.Product) (domain.Product, error) {
	r.updated = append(r.updated, p)
	// Behave like a single-product store so a subsequent FindById (e.g. the
	// GetDetails call after archive/unarchive) reflects the mutation.
	r.byId = p
	return p, nil
}

func (r *fakeProductRepo) Delete(_ context.Context, _ string, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = append(r.deleted, id)
	return nil
}

// fakeVariantRepo satisfies port.VariantRepository.
type fakeVariantRepo struct {
	port.VariantRepository
	byId       domain.Variant
	byIdErr    error
	created    []domain.Variant
	listResult []domain.Variant
	listTotal  int
}

func (r *fakeVariantRepo) FindById(context.Context, string, string) (domain.Variant, error) {
	if r.byIdErr != nil {
		return domain.Variant{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeVariantRepo) Create(_ context.Context, v domain.Variant) (domain.Variant, error) {
	r.created = append(r.created, v)
	return v, nil
}

func (r *fakeVariantRepo) FindByProductId(context.Context, string, string, domain.Pagination) ([]domain.Variant, int, error) {
	total := r.listTotal
	if total == 0 {
		total = len(r.listResult)
	}
	return r.listResult, total, nil
}

func (r *fakeVariantRepo) Update(_ context.Context, v domain.Variant) (domain.Variant, error) {
	return v, nil
}
func (r *fakeVariantRepo) Delete(context.Context, string, string) error { return nil }

// fakePriceRepo satisfies port.PriceRepository.
type fakePriceRepo struct {
	port.PriceRepository
	byId       domain.Price
	byIdErr    error
	created    []domain.Price
	listResult []domain.Price
	listTotal  int
}

func (r *fakePriceRepo) FindById(context.Context, string, string) (domain.Price, error) {
	if r.byIdErr != nil {
		return domain.Price{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakePriceRepo) Create(_ context.Context, p domain.Price) (domain.Price, error) {
	r.created = append(r.created, p)
	return p, nil
}

func (r *fakePriceRepo) FindByVariantId(context.Context, string, string, domain.Pagination) ([]domain.Price, int, error) {
	total := r.listTotal
	if total == 0 {
		total = len(r.listResult)
	}
	return r.listResult, total, nil
}

func (r *fakePriceRepo) Update(_ context.Context, p domain.Price) (domain.Price, error) {
	return p, nil
}
func (r *fakePriceRepo) Delete(context.Context, string, string) error { return nil }

func (r *fakePriceRepo) FindByIds(_ context.Context, _ string, ids []string) ([]domain.Price, error) {
	// Match against the canned byId / listResult.
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []domain.Price
	if want[r.byId.Id] {
		out = append(out, r.byId)
	}
	for _, p := range r.listResult {
		if want[p.Id] {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePriceRepo) FindByVariantIds(context.Context, string, []string) ([]domain.Price, error) {
	return r.listResult, nil
}

// fakeSessionRepo satisfies port.SessionRepository.
type fakeSessionRepo struct {
	port.SessionRepository
	byId      domain.Session
	byIdErr   error
	created   []domain.Session
	createErr error
}

func (r *fakeSessionRepo) FindById(context.Context, string, string) (domain.Session, error) {
	if r.byIdErr != nil {
		return domain.Session{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeSessionRepo) Create(_ context.Context, s domain.Session) (domain.Session, error) {
	if r.createErr != nil {
		return domain.Session{}, r.createErr
	}
	r.created = append(r.created, s)
	return s, nil
}

// fakeOrgRepo satisfies port.OrgRepository.
type fakeOrgRepo struct {
	port.OrgRepository
	created   []domain.Org
	createErr error
}

func (r *fakeOrgRepo) Create(_ context.Context, o domain.Org) (domain.Org, error) {
	if r.createErr != nil {
		return domain.Org{}, r.createErr
	}
	r.created = append(r.created, o)
	return o, nil
}

// fakePspRepo satisfies port.PspRepository.
type fakePspRepo struct {
	port.PspRepository
	byId    domain.PspConfig
	byIdErr error
	created []domain.PspConfig
}

func (r *fakePspRepo) FindById(context.Context, string, string) (domain.PspConfig, error) {
	if r.byIdErr != nil {
		return domain.PspConfig{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakePspRepo) Create(_ context.Context, c domain.PspConfig) (domain.PspConfig, error) {
	r.created = append(r.created, c)
	return c, nil
}

// fakeSettingRepo satisfies port.SettingRepository.
type fakeSettingRepo struct {
	port.SettingRepository
	byId      domain.Setting
	byIdErr   error
	created   []domain.Setting
	createErr error
}

func (r *fakeSettingRepo) FindById(context.Context, string, string, string) (domain.Setting, error) {
	if r.byIdErr != nil {
		return domain.Setting{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeSettingRepo) Create(_ context.Context, s domain.Setting) (domain.Setting, error) {
	if r.createErr != nil {
		return domain.Setting{}, r.createErr
	}
	r.created = append(r.created, s)
	return s, nil
}

// fakeSecretCipher is a reversible stand-in for port.SecretCipher that still
// enforces the (orgId, id) binding the real AES-GCM AAD provides.
type fakeSecretCipher struct{}

func (fakeSecretCipher) Encrypt(orgId, id string, plaintext []byte) (string, error) {
	return "enc[" + orgId + ":" + id + "]" + string(plaintext), nil
}

func (fakeSecretCipher) Decrypt(orgId, id string, envelope string) ([]byte, error) {
	prefix := "enc[" + orgId + ":" + id + "]"
	if !strings.HasPrefix(envelope, prefix) {
		return nil, errors.New("envelope failed authentication")
	}
	return []byte(strings.TrimPrefix(envelope, prefix)), nil
}

// fakeApiKeyRepo satisfies port.ApiKeyRepository.
type fakeApiKeyRepo struct {
	port.ApiKeyRepository
	created []domain.ApiKey
}

func (r *fakeApiKeyRepo) Create(_ context.Context, k domain.ApiKey) (domain.ApiKey, error) {
	r.created = append(r.created, k)
	return k, nil
}

// fakeMetadataRepo satisfies port.MetadataStoreRepository (empty surface
// suffices for what the org constructor invokes).
type fakeMetadataRepo struct {
	port.MetadataStoreRepository
}

// fakeAuthProvider satisfies port.AuthProvider with a do-nothing impl. The
// org service skips it entirely when Owner.Id is "".
type fakeAuthProvider struct {
	port.AuthProvider
}

func (fakeAuthProvider) CreateOrg(context.Context, domain.Org, string) (port.CreateOrgResponse, error) {
	return port.CreateOrgResponse{ExternalId: "ext_1"}, nil
}

// fakeOrderRepo satisfies port.OrderRepository.
type fakeOrderRepo struct {
	port.OrderRepository
	byId       domain.Order
	byIdErr    error
	created    []domain.Order
	updated    []domain.Order
	listResult []domain.Order
	listTotal  int
	listErr    error
	items      []domain.OrderItem
}

func (r *fakeOrderRepo) FindById(context.Context, string, string) (domain.Order, error) {
	if r.byIdErr != nil {
		return domain.Order{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeOrderRepo) Create(_ context.Context, o domain.Order) (domain.Order, error) {
	r.created = append(r.created, o)
	return o, nil
}

func (r *fakeOrderRepo) Update(_ context.Context, o domain.Order) (domain.Order, error) {
	r.updated = append(r.updated, o)
	r.byId = o
	return o, nil
}

func (r *fakeOrderRepo) SetPaymentSession(_ context.Context, _, _ string, session any) error {
	r.byId.PaymentSession = session
	return nil
}

func (r *fakeOrderRepo) Find(context.Context, string, domain.Pagination) ([]domain.Order, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	total := r.listTotal
	if total == 0 {
		total = len(r.listResult)
	}
	return r.listResult, total, nil
}

func (r *fakeOrderRepo) FindOrderItemById(context.Context, string, string) (domain.OrderItem, error) {
	return domain.OrderItem{}, nil
}

func (r *fakeOrderRepo) CreateOrderItem(_ context.Context, oi domain.OrderItem) (domain.OrderItem, error) {
	return oi, nil
}

func (r *fakeOrderRepo) UpdateOrderItem(_ context.Context, oi domain.OrderItem) (domain.OrderItem, error) {
	return oi, nil
}

func (r *fakeOrderRepo) FindOrderItemsByOrderId(context.Context, string, string) ([]domain.OrderItem, error) {
	return r.items, nil
}

func (r *fakeOrderRepo) FindOrderItemsBySubscriptionId(context.Context, string, string) ([]domain.OrderItem, error) {
	return r.items, nil
}

// fakeSubRepo satisfies port.SubscriptionRepository.
type fakeSubRepo struct {
	port.SubscriptionRepository
	byId       domain.Subscription
	byIdErr    error
	byOrderId  []domain.Subscription
	created    []domain.Subscription
	updated    []domain.Subscription
	listResult []domain.Subscription
	listTotal  int
	listErr    error
}

func (r *fakeSubRepo) FindById(context.Context, string, string) (domain.Subscription, error) {
	if r.byIdErr != nil {
		return domain.Subscription{}, r.byIdErr
	}
	return r.byId, nil
}

func (r *fakeSubRepo) FindByIdForUpdate(ctx context.Context, orgId, id string) (domain.Subscription, error) {
	return r.FindById(ctx, orgId, id)
}

func (r *fakeSubRepo) Create(_ context.Context, s domain.Subscription) (domain.Subscription, error) {
	r.created = append(r.created, s)
	return s, nil
}

func (r *fakeSubRepo) Update(_ context.Context, s domain.Subscription) (domain.Subscription, error) {
	r.updated = append(r.updated, s)
	r.byId = s
	return s, nil
}

func (r *fakeSubRepo) FindByOrderId(context.Context, string, string) ([]domain.Subscription, error) {
	return r.byOrderId, nil
}

func (r *fakeSubRepo) Find(context.Context, string, domain.Pagination) ([]domain.Subscription, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	total := r.listTotal
	if total == 0 {
		total = len(r.listResult)
	}
	return r.listResult, total, nil
}

// fakePaymentRepo satisfies port.PaymentRepository.
type fakePaymentRepo struct {
	port.PaymentRepository
	bySub      []domain.Payment
	bySubTotal int
	created    []domain.Payment
}

func (r *fakePaymentRepo) FindBySubscriptionId(context.Context, string, string, domain.Pagination) ([]domain.Payment, int, error) {
	total := r.bySubTotal
	if total == 0 {
		total = len(r.bySub)
	}
	return r.bySub, total, nil
}

func (r *fakePaymentRepo) Create(_ context.Context, p domain.Payment) (domain.Payment, error) {
	r.created = append(r.created, p)
	return p, nil
}

// fakeWhSubRepo satisfies port.WebhookSubscriptionRepository.
type fakeWhSubRepo struct {
	port.WebhookSubscriptionRepository
	created   []domain.WebhookSubscription
	createErr error
}

func (r *fakeWhSubRepo) Create(_ context.Context, w domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	if r.createErr != nil {
		return domain.WebhookSubscription{}, r.createErr
	}
	r.created = append(r.created, w)
	return w, nil
}

// fakeIdempRepo satisfies port.IdempotencyKeyRepository for the new
// Claim/Release contract. `exists` controls who wins the claim: when
// true, this fake represents a sibling delivery that already owns the
// row, so Claim returns false (and the caller short-circuits).
type fakeIdempRepo struct {
	port.IdempotencyKeyRepository
	exists    bool
	existsErr error
	created   []string
	released  []string
}

func (r *fakeIdempRepo) Claim(_ context.Context, key string, _ time.Time) (bool, error) {
	if r.existsErr != nil {
		return false, r.existsErr
	}
	if r.exists {
		return false, nil
	}
	r.created = append(r.created, key)
	return true, nil
}

func (r *fakeIdempRepo) Release(_ context.Context, key string) error {
	r.released = append(r.released, key)
	return nil
}

// fakeDunningRepo satisfies port.DunningRepository for handlers' needs.
type fakeDunningRepo struct {
	port.DunningRepository
	campaign        domain.DunningCampaign
	campaignErr     error
	listCampaigns   []domain.DunningCampaign
	listAttempts    []domain.DunningAttempt
	listComms       []domain.DunningCommunication
	listConfigs     []domain.DunningConfiguration
	cfg             domain.DunningConfiguration
	cfgErr          error
	token           domain.PaymentUpdateToken
	tokenErr        error
	history         domain.CustomerDunningHistory
	historyErr      error
	createdCfg      []domain.DunningConfiguration
	updatedCfg      []domain.DunningConfiguration
	createdToken    []domain.PaymentUpdateToken
	updatedCampaign []domain.DunningCampaign
}

func (r *fakeDunningRepo) FindCampaignById(context.Context, string, string) (domain.DunningCampaign, error) {
	if r.campaignErr != nil {
		return domain.DunningCampaign{}, r.campaignErr
	}
	return r.campaign, nil
}

func (r *fakeDunningRepo) FindCampaigns(context.Context, string, domain.Pagination) ([]domain.DunningCampaign, int, error) {
	return r.listCampaigns, len(r.listCampaigns), nil
}

func (r *fakeDunningRepo) FindActiveCampaignForSubscription(context.Context, string, string) (domain.DunningCampaign, error) {
	return r.campaign, r.campaignErr
}

func (r *fakeDunningRepo) UpdateCampaign(_ context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	r.updatedCampaign = append(r.updatedCampaign, c)
	r.campaign = c
	return c, nil
}

func (r *fakeDunningRepo) FindAttemptsByCampaignId(context.Context, string, string, domain.Pagination) ([]domain.DunningAttempt, int, error) {
	return r.listAttempts, len(r.listAttempts), nil
}

func (r *fakeDunningRepo) FindCommunicationsByCampaignId(context.Context, string, string, domain.Pagination) ([]domain.DunningCommunication, int, error) {
	return r.listComms, len(r.listComms), nil
}

func (r *fakeDunningRepo) FindConfigurations(context.Context, string, domain.Pagination) ([]domain.DunningConfiguration, int, error) {
	return r.listConfigs, len(r.listConfigs), nil
}

func (r *fakeDunningRepo) FindConfigurationById(context.Context, string, string) (domain.DunningConfiguration, error) {
	if r.cfgErr != nil {
		return domain.DunningConfiguration{}, r.cfgErr
	}
	return r.cfg, nil
}

func (r *fakeDunningRepo) CreateConfiguration(_ context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	r.createdCfg = append(r.createdCfg, c)
	r.cfg = c
	return c, nil
}

func (r *fakeDunningRepo) UpdateConfiguration(_ context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	r.updatedCfg = append(r.updatedCfg, c)
	r.cfg = c
	return c, nil
}

func (r *fakeDunningRepo) FindTokenById(context.Context, string, string) (domain.PaymentUpdateToken, error) {
	if r.tokenErr != nil {
		return domain.PaymentUpdateToken{}, r.tokenErr
	}
	return r.token, nil
}

func (r *fakeDunningRepo) CreateToken(_ context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	r.createdToken = append(r.createdToken, t)
	r.token = t
	return t, nil
}

func (r *fakeDunningRepo) UpdateToken(_ context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	r.token = t
	return t, nil
}

func (r *fakeDunningRepo) GetCustomerDunningHistory(context.Context, string, string) (domain.CustomerDunningHistory, error) {
	if r.historyErr != nil {
		return domain.CustomerDunningHistory{}, r.historyErr
	}
	return r.history, nil
}

func (r *fakeDunningRepo) FindConfigurationsByPriority(context.Context, string) ([]domain.DunningConfiguration, error) {
	return r.listConfigs, nil
}

// assertErrorEnvelope verifies the standard ApiError envelope shape with the
// supplied code and a substring of the message field. Reused by many handler
// tests for the validation/authz/service-error paths.
func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) ApiError {
	t.Helper()
	require.Equal(t, wantStatus, rec.Code, "body=%s", rec.Body.String())
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got ApiError
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got), "body=%s", rec.Body.String())
	require.Equal(t, wantCode, got.Code, "body=%s", rec.Body.String())
	return got
}
