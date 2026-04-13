package cedar

import (
	"github.com/cedar-policy/cedar-go"
	"io"
	"os"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type CedarAuthz struct {
	logger    port.Logger
	policySet *cedar.PolicySet
	entities  cedar.EntityMap
}

func NewCedarAuthz(logger port.Logger, env lib.Env) port.Authz {
	config, err := openConfig(env.CedarPolicyFile)
	if err != nil {
		logger.Fatal("cannot read cedar policy", "error", err)
	}
	var policy cedar.Policy
	if err := policy.UnmarshalCedar(config); err != nil {
		logger.Fatal("failed to unmarshal cedar policy", "error", err)
	}

	ps := cedar.NewPolicySet()
	ps.Add("policy0", &policy)

	var entities cedar.EntityMap
	logger.Info("loaded cedar policy", "file", env.CedarPolicyFile)
	return CedarAuthz{
		logger:    logger,
		policySet: ps,
		entities:  entities,
	}
}

func (a CedarAuthz) Enforce(user port.AuthUser, action port.Action, resource string) bool {

	role := user.PrimaryRole

	req := cedar.Request{
		Principal: cedar.NewEntityUID("Role", cedar.String(role)),
		Action:    cedar.NewEntityUID("Action", cedar.String(action)),
		Resource:  cedar.NewEntityUID("Resource", cedar.String(resource)),
		Context: cedar.NewRecord(cedar.RecordMap{
			"demoRequest": cedar.True,
		}),
	}

	ok, d := a.policySet.IsAuthorized(a.entities, req)
	a.logger.Info("cedar authz result", "ok", ok, "decision", d)
	return bool(ok)
}

func openConfig(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}
