package lib

import "github.com/segmentio/ksuid"

func GenerateId(resource string) string {
	switch resource {
	case "user":
		return "usr_" + ksuid.New().String()
	case "order":
		return "ord_" + ksuid.New().String()
	case "account":
		return "acct_" + ksuid.New().String()
	case "customer":
		return "cus_" + ksuid.New().String()
	default:
		return ksuid.New().String()
	}

}
