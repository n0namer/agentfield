package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

var _ AsyncExecutionIdempotencyStore = (*LocalStorage)(nil)

func TestR17AcceptanceSourceBindings(t *testing.T) {
	bindings := []struct {
		name   string
		path   string
		digest string
	}{
		{
			name:   "storage_r14_contract",
			path:   "execution_idempotency.go",
			digest: "7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2",
		},
		{
			name:   "handler_r14_contract",
			path:   "../handlers/execute.go",
			digest: "be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1",
		},
		{
			name:   "http_canary_reviewed_snapshot",
			path:   "../handlers/northstar_r16_http_canary_test.go",
			digest: "d0801c513c08ec16426edd22dea3bb9f4b8df24bc2ca31808944e210dd53d5ba",
		},
	}
	for _, binding := range bindings {
		t.Run(binding.name, func(t *testing.T) {
			source, err := os.ReadFile(binding.path)
			if err != nil {
				t.Fatalf("source binding unavailable: %v", err)
			}
			digest := sha256.Sum256(source)
			if actual := hex.EncodeToString(digest[:]); actual != binding.digest {
				t.Fatalf("source drift for %s: got %s, want %s; re-review rather than updating the pin blindly", binding.path, actual, binding.digest)
			}
		})
	}
}

func TestR17AcceptanceOriginalCanaryPresence(t *testing.T) {
	canaries := []struct {
		path   string
		prefix string
	}{
		{path: "northstar_r16_sql_canary_test.go", prefix: "TestR16SQL"},
		{path: "../handlers/northstar_r16_http_canary_test.go", prefix: "TestR16HTTP"},
	}
	for _, canary := range canaries {
		t.Run(canary.prefix, func(t *testing.T) {
			source, err := os.ReadFile(canary.path)
			if err != nil {
				t.Fatalf("FIXTURE/EVIDENCE BLOCKED: original canary unavailable: %v", err)
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), canary.path, source, 0)
			if err != nil {
				t.Fatalf("FIXTURE RED: original canary is not valid Go: %v", err)
			}
			for _, declaration := range parsed.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if ok && r17CanaryTestDeclaration(function, canary.prefix) {
					return
				}
			}
			t.Fatalf("FIXTURE RED: no %s test declaration with signature func(*testing.T); presence is not runtime or receipt verification", canary.prefix)
		})
	}
}

func r17CanaryTestDeclaration(function *ast.FuncDecl, prefix string) bool {
	if function.Recv != nil || function.Body == nil || !strings.HasPrefix(function.Name.Name, prefix) {
		return false
	}
	if function.Type.TypeParams != nil || function.Type.Params == nil || len(function.Type.Params.List) != 1 {
		return false
	}
	if function.Type.Results != nil && len(function.Type.Results.List) != 0 {
		return false
	}
	parameter := function.Type.Params.List[0]
	if len(parameter.Names) > 1 {
		return false
	}
	pointer, ok := parameter.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "T" {
		return false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	return ok && qualifier.Name == "testing"
}

func TestR17AcceptanceCanaryDeclarationBoundaries(t *testing.T) {
	cases := []struct {
		name        string
		declaration string
		want        bool
	}{
		{name: "named_test", declaration: "func TestR16SQLCanary(test *testing.T) {}", want: true},
		{name: "unnamed_test", declaration: "func TestR16SQLCanary(*testing.T) {}", want: true},
		{name: "no_parameter", declaration: "func TestR16SQLCanary() {}"},
		{name: "wrong_parameter", declaration: "func TestR16SQLCanary(test *testing.B) {}"},
		{name: "value_parameter", declaration: "func TestR16SQLCanary(test testing.T) {}"},
		{name: "multiple_parameters", declaration: "func TestR16SQLCanary(first, second *testing.T) {}"},
		{name: "method_not_test", declaration: "func (suite *Suite) TestR16SQLCanary(test *testing.T) {}"},
		{name: "returns_value", declaration: "func TestR16SQLCanary(test *testing.T) error { return nil }"},
		{name: "generic_function", declaration: "func TestR16SQLCanary[Value any](test *testing.T) {}"},
		{name: "wrong_prefix", declaration: "func TestOtherCanary(test *testing.T) {}"},
		{name: "no_body", declaration: "func TestR16SQLCanary(test *testing.T)"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, err := parser.ParseFile(token.NewFileSet(), "canary.go", "package storage\n"+testCase.declaration, 0)
			if err != nil {
				t.Fatal(err)
			}
			function, ok := parsed.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatal("fixture is not a function declaration")
			}
			if actual := r17CanaryTestDeclaration(function, "TestR16SQL"); actual != testCase.want {
				t.Fatalf("canary declaration accepted=%v, want %v", actual, testCase.want)
			}
		})
	}
}

func TestR17AcceptanceNilPayloadRejectsBeforeDatabase(t *testing.T) {
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name    string
		store   *LocalStorage
		context context.Context
	}{
		{name: "nil_receiver", context: context.Background()},
		{name: "uninitialized_database", store: &LocalStorage{}, context: context.Background()},
		{name: "cancelled_context", store: &LocalStorage{}, context: cancelledContext},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := testCase.store.CreateAsyncExecutionIdempotent(testCase.context, "r17-scope", "r17-target", "r17-key", "r17-digest", nil)
			if err == nil || err.Error() != "nil execution payload" || result != nil {
				t.Fatalf("nil payload must fail before database access: result=%v error=%v", result, err)
			}
		})
	}
}
