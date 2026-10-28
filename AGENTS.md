# Repository Development Guidelines

## Go Variable Declarations and Assignments

- When adding or modifying Go code, declare and assign variables individually by
  default. Avoid declarations or assignments involving multiple variables, such
  as `a, b := x, y`, `var a, b T`, and `a, b = x, y`.
- An ordinary declaration or assignment involving multiple variables may remain
  if it fits entirely on one formatted line of at most **60 characters**.
  Count the complete line, including indentation but excluding the newline.
- When such a statement exceeds the limit, split it into individual declarations
  or assignments. Do not bypass the rule by merely wrapping lines or declaring
  variables separately before assigning them together.
- Preserve evaluation order and simultaneous assignment semantics when splitting
  assignments. For swaps, cross-references, or side effects, first save the
  original values or computed results in temporary variables, then assign each
  variable individually.
- **Exception for multiple return values:** `a, b := f()` and `a, b = f()` are
  exempt from the multiple-variable rule and the 60-character limit. Variables
  need not be declared individually beforehand. If the `f()` call itself is too
  long, extract its arguments into local variables to keep the call clear. Do
  not call the function repeatedly. `_` does not count as a named variable.
- When splitting loop variables, preserve their separate scope for each
  iteration.

```go
first := make(chan struct{})
second := make(chan struct{})

ctx, cancel := context.WithCancel(parent)
```

## JSON Encoding and Decoding

- Use `encoding/json/v2` for JSON encoding and decoding in Go source, tests,
  and examples. Do not introduce `encoding/json` or third-party JSON codecs.
- Migration documents may describe the old `encoding/json` APIs and behavior
  at tag `v0.30.2`, but examples intended for v1 must use `encoding/json/v2`.

## Code Identifiers in Markdown

- When adding or modifying any Markdown document, including READMEs, migration
  guides, and skill documentation, use backticks for Go identifiers in prose,
  headings, lists, and tables. This includes package names, types, interfaces,
  methods, functions, fields, constants, variables, and type parameters, such as
  `context.Context`, `Module`, `App.Wait(ctx)`, `StageInit`, and `moduleName`.
- Keep a complete qualified name, call, or signature within one inline code
  span, such as `app.WithDependencies` or `Dependencies() []string`; do not split
  it across multiple spans. Also use inline code for predeclared identifiers such
  as `nil`, `true`, and `false`, and for keywords, type expressions, and struct
  tags when referring to them as code.
- Distinguish code names from natural language. Do not mechanically format
  ordinary verbs or the Go language name as code. Do not add inline backticks
  inside code blocks, and preserve the syntax of metadata such as YAML front
  matter.
- Use inline code when link text names a code identifier, such as [`app`](app),
  while keeping the link destination unchanged. After editing, check code
  blocks, inline code, links, and heading anchors to ensure correct Markdown
  rendering.

## Migration Documentation Maintenance

- All migration documents must describe API and behavior differences from
  **v0.30.2 to v1**. The baseline is always Git tag `v0.30.2`, and the target is
  the v1 implementation in the source tree containing the document. Do not turn
  these guides into migrations between v1.0.0-beta releases or other
  intermediate v1 versions.
- Verify old APIs, signatures, and behavior against the source at tag
  `v0.30.2`, and verify new APIs and behavior against the target v1 source.
  Do not present intermediate APIs that existed only during v1 development
  as v0.30.2 APIs.
- When changing v1 public APIs, behavior, lifecycle, toolchain requirements,
  or anything else affecting migration, update the relevant migration documents
  at the same time. Maintain a complete migration path directly from v0.30.2 to
  the target v1 implementation; do not append rename histories between beta
  releases as migration steps.
- Maintain the root and package-level `MIGRATION.md` files, the Migration
  section of `stringx/README.md`, and
  `.agents/skills/go-toolkit/references/migration-v0.30.2-to-v1.md`. Also check
  migration scope, indexes, and links in `README.md` and skill documentation to
  keep them consistent.
- When editing migration documentation, explain the actual impact and
  replacements, verify the old and new APIs in examples, and validate related
  documentation links.
