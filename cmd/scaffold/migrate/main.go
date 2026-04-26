package main

import (
	"bufio"
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

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: make migrations name=<feature-name>")
		fmt.Fprintln(os.Stderr, "       make migrations up")
		fmt.Fprintln(os.Stderr, "       make migrations down")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		runMigrations(true)
	case "down":
		runMigrations(false)
	default:
		generateMigration(os.Args[1])
	}
}

func runMigrations(up bool) {
	dbURL := buildDBURL()
	migrationsDir := "file://app/infra/database/migrations"

	m, err := migrate.New(migrationsDir, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing migrate: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	versionBefore, _, _ := m.Version()

	if up {
		err = m.Up()
	} else {
		err = m.Steps(-1)
	}

	if err == migrate.ErrNoChange {
		fmt.Println("  no changes to apply.")
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration error: %v\n", err)
		os.Exit(1)
	}

	versionAfter, _, _ := m.Version()

	if up {
		fmt.Printf("✓ Migrations applied. version: %d → %d\n", versionBefore, versionAfter)
	} else {
		fmt.Printf("✓ Rolled back migration %d → version now: %d\n", versionBefore, versionAfter)
	}
}

func buildDBURL() string {
	env := loadEnv(".env")
	host := envOr(env, "DB_HOST", "localhost")
	port := envOr(env, "DB_PORT", "5432")
	user := envOr(env, "DB_USER", "postgres")
	pass := envOr(env, "DB_PASSWORD", "postgres")
	name := envOr(env, "DB_NAME", "go_boilerplate")
	ssl := envOr(env, "DB_SSL_MODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, ssl)
}

func loadEnv(path string) map[string]string {
	env := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return env
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return env
}

func envOr(env map[string]string, key, fallback string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return fallback
}

func generateMigration(name string) {
	name = strings.ToLower(strings.TrimSpace(name))
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

	fmt.Printf("\n✓ Migration for '%s' generated. Review and edit before running make migrations up.\n", name)
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
	sb.WriteString("    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n")
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
