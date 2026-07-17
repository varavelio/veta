package js

import "testing"

func BenchmarkRunnerCallSameSource(b *testing.B) {
	runner := New(WithRuntime(Runtime{"siteName": "Veta"}))
	source := Source{Name: "filter.js", Code: `
		export default function({ siteName }, input, suffix) {
			return siteName + ":" + input + ":" + suffix;
		}
	`}

	b.ReportAllocs()

	for b.Loop() {
		result, err := runner.Call(source, "hello", "world")
		if err != nil {
			b.Fatal(err)
		}
		if result.Export() != "Veta:hello:world" {
			b.Fatalf("unexpected result %v", result.Export())
		}
	}
}
