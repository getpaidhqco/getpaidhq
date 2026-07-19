package ids

import "github.com/segmentio/ksuid"

func Generate(resource string) string {
	switch resource {
	case "user":
		return "usr_" + ksuid.New().String()
	case "order":
		return "ord_" + ksuid.New().String()
	case "order_item":
		return "item_" + ksuid.New().String()
	case "org":
		return "org_" + ksuid.New().String()
	case "customer":
		return "cus_" + ksuid.New().String()
	case "cartitem":
		return "ci_" + ksuid.New().String()
	case "payment_method":
		return "pm_" + ksuid.New().String()
	case "dunning_campaign":
		return "dcam_" + ksuid.New().String()
	case "dunning_attempt":
		return "datt_" + ksuid.New().String()
	case "dunning_communication":
		return "dcom_" + ksuid.New().String()
	case "dunning_configuration":
		return "dcfg_" + ksuid.New().String()
	case "payment_update_token":
		return "tok_" + ksuid.New().String()
	case "coupon":
		return "coup_" + ksuid.New().String()
	default:
		return resource + "_" + ksuid.New().String()
	}

}
