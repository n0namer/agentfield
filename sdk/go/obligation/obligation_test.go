package obligation

import (
	"context"
	"errors"
	"testing"
)

func TestEvaluateReobservesAndRemainingDropsSatisfied(t *testing.T) {
	specs := []Spec{
		{
			Obligation: Obligation{ID: "commit", Kind: KindState, Status: StatusMissing},
			Validate: func(context.Context, Obligation) (Status, string, error) {
				return StatusSatisfied, "HEAD contains intended change", nil
			},
		},
		{
			Obligation: Obligation{ID: "tests", Kind: KindEvidence, Status: StatusSatisfied},
			Validate: func(context.Context, Obligation) (Status, string, error) {
				return StatusMissing, "no current-worktree test evidence", nil
			},
		},
	}

	got, err := Evaluate(context.Background(), specs)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].ID != "commit" || got[0].Status != StatusSatisfied {
		t.Fatalf("commit = %+v", got[0])
	}
	if got[1].ID != "tests" || got[1].Status != StatusMissing {
		t.Fatalf("tests = %+v", got[1])
	}
	remaining := Remaining(got)
	if len(remaining) != 1 || remaining[0].ID != "tests" {
		t.Fatalf("remaining = %+v", remaining)
	}
}

func TestEvaluateRejectsDuplicateIDs(t *testing.T) {
	_, err := Evaluate(context.Background(), []Spec{
		{Obligation: Obligation{ID: "x"}},
		{Obligation: Obligation{ID: "x"}},
	})
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestEvaluatePropagatesValidatorError(t *testing.T) {
	want := errors.New("probe failed")
	_, err := Evaluate(context.Background(), []Spec{{
		Obligation: Obligation{ID: "tests"},
		Validate: func(context.Context, Obligation) (Status, string, error) {
			return StatusUnknown, "", want
		},
	}})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped %v", err, want)
	}
}
