package service

import (
	"strings"
	"testing"
)

func TestResolveLinkRelationshipCanonical(t *testing.T) {
	t.Parallel()

	for _, canonical := range CanonicalLinkRelationships {
		got, err := ResolveLinkRelationship(canonical)
		if err != nil {
			t.Fatalf("canonical %q: %v", canonical, err)
		}
		if got.Value != canonical || got.Swap {
			t.Fatalf("canonical %q resolved to %+v", canonical, got)
		}
	}
}

func TestResolveLinkRelationshipAliases(t *testing.T) {
	t.Parallel()

	cases := map[string]ResolvedRelationship{
		"depends":       {Value: "is dependent by"},
		"is blocked by": {Value: "is dependent by"},
		"duplicate":     {Value: "duplicates"},
		"blocks":        {Value: "is dependent by", Swap: true},
		"depends upon":  {Value: "is dependent by", Swap: true},
		"BLOCKER":       {Value: "is dependent by", Swap: true},
	}
	for input, want := range cases {
		got, err := ResolveLinkRelationship(input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if got != want {
			t.Fatalf("%q = %+v, want %+v", input, got, want)
		}
	}
}

func TestResolveLinkRelationshipParentManaged(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"subtask", "is subtask of", "parent"} {
		_, err := ResolveLinkRelationship(input)
		if err == nil {
			t.Fatalf("%q: expected error", input)
		}
		if !strings.Contains(err.Error(), "parent field") {
			t.Fatalf("%q error = %q, want it to mention parent field", input, err)
		}
	}
}

func TestResolveLinkRelationshipInvalid(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "banana"} {
		_, err := ResolveLinkRelationship(input)
		if err == nil {
			t.Fatalf("%q: expected error", input)
		}
		if !strings.Contains(err.Error(), "is dependent by") {
			t.Fatalf("%q error = %q, want accepted values listed", input, err)
		}
	}
}
