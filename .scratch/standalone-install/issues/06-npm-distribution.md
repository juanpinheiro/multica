# Issue 06: npm distribution — `multica` package + per-platform binaries

**Status:** `ready-for-agent`
**Model:** `sonnet`

## Parent

`.scratch/standalone-install/PRD.md`

## What to build

Package the runtime so a developer installs it with `npm i -g multica` and then runs `multica up`. The published `multica` package is a thin JS wrapper with a `bin` shim that dispatches to the platform-specific native artifact. Per-platform packages (e.g. `multica-win32-x64`, `multica-darwin-arm64`, `multica-linux-x64`) are declared as `optionalDependencies` and carry the Go binary (server + daemon + supervisor) plus the embedded Postgres; the Next.js `output: "standalone"` web bundle and migrations ship in the main JS package. This is the esbuild / next-swc distribution pattern.

Include the build/release tooling that produces the per-platform artifacts from the Go build + web build, and assembles the npm packages. Windows must be a first-class target.

## Acceptance criteria

- [ ] `npm i -g multica` installs the wrapper plus only the current platform's native artifact via `optionalDependencies`.
- [ ] After install, `multica up` runs the full stack (the supervisor, embedded Postgres, and the spawned monitor from Issues 02–03) from the installed package.
- [ ] An unsupported platform fails at install time with a clear message naming the platform, not a cryptic error at `multica up`.
- [ ] The `bin` shim correctly resolves and invokes the platform binary on macOS, Linux, and Windows.
- [ ] Build/release tooling produces the per-platform artifacts and the JS package(s) reproducibly from the Go + web builds.
- [ ] A short install doc covers `npm i -g multica` → `multica up`.

## Blocked by

- Issue 03 (monitor spawn — packaging needs the supervisor binary and the web bundle)
