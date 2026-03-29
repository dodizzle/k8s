# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go CLI tool (`k8s`) for managing Kubernetes contexts, namespaces, and GCP projects. It wraps `kubectl` and `gcloud` commands with an interactive numbered-menu interface.

## Build & Run

```bash
go build -o bin/k8s k8s.go    # build binary
go run k8s.go -cc              # run directly
```

No tests exist yet. No linter is configured.

## Architecture

Single-file Go CLI (`k8s.go`) — everything lives in `package main`. The binary is a thin interactive wrapper around `kubectl` and `gcloud` shell commands using `os/exec`.

**Pattern:** Most commands follow the same flow:
1. Shell out to `kubectl`/`gcloud` to get a list (contexts, namespaces, projects)
2. Parse output into a numbered map, render with `tablewriter`
3. Prompt user for a number via `bufio.Reader`
4. Shell out again to apply the selection

**CLI flags:** parsed manually via `os.Args` (no flag library). Flags: `-cc`, `-cn`, `-cp`, `-dc`, `-lc`, `-ln`, `-lp`, `-rc`, `-t`.

**External dependencies:** only `github.com/olekukonko/tablewriter` for table formatting.
