# Contract Test Dependency Justification

## Schema Validator: kin-openapi

**Dependency**: `github.com/getkin/kin-openapi` v0.133.0

### Why kin-openapi?

| Criterion | Justification |
|-----------|---------------|
| **Minimal** | Core library, no heavy framework dependencies |
| **Purpose-built** | Designed specifically for OpenAPI 3.x validation |
| **Battle-tested** | 4.5k+ GitHub stars, used in production systems |
| **Go-native** | Pure Go implementation, no external binaries |
| **Maintenance** | Actively maintained, last release within 3 months |
| **License** | MIT (compatible with Penshort) |

### Alternatives Considered

| Alternative | Why Not? |
|-------------|----------|
| `swaggo/swag` | Code generation focus, too heavy for validation-only |
| `deepmap/oapi-codegen` | Generates server stubs (not needed) |
| `go-swagger` | Swagger 2.0 focus, heavyweight |
| Manual validation | Reinventing wheel, error-prone |

### Dependency Cost

```
Direct dependencies added: 12
- github.com/getkin/kin-openapi
- github.com/gorilla/mux (router matching)
- github.com/go-openapi/{jsonpointer,swag} (schema refs)
- github.com/mohae/deepcopy (schema cloning)
- github.com/perimeterx/marshmallow (JSON handling)
- Others (YAML parsing, utilities)

Binary size impact: ~2MB (negligible for server app)
```

### What It Does

1. **Parses OpenAPI 3.x YAML/JSON** - Validates syntax and structure
2. **Validates Requests** - Checks path params, query strings, headers
3. **Validates Responses** - Checks status codes, schemas, content-types
4. **Reference Resolution** - Handles `$ref` pointers correctly

### Usage in Penshort

```go
// Load spec
spec, _ := openapi3.NewLoader().LoadFromFile("openapi.yaml")

// Validate response against schema
err := openapi3filter.ValidateResponse(ctx, responseInput)
```

**LOC**: ~250 lines in `contract_test.go`  
**Complexity**: Low (library handles hard parts)  
**Maintenance burden**: Minimal (stable API)

---

## Conclusion

`kin-openapi` is the **minimal, justified choice** for OpenAPI contract testing in Go.

- ✅ No code generation bloat
- ✅ Runtime validation only
- ✅ Industry standard library
- ✅ Active maintenance
