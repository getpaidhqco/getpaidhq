package cedar

import (
	"io"
	"log"
	"os"

	"github.com/cedar-policy/cedar-go"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type CedarAuthz struct {
	logger    port.Logger
	policySet *cedar.PolicySet
}

func NewCedarAuthz(logger port.Logger, env lib.Env) port.Authz {
	config, err := openConfig(env.CedarPolicyFile)
	if err != nil {
		log.Fatal("cannot read cedar policy")
	}

	// Parse the WHOLE document: policy.cedar declares multiple permit rules
	// (admin / owner / member). cedar.Policy.UnmarshalCedar only reads the first
	// statement, which silently dropped the owner+member rules and denied every
	// non-admin role. NewPolicySetFromBytes loads them all.
	ps, err := cedar.NewPolicySetFromBytes(env.CedarPolicyFile, config)
	if err != nil {
		log.Fatal(err)
	}

	logger.Infof("Loaded cedar policy from %s (%d rules)", env.CedarPolicyFile, len(ps.Map()))
	return CedarAuthz{
		logger:    logger,
		policySet: ps,
	}
}

func (a CedarAuthz) Enforce(user port.AuthUser, action port.Action, resource string) bool {
	role := user.PrimaryRole

	principalUID := cedar.NewEntityUID("Role", cedar.String(role))
	resourceUID := cedar.NewEntityUID("Resource", cedar.String(resource))

	// The owner/member rules guard on `principal.org_id == resource.org_id`, so
	// both entities must exist in the map and carry an org_id attribute. The
	// data layer already scopes every lookup by AuthUser.OrgId (you can only
	// read your own org's rows), and handlers don't pass the resource's owning
	// org, so we assert both org_ids as the caller's. That makes Cedar a
	// role×action gate here; cross-org isolation lives in the repositories.
	orgAttr := cedar.NewRecord(cedar.RecordMap{"org_id": cedar.String(user.OrgId)})
	entities := cedar.EntityMap{
		principalUID: cedar.Entity{UID: principalUID, Attributes: orgAttr},
		resourceUID:  cedar.Entity{UID: resourceUID, Attributes: orgAttr},
	}

	req := cedar.Request{
		Principal: principalUID,
		Action:    cedar.NewEntityUID("Action", cedar.String(action)),
		Resource:  resourceUID,
		Context:   cedar.NewRecord(cedar.RecordMap{}),
	}

	ok, _ := a.policySet.IsAuthorized(entities, req)
	a.logger.Debugf("Cedar authz: role=%s action=%s allowed=%v", role, action, bool(ok))
	return bool(ok)
}

func openConfig(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	return io.ReadAll(file)
}
