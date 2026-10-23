# A Collections Of Base Tool Kits

[![codecov](https://codecov.io/gh/xgfone/go-toolkit/branch/main/graph/badge.svg)](https://codecov.io/gh/xgfone/go-toolkit)
[![Build Status](https://github.com/xgfone/go-toolkit/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-toolkit/actions/workflows/go.yml)
[![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-toolkit)](https://pkg.go.dev/github.com/xgfone/go-toolkit)
![Minimum Go Version](https://img.shields.io/github/go-mod/go-version/xgfone/go-toolkit?label=Go%2B)
![Latest SemVer](https://img.shields.io/github/v/tag/xgfone/go-toolkit?sort=semver)

Packages under [`exp/`](exp) are experimental and carry no compatibility guarantees,
including in v1 releases. They may change, be removed, or move outside `exp/` to
become stable packages at any time.

Current v1 development requires Go 1.27 or newer and has no third-party module
dependencies. Import the required subpackage, such as
`github.com/xgfone/go-toolkit/iterx`; the module root is not a Go API package.
Use documentation from the release tag or commit selected by your application.

## Packages

| Task                                                               | Packages                                                                                                                           |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| Application lifecycle, modules, and build version command          | [`app`](app), [`app/module`](app/module), [`app/cmd/version`](app/cmd/version)                                                     |
| HTTP clients, request binding, context handlers, and responses     | [`httpx`](httpx)                                                                                                                   |
| Routing, middleware, CORS, and request logs                        | [`httpx/router`](httpx/router), [`httpx/middleware`](httpx/middleware), [`httpx/cors`](httpx/cors), [`httpx/logger`](httpx/logger) |
| Lazy iterator transforms and aggregation                           | [`iterx`](iterx)                                                                                                                   |
| Slice conversion, grouping, and deduplication; map conversion      | [`slicex`](slicex), [`mapx`](mapx)                                                                                                 |
| String helpers, masks, and generators                              | [`stringx`](stringx/README.md)                                                                                                     |
| Struct binding, defaults, and validation                           | [`structx`](structx), [`validation`](validation)                                                                                   |
| Error codes, wrapped errors, and result responses                  | [`codeint`](codeint), [`errorx`](errorx), [`result`](result)                                                                       |
| Clocks, cancellable waits, random values, and integer-list parsing | [`timex`](timex), [`random`](random), [`strconvx`](strconvx)                                                                       |
| Networking and address conversion                                  | [`netx`](netx), [`netx/netipx`](netx/netipx)                                                                                       |
| Comment stripping, I/O adapters, reflection, and runtime helpers   | [`bytex`](bytex), [`iox`](iox), [`reflectx`](reflectx), [`runtimex`](runtimex)                                                     |
| Zero-copy string/byte conversions                                  | [`unsafex`](unsafex)                                                                                                               |
| Experimental streams and redaction                                 | [`exp/iterx`](exp/iterx), [`exp/redact`](exp/redact); see [experimental policy](exp/README.md)                                     |

Package comments and example tests document usage and behavior.
For migration from v0.30.2 to v1, see [MIGRATION.md](MIGRATION.md).

## AI Agent skill

The [go-toolkit skill](.agents/skills/go-toolkit/SKILL.md) guides agents through
package selection, version-aware documentation lookup, and important usage
contracts. Use the documentation from the tag or revision selected by the
application; that revision's APIs remain authoritative.

Adding this Go dependency does **not** install the skill or load this library's
agent instructions into the consuming project's session. Install the skill
separately in the consuming project's `.agents/skills` directory, or in
`~/.agents/skills` for use across your projects. These are supported discovery
locations for [Codex skills](https://learn.chatgpt.com/docs/build-skills).
Other agents may require a different location or an explicit instruction to
read `SKILL.md`.

### Install in a consuming project

Obtain a checkout of this repository at a tag or commit containing the skill.
Run these commands in the consuming project's root, replacing the example
source path with that checkout's path:

```shell
TOOLKIT_SOURCE=/path/to/go-toolkit
mkdir -p .agents/skills
cp -R "$TOOLKIT_SOURCE/.agents/skills/go-toolkit" .agents/skills/
```

Commit the installed `.agents/skills/go-toolkit` directory in the consuming
project to share it with the team. Record the source tag or commit when installing
or updating it. Older tags may not contain the skill; choose a revision containing
the needed guidance, then let the skill resolve the application's actual dependency.

For a personal installation instead, copy the same directory to the user scope:

```shell
TOOLKIT_SOURCE=/path/to/go-toolkit
mkdir -p "$HOME/.agents/skills"
cp -R "$TOOLKIT_SOURCE/.agents/skills/go-toolkit" "$HOME/.agents/skills/"
```

For a copied installation, copy the whole skill directory, including
`references/`, when installing or updating. The installed skill locates source
and documentation with
`go list -m -json github.com/xgfone/go-toolkit`, including local replacements.
The copied files do not depend on the original installation checkout remaining
at that path. If documents are missing locally, the agent needs access to the
matching repository revision.

### Install from a local checkout with a symlink

For personal use, Codex also supports a directory symlink in `~/.agents/skills`.
This links the whole skill directory, including `references/`, to a local
checkout so updates need no copying. Use an absolute source path and keep that
checkout at a stable location:

```shell
TOOLKIT_SOURCE=/absolute/path/to/go-toolkit
mkdir -p "$HOME/.agents/skills"
test -f "$TOOLKIT_SOURCE/.agents/skills/go-toolkit/SKILL.md" &&
    test ! -e "$HOME/.agents/skills/go-toolkit" &&
    test ! -L "$HOME/.agents/skills/go-toolkit" &&
    ln -s "$TOOLKIT_SOURCE/.agents/skills/go-toolkit" "$HOME/.agents/skills/go-toolkit"
```

The checks require a valid source and refuse any existing destination, including
a broken symlink. To convert an existing copied directory, first move it to a
backup outside the scanned skills directory, then run the installation above:

```shell
test -d "$HOME/.agents/skills/go-toolkit" &&
    test ! -L "$HOME/.agents/skills/go-toolkit" &&
    mkdir -p "$HOME/.agents/skill-backups" &&
    TOOLKIT_SKILL_BACKUP=$(mktemp -d "$HOME/.agents/skill-backups/go-toolkit.XXXXXX") &&
    mv "$HOME/.agents/skills/go-toolkit" "$TOOLKIT_SKILL_BACKUP/go-toolkit" &&
    printf 'Backup: %s\n' "$TOOLKIT_SKILL_BACKUP"
```

The conversion keeps the original files and leaves existing symlinks alone.
Keep project-scoped copies in version control for team sharing; an absolute
symlink to your personal checkout is specific to your machine.

### Keep the skill up to date

A copied installation needs a new copy of the entire skill directory from the
chosen source revision. A symlink reads the linked checkout directly: updates to
`SKILL.md` and files under `references/` are immediately visible through the
user-level entry, with no reinstallation. It does not fetch Git changes
automatically; update the source checkout to the desired branch, tag, or commit
yourself. Branch switches and uncommitted skill edits are also visible through
the link. Moving the checkout or selecting a revision without the skill directory
makes the link unusable.

Update the repository's `SKILL.md` and references when API guidance changes;
linking files does not generate new guidance from code changes. The consuming
project's actual dependency version still determines which API documentation
to use. If an update is not reflected in Codex, restart it so the skill can be
discovered again.

### Use the skill

In Codex CLI or the IDE extension, invoke the installed skill explicitly:

```text
$go-toolkit Parse valid integers lazily with iterx and collect the first two.
```

Codex can also select the skill when the task matches its description. For
consistent use, add this rule to the consuming project's root `AGENTS.md`:

```markdown
When using or fixing github.com/xgfone/go-toolkit, use the go-toolkit skill.
First resolve the effective dependency version and read that version's package docs.
Packages under exp/ have no compatibility guarantees, including in v1.
```

If the skill does not appear, restart the agent session. Keep the skill updated
with library releases and use only guidance that matches the application's
selected revision.
