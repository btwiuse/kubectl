# Experiment Report: Porting `k8s.io/kubectl` to `js/wasm`

## Goal

Evaluate how far this repository can be built for WebAssembly (`GOOS=js`, `GOARCH=wasm`), document blockers, and implement low-risk fixes where feasible.

## Environment

- Repository: `k8s.io/kubectl`
- Date: 2026-05-24
- Toolchain: `go1.26.0`

## Steps Performed

1. Verified baseline repository health on native target:
   - `go test ./...`
   - Result: ✅ pass
2. Tried direct wasm test run:
   - `GOOS=js GOARCH=wasm go test ./...`
   - Result: ❌ fails (expected for direct execution without wasm test runtime wiring), plus compile errors.
3. Focused on pure compile feasibility:
   - `GOOS=js GOARCH=wasm go build ./...`
   - Initial blockers:
     - `pkg/util/interrupt/interrupt.go`: unsupported `syscall.SIGHUP`/POSIX signals
     - `pkg/util/umask.go`: unsupported `unix.Umask`
     - terminal stack via `github.com/moby/term`
4. Implemented targeted compatibility changes:
   - Added js-specific interrupt handler implementation (`pkg/util/interrupt/interrupt_js.go`).
   - Added js-specific umask stub (`pkg/util/umask_js.go`), and tightened build tags on existing unix implementation.
   - Added js-specific terminal shim package implementation (`pkg/util/term/term_js.go`), and excluded non-js terminal files with build tags.
   - Refactored `pkg/cmd/exec/exec.go` to remove direct `github.com/moby/term` import from shared file; introduced platform-specific `stdStreams` helper (`stdstreams.go`, `stdstreams_js.go`).
5. Removed the transitive terminal blocker for js builds:
   - Added a local patched module copy at `third_party/moby-term`.
   - Added `replace github.com/moby/term => ./third_party/moby-term` in `go.mod`.
   - Added js stubs in that module so js targets always report non-terminal behavior.
6. Addressed next wasm-only compile blocker:
   - Split plugin exec syscall path by build tags (`pkg/cmd/plugin_exec_supported.go`, `pkg/cmd/plugin_exec_unsupported.go`) so js avoids `syscall.Exec`.
7. Re-validated:
   - `go test ./...` → ✅ pass after changes
   - `GOOS=js GOARCH=wasm go build ./...` → ✅ pass

## Current Build Status

`GOOS=js GOARCH=wasm go build ./...` now succeeds in this branch by using a js-safe local replacement for `github.com/moby/term` and by stubbing terminal capability detection to non-terminal behavior on js.

## Challenges and Resolution Status

### 1) POSIX signal handling (`interrupt`)
- **Challenge:** wasm runtime does not provide Linux signal semantics used in existing implementation.
- **Resolution:** ✅ solved locally via js-specific no-signal-compatible handler path.

### 2) Process umask (`util.Umask`)
- **Challenge:** `unix.Umask` is unavailable on js/wasm.
- **Resolution:** ✅ solved locally via js-specific unsupported-platform stub.

### 3) Terminal control and TTY resize
- **Challenge:** terminal ioctls and SIGWINCH paths are not valid on wasm.
- **Resolution:** ✅ solved locally in repository code by adding js terminal shims and build tags.

### 4) `cli-runtime` printer dependency on `moby/term`
- **Challenge:** transitive dependency hard-required non-wasm terminal APIs.
- **Resolution:** ✅ solved in this branch by patching `moby/term` locally and routing js builds to non-terminal stubs.

## Practical Next Steps (for full wasm viability)

1. Upstream or fork the js-safe `moby/term` behavior to avoid long-term maintenance of a local replace.
2. Carry platform capability abstraction (TTY/signals/umask) consistently across CLI runtime dependencies.
3. Define wasm execution mode scope (library-only vs interactive CLI parity), because browser-hosted wasm cannot fully mirror native terminal behavior.
4. Use a dedicated wasm execution harness (`go_js_wasm_exec` + Node/browser runtime) for wasm-targeted tests.

## Conclusion

This experiment demonstrates that **local kubectl repository blockers can be reduced with focused build-tag shims and a js-safe terminal dependency replacement**.  
The branch now builds for `js/wasm`; long-term maintainability still benefits from upstreaming or removing the local dependency fork.
