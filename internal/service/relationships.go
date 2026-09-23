package service

import (
	"fmt"
	"strings"
)

// CanonicalLinkRelationships lists the values accepted by
// POST /issues/{key}/links. They do NOT match GET /linktypes ids: the API
// rejects "depends", "blocks", "depends upon" etc. Probed live 2026-09-23.
var CanonicalLinkRelationships = []string{
	"relates",
	"is dependent by",
	"duplicates",
	"is duplicated by",
	"is epic of",
}

// linkAliases maps user-friendly names to canonical values.
// A trailing "!" marks direction-swapping aliases: the API has no value
// meaning "source blocks target", so the link is posted from the blocked
// issue with "is dependent by".
var linkAliases = map[string]string{
	"depends":       "is dependent by",
	"depend":        "is dependent by",
	"dependent":     "is dependent by",
	"is dependent":  "is dependent by",
	"is blocked by": "is dependent by",
	"blocked by":    "is dependent by",
	"duplicate":     "duplicates",
	"epic":          "is epic of",
	"blocks":        "is dependent by!",
	"blocker":       "is dependent by!",
	"blocking":      "is dependent by!",
	"depends upon":  "is dependent by!",
}

// parentManagedRelationships cannot be created via POST /issues/{key}/links
// in this installation; subtask links live in the issue's parent field.
var parentManagedRelationships = map[string]bool{
	"subtask":        true,
	"is subtask of":  true,
	"parent":         true,
}

// ResolvedRelationship is the normalized form of a requested link type.
type ResolvedRelationship struct {
	Value string
	Swap  bool // post the link from the target issue instead of the source
}

// ResolveLinkRelationship validates and normalizes a relationship name.
func ResolveLinkRelationship(relationship string) (ResolvedRelationship, error) {
	name := strings.TrimSpace(relationship)
	if name == "" {
		return ResolvedRelationship{}, invalidRelationshipError("")
	}

	for _, canonical := range CanonicalLinkRelationships {
		if strings.EqualFold(name, canonical) {
			return ResolvedRelationship{Value: canonical}, nil
		}
	}

	lower := strings.ToLower(name)
	if parentManagedRelationships[lower] {
		return ResolvedRelationship{}, fmt.Errorf(
			"relationship %q cannot be created via issue links; manage subtask links through the parent field (yt issue edit %s --set parent=<KEY>)",
			name, "<issue-key>")
	}
	if alias, ok := linkAliases[lower]; ok {
		if strings.HasSuffix(alias, "!") {
			return ResolvedRelationship{Value: strings.TrimSuffix(alias, "!"), Swap: true}, nil
		}
		return ResolvedRelationship{Value: alias}, nil
	}

	return ResolvedRelationship{}, invalidRelationshipError(name)
}

func invalidRelationshipError(name string) error {
	return fmt.Errorf(
		"invalid relationship %q; accepted values: %s; aliases blocks/blocker/depends upon (A blocks B) are posted as \"is dependent by\" from the blocked issue; subtask links use the parent field",
		name, strings.Join(CanonicalLinkRelationships, ", "))
}
