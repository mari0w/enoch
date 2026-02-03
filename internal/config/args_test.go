package config

import "testing"

func TestSplitArgsSimple(t *testing.T) {
	args, err := SplitArgs("a b c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b", "c"}
	assertEqualSlice(t, args, want)
}

func TestSplitArgsQuotes(t *testing.T) {
	args, err := SplitArgs("a \"b c\" 'd e'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b c", "d e"}
	assertEqualSlice(t, args, want)
}

func TestSplitArgsEscapes(t *testing.T) {
	args, err := SplitArgs("a\\ b \"c\\\"d\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a b", "c\"d"}
	assertEqualSlice(t, args, want)
}

func TestSplitArgsErrors(t *testing.T) {
	_, err := SplitArgs("\"")
	if err == nil {
		t.Fatalf("expected unterminated quote error")
	}
	_, err = SplitArgs("abc\\")
	if err == nil {
		t.Fatalf("expected unterminated escape error")
	}
}

func assertEqualSlice(t *testing.T, got, want []string) {
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d mismatch: got %q want %q", i, got[i], want[i])
		}
	}
}
