package itst

import (
	"strings"
	"testing"
)

func RunTest(testNames ...string) func(*testing.T) {
	var subTestName = strings.NewReplacer(" ", "_")

	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			if testName == "" || testName == "*" {
				return
			}

			want := subTestName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				return
			}
		}
		t.SkipNow()
	}
}

func SkipTest(testNames ...string) func(*testing.T) {
	var subTestName = strings.NewReplacer(" ", "_")

	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			want := subTestName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				t.SkipNow()
			}
		}
	}
}

func DoNotTest(testSuffixes ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		idx := strings.LastIndex(t.Name(), ".")
		for _, suffix := range testSuffixes {
			if t.Name() == t.Name()[:idx]+"."+suffix {
				t.SkipNow()
			}
		}
	}
}
