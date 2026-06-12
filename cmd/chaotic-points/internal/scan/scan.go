// Package scan discovers chaos.Point / chaos.PointWith call sites in a module
// and gates rule configs against the discovered point names.
package scan

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"sort"

	"golang.org/x/tools/go/packages"
)

// chaosPkgPath is the import path whose Point/PointWith calls we discover.
const chaosPkgPath = "github.com/RomanAgaltsev/chaotic/chaos"

// Point is one discovered chaos.Point / chaos.PointWith call site.
type Point struct {
	Name    string // constant-string name; empty when Dynamic
	File    string
	Line    int
	Dynamic bool // true when the name argument is not a constant string
}

const loadMode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
	packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps

// Scan discovers points in the packages matching patterns, resolved from the
// current working directory.
func Scan(patterns ...string) ([]Point, error) {
	return load(&packages.Config{Mode: loadMode}, patterns)
}

// Dir is Scan with an explicit module directory (used by tests and by the
// CLI when a -C/dir flag is given).
func Dir(dir string, patterns ...string) ([]Point, error) {
	return load(&packages.Config{Mode: loadMode, Dir: dir}, patterns)
}

func load(cfg *packages.Config, patterns []string) ([]Point, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	// Resolve packages relative to the target module alone. Without this, an
	// ambient go.work above the scanned directory makes `go list` reject any
	// module not listed in its `use` block (e.g. testdata fixtures).
	if cfg.Env == nil {
		cfg.Env = append(os.Environ(), "GOWORK=off")
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, fmt.Errorf("scan: %d package load error(s)", n)
	}
	var pts []Point
	for _, p := range pkgs {
		for _, file := range p.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isChaosPoint(p.TypesInfo, call) || len(call.Args) < 2 {
					return true
				}
				pos := p.Fset.Position(call.Pos())
				pt := Point{File: pos.Filename, Line: pos.Line}
				if tv, ok := p.TypesInfo.Types[call.Args[1]]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
					pt.Name = constant.StringVal(tv.Value)
				} else {
					pt.Dynamic = true
				}
				pts = append(pts, pt)
				return true
			})
		}
	}
	sort.Slice(pts, func(i, j int) bool {
		if pts[i].Name != pts[j].Name {
			return pts[i].Name < pts[j].Name
		}
		if pts[i].File != pts[j].File {
			return pts[i].File < pts[j].File
		}
		return pts[i].Line < pts[j].Line
	})
	return pts, nil
}

// isChaosPoint reports whether call is a call to chaos.Point or chaos.PointWith,
// resolved through type info so import aliases are handled and same-named funcs
// in other packages are not matched.
func isChaosPoint(info *types.Info, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := info.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != chaosPkgPath {
		return false
	}
	return fn.Name() == "Point" || fn.Name() == "PointWith"
}
