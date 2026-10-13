# Technology Decision Record (TDR-0001)

## Selection of Primary Implementation Language for Knodexa

- **Status:** Accepted
- **Date:** 2026-10-13
- **Decision Owner:** Principal Systems Engineer

---

## 1. Context and Problem Statement

Knodexa is an open-source, local-first engineering memory system for AI coding agents. It provides a CLI tool and an in-process / stdio Model Context Protocol (MCP) server that operates directly inside user repositories alongside AI coding agents.

The core runtime characteristics required are:
1. **Instantaneous CLI invocation** (sub-20ms) to ensure interactive commands and agent tool calls introduce near-zero latency overhead.
2. **Zero-dependency, single-binary distribution** across macOS, Linux, and Windows without requiring Python virtual environments, Node runtimes, or dynamic C libraries.
3. **Robust, embedded SQLite capabilities** with bulletproof concurrent read/write behavior and in-memory test capability.
4. **First-class filesystem, process (Git), and JSON-RPC 2.0 (MCP) primitives**.
5. **Deterministic, garbage-collected or memory-safe execution** with minimal cognitive overhead and high maintainability.

The candidate languages evaluated are **Go**, **Rust**, **C++**, and **Python**.

---

## 2. Evaluation Matrix

| Criterion | Go | Rust | C++ | Python |
| :--- | :---: | :---: | :---: | :---: |
| **Single-Binary Distribution** | **Exceptional** (Static executable, cross-compiles with `GOOS`/`GOARCH`) | **Exceptional** (Static binary, requires cross-toolchains) | **Moderate** (Complex cross-platform toolchains, libc issues) | **Poor** (Requires PyInstaller/uv/venv, heavy bundle, fragile) |
| **Startup Speed** | **Fast (< 10ms)** | **Fastest (< 3ms)** | **Fastest (< 3ms)** | **Slow (100–300ms interpreter cold start)** |
| **Memory Footprint** | **Low (10–25MB resident)** | **Minimal (3–10MB resident)** | **Minimal (3–10MB resident)** | **Moderate-High (40–80MB)** |
| **SQLite Ecosystem** | **Excellent** (`modernc.org/sqlite` pure-Go, zero CGO; or `mattn/go-sqlite3`) | **Excellent** (`rusqlite`) | **Good** (Native `sqlite3.c`, manual bindings) | **Good** (`sqlite3` stdlib, but slow async/concurrency) |
| **MCP Implementation** | **First-Class** (Clean JSON-RPC 2.0 over stdio, typed schemas, lightweight goroutines) | **Good** (Requires `tokio`, `serde_json`, heavier async machinery) | **Poor** (No standard ecosystem; verbose JSON/RPC setup) | **Good** (Official Python SDK exists, but incurs startup penalty) |
| **Filesystem & YAML/Markdown** | **Standardized & Battle-tested** (`os`, `path/filepath`, `gopkg.in/yaml.v3`, goldmark) | **Strong** (Strong typing, but slower compile times) | **Verbose & Fragile** (Manual file handling, external libraries) | **Easy**, but lacking static type guarantees across refactors |
| **Git Integration** | **Fast & Direct** (`os/exec` to git CLI or `go-git`) | **Good** (`git2` / libgit2 C-bindings or CLI) | **Complex** (`libgit2` linking) | **Slow** (`GitPython` is notorious for subprocess overhead) |
| **Concurrency Model** | **Best-in-class** (Goroutines, channels, native sync) | **Strong** (Send/Sync, fearless concurrency, but high syntax overhead) | **Complex** (Pointers, data races, manual locks) | **Weak** (GIL limitations, asyncio complexity) |
| **Maintainability & Readability** | **High** (Explicit error handling, uniform style, fast compilation) | **High** (Strong type system, but high cognitive load & borrow checker churn) | **Low** (Memory safety hazards, legacy idiom collisions) | **Moderate** (Dynamic typing causes silent bugs unless strictly linted) |
| **Cross-Platform Support** | **Tier-1 Native** (Windows, macOS, Linux, arm64/amd64 out-of-the-box) | **Tier-1 Native** | **Challenging** across compilers (MSVC/Clang/GCC) | **Fragile** packaging across OS environments |
| **No-LLM Core Engine Principle** | **Native** (Small, deterministic stdlib focus) | **Native** | **Native** | High temptation to pull in heavyweight AI frameworks |

---

## 3. Comparative Analysis

### 3.1 Python
While Python is ubiquitous in AI orchestration, it is fundamentally ill-suited for Knodexa's core mission:
- **CLI Startup Latency:** Python's 150-300ms interpreter boot time degrades agent loops where tools are invoked dozens of times.
- **Distribution Friction:** Packaging a CLI with SQLite, YAML parsers, and MCP support requires virtual environments, `uv`, or bloated PyInstaller executables.
- **Type Safety:** Maintaining graph consistency and file serialization across long-term refactors requires strict compile-time guarantees.

### 3.2 C++
C++ offers raw performance but introduces significant liabilities:
- Dependency management (CMake/vcpkg/Conan) adds substantial maintenance burden.
- Memory safety risks (dangling pointers, buffer overflows) during untrusted YAML/agent input parsing.
- Cross-platform build complexity.

### 3.3 Rust
Rust is a strong contender with exceptional performance and memory safety. However:
- Compile times for Rust projects with async runtimes (`tokio`), SQLite (`rusqlite`), and serialization (`serde`) are significantly higher than Go.
- For Knodexa’s workload (deterministic file parsing, graph indexing in SQLite, and stdio JSON-RPC), Go provides equivalent end-user throughput with vastly simpler code ergonomics and faster build/test feedback loops.

### 3.4 Go
Go matches the exact technical sweet spot for Knodexa:
- **Instant Compilation and Test Execution:** Tests run in milliseconds, ensuring fast CI and local iteration.
- **Pure-Go SQLite Option:** Using `modernc.org/sqlite` enables a 100% CGO-free, statically compiled single binary that can be compiled for any target (`GOOS=linux GOARCH=arm64 go build`) without cross-compilers.
- **Native Concurrency:** Goroutines and standard `sync` primitives make handling concurrent MCP requests and background indexing effortless.
- **Single Static Binary:** The final binary contains the CLI, the SQLite engine, and the MCP server in a single ~15MB self-contained executable.

---

## 4. Decision

We choose **Go (version 1.24+)** as the sole language for Knodexa.

### Rules of Engagement:
1. **Single Backend Language:** Do not mix Go with Python or Rust backend services.
2. **CGO-Free Builds:** Target pure-Go SQLite (`modernc.org/sqlite`) to guarantee effortless cross-compilation and distribution.
3. **Standard Library First:** Rely on standard library packages (`net/rpc`, `encoding/json`, `os/exec`, `path/filepath`, `sync`) wherever practical.
4. **Deterministic Storage:** Never embed LLM API clients inside the core engine. Knodexa is deterministic infrastructure consumed by external agents.
