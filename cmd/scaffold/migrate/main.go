package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: make migrations name=<feature-name>")
		os.Exit(1)
	}

	name := strings.ToLower(strings.TrimSpace(os.Args[1]))
	if name == "" {
		fmt.Fprintln(os.Stderr, "feature name cannot be empty")
		os.Exit(1)
	}

	modelPath := filepath.Join("app", "features", name, "model.go")
	migrationsDir := filepath.Join("app", "infra", "database", "migrations")

	fields, err := parseModelFields(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing model: %v\n", err)
		os.Exit(1)
	}

	seqNum, err := nextSeqNum(migrationsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading migrations dir: %v\n", err)
		os.Exit(1)
	}

	prefix := fmt.Sprintf("%06d_create_%s", seqNum, name)
	upPath := filepath.Join(migrationsDir, prefix+".up.sql")
	downPath := filepath.Join(migrationsDir, prefix+".down.sql")

	if err := os.WriteFile(upPath, []byte(generateUpSQL(name, fields)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", upPath, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", upPath)

	if err := os.WriteFile(downPath, []byte(generateDownSQL(name)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", downPath, err)
		os.Exit(1)
	}
	fmt.Printf("  created  %s\n", downPath)

	fmt.Printf("\n✓ Migration for '%s' generated. Review and edit before running make migrate-up.\n", name)
}

type fieldDef struct {
	Column   string
	SQLType  string
	Nullable bool
}

func parseModelFields(path string) ([]fieldDef, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var fields []fieldDef
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if !embedsModelBase(structType) {
				continue
			}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				for _, nameIdent := range field.Names {
					if !nameIdent.IsExported() {
						continue
					}
					col := columnName(nameIdent.Name, field.Tag)
					sqlType, nullable := goTypeToSQL(field.Type)
					fields = append(fields, fieldDef{Column: col, SQLType: sqlType, Nullable: nullable})
				}
			}
		}
	}
	return fields, nil
}

func embedsModelBase(s *ast.StructType) bool {
	for _, field := range s.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		if sel, ok := field.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Base" {
			return true
		}
	}
	return false
}

func columnName(fieldName string, tag *ast.BasicLit) string {
	if tag != nil {
		raw := strings.Trim(tag.Value, "`")
		if db := reflect.StructTag(raw).Get("db"); db != "" && db != "-" {
			return strings.Split(db, ",")[0]
		}
	}
	return toSnakeCase(fieldName)
}

func goTypeToSQL(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return identToSQL(t.Name), false
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return selectorToSQL(pkg.Name, t.Sel.Name), false
		}
		return "TEXT", false
	case *ast.StarExpr:
		sql, _ := goTypeToSQL(t.X)
		return sql, true
	case *ast.ArrayType, *ast.MapType:
		return "JSONB", false
	default:
		return "TEXT", false
	}
}

func identToSQL(name string) string {
	switch name {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float32", "float64":
		return "NUMERIC"
	case "bool":
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func selectorToSQL(pkg, sel string) string {
	switch {
	case pkg == "time" && sel == "Time":
		return "TIMESTAMPTZ"
	case pkg == "uuid" && sel == "UUID":
		return "UUID"
	default:
		return "TEXT"
	}
}

func generateUpSQL(table string, fields []fieldDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table))
	sb.WriteString("    id         UUID PRIMARY KEY,\n")
	for _, f := range fields {
		null := " NOT NULL"
		if f.Nullable {
			null = ""
		}
		sb.WriteString(fmt.Sprintf("    %-20s %s%s,\n", f.Column, f.SQLType, null))
	}
	sb.WriteString("    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n")
	sb.WriteString("    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n")
	sb.WriteString(");\n")
	return sb.String()
}

func generateDownSQL(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", table)
}

func nextSeqNum(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1, nil
	}
	re := regexp.MustCompile(`^(\d+)_`)
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

func toSnakeCase(s string) string {
	var out []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
