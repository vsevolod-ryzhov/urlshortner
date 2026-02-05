package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	if err := run(root, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(root string, verbose bool) error {
	if verbose {
		fmt.Printf("Scanning project from root: %s\n", root)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  root,
	}

	pkgs, errPkg := packages.Load(cfg, "./...")
	if errPkg != nil {
		return fmt.Errorf("failed to load packages: %w", errPkg)
	}

	if verbose {
		fmt.Printf("Found %d packages\n", len(pkgs))
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			if verbose {
				fmt.Printf("Package %s has errors: %v\n", pkg.Name, pkg.Errors)
			}
			continue
		}

		if verbose {
			fmt.Printf("Processing package: %s (dir: %s)\n", pkg.Name, pkg.PkgPath)
		}

		structs, errStructs := findResetStructs(pkg)
		if errStructs != nil {
			if verbose {
				fmt.Printf("  Error scanning package %s: %v\n", pkg.Name, errStructs)
			}
			continue
		}

		if len(structs) == 0 {
			if verbose {
				fmt.Printf("  No structures with // generate:reset found\n")
			}
			continue
		}

		if verbose {
			fmt.Printf("  Found %d structures with // generate:reset:\n", len(structs))
			for _, s := range structs {
				fmt.Printf("    - %s (file: %s, line: %d)\n", s.Name, s.File, s.Line)
			}
		}

		if err := generateResetFile(pkg, structs, verbose); err != nil {
			return fmt.Errorf("failed to generate reset file for package %s: %w", pkg.Name, err)
		}
	}

	return nil
}

type StructInfo struct {
	Name     string
	File     string
	Line     int
	Fields   []FieldInfo
	Type     *types.Struct
	TypeName string
}

type FieldInfo struct {
	Name     string
	Type     types.Type
	TypeStr  string
	IsPtr    bool
	IsSlice  bool
	IsMap    bool
	IsStruct bool
}

func findResetStructs(pkg *packages.Package) ([]StructInfo, error) {
	var structs []StructInfo

	for _, file := range pkg.Syntax {
		posPkg := pkg.Fset.Position(file.Package)
		fileName := posPkg.Filename

		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}

			if _, isStruct := typeSpec.Type.(*ast.StructType); !isStruct {
				return true
			}

			if !hasGenerateResetComment(typeSpec, file.Comments, pkg.Fset) {
				return true
			}

			obj := pkg.TypesInfo.Defs[typeSpec.Name]
			if obj == nil {
				return true
			}

			named, ok := obj.Type().(*types.Named)
			if !ok {
				return true
			}

			structTypeObj, ok := named.Underlying().(*types.Struct)
			if !ok {
				return true
			}

			fields := collectFieldInfo(structTypeObj)

			pos := pkg.Fset.Position(typeSpec.Pos())
			relPath, _ := filepath.Rel(".", fileName)

			structs = append(structs, StructInfo{
				Name:     typeSpec.Name.Name,
				File:     relPath,
				Line:     pos.Line,
				Fields:   fields,
				Type:     structTypeObj,
				TypeName: named.String(),
			})

			return true
		})
	}

	return structs, nil
}

func hasGenerateResetComment(typeSpec *ast.TypeSpec, comments []*ast.CommentGroup, fset *token.FileSet) bool {
	if typeSpec.Doc != nil {
		for _, comment := range typeSpec.Doc.List {
			if strings.Contains(comment.Text, "generate:reset") {
				return true
			}
		}
	}

	typePos := fset.Position(typeSpec.Pos())
	for _, commentGroup := range comments {
		if commentGroup.Pos() > typeSpec.Pos() {
			continue
		}

		commentPos := fset.Position(commentGroup.End())
		if commentPos.Line == typePos.Line-1 || commentPos.Line == typePos.Line {
			for _, comment := range commentGroup.List {
				if strings.Contains(comment.Text, "generate:reset") {
					return true
				}
			}
		}
	}

	return false
}

func collectFieldInfo(structType *types.Struct) []FieldInfo {
	var fields []FieldInfo

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if !field.Exported() {
			continue
		}

		fieldType := field.Type()
		typeStr := types.TypeString(fieldType, func(p *types.Package) string {
			return p.Name()
		})

		fields = append(fields, FieldInfo{
			Name:     field.Name(),
			Type:     fieldType,
			TypeStr:  typeStr,
			IsPtr:    isPointerType(fieldType),
			IsSlice:  isSliceType(fieldType),
			IsMap:    isMapType(fieldType),
			IsStruct: isStructType(fieldType),
		})
	}

	return fields
}

func isPointerType(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

func isSliceType(t types.Type) bool {
	_, ok := t.(*types.Slice)
	return ok
}

func isMapType(t types.Type) bool {
	_, ok := t.(*types.Map)
	return ok
}

func isStructType(t types.Type) bool {
	switch t := t.(type) {
	case *types.Named:
		_, ok := t.Underlying().(*types.Struct)
		return ok
	case *types.Struct:
		return true
	default:
		return false
	}
}

func generateResetFile(pkg *packages.Package, structs []StructInfo, verbose bool) error {
	pkgDir := filepath.Dir(pkg.Fset.Position(pkg.Syntax[0].Package).Filename)

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "// Code generated by \"reset\"; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkg.Name)

	imports := collectImports(structs)
	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&buf, "\t%q\n", imp)
		}
		buf.WriteString(")\n\n")
	}

	sort.Slice(structs, func(i, j int) bool {
		return structs[i].Name < structs[j].Name
	})

	for _, s := range structs {
		if verbose {
			fmt.Printf("  Generating Reset() method for %s\n", s.Name)
		}
		generateResetMethod(&buf, s)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not format generated code: %v\n", err)
		formatted = buf.Bytes()
	}

	outputFile := filepath.Join(pkgDir, "reset.gen.go")
	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if verbose {
		fmt.Printf("  Generated: %s\n", outputFile)
	}

	return nil
}

func collectImports(structs []StructInfo) []string {
	imports := make(map[string]bool)

	for _, s := range structs {
		for _, field := range s.Fields {
			if named, ok := field.Type.(*types.Named); ok {
				pkg := named.Obj().Pkg()
				if pkg != nil {
					imports[pkg.Path()] = true
				}
			}

			if ptr, ok := field.Type.(*types.Pointer); ok {
				if named, ok := ptr.Elem().(*types.Named); ok {
					pkg := named.Obj().Pkg()
					if pkg != nil {
						imports[pkg.Path()] = true
					}
				}
			}
		}
	}

	var result []string
	for imp := range imports {
		result = append(result, imp)
	}
	sort.Strings(result)
	return result
}

func generateResetMethod(buf *bytes.Buffer, s StructInfo) {
	fmt.Fprintf(buf, "func (x *%s) Reset() {\n", s.Name)
	fmt.Fprintf(buf, "\tif x == nil {\n")
	fmt.Fprintf(buf, "\t\treturn\n")
	fmt.Fprintf(buf, "\t}\n\n")

	for _, field := range s.Fields {
		generateFieldReset(buf, "x", field, 1)
	}

	fmt.Fprintf(buf, "}\n\n")
}

func generateFieldReset(buf *bytes.Buffer, prefix string, field FieldInfo, indentLevel int) {
	indent := strings.Repeat("\t", indentLevel)
	fieldAccess := fmt.Sprintf("%s.%s", prefix, field.Name)

	switch {
	case field.IsPtr:
		fmt.Fprintf(buf, "%sif %s != nil {\n", indent, fieldAccess)
		ptrType := field.Type.(*types.Pointer)
		t := ptrType.Elem()

		switch elemType := t.(type) {
		case *types.Basic:
			generateBasicFieldReset(buf, fieldAccess, elemType, indentLevel+1)
		case *types.Slice:
			fmt.Fprintf(buf, "%s\t%s = %s[:0]\n", indent, fieldAccess, fieldAccess)
		case *types.Map:
			fmt.Fprintf(buf, "%s\tclear(%s)\n", indent, fieldAccess)
		case *types.Named:
			if isStructType(elemType) {
				fmt.Fprintf(buf, "%s\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", indent, fieldAccess)
				fmt.Fprintf(buf, "%s\t\tresetter.Reset()\n", indent)
				fmt.Fprintf(buf, "%s\t}\n", indent)
			} else {
				if basic, ok := elemType.Underlying().(*types.Basic); ok {
					generateBasicFieldReset(buf, "*"+fieldAccess, basic, indentLevel+1)
				}
			}
		case *types.Struct:
			fmt.Fprintf(buf, "%s\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", indent, fieldAccess)
			fmt.Fprintf(buf, "%s\t\tresetter.Reset()\n", indent)
			fmt.Fprintf(buf, "%s\t}\n", indent)
		default:
			fmt.Fprintf(buf, "%s\t// %s: unsupported pointer type for Reset\n", indent, field.Name)
		}
		fmt.Fprintf(buf, "%s}\n", indent)

	case field.IsSlice:
		fmt.Fprintf(buf, "%s%s = %s[:0]\n", indent, fieldAccess, fieldAccess)

	case field.IsMap:
		fmt.Fprintf(buf, "%sclear(%s)\n", indent, fieldAccess)

	case field.IsStruct:
		fmt.Fprintf(buf, "%sif resetter, ok := interface{}(&%s).(interface{ Reset() }); ok {\n", indent, fieldAccess)
		fmt.Fprintf(buf, "%s\tresetter.Reset()\n", indent)
		fmt.Fprintf(buf, "%s}\n", indent)

	default:
		if basic, ok := field.Type.(*types.Basic); ok {
			generateBasicFieldReset(buf, fieldAccess, basic, indentLevel)
		} else {
			fmt.Fprintf(buf, "%s// %s: unsupported type %s for Reset\n", indent, field.Name, field.TypeStr)
		}
	}
}

func generateBasicFieldReset(buf *bytes.Buffer, fieldAccess string, basic *types.Basic, indentLevel int) {
	indent := strings.Repeat("\t", indentLevel)

	switch basic.Kind() {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr, types.Float32, types.Float64, types.Complex64, types.Complex128:
		fmt.Fprintf(buf, "%s%s = 0\n", indent, fieldAccess)
	case types.String:
		fmt.Fprintf(buf, "%s%s = \"\"\n", indent, fieldAccess)
	case types.Bool:
		fmt.Fprintf(buf, "%s%s = false\n", indent, fieldAccess)
	default:
		fmt.Fprintf(buf, "%s// %s: unsupported basic type %s\n", indent, fieldAccess, basic.Name())
	}
}
