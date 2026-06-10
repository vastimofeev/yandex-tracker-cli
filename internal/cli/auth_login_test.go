package cli

import (
	"testing"

	"github.com/vastimofeev/yandex-tracker-cli/internal/model"
)

func TestInferOrgIDFromLoovTeamEmail(t *testing.T) {
	t.Parallel()

	orgID, ok := inferOrgIDFromUser(&model.User{Email: "user@loov.team"})
	if !ok {
		t.Fatal("inferOrgIDFromUser() ok = false, want true")
	}
	if orgID != loovTeamOrgID {
		t.Fatalf("orgID = %q, want %q", orgID, loovTeamOrgID)
	}
}

func TestInferOrgIDFromLoovTeamLoginFallback(t *testing.T) {
	t.Parallel()

	orgID, ok := inferOrgIDFromUser(&model.User{Login: "user@loov.team"})
	if !ok {
		t.Fatal("inferOrgIDFromUser() ok = false, want true")
	}
	if orgID != loovTeamOrgID {
		t.Fatalf("orgID = %q, want %q", orgID, loovTeamOrgID)
	}
}

func TestInferOrgIDIgnoresOtherDomains(t *testing.T) {
	t.Parallel()

	orgID, ok := inferOrgIDFromUser(&model.User{Email: "user@example.com"})
	if ok {
		t.Fatalf("inferOrgIDFromUser() ok = true, orgID = %q; want false", orgID)
	}
}
