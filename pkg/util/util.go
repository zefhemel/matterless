package util

func StringSliceContains(slice []string, s string) bool {
	for _, a := range slice {
		if a == s {
			return true
		}
	}
	return false
}

func ReverseStringSlice(ss []string) []string {
	ss2 := make([]string, 0, len(ss))
	for i := len(ss) - 1; i >= 0; i-- {
		ss2 = append(ss2, ss[i])
	}
	return ss2
}
