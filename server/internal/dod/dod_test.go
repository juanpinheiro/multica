package dod

import (
	"reflect"
	"testing"
)

func a(id string) Assertion       { return Assertion{ID: id} }
func r(id string, ok bool) Result { return Result{AssertionID: id, Passed: ok} }

func TestMilestoneSatisfied(t *testing.T) {
	cases := []struct {
		name       string
		assertions []Assertion
		results    []Result
		want       bool
	}{
		{"no assertions is vacuously satisfied", nil, nil, true},
		{"single passing assertion", []Assertion{a("x")}, []Result{r("x", true)}, true},
		{"single unvalidated assertion fails", []Assertion{a("x")}, nil, false},
		{"single failing assertion fails", []Assertion{a("x")}, []Result{r("x", false)}, false},
		{
			"all passing",
			[]Assertion{a("x"), a("y")},
			[]Result{r("x", true), r("y", true)},
			true,
		},
		{
			"one of two failing",
			[]Assertion{a("x"), a("y")},
			[]Result{r("x", true), r("y", false)},
			false,
		},
		{
			"missing one verdict fails",
			[]Assertion{a("x"), a("y")},
			[]Result{r("x", true)},
			false,
		},
		{
			"any failing verdict for an assertion fails it",
			[]Assertion{a("x")},
			[]Result{r("x", true), r("x", false)},
			false,
		},
		{
			"stray result for unknown assertion is ignored",
			[]Assertion{a("x")},
			[]Result{r("x", true), r("ghost", false)},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MilestoneSatisfied(c.assertions, c.results); got != c.want {
				t.Errorf("MilestoneSatisfied() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestFailedAssertions(t *testing.T) {
	cases := []struct {
		name       string
		assertions []Assertion
		results    []Result
		want       []string
	}{
		{"none failed when all pass", []Assertion{a("x")}, []Result{r("x", true)}, nil},
		{"unvalidated assertion is failed", []Assertion{a("x")}, nil, []string{"x"}},
		{
			"preserves input order of failures",
			[]Assertion{a("x"), a("y"), a("z")},
			[]Result{r("y", true)},
			[]string{"x", "z"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ids(FailedAssertions(c.assertions, c.results))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("FailedAssertions() = %v, want %v", got, c.want)
			}
		})
	}
}

func ids(as []Assertion) []string {
	if len(as) == 0 {
		return nil
	}
	out := make([]string, len(as))
	for i, x := range as {
		out[i] = x.ID
	}
	return out
}
