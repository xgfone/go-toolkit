# Experimental packages

All packages under `exp/` are experimental and carry no compatibility guarantees,
including in v1 releases. Their APIs may change or be removed at any time. A package
may also move outside `exp/` when it is ready to become a supported, stable API.

These packages belong to the main `github.com/xgfone/go-toolkit` module; they do not
have separate module versions.

- [`exp/iterx`](iterx): lazy, chainable `Stream` and `Stream2` wrappers for Go iterators.

Experimental APIs use Rust-inspired names where the semantics fit Go. Selected
short names may be provided as aliases; each alias documents its primary method.
