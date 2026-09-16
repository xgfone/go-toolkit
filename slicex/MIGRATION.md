# slicex: v0.30.2 to v1.0.0

This guide compares v0.30.2 with v1.0.0. Examples omit imports and declarations of existing variables.

## 1. Additions

- `GroupBy`: group elements by a key, preserving order within each group.
- `HasDuplicates`: detect duplicate elements, including non-adjacent ones.
- `Unique`: remove duplicates while preserving first-occurrence order and nilness.

`FilterTo` is the renamed v0.30.2 `Filter`; see section 3.

## 2. Removals

| Previous API                         | Replacement                                                                           |
| ------------------------------------ | ------------------------------------------------------------------------------------- |
| `Convert(s, f)`                      | `slicex.To(s, f)`                                                                     |
| `Empty(s)`                           | `if s == nil { s = S{} }`, where S is the original slice type.                        |
| `ValuesFunc(s, f)`                   | `iterx.To(slices.Values(s), f)`; transformation remains lazy.                         |
| `Collect(capacity, seq)`             | Preallocate, then call `slices.AppendSeq`; see below.                                 |
| `Merge(ss...)`                       | Usually `slices.Concat(ss...)`; to preserve the original sharing behavior, see below. |
| `ContainsAllFunc(super, sub, equal)` | Nested `slices.ContainsFunc` calls; see below.                                        |
| `ContainsAll(super, sub)`            | Nested searches for small inputs; to retain the original linear algorithm, see below. |

**Collecting an iterator:** preserve nil for empty input when capacity is non-positive.

```go
var out []T
if capacity > 0 {
    out = make([]T, 0, capacity)
}
out = slices.AppendSeq(out, seq)
```

**Containment:** replace `ContainsAllFunc`, preserving comparison order, duplicate handling, and short-circuiting.

```go
all := !slices.ContainsFunc(sub, func(b E2) bool {
    return !slices.ContainsFunc(super, func(a E1) bool {
        return equal(a, b)
    })
})
```

For `ContainsAll`, the inner call can be `slices.Contains(super, b)`, but time becomes
O(nm). To retain expected O(n+m) time and map-key comparison semantics, define:

```go
func containsAll[S1 ~[]E, S2 ~[]E, E comparable](super S1, sub S2) bool {
    if len(sub) == 0 {
        return true
    }
    if len(super) == 0 {
        return false
    }

    seen := make(map[E]struct{}, len(super))
    for _, v := range super {
        seen[v] = struct{}{}
    }

    for _, v := range sub {
        if _, ok := seen[v]; !ok {
            return false
        }
    }

    return true
}
```

**Concatenating slices:** `slices.Concat` copies a single non-empty input and returns nil
when the total length is zero. To preserve the old `Merge` behavior, including sharing
for a single input, non-nil results for multiple empty inputs, and capacity allocation:

```go
func merge[S ~[]E, E any](ss ...S) S {
    if len(ss) == 0 {
        return nil
    }
    if len(ss) == 1 {
        return ss[0]
    }

    n := 0
    for _, s := range ss {
        n += len(s)
    }

    out := make(S, 0, n)
    for _, s := range ss {
        out = append(out, s...)
    }

    return out
}
```

## 3. Breaking Changes

The minimum Go version increases from 1.24 to 1.27. Upgrade your toolchain before updating the module.

`Filter(s, convert)` is renamed to `FilterTo(s, convert)`. The callback still returns
`(value, bool)` and behavior is unchanged.

```go
out := slicex.FilterTo([]string{"12", "bad"}, func(s string) (int, bool) {
    n, err := strconv.Atoi(s)
    return n, err == nil
}) // []int{12}
```
