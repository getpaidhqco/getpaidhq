package workflows

// containsKey reports whether target appears in keys. Used by the durable
// runners' OR-condition wait branches to identify which condition fired.
func containsKey(keys []string, target string) bool {
	for _, k := range keys {
		if k == target {
			return true
		}
	}
	return false
}
