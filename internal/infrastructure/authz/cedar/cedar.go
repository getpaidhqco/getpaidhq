package cedar

import (
	"github.com/cedar-policy/cedar-go"
	"io"
	"log"
	"os"
	"payloop/internal/api/authn"
	"payloop/internal/application/lib/authz"
	"payloop/internal/lib"
)

type CedarAuthz struct {
	logger    lib.Logger
	policySet *cedar.PolicySet
	entities  cedar.EntityMap
}

func NewCedarAuthz(logger lib.Logger, env lib.Env) authz.Authz {
	config, err := openConfig(env.CedarPolicyFile)
	if err != nil {
		log.Fatal("☠️ cannot read cedar policy")
	}
	var policy cedar.Policy
	if err := policy.UnmarshalCedar(config); err != nil {
		log.Fatal(err)
	}

	ps := cedar.NewPolicySet()
	ps.Add("policy0", &policy)

	var entities cedar.EntityMap
	logger.Infof("Loaded cedar policy from %s", env.CedarPolicyFile)
	return CedarAuthz{
		logger:    logger,
		policySet: ps,
		entities:  entities,
	}
}

func (a CedarAuthz) Enforce(user authn.User, action authz.Action, resource string) bool {

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
	a.logger.Infof("Cedar authz result: %v, decision: %v", ok, d)
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
