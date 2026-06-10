package gorege_test

import (
	"testing"

	"github.com/yplog/gorege"
)

func benchEngine(t testing.TB) *gorege.Engine {
	t.Helper()
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("membership", "Gold member", "Regular member", "Guest"),
			gorege.Dim("day", "Mon", "Tue", "Wed", "Thu", "Fri"),
			gorege.Dim("facility", "Swimming pool", "Gym", "Sauna"),
		),
		gorege.WithRules(
			gorege.Allow("Gold member", gorege.Wildcard, gorege.Wildcard),
			gorege.Deny("Guest", gorege.AnyOf("Mon", "Tue"), "Sauna"),
			gorege.Allow(gorege.AnyOf("Guest", "Regular member"), gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func BenchmarkCheck(b *testing.B) {
	e := benchEngine(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Check("Guest", "Wed", "Sauna")
	}
}

func BenchmarkExplain(b *testing.B) {
	e := benchEngine(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Explain("Guest", "Wed", "Sauna")
	}
}

func BenchmarkClosest(b *testing.B) {
	e, _, err := gorege.New(
		gorege.WithDimensions(
			gorege.Dim("role", "u", "v"),
			gorege.Dim("flag", "0", "1"),
		),
		gorege.WithRules(
			gorege.Deny("u", "0"),
			gorege.Allow(gorege.Wildcard, gorege.Wildcard),
		),
	)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Closest("u", "0")
	}
}

func BenchmarkNewSkipAnalysis(b *testing.B) {
	for b.Loop() {
		_, _, _ = gorege.New(
			gorege.WithAnalysisLimit(-1),
			gorege.WithDimensions(
				gorege.Dim("membership", "Gold member", "Regular member", "Guest"),
				gorege.Dim("day", "Mon", "Tue", "Wed", "Thu", "Fri"),
				gorege.Dim("facility", "Swimming pool", "Gym", "Sauna"),
			),
			gorege.WithRules(
				gorege.Allow("Gold member", gorege.Wildcard, gorege.Wildcard),
				gorege.Deny("Guest", gorege.AnyOf("Mon", "Tue"), "Sauna"),
				gorege.Allow(gorege.AnyOf("Guest", "Regular member"), gorege.Wildcard, gorege.Wildcard),
			),
		)
	}
}

func BenchmarkNewFromConfigSkipAnalysis(b *testing.B) {
	cfg := gorege.Config{
		Dimensions: []gorege.DimensionConfig{
			{Name: "membership", Values: []string{"Gold member", "Regular member", "Guest"}},
			{Name: "day", Values: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}},
			{Name: "facility", Values: []string{"Swimming pool", "Gym", "Sauna"}},
		},
		Rules: []gorege.RuleConfig{
			{Action: "ALLOW", Conditions: []any{"Gold member", "*", "*"}},
			{Action: "DENY", Conditions: []any{"Guest", []string{"Mon", "Tue"}, "Sauna"}},
			{Action: "ALLOW", Conditions: []any{[]string{"Guest", "Regular member"}, "*", "*"}},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = gorege.NewFromConfig(cfg, gorege.WithAnalysisLimit(-1))
	}
}
