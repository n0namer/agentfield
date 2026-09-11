package obligation

import (
	"context"
	"fmt"
	"sort"
)

type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusSatisfied Status = "satisfied"
	StatusMissing   Status = "missing"
	StatusInvalid   Status = "invalid"
	StatusBlocked   Status = "blocked"
)

type Kind string

const (
	KindState    Kind = "state"
	KindEvidence Kind = "evidence"
	KindSemantic Kind = "semantic"
	KindResponse Kind = "response"
)

type Obligation struct {
	ID     string
	Kind   Kind
	Status Status
	Reason string
}

type Validator func(context.Context, Obligation) (Status, string, error)

type Spec struct {
	Obligation Obligation
	Validate   Validator
}

// Evaluate re-observes every obligation and returns a deterministic snapshot.
// A validator is authoritative over any stale caller-supplied status.
func Evaluate(ctx context.Context, specs []Spec) ([]Obligation, error) {
	out := make([]Obligation, 0, len(specs))
	seen := make(map[string]struct{}, len(specs))
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

// Remaining returns only obligations that still need work or a decision.
func Remaining(snapshot []Obligation) []Obligation {
	out := make([]Obligation, 0, len(snapshot))
	for _, o := range snapshot {
		if o.Status != StatusSatisfied {
			out = append(out, o)
		}
	}
	return out
}
