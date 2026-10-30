# AGENTS.md — go-tpkt

## What this is

Pure Go TPKT framing (RFC 1006, RFC 2126): encode, decode and stream ISO transport packets. Profile: **foundational-transport**. Other OT Fabric libraries build on it, so compatibility matters more here than anywhere else.

## Organisation standards

This repository follows **otfabric/engineering policy 1.0**, profile **foundational-transport** (`.otfabric/policy.yaml`, mode `baseline`). Where this file and the organisation standards disagree, this file wins here — say so in your report.

## Where the truth lives

| Question | Source |
|---|---|
| Wire behaviour | RFC 1006, RFC 2126 |
| Errors | ERRORS.md (1:1 with exported errors) |
| Release history | RELEASE.md (`## vX.Y.Z` … `---`) |

## Hard rules

1. Never lower the coverage minimum (90), add lint suppressions, or skip/delete tests to make CI pass.
2. Exported API changes are breaking for go-cotp, and through it go-s7comm, go-mms and go-iec61850: this library stays on v0 or v1, and a breaking change is listed under "Breaking" in RELEASE.md.
3. No new runtime dependency without an ADR.

## Commands

| When you change… | Run |
|---|---|
| anything | `make check` |
| a decoder | `make fuzz` |

## Definition of Done

`make check` green; tests for new behaviour incl. error paths; fuzz target for new decoders; ERRORS.md updated; RELEASE.md section for user-visible changes.

## Git and releases

Branches and PRs. Do not push to `main`, tag or run the Release workflow unless the maintainer asks in the session.
