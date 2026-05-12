package workflows

import "github.com/hatchet-dev/hatchet/pkg/worker"

// The Hatchet Go SDK marks worker.WaitResult.Keys / Unmarshal as
// "internal" (SA1019), but exposes no public replacement for the
// OR-condition wait pattern that DurableContext.WaitFor returns. The
// SDK's own e2e tests use these same methods. Until upstream
// provides a stable alternative, centralise the suppression here.

func waitedKeys(r *worker.WaitResult) []string {
	if r == nil {
		return nil
	}
	return r.Keys() //nolint:staticcheck // SA1019: no public alternative for OrCondition waits.
}

func unmarshalWaited(r *worker.WaitResult, key string, dst any) error {
	return r.Unmarshal(key, dst) //nolint:staticcheck // SA1019: no public alternative for OrCondition waits.
}
