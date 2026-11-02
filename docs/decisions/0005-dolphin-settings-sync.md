# ADR-0005: Sync curated Dolphin settings back from the expert GUI

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-11-02 |
| **Deciders** | Ryan Wendell |

## Context

The shell owns a curated `settings.toml` and writes it into Dolphin's INI
files before every launch. Expert users can open the real Dolphin GUI from
Expert tools, where they can change the same settings plus many others. Today
those GUI changes live only in the INI files, and the next shell launch
overwrites the curated subset.

## Decision

Before opening the Dolphin settings GUI, write the curated shell settings into
the INIs. After the GUI exits, read the curated keys back:

- If a curated key changed, update `settings.toml` from the INI value.
- If a key is absent or cannot be parsed, leave the shell setting unchanged.
- Keys the shell does not curate stay in the INI files and are not copied into
  `settings.toml`.

The curated keys are the ones the shell exposes: renderer, resolution, aspect,
VSync, MSAA, audio backend and latency, Slippi music, shader precompilation,
shader compilation mode, netplay ping, and FPS display. The beta and stable
builds use different INI sections for some keys; the mapping lives in
`backend/internal/dolphin/managed.go`.

## Consequences

- A GUI change to a curated setting survives the next shell launch.
- A GUI change to an uncurated setting remains in the INI but is not visible in
  the shell. This is intentional: the shell only promises its curated subset.
- The sync is best effort. If Dolphin writes a key the shell cannot map, the
  shell keeps its previous value.
