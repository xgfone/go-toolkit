# mapx: v0.30.2 to v1.0.0

This guide compares v0.30.2 with v1.0.0. Examples omit imports and declarations of existing variables.

## 1. Additions

None.

## 2. Removals

| Previous API                          | Replacement                                                                     |
| ------------------------------------- | ------------------------------------------------------------------------------- |
| `Convert(m, f)`                       | `mapx.To(m, f)`                                                                 |
| `Empty(m)`                            | `if m == nil { m = M{} }`, where M is the original map type.                    |
| `NewSMap[T](capacity)`                | `make(mapx.SMap[T], capacity)`                                                  |
| `Collect(capacity, seq)`              | `make` followed by `maps.Insert`; see below.                                    |
| `Keys(m)` / `Values(m)`               | Collect `maps.Keys(m)` / `maps.Values(m)` into a preallocated slice; see below. |
| `KeysFunc(m, f)` / `ValuesFunc(m, f)` | Add an `iterx.To` transformation before collecting; see below.                  |
| `All(m)` / `Pair`                     | Use `for k, v := range maps.All(m)`; to retain the Pair iterator, see below.    |
| `Filter(m, convert)`                  | `iterx.FilterTo2` with preallocated collection; see below.                      |

**Collecting an iterator:** preserve the capacity hint, non-nil results for empty input,
and later values overwriting earlier entries with the same key.

```go
out := make(map[K]V, max(0, capacity))
maps.Insert(out, seq)
```

**Collecting keys and values:** preserve preallocation and non-nil slices for empty input.
Order remains unspecified.

```go
keys := slices.AppendSeq(make([]K, 0, len(m)), maps.Keys(m))
values := slices.AppendSeq(make([]V, 0, len(m)), maps.Values(m))

// Replaces ValuesFunc; for KeysFunc, use maps.Keys instead of maps.Values.
converted := slices.AppendSeq(make([]T, 0, len(m)), iterx.To(maps.Values(m), convert))
```

**Filtering and converting:** replace v0.30.2 `Filter`, preserving
nilness and preallocation and calling the callback once per entry.

```go
var out map[K2]V2
if m != nil {
    out = make(map[K2]V2, len(m))
    maps.Insert(out, iterx.FilterTo2(maps.All(m), convert))
}
```

This preserves support for NaN keys and overwriting duplicate converted keys.
For a local implementation, check for nil, preallocate, then iterate over m and
write the converted key/value only when the callback returns true.

**Retaining the Pair iterator:** if callers require `iter.Seq[Pair]`, define it locally:

```go
type Pair[K comparable, V any] struct {
    Key   K
    Value V
}

func pairs[M ~map[K]V, K comparable, V any](m M) iter.Seq[Pair[K, V]] {
    return func(yield func(Pair[K, V]) bool) {
        for k, v := range m {
            if !yield(Pair[K, V]{Key: k, Value: v}) {
                return
            }
        }
    }
}
```

## 3. Breaking Changes

The minimum Go version increases from 1.24 to 1.27. Upgrade your toolchain before updating the module.

Beyond the removals above, `To`, `SMap.Get`, and `SMap.IsZero` retain their v0.30.2 signatures and behavior.
