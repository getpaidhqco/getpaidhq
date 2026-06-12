package hatchet

import "time"

// Config carries the Hatchet adapter's settings. The composition root
// (internal/config/app.go) maps env vars onto this struct; the adapter never
// sees the global env.
//
// Note: the Hatchet SDK client itself reads HATCHET_CLIENT_TOKEN,
// HATCHET_CLIENT_HOST_PORT, HATCHET_CLIENT_NAMESPACE and
// HATCHET_CLIENT_TLS_STRATEGY directly from the process environment —
// HostPort/Namespace here are informational (logging) until the client is
// constructed programmatically.
type Config struct {
	HostPort  string
	Namespace string

	// BillingSweepInterval is how often the billing-sweep cron fires
	// (HATCHET_BILLING_SWEEP_INTERVAL). Normalized by workflows.SweepCadence
	// to whole minutes in [1m, 60m].
	BillingSweepInterval time.Duration
}
