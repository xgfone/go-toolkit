# stringx

## Desensitizers

`Desensitizer` exposes `Desensitize(string) string`. `DesensitizerFunc` adapts a
custom function. `NewDesensitizer` returns a concrete `MaskDesensitizer` whose
configuration is private and accessible through `Left()`, `Right()`, and `Chars()`.
Its `WithLeft`, `WithRight`, and `WithChars` methods return independent values.

```go
d := stringx.NewDesensitizer(3, 4).WithChars("***")
masked := d.Desensitize("13812345678") // "138***5678"
phone := stringx.PhoneDesensitizer().Desensitize("13812345678")
email := stringx.EmailDesensitizer().Desensitize("alice@example.com") // "a****@example.com"
```

Retained lengths count runes, and negative counts panic. If the input has no
more runes than the combined retained counts, it is replaced entirely. An empty
replacement uses `"****"` for non-empty input and preserves empty input. The zero
value follows this empty-replacement behavior.

`SetPhoneDesensitizer`, `SetEmailDesensitizer`, `SetShortDesensitizer`,
`SetDefaultDesensitizer`, and `SetPasswordDesensitizer` accept custom `Desensitizer`
implementations. The initial
implementations are shared pointers; getters return the stored interface without
creating a new instance. Getters and setters are safe to call concurrently, and
replacing a preset does not change previously retrieved instances. Custom
implementations must provide their own concurrency guarantees. Nil interfaces
and typed nils are rejected before replacing the current implementation.

The default email implementation preserves the first rune of the local part and
the complete domain, replacing the rest of the local part with `"****"`. A
single-rune local part is entirely masked: `a@example.com` becomes
`****@example.com`. Empty input stays empty, and unparseable input becomes `"****"`.
It uses `net/mail.ParseAddress` to parse one address; display names and comments
are discarded. Unicode local parts are masked by rune, and address lists are
not supported. Use `SetEmailDesensitizer` to replace this policy.

`NewEmailDesensitizer(local)` returns a `Desensitizer` using a copy of the supplied
`MaskDesensitizer`; its concrete implementation is private. Configure the local
mask before creating the email desensitizer, or create another one to use a new
configuration. A zero-value mask hides the entire local part. Empty input and
invalid-input handling are independent of the custom local-part mask.

```go
d := stringx.NewEmailDesensitizer(
    stringx.NewDesensitizer(2, 1).WithChars("***"),
)
masked := d.Desensitize("abcdef@example.com") // "ab***f@example.com"
stringx.SetEmailDesensitizer(d)
```

```go
stringx.SetPhoneDesensitizer(stringx.DesensitizerFunc(func(s string) string {
    return "[phone]"
}))
```

## Generators

`Generator` exposes `MinLen() int`, `Generate(int) string`, and
`Append([]byte, int) []byte`. All lengths count bytes, including affixes.
Non-positive lengths use `MinLen()`; positive lengths below the minimum panic.
`Append` preserves the existing buffer and appends the requested number of bytes.

```go
g := stringx.NewAffixGenerator(stringx.DateTimeMilliRandGenerator()).
    WithPrefix("order_").WithSuffix("!")
s := g.Generate(32)
buf := g.Append([]byte("id="), 32) // 35 bytes, including the existing "id="

stringx.SetDefaultGenerator(g)
s = stringx.DefaultGenerator().Generate(32)
```

`NewGenerator(minLen, appendFunc)` returns a `FuncGenerator`. The callback must
preserve the existing buffer and append exactly the validated byte length passed
to it; incorrect output lengths panic. `Generate` copies the returned bytes into
a string so later buffer changes do not modify the string. Both generator types
require initialization; an `AffixGenerator` zero value may be initialized with
`WithGenerator`.

Default access and replacement use `atomic.Value` with a fixed struct wrapper,
allowing replacements with different concrete types without a read lock.
A previously retrieved generator
keeps its original implementation after replacement. Custom generators and
callbacks must provide their own concurrency guarantees, and callers must
synchronize access to shared buffers. The time presets use `timex.Now`; configure
`timex` before concurrent generation and supply a concurrency-safe clock function.

## Migration

This redesign changes public API names and signatures:

| Previous API | Replacement |
| --- | --- |
| Concrete `Desensitizer` | `MaskDesensitizer`; `Desensitizer` is now an interface |
| `d.Left`, `d.Right`, `d.Chars` | `d.Left()`, `d.Right()`, `d.Chars()`; configure with `WithXxx` |
| Desensitizer preset variables | Same names as getter functions, plus corresponding setters such as `SetPhoneDesensitizer(d)` |
| `Builder`, `NewBuilder(g)` | `AffixGenerator`, `NewAffixGenerator(g)` |
| `b.Build(n)` | `g.Generate(n)` |
| `g.Generate(dst, n)` | `g.Append(dst, n)` |
| `DefaultBuilder` | `DefaultGenerator()` and `SetDefaultGenerator(g)` |
| Time generator preset variables | Same names as getter functions |
| Package-level `Generate(dst, n, timeCallback)` | Internal helper; use a time preset or `NewGenerator` with a custom append callback |

The `NewGenerator` constructor now returns concrete `FuncGenerator`, implementing
the new `Generator` interface. Existing custom generator types need the new
string-returning `Generate` method and the renamed `Append` method. Configure
custom masks through `NewDesensitizer(...).WithXxx(...)`; preset getters return
the behavior interface without configuration methods.
