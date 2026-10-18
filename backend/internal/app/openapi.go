package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/health"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// OpenAPIPath is where the generated document is committed (rest.md).
const OpenAPIPath = "docs/api/openapi.yaml"

// OpenAPI builds the OpenAPI 3.1 document of every implemented REST route
// from the route registry (httpserver.WithDoc) and the Go request/response
// types, so the contract cannot drift from the code (roadmap 1.6.7).
func OpenAPI() ([]byte, error) {
	logger := slog.New(slog.DiscardHandler)
	srv := httpserver.New(config.HTTP{}, config.EnvProd, logger)
	health.Register(srv, logger)
	registerModules(srv, moduleDeps{logger: logger})

	g := &schemaGen{schemas: map[string]any{}, owners: map[string]reflect.Type{}, opIDs: map[string]bool{}}
	paths := map[string]map[string]any{}
	for _, r := range srv.Routes() {
		if r.Doc == nil {
			return nil, fmt.Errorf("route %q has no httpserver.WithDoc", r.Pattern)
		}
		method, path, ok := strings.Cut(r.Pattern, " ")
		if !ok {
			return nil, fmt.Errorf("route %q has no method", r.Pattern)
		}
		op, err := g.operation(method, path, *r.Doc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", r.Pattern, err)
		}
		if paths[path] == nil {
			paths[path] = map[string]any{}
		}
		paths[path][strings.ToLower(method)] = op
	}

	g.schemas["ErrorEnvelope"] = map[string]any{
		"type":     "object",
		"required": []string{"error"},
		"properties": map[string]any{"error": map[string]any{
			"type":     "object",
			"required": []string{"code", "message", "request_id"},
			"properties": map[string]any{
				"code":       map[string]any{"type": "string", "description": "Stable code from docs/api/errors.md"},
				"message":    map[string]any{"type": "string", "description": "Localized fallback text (Accept-Language)"},
				"details":    map[string]any{"type": "object", "additionalProperties": true},
				"request_id": map[string]any{"type": "string", "format": "uuid"},
			},
		}},
	}

	doc := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "BrainVolt API",
			"version":     "v1",
			"description": "Generated from the backend code (internal/app/openapi.go); do not edit by hand. Conventions: docs/api/conventions.md, error codes: docs/api/errors.md.",
		},
		"servers": []any{
			map[string]any{"url": "http://localhost:8080", "description": "local development"},
		},
		"paths": paths,
		"components": map[string]any{
			"schemas": g.schemas,
			"securitySchemes": map[string]any{
				//nolint:gosec // G101 false positive: describes the auth scheme, holds no secret
				"firebase": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "Firebase ID token"},
			},
		},
	}
	return yaml.Marshal(doc)
}

// schemaGen turns Go types into JSON Schema (OpenAPI 3.1 dialect).
type schemaGen struct {
	schemas map[string]any
	owners  map[string]reflect.Type
	opIDs   map[string]bool
}

var pathParam = regexp.MustCompile(`\{([a-z_]+)\}`)

// generalErrors apply to every authenticated route; status-dependent codes
// (ACCOUNT_SUSPENDED, ACCOUNT_DELETION_PENDING) are listed per route.
var generalErrors = []apierror.Code{
	apierror.Unauthenticated, apierror.TokenExpired, apierror.RateLimited,
	apierror.MalformedRequest, apierror.Internal, apierror.Unavailable,
}

func (g *schemaGen) operation(method, path string, d httpserver.Doc) (map[string]any, error) {
	id := operationID(method, path)
	if g.opIDs[id] {
		return nil, fmt.Errorf("duplicate operationId %s", id)
	}
	g.opIDs[id] = true
	op := map[string]any{
		"summary":     d.Summary,
		"tags":        []string{d.Tag},
		"operationId": id,
		"x-auth":      d.Auth,
	}
	var params []any
	for _, m := range pathParam.FindAllStringSubmatch(path, -1) {
		params = append(params, map[string]any{"name": m[1], "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
	}
	for _, qp := range d.Query {
		schema := map[string]any{"type": qp.Type}
		if qp.Format != "" {
			schema["format"] = qp.Format
		}
		if len(qp.Enum) > 0 {
			schema["enum"] = qp.Enum
		}
		p := map[string]any{"name": qp.Name, "in": "query", "required": false, "schema": schema}
		if qp.Description != "" {
			p["description"] = qp.Description
		}
		params = append(params, p)
	}
	if d.Auth != "public" {
		op["security"] = []any{map[string]any{"firebase": []string{}}}
		params = append(params, map[string]any{
			"name": "Accept-Language", "in": "header", "required": false,
			"schema": map[string]any{"type": "string", "enum": []string{"tr", "en"}},
		})
	}
	if d.Idempotency {
		params = append(params, map[string]any{
			"name": "Idempotency-Key", "in": "header", "required": true,
			"schema":      map[string]any{"type": "string", "format": "uuid"},
			"description": "Client-generated UUID; repeats within 24 h replay the stored response (conventions.md §9)",
		})
	}
	if len(params) > 0 {
		op["parameters"] = params
	}
	if d.Request != nil {
		schema, err := g.ref(reflect.TypeOf(d.Request))
		if err != nil {
			return nil, err
		}
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}

	responses := map[string]any{}
	status := d.Status
	if status == 0 {
		status = http.StatusOK
		if d.Response == nil {
			status = http.StatusNoContent
		}
	}
	ok := map[string]any{"description": http.StatusText(status)}
	if d.Response != nil {
		schema, err := g.ref(reflect.TypeOf(d.Response))
		if err != nil {
			return nil, err
		}
		ok["content"] = map[string]any{"application/json": map[string]any{"schema": schema}}
	}
	responses[strconv.Itoa(status)] = ok

	codes := slices.Clone(d.Errors)
	if d.Auth != "public" {
		codes = append(codes, generalErrors...)
	}
	byStatus := map[int][]string{}
	for _, c := range codes {
		s := c.HTTPStatus()
		if !slices.Contains(byStatus[s], string(c)) {
			byStatus[s] = append(byStatus[s], string(c))
		}
	}
	for s, cs := range byStatus {
		sort.Strings(cs)
		responses[strconv.Itoa(s)] = map[string]any{
			"description": "Error codes: " + strings.Join(cs, ", "),
			"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ErrorEnvelope"}}},
		}
	}
	op["responses"] = responses
	return op, nil
}

// operationID is method + path in camel case, e.g. postV1MeDeletion.
func operationID(method, path string) string {
	id := strings.ToLower(method)
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		part = strings.Trim(part, "{}")
		for _, w := range strings.FieldsFunc(part, func(r rune) bool { return r == '-' || r == '_' }) {
			id += strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return id
}

type enumer interface{ EnumValues() []string }

var (
	timeType   = reflect.TypeFor[time.Time]()
	uuidType   = reflect.TypeFor[uuid.UUID]()
	enumerType = reflect.TypeFor[enumer]()
)

// ref returns a $ref for named structs (registered in components) or an
// inline schema for everything else.
func (g *schemaGen) ref(t reflect.Type) (map[string]any, error) {
	if t.Kind() == reflect.Struct && t != timeType && t.Name() != "" {
		name := t.Name()
		if owner, seen := g.owners[name]; seen {
			if owner != t {
				return nil, fmt.Errorf("two types named %s (%s, %s): rename one", name, owner.PkgPath(), t.PkgPath())
			}
		} else {
			g.owners[name] = t
			s, err := g.object(t)
			if err != nil {
				return nil, err
			}
			g.schemas[name] = s
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}, nil
	}
	return g.inline(t)
}

func (g *schemaGen) inline(t reflect.Type) (map[string]any, error) {
	if t.Implements(enumerType) {
		v, _ := reflect.Zero(t).Interface().(enumer)
		return map[string]any{"type": "string", "enum": v.EnumValues()}, nil
	}
	switch t {
	case timeType:
		return map[string]any{"type": "string", "format": "date-time"}, nil
	case uuidType:
		return map[string]any{"type": "string", "format": "uuid"}, nil
	}
	switch t.Kind() {
	case reflect.Pointer:
		s, err := g.ref(t.Elem())
		if err != nil {
			return nil, err
		}
		return nullable(s), nil
	case reflect.String:
		return map[string]any{"type": "string"}, nil
	case reflect.Bool:
		return map[string]any{"type": "boolean"}, nil
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}, nil
	case reflect.Float64:
		return map[string]any{"type": "number"}, nil
	case reflect.Slice:
		items, err := g.ref(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": items}, nil
	case reflect.Map:
		values, err := g.ref(t.Elem())
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "object", "additionalProperties": values}, nil
	case reflect.Struct:
		return g.object(t)
	default:
		return nil, fmt.Errorf("unsupported type %s", t)
	}
}

func nullable(s map[string]any) map[string]any {
	if _, isRef := s["$ref"]; isRef {
		return map[string]any{"oneOf": []any{s, map[string]any{"type": "null"}}}
	}
	out := map[string]any{}
	for k, v := range s {
		out[k] = v
	}
	out["type"] = []any{s["type"], "null"}
	return out
}

func (g *schemaGen) object(t reflect.Type) (map[string]any, error) {
	props := map[string]any{}
	var required []string
	for f := range t.Fields() {
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		s, err := g.ref(f.Type)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}
		props[name] = s
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		sort.Strings(required)
		out["required"] = required
	}
	return out, nil
}
