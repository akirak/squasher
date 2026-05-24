---
name: update-nix-vendor-hash
description: Update Nix `vendorHash` values for Go packages built with `buildGoModule` after `go.mod`, `go.sum`, or vendored dependencies change. Use when a flake or Nix package fails with a fixed-output hash mismatch, `inconsistent vendoring`, or a stale `vendorHash`, and Codex needs to replace the hash with a fake value, rebuild, capture the expected hash, and verify the final build.
---

# Update Nix Vendor Hash

## Overview

Refresh a Go package's `vendorHash` by forcing Nix to reveal the correct fixed-output hash, then rebuilding with the updated value.

## Workflow

1. Locate the Go package definition.
   Look for `buildGoModule` and identify the exact `vendorHash` field and the build target the user expects, such as `.#package-name`.

2. Replace the current hash with a fake hash.
   Use `sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=` unless the package already uses another conventional fake hash value for the same workflow.

3. Rebuild the package.
   Run the narrowest build target possible, typically `nix build .#package-name`.
   If Nix fails because the sandbox cannot access user cache paths, rerun the build with escalated permissions instead of changing the package logic.

4. Capture the expected hash.
   Read the fixed-output mismatch error and extract the `got:` hash value. Do not use the `specified:` fake hash.

5. Update `vendorHash`.
   Replace the fake hash with the expected hash exactly as reported by Nix.

6. Verify the result.
   Rebuild the same target and confirm the build exits successfully.

7. Report the outcome.
   Mention the file changed, the final hash, and whether the verification build succeeded.

## Guardrails

- Update only the relevant `vendorHash`; do not rewrite unrelated package attributes.
- Prefer the smallest possible build target over `nix build` for the whole flake.
- If multiple Go packages exist, match the package that corresponds to the user's failing derivation or attribute path.
- If the build fails for reasons unrelated to the vendor hash after the update, report that separately instead of guessing a new hash.
