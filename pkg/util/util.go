package util

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

func TokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

var safeFilenameRE = regexp.MustCompile("[^A-Za-z0-9_]")

func SafeFilename(s string) string {
	return safeFilenameRE.ReplaceAllString(s, "_")
}

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

func FlatStringMap(m map[string][]string) map[string]string {
	m2 := map[string]string{}
	for k, vs := range m {
		if len(vs) > 0 {
			m2[k] = vs[0]
		}
	}
	return m2
}

func ListStringMap(m map[string]string) map[string][]string {
	m2 := map[string][]string{}
	for k, v := range m {
		m2[k] = []string{v}
	}
	return m2
}
