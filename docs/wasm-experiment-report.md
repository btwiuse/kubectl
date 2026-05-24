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
5. Re-validated:
   - `go test ./...` → ✅ pass after changes
   - `GOOS=js GOARCH=wasm go build ./...` → ❌ still fails

## Remaining Blocker

The remaining wasm build blocker is external:

- `github.com/moby/term` does not compile for `js/wasm` (`golang.org/x/sys/unix` terminal ioctl symbols unavailable).
- Dependency path discovery shows `k8s.io/cli-runtime/pkg/printers` imports `github.com/moby/term`, and `kubectl` depends broadly on `cli-runtime/pkg/printers`.

This means the project is still not globally buildable for `js/wasm` without upstream or deeper dependency-level changes.

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
- **Challenge:** transitive dependency still hard-requires non-wasm terminal APIs.
- **Resolution:** ⚠️ partially solved. Direct imports in this repo were reduced, but upstream/transitive dependency still blocks full build.

## Practical Next Steps (for full wasm viability)

1. Add/obtain wasm-safe terminal capability layer in `k8s.io/cli-runtime/pkg/printers` (or remove hard dependency on `moby/term` when `GOOS=js`).
2. Carry platform capability abstraction (TTY/signals/umask) consistently across CLI runtime dependencies.
3. Define wasm execution mode scope (library-only vs interactive CLI parity), because browser-hosted wasm cannot fully mirror native terminal behavior.
4. Use a dedicated wasm execution harness (`go_js_wasm_exec` + Node/browser runtime) for any future wasm-targeted tests.

## Conclusion

This experiment demonstrates that **local kubectl repository blockers can be reduced with focused build-tag shims**, but **full `js/wasm` port is currently blocked by transitive terminal dependencies in `k8s.io/cli-runtime` (`moby/term`)**.  
The codebase is now better structured for platform-specific behavior, but complete wasm support requires upstream dependency changes.
