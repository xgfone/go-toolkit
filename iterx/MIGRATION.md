# iterx: v0.30.2 to v1.0.0

This guide compares v0.30.2 with v1.0.0. Examples omit imports and declarations of existing variables.

## 1. Additions

- `To2`: transform two-value iterators; `Keys` / `Values`: project either value.
- `FilterTo` / `FilterTo2`: transform and filter in a single callback.
- `Find`, `Take`, `Drop`, `Concat`, `Reduce`: search, limit, skip, concatenate, and fold.
- `SumFunc` / `CountFunc`: replace the callback forms of `Sum` / `Count`; see section 3.

## 2. Removals

| Previous API         | Replacement                                                                                                      |
| -------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `All(seq, p)`        | Find the first element that fails p; return true if none is found. See below.                                    |
| `Any(seq, p)`        | `_, ok := iterx.Find(seq, p)`; ok is the result.                                                                 |
| `Seq(seq2, f)`       | Use `Keys` / `Values` for one side, or the composition below when both values are needed.                        |
| `Seq2(seq, f)`       | For containers, use `slices.All` / `maps.All` with `iterx.To2`; for arbitrary Seq inputs, use the adapter below. |
| `Integer` / `Number` | Define the constraints locally if needed; see below.                                                             |

Replace `All`, preserving short-circuiting and true for empty input:

```go
_, failed := iterx.Find(seq, func(v T) bool { return !predicate(v) })
all := !failed
```

Replace `Seq(seq2, convert)`, preserving laziness, order, and early termination:

```go
out := iterx.Values(iterx.To2(seq2, func(k K, v V) (K, R) {
    return k, convert(k, v)
}))
```

There is no direct replacement for arbitrary Seq-to-Seq2 conversion. Define this adapter locally:

```go
func seq2[T, K, V any](seq iter.Seq[T], convert func(T) (K, V)) iter.Seq2[K, V] {
    return func(yield func(K, V) bool) {
        for v := range seq {
            if !yield(convert(v)) {
                return
            }
        }
    }
}
```

To retain the original numeric constraints:

```go
type Integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Number interface {
    Integer | ~float32 | ~float64
}
```

## 3. Breaking Changes

The minimum Go version increases from 1.24 to 1.27. Upgrade your toolchain before updating the module.

| Previous call         | New call                  | Notes                                          |
| --------------------- | ------------------------- | ---------------------------------------------- |
| `iterx.Map(seq, f)`   | `iterx.To(seq, f)`        | Renamed; transformation remains lazy.          |
| `iterx.Sum(seq, f)`   | `iterx.SumFunc(seq, f)`   | `Sum(seq)` now sums numeric elements directly. |
| `iterx.Count(seq, p)` | `iterx.CountFunc(seq, p)` | `Count(seq)` now counts all elements.          |

```go
sum := iterx.SumFunc(slices.Values([]string{"a", "bb"}), func(s string) int {
    return len(s)
}) // 3

count := iterx.Count(slices.Values([]string{"a", "bb"})) // 2
```
