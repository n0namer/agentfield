package obligation

import (
	"context"
	"fmt"
	"sort"
)

// Status describes the current state of a contract obligation.
type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusSatisfied  Status = "satisfied"
	StatusMissing    Status = "missing"
	StatusInvalid    Status = "invalid"
	StatusBlocked    Status = "blocked"
)

// Kind describes how an obligation should be proven.
type Kind string

const (
	KindState    Kind = "state"
	KindEvidence Kind = "evidence"
	KindSemantic Kind = "semantic"
	KindResponse Kind = "response"
)

// Obligation is an atomic, independently validatable part of a contract.
type Obligation struct {
	ID      string
	Kind   Kind
	Status Status
	Reason  string
}

// Validator observes authoritative current state for one obligation.
// It must not mutate the target just to decide whether the obligation is satisfied.
type Validator func(context.Context, Obligation) (Status, string, error)

// Spec binds an obligation to its read-only validator.
type Spec struct {
	Obligation Obligation
	Validate   Validator
}

// Evaluate re-observes every obligation and returns a deterministic snapshot.
// Itnever trusts a stale caller-supplied Status when a validator is available.
func Evaluate(ctx context.Context, specs []Spec) ([]Obligation, error) {
	out := make([]Obligation, 0, len(specs))
	seen := make(map[string]struct, len(specs))
	for _, spec := range specs {
		o := spec.Obligation
		if o.ID == "" {
			return nil, fmt.Errorf("obligation id must not be empty")
		}
		if _, exists := seen[o.ID]; exists {
			return nil, fmt.Errorf("duplicate obligation id %q", o.ID)
		}
		seen[o.ID] = struct{}{}
		if spec.Validate != nil {
			status, reason, err := spec.Validate(ctx, o)
			if err != nil {
				return nil, fmt.Errorf("validate %s: %w", o.ID, err)
			}
			o.Status = status
			o.Reason = reason
		} else if o.Status == "" {
			o.Status = StatusUnknown
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Remaining returns only obligations that still need work or decision.
// Satisfied obligations are dropped so a continuation can be bounded to the smallest remaining scope.
func Remaining(snapshot []Obligation) []Obligation {
	out := make([]Obligation, 0, len(snapshot))
	for _, o := range snapshot {
		if o.Status != StatusSatisfied {
			out = append(out, o)
		}
	}
	return out
}
