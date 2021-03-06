package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestDecodeMovedBlock(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
		End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
	}

	foo_expr := hcltest.MockExprTraversalSrc("test_instance.foo")
	bar_expr := hcltest.MockExprTraversalSrc("test_instance.bar")

	foo_index_expr := hcltest.MockExprTraversalSrc("test_instance.foo[1]")
	bar_index_expr := hcltest.MockExprTraversalSrc("test_instance.bar[\"one\"]")

	mod_foo_expr := hcltest.MockExprTraversalSrc("module.foo")
	mod_bar_expr := hcltest.MockExprTraversalSrc("module.bar")

	tests := map[string]struct {
		input *hcl.Block
		want  *Moved
		err   string
	}{
		"success": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustTargetFromExpr(foo_expr),
				To:        mustTargetFromExpr(bar_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"indexed resources": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_index_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_index_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustTargetFromExpr(foo_index_expr),
				To:        mustTargetFromExpr(bar_index_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"modules": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: mod_foo_expr,
						},
						"to": {
							Name: "to",
							Expr: mod_bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustTargetFromExpr(mod_foo_expr),
				To:        mustTargetFromExpr(mod_bar_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"error: missing argument": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustTargetFromExpr(foo_expr),
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: type mismatch": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"to": {
							Name: "to",
							Expr: foo_expr,
						},
						"from": {
							Name: "from",
							Expr: mod_foo_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				To:        mustTargetFromExpr(foo_expr),
				From:      mustTargetFromExpr(mod_foo_expr),
				DeclRange: blockRange,
			},
			"Invalid \"moved\" targets",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeMovedBlock(test.input)

			if diags.HasErrors() {
				if test.err == "" {
					t.Fatalf("unexpected error: %s", diags.Errs())
				}
				if gotErr := diags[0].Summary; gotErr != test.err {
					t.Errorf("wrong error, got %q, want %q", gotErr, test.err)
				}
			} else if test.err != "" {
				t.Fatal("expected error")
			}

			if !cmp.Equal(got, test.want) {
				t.Fatalf("wrong result: %s", cmp.Diff(got, test.want))
			}
		})
	}
}

func mustTargetFromExpr(expr hcl.Expression) *addrs.Target {
	traversal, hcldiags := hcl.AbsTraversalForExpr(expr)
	if hcldiags.HasErrors() {
		panic(hcldiags.Errs())
	}

	target, diags := addrs.ParseTarget(traversal)
	if diags.HasErrors() {
		panic(diags.Err())
	}

	return target
}
