package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

type featureData struct {
	Name       string // e.g. "products"
	NameTitle  string // e.g. "Products"
	NameSingle string // e.g. "Product"
	PkgDB      string // e.g. "productsdb"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: make feature name=<feature-name>")
		fmt.Fprintln(os.Stderr, "       make feature rm name=<feature-name>")
		os.Exit(1)
	}

	subCmd := "create"
	nameArg := os.Args[1]
	if os.Args[1] == "rm" || os.Args[1] == "create" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "feature name required")
			os.Exit(1)
		}
		subCmd = os.Args[1]
		nameArg = os.Args[2]
	}

	name := strings.ToLower(strings.TrimSpace(nameArg))
	if name == "" {
		fmt.Fprintln(os.Stderr, "feature name cannot be empty")
		os.Exit(1)
	}

	data := featureData{
		Name:       name,
		NameTitle:  title(name),
		NameSingle: singular(title(name)),
		PkgDB:      name + "db",
	}

	if subCmd == "rm" {
		removeFeature(data)
		return
	}

	featureDir := filepath.Join("app", "features", name)
	dbDir := filepath.Join("app", "infra", "database", name)

	files := []struct {
		path    string
		content string
	}{
		{filepath.Join(featureDir, "model.go"), modelTpl},
		{filepath.Join(featureDir, "errors.go"), errorsTpl},
		{filepath.Join(featureDir, "repository.go"), repositoryTpl},
		{filepath.Join(featureDir, "dto.go"), dtoTpl},
		{filepath.Join(featureDir, "service.go"), serviceTpl},
		{filepath.Join(featureDir, "handler.go"), handlerTpl},
		{filepath.Join(featureDir, "routes.go"), routesTpl},
		{filepath.Join(dbDir, "pg_repository.go"), pgRepositoryTpl},
	}

	for _, f := range files {
		if err := writeTemplate(f.path, f.content, data); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", f.path, err)
			os.Exit(1)
		}
		fmt.Printf("  created  %s\n", f.path)
	}

	if err := injectContainer(data); err != nil {
		fmt.Fprintf(os.Stderr, "error updating app/bootstrap/container.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  updated  app/bootstrap/container.go")

	if err := injectBootstrapRoutes(data); err != nil {
		fmt.Fprintf(os.Stderr, "error updating app/bootstrap/routes.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  updated  app/bootstrap/routes.go")

	fmt.Printf("\n✓ Feature '%s' scaffolded and wired successfully.\n\n", name)
	fmt.Println("Next steps:")
	fmt.Printf("  1. Fill model:     app/features/%s/model.go\n", name)
	fmt.Printf("  2. Fill DTOs:      app/features/%s/dto.go\n", name)
	fmt.Printf("  3. Run migration:  make migrations name=%s\n", name)
	fmt.Printf("  4. Fill repo:      app/infra/database/%s/pg_repository.go\n", name)
}

func removeFeature(data featureData) {
	featureDir := filepath.Join("app", "features", data.Name)
	dbDir := filepath.Join("app", "infra", "database", data.Name)
	migrationsDir := filepath.Join("app", "infra", "database", "migrations")

	if err := os.RemoveAll(featureDir); err != nil {
		fmt.Fprintf(os.Stderr, "error removing %s: %v\n", featureDir, err)
		os.Exit(1)
	}
	fmt.Printf("  removed  %s\n", featureDir)

	if err := os.RemoveAll(dbDir); err != nil {
		fmt.Fprintf(os.Stderr, "error removing %s: %v\n", dbDir, err)
		os.Exit(1)
	}
	fmt.Printf("  removed  %s\n", dbDir)

	removeMigrationFiles(migrationsDir, data.Name)

	if err := removeFromContainer(data); err != nil {
		fmt.Fprintf(os.Stderr, "error updating app/bootstrap/container.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  updated  app/bootstrap/container.go")

	if err := removeFromBootstrapRoutes(data); err != nil {
		fmt.Fprintf(os.Stderr, "error updating app/bootstrap/routes.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  updated  app/bootstrap/routes.go")

	fmt.Printf("\n✓ Feature '%s' removed.\n", data.Name)
	fmt.Println("  note: run 'make migrate-down' before removing if table exists in DB.")
}

func removeMigrationFiles(dir, name string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	pattern := "_create_" + name + "."
	for _, e := range entries {
		if strings.Contains(e.Name(), pattern) {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: could not remove %s: %v\n", path, err)
				continue
			}
			fmt.Printf("  removed  %s\n", path)
		}
	}
}

func removeFromContainer(data featureData) error {
	return modifyFile("app/bootstrap/container.go", func(content string) (string, error) {
		tokens := []string{
			data.Name + "Feature",
			"db" + data.NameTitle,
			data.NameTitle + "Handler",
			data.Name + "Repo",
			data.Name + "Svc",
			data.Name + "Handler",
		}
		return removeLines(content, tokens...), nil
	})
}

func removeFromBootstrapRoutes(data featureData) error {
	return modifyFile("app/bootstrap/routes.go", func(content string) (string, error) {
		tokens := []string{
			data.Name + "Feature",
		}
		return removeLines(content, tokens...), nil
	})
}

func removeLines(content string, tokens ...string) string {
	lines := strings.Split(content, "\n")
	out := lines[:0]
	for _, line := range lines {
		keep := true
		for _, tok := range tokens {
			if strings.Contains(line, tok) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// ---------------------------------------------------------------------------
// Injection helpers
// ---------------------------------------------------------------------------

func injectContainer(data featureData) error {
	return modifyFile("app/bootstrap/container.go", func(content string) (string, error) {
		var err error

		importLines := fmt.Sprintf(
			"\t%sFeature \"go-boilerplate/app/features/%s\"\n\tdb%s \"go-boilerplate/app/infra/database/%s\"",
			data.Name, data.Name, data.NameTitle, data.Name)
		if content, err = injectBefore(content, "\t// scaffold:container-imports", importLines); err != nil {
			return "", err
		}

		fieldLine := fmt.Sprintf("\t%sHandler *%sFeature.Handler", data.NameTitle, data.Name)
		if content, err = injectBefore(content, "\t// scaffold:container-fields", fieldLine); err != nil {
			return "", err
		}

		wireLines := fmt.Sprintf(
			"\t%sRepo    := db%s.NewPgRepository(db)\n\t%sSvc     := %sFeature.NewService(%sRepo)\n\t%sHandler := %sFeature.NewHandler(%sSvc)",
			data.Name, data.NameTitle,
			data.Name, data.Name, data.Name,
			data.Name, data.Name, data.Name)
		if content, err = injectBefore(content, "\t// scaffold:container-wire", wireLines); err != nil {
			return "", err
		}

		initLine := fmt.Sprintf("\t\t%sHandler: %sHandler,", data.NameTitle, data.Name)
		if content, err = injectBefore(content, "\t\t// scaffold:container-init", initLine); err != nil {
			return "", err
		}

		return content, nil
	})
}

func injectBootstrapRoutes(data featureData) error {
	return modifyFile("app/bootstrap/routes.go", func(content string) (string, error) {
		var err error

		importLine := fmt.Sprintf("\t%sFeature \"go-boilerplate/app/features/%s\"", data.Name, data.Name)
		if content, err = injectBefore(content, "\t// scaffold:feature-imports", importLine); err != nil {
			return "", err
		}

		routeLine := fmt.Sprintf("\t%sFeature.RegisterRoutes(v1.Group(\"/%s\"), c.%sHandler)", data.Name, data.Name, data.NameTitle)
		if content, err = injectBefore(content, "\t// scaffold:feature-routes", routeLine); err != nil {
			return "", err
		}

		return content, nil
	})
}

func modifyFile(path string, transform func(string) (string, error)) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	result, err := transform(string(b))
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(result), 0644)
}

func injectBefore(content, marker, injection string) (string, error) {
	if !strings.Contains(content, marker) {
		return "", fmt.Errorf("sentinel %q not found — was it manually removed?", marker)
	}
	return strings.Replace(content, marker, injection+"\n"+marker, 1), nil
}

// ---------------------------------------------------------------------------
// File generation helpers
// ---------------------------------------------------------------------------

func writeTemplate(path, tpl string, data featureData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

func title(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// singular strips a trailing 's' for common plural → singular conversion.
// Good enough for scaffold naming (products→Product, users→User).
// Complex cases (categories→Category) need manual rename.
func singular(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "s") && len(s) > 1 {
		return s[:len(s)-1]
	}
	return s
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

const modelTpl = `package {{.Name}}

import "go-boilerplate/app/shared/model"

type {{.NameSingle}} struct {
	model.Base
	// TODO: add fields
}
`

const errorsTpl = `package {{.Name}}

import (
	"net/http"

	"go-boilerplate/app/shared/apperror"
)

var (
	Err{{.NameSingle}}NotFound = apperror.New(http.StatusNotFound, "{{.Name}} not found")
	Err{{.NameSingle}}Conflict = apperror.New(http.StatusConflict, "{{.Name}} already exists")
)
`

const repositoryTpl = `package {{.Name}}

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, item *{{.NameSingle}}) error
	FindByID(ctx context.Context, id uuid.UUID) (*{{.NameSingle}}, error)
	Update(ctx context.Context, item *{{.NameSingle}}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*{{.NameSingle}}, error)
}
`

const dtoTpl = `package {{.Name}}

type Create{{.NameSingle}}Request struct {
	// TODO: add fields
}

type Update{{.NameSingle}}Request struct {
	// TODO: add fields
}

type {{.NameSingle}}Response struct {
	ID string ` + "`" + `json:"id"` + "`" + `
	// TODO: add fields
}
`

const serviceTpl = `package {{.Name}}

import (
	"context"
	"time"

	"github.com/google/uuid"

	"go-boilerplate/app/shared/model"
)

type Service interface {
	Create(ctx context.Context, req Create{{.NameSingle}}Request) (*{{.NameSingle}}Response, error)
	GetByID(ctx context.Context, id uuid.UUID) (*{{.NameSingle}}Response, error)
	Update(ctx context.Context, id uuid.UUID, req Update{{.NameSingle}}Request) (*{{.NameSingle}}Response, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*{{.NameSingle}}Response, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req Create{{.NameSingle}}Request) (*{{.NameSingle}}Response, error) {
	item := &{{.NameSingle}}{
		Base: model.Base{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return toResponse(item), nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*{{.NameSingle}}Response, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(item), nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req Update{{.NameSingle}}Request) (*{{.NameSingle}}Response, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return toResponse(item), nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) List(ctx context.Context) ([]*{{.NameSingle}}Response, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*{{.NameSingle}}Response, len(items))
	for i, item := range items {
		result[i] = toResponse(item)
	}
	return result, nil
}

func toResponse(item *{{.NameSingle}}) *{{.NameSingle}}Response {
	return &{{.NameSingle}}Response{
		ID: item.ID.String(),
	}
}
`

const handlerTpl = `package {{.Name}}

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go-boilerplate/app/shared/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c echo.Context) error {
	var req Create{{.NameSingle}}Request
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	resp, err := h.svc.Create(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, resp)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, err)
	}
	resp, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

func (h *Handler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, err)
	}
	var req Update{{.NameSingle}}Request
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	resp, err := h.svc.Update(c.Request().Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}

func (h *Handler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, err)
	}
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return response.Error(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) List(c echo.Context) error {
	resp, err := h.svc.List(c.Request().Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, resp)
}
`

const routesTpl = `package {{.Name}}

import "github.com/labstack/echo/v4"

func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("", h.Create)
	g.GET("/:id", h.GetByID)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.GET("", h.List)
}
`

const pgRepositoryTpl = `package {{.PkgDB}}

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	feat "go-boilerplate/app/features/{{.Name}}"
)

type pgRepository struct {
	db *sql.DB
}

func NewPgRepository(db *sql.DB) feat.Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, item *feat.{{.NameSingle}}) error {
	const q = ` + "`" + `INSERT INTO {{.Name}} (id, created_at, updated_at) VALUES ($1, $2, $3)` + "`" + `
	_, err := r.db.ExecContext(ctx, q, item.ID, item.CreatedAt, item.UpdatedAt)
	return err
}

func (r *pgRepository) FindByID(ctx context.Context, id uuid.UUID) (*feat.{{.NameSingle}}, error) {
	const q = ` + "`" + `SELECT id, created_at, updated_at FROM {{.Name}} WHERE id = $1` + "`" + `
	item := &feat.{{.NameSingle}}{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, feat.Err{{.NameSingle}}NotFound
	}
	return item, err
}

func (r *pgRepository) Update(ctx context.Context, item *feat.{{.NameSingle}}) error {
	const q = ` + "`" + `UPDATE {{.Name}} SET updated_at = $1 WHERE id = $2` + "`" + `
	_, err := r.db.ExecContext(ctx, q, item.UpdatedAt, item.ID)
	return err
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = ` + "`" + `DELETE FROM {{.Name}} WHERE id = $1` + "`" + `
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *pgRepository) List(ctx context.Context) ([]*feat.{{.NameSingle}}, error) {
	const q = ` + "`" + `SELECT id, created_at, updated_at FROM {{.Name}} ORDER BY created_at DESC` + "`" + `
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*feat.{{.NameSingle}}
	for rows.Next() {
		item := &feat.{{.NameSingle}}{}
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
`
