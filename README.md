# A Collections Of Base Tool Kits

[![codecov](https://codecov.io/gh/xgfone/go-toolkit/branch/main/graph/badge.svg)](https://codecov.io/gh/xgfone/go-toolkit)
[![Build Status](https://github.com/xgfone/go-toolkit/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-toolkit/actions/workflows/go.yml)
[![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-toolkit)](https://pkg.go.dev/github.com/xgfone/go-toolkit)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-toolkit/main/LICENSE)
![Minimum Go Version](https://img.shields.io/github/go-mod/go-version/xgfone/go-toolkit?label=Go%2B)
![Latest SemVer](https://img.shields.io/github/v/tag/xgfone/go-toolkit?sort=semver)

## Requirements

Go 1.27 or later is required. Generic `httpx.Context.BindBody`, `BindQuery`,
`BindHeader`, and `BindPath` methods are available without version build tags.

## Migrating removed APIs

- Replace `codeint.Error.TryError` with `Error.Wrap`.
- Replace `jsonx.Marshal` and `jsonx.Unmarshal` with `MarshalBytes` and `UnmarshalBytes`.
- Replace `logger.Config.Logger` with `Config.Middleware`.
- Replace direct access to `timex.Location` with `GetLocation` and `SetLocation`.
- Keep time format settings in the caller; `timex.Format`, `Formats`, and their
  getters/setters have been removed.
- Define upload interfaces in the consuming package; `iox.Uploader` and
  `UploaderFunc` have been removed.
