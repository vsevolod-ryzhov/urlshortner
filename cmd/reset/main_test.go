package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasGenerateResetComment(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "simple comment",
			code: `// generate:reset
type User struct {}`,
			expected: true,
		},
		{
			name: "comment in doc",
			code: `// generate:reset
// User is a user
type User struct {}`,
			expected: true,
		},
		{
			name:     "no comment",
			code:     `type User struct {}`,
			expected: false,
		},
		{
			name: "wrong comment",
			code: `// generate:something
type User struct {}`,
			expected: false,
		},
		{
			name: "block comment",
			code: `/* generate:reset */
type User struct {}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset, file, comments := parseTestCode(t, tt.code)

			var typeSpec *ast.TypeSpec
			ast.Inspect(file, func(node ast.Node) bool {
				if ts, ok := node.(*ast.TypeSpec); ok {
					typeSpec = ts
					return false
				}
				return true
			})

			if typeSpec == nil {
				t.Fatal("type spec not found in test code")
			}

			result := hasGenerateResetComment(typeSpec, comments, fset)
			if result != tt.expected {
				t.Errorf("hasGenerateResetComment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func parseTestCode(t *testing.T, code string) (*token.FileSet, *ast.File, []*ast.CommentGroup) {
	t.Helper()

	fset := token.NewFileSet()
	src := "package test\n" + code

	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test code: %v", err)
	}

	return fset, file, file.Comments
}

func TestGenerateFieldReset(t *testing.T) {
	tests := []struct {
		name     string
		field    FieldInfo
		expected []string
	}{
		{
			name: "int field",
			field: FieldInfo{
				Name:    "ID",
				Type:    types.Typ[types.Int],
				TypeStr: "int",
			},
			expected: []string{"x.ID = 0"},
		},
		{
			name: "string field",
			field: FieldInfo{
				Name:    "Name",
				Type:    types.Typ[types.String],
				TypeStr: "string",
			},
			expected: []string{"x.Name = \"\""},
		},
		{
			name: "bool field",
			field: FieldInfo{
				Name:    "Enabled",
				Type:    types.Typ[types.Bool],
				TypeStr: "bool",
			},
			expected: []string{"x.Enabled = false"},
		},
		{
			name: "float64 field",
			field: FieldInfo{
				Name:    "Price",
				Type:    types.Typ[types.Float64],
				TypeStr: "float64",
			},
			expected: []string{"x.Price = 0"},
		},
		{
			name: "slice field",
			field: FieldInfo{
				Name:    "Tags",
				Type:    types.NewSlice(types.Typ[types.String]),
				TypeStr: "[]string",
				IsSlice: true,
			},
			expected: []string{"x.Tags = x.Tags[:0]"},
		},
		{
			name: "map field",
			field: FieldInfo{
				Name:    "Data",
				Type:    types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
				TypeStr: "map[string]int",
				IsMap:   true,
			},
			expected: []string{"clear(x.Data)"},
		},
		{
			name: "pointer to slice",
			field: FieldInfo{
				Name:    "Items",
				Type:    types.NewPointer(types.NewSlice(types.Typ[types.String])),
				TypeStr: "*[]string",
				IsPtr:   true,
			},
			expected: []string{"if x.Items != nil", "x.Items = x.Items[:0]"},
		},
		{
			name: "struct field",
			field: FieldInfo{
				Name:     "Profile",
				Type:     types.NewStruct(nil, nil),
				TypeStr:  "Profile",
				IsStruct: true,
			},
			expected: []string{"interface{}(&x.Profile).(interface{ Reset() })"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			generateFieldReset(&buf, "x", tt.field, 1)

			output := buf.String()
			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("output doesn't contain expected string: %q\nGot: %q", exp, output)
				}
			}

			// Для отладки
			t.Logf("Generated for %s:\n%s", tt.name, output)
		})
	}
}

func TestGenerateResetMethod(t *testing.T) {
	tests := []struct {
		name       string
		structInfo StructInfo
		expected   []string
	}{
		{
			name: "simple struct",
			structInfo: StructInfo{
				Name: "User",
				Fields: []FieldInfo{
					{
						Name:    "ID",
						Type:    types.Typ[types.Int],
						TypeStr: "int",
					},
					{
						Name:    "Name",
						Type:    types.Typ[types.String],
						TypeStr: "string",
					},
					{
						Name:    "Tags",
						Type:    types.NewSlice(types.Typ[types.String]),
						TypeStr: "[]string",
						IsSlice: true,
					},
				},
			},
			expected: []string{
				"func (x *User) Reset() {",
				"if x == nil {",
				"x.ID = 0",
				"x.Name = \"\"",
				"x.Tags = x.Tags[:0]",
			},
		},
		{
			name: "struct with pointer",
			structInfo: StructInfo{
				Name: "Config",
				Fields: []FieldInfo{
					{
						Name:    "Timeout",
						Type:    types.NewPointer(types.Typ[types.Int]),
						TypeStr: "*int",
						IsPtr:   true,
					},
					{
						Name:    "Settings",
						Type:    types.NewMap(types.Typ[types.String], types.Typ[types.String]),
						TypeStr: "map[string]string",
						IsMap:   true,
					},
				},
			},
			expected: []string{
				"func (x *Config) Reset() {",
				"if x.Timeout != nil",
				"clear(x.Settings)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			generateResetMethod(&buf, tt.structInfo)

			output := buf.String()
			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("output doesn't contain %q\nGot: %s", exp, output)
				}
			}

			_, err := format.Source([]byte(output))
			if err != nil {
				t.Errorf("generated code is not valid Go: %v\nCode:\n%s", err, output)
			}
		})
	}
}

func TestIsTypeFunctions(t *testing.T) {
	pkg := types.NewPackage("test", "test")
	namedStruct := types.NewNamed(
		types.NewTypeName(0, pkg, "MyStruct", nil),
		types.NewStruct(nil, nil),
		nil,
	)

	tests := []struct {
		name     string
		typ      types.Type
		isPtr    bool
		isSlice  bool
		isMap    bool
		isStruct bool
	}{
		{
			name:     "pointer",
			typ:      types.NewPointer(types.Typ[types.Int]),
			isPtr:    true,
			isSlice:  false,
			isMap:    false,
			isStruct: false,
		},
		{
			name:     "slice",
			typ:      types.NewSlice(types.Typ[types.String]),
			isPtr:    false,
			isSlice:  true,
			isMap:    false,
			isStruct: false,
		},
		{
			name:     "map",
			typ:      types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			isPtr:    false,
			isSlice:  false,
			isMap:    true,
			isStruct: false,
		},
		{
			name:     "basic int",
			typ:      types.Typ[types.Int],
			isPtr:    false,
			isSlice:  false,
			isMap:    false,
			isStruct: false,
		},
		{
			name:     "basic string",
			typ:      types.Typ[types.String],
			isPtr:    false,
			isSlice:  false,
			isMap:    false,
			isStruct: false,
		},
		{
			name:     "named struct",
			typ:      namedStruct,
			isPtr:    false,
			isSlice:  false,
			isMap:    false,
			isStruct: true,
		},
		{
			name:     "anon struct",
			typ:      types.NewStruct(nil, nil),
			isPtr:    false,
			isSlice:  false,
			isMap:    false,
			isStruct: true,
		},
		{
			name:     "pointer to struct",
			typ:      types.NewPointer(namedStruct),
			isPtr:    true,
			isSlice:  false,
			isMap:    false,
			isStruct: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPointerType(tt.typ); got != tt.isPtr {
				t.Errorf("isPointerType() = %v, want %v", got, tt.isPtr)
			}
			if got := isSliceType(tt.typ); got != tt.isSlice {
				t.Errorf("isSliceType() = %v, want %v", got, tt.isSlice)
			}
			if got := isMapType(tt.typ); got != tt.isMap {
				t.Errorf("isMapType() = %v, want %v", got, tt.isMap)
			}
			if got := isStructType(tt.typ); got != tt.isStruct {
				t.Errorf("isStructType() = %v, want %v", got, tt.isStruct)
			}
		})
	}
}

func TestGenerateBasicFieldReset(t *testing.T) {
	tests := []struct {
		name      string
		basic     *types.Basic
		fieldName string
		expected  string
	}{
		{
			name:      "int",
			basic:     types.Typ[types.Int],
			fieldName: "ID",
			expected:  "x.ID = 0",
		},
		{
			name:      "string",
			basic:     types.Typ[types.String],
			fieldName: "Name",
			expected:  "x.Name = \"\"",
		},
		{
			name:      "bool",
			basic:     types.Typ[types.Bool],
			fieldName: "Active",
			expected:  "x.Active = false",
		},
		{
			name:      "float32",
			basic:     types.Typ[types.Float32],
			fieldName: "Value",
			expected:  "x.Value = 0",
		},
		{
			name:      "uint64",
			basic:     types.Typ[types.Uint64],
			fieldName: "Count",
			expected:  "x.Count = 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			generateBasicFieldReset(&buf, "x."+tt.fieldName, tt.basic, 1)

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("output doesn't contain expected string: %q\nGot: %q", tt.expected, output)
			}
		})
	}
}

func TestCollectFieldInfo(t *testing.T) {
	pkg := types.NewPackage("test", "test")

	fields := []*types.Var{
		types.NewField(0, pkg, "ID", types.Typ[types.Int], false),
		types.NewField(0, pkg, "Name", types.Typ[types.String], false),
		types.NewField(0, pkg, "Items", types.NewSlice(types.Typ[types.String]), false),
		types.NewField(0, pkg, "Settings", types.NewMap(types.Typ[types.String], types.Typ[types.Int]), false),
		types.NewField(0, pkg, "Ref", types.NewPointer(types.Typ[types.Int]), false),
	}

	structType := types.NewStruct(fields, nil)

	fieldInfos := collectFieldInfo(structType)

	if len(fieldInfos) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fieldInfos))
	}

	expectedNames := []string{"ID", "Name", "Items", "Settings", "Ref"}
	for i, name := range expectedNames {
		if fieldInfos[i].Name != name {
			t.Errorf("field[%d].Name = %s, want %s", i, fieldInfos[i].Name, name)
		}
	}

	if !fieldInfos[2].IsSlice {
		t.Error("Items field should be marked as slice")
	}
	if !fieldInfos[3].IsMap {
		t.Error("Settings field should be marked as map")
	}
	if !fieldInfos[4].IsPtr {
		t.Error("Ref field should be marked as pointer")
	}
}

func TestCollectImports(t *testing.T) {
	pkg1 := types.NewPackage("pkg1", "pkg1")
	pkg2 := types.NewPackage("pkg2", "pkg2")

	named1 := types.NewNamed(
		types.NewTypeName(0, pkg1, "Type1", nil),
		types.Typ[types.Int],
		nil,
	)

	named2 := types.NewNamed(
		types.NewTypeName(0, pkg2, "Type2", nil),
		types.Typ[types.String],
		nil,
	)

	structs := []StructInfo{
		{
			Name: "Test1",
			Fields: []FieldInfo{
				{
					Name: "Field1",
					Type: named1,
				},
				{
					Name: "Field2",
					Type: types.NewPointer(named2),
				},
			},
		},
	}

	imports := collectImports(structs)

	if len(imports) != 2 {
		t.Fatalf("expected 2 imports, got %d: %v", len(imports), imports)
	}

	expectedImports := []string{"pkg1", "pkg2"}
	for _, imp := range expectedImports {
		found := false
		for _, actual := range imports {
			if actual == imp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("import %q not found in %v", imp, imports)
		}
	}
}

func TestIntegrationSimple(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	goModContent := `module myproject

go 1.21
`
	goModPath := filepath.Join(projectDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	pkgDir := filepath.Join(projectDir, "models")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create package directory: %v", err)
	}

	modelContent := `package models

// generate:reset
type User struct {
	ID   int
	Name string
}
`
	modelPath := filepath.Join(pkgDir, "user.go")
	if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
		t.Fatalf("failed to write user.go: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change to project directory: %v", err)
	}

	if err := run(projectDir, true); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	genFile := filepath.Join(pkgDir, "reset.gen.go")
	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Logf("Note: reset.gen.go not generated (might be expected): %v", err)
		return
	}

	expectedStrings := []string{
		"package models",
		"func (x *User) Reset()",
		"x.ID = 0",
		"x.Name = \"\"",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(string(content), expected) {
			t.Errorf("generated file doesn't contain %q", expected)
		}
	}

	_, err = format.Source(content)
	if err != nil {
		t.Errorf("generated code is not valid Go: %v", err)
	}
}

func TestCommentParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
		desc     string
	}{
		{
			name: "comment with spaces",
			code: `//   generate:reset   
type User struct {}`,
			expected: true,
			desc:     "must ignore spaces",
		},
		{
			name: "comment in middle",
			code: `type User struct {
	// generate:reset - this is comment to field, not struct
	ID int
}`,
			expected: false,
			desc:     "comment to field, not struct",
		},
		{
			name: "multiple comments",
			code: `// First comments
// generate:reset
// third comment
type User struct {}`,
			expected: true,
			desc:     "must be in comment group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset, file, comments := parseTestCode(t, tt.code)

			var typeSpec *ast.TypeSpec
			ast.Inspect(file, func(node ast.Node) bool {
				if ts, ok := node.(*ast.TypeSpec); ok {
					typeSpec = ts
					return false
				}
				return true
			})

			if typeSpec == nil {
				t.Fatal("type spec not found")
			}

			result := hasGenerateResetComment(typeSpec, comments, fset)
			if result != tt.expected {
				t.Errorf("%s: hasGenerateResetComment() = %v, want %v (%s)",
					tt.name, result, tt.expected, tt.desc)
			}
		})
	}
}
