# ADR-0003: Use the CachyOS BORE kernel

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-10-28 |
| **Deciders** | Ryan Wendell |

## Context

SlippiOS ships one kernel for every machine that boots the image. Melee is
latency-sensitive and CPU-bound at 1x native resolution, so the kernel choice
matters more than the GPU driver for frame pacing.

We compared three kernels on the EM680 (Ryzen 7 6800U, Radeon 680M) with the
same settings:

- Display: 640x480 at 119.52 Hz
- Dolphin: 1x native, Vulkan
- CPU: `performance` governor and `performance` EPP
- Replays: `finalizedFrame.slp` (singles) and `doubles.slp` (4-player)
- Shell: stopped, so the Godot UI did not compete for CPU

The shell was running during the first comparison and idled at about 20% CPU.
That kept the CPU near 3.06 GHz and invalidated the kernel comparison. After
stopping the shell, both CachyOS kernels boosted to about 4.4 GHz.

| Kernel | Singles | 4-player | CPU max |
|---|---:|---:|---:|
| CachyOS BORE 7.3.0 | 0.9945 | 0.8633 | 4.40 GHz |
| CachyOS LTS 6.18.55 | 0.9940 | 0.8596 | 4.42 GHz |
| Stock NixOS 6.18.57 | 0.9822 | 0.7992 | 3.06 GHz |

The stock result was measured with the shell running, so treat it as
directional rather than clean. It also uses `madvise` transparent hugepages
while the CachyOS kernels use `always`.

## Decision

Release images use the CachyOS BORE kernel (`linux-cachyos-bore`, 7.3.0). BORE
was fastest in both clean replays. The margins are small — 0.05% on singles and
0.4% on 4-player — but singles is the priority, and BORE also won 4-player.

LTS remains the fallback for comparison and for anyone who prefers the longer
support branch. Dev configurations can override the kernel variant. The
CachyOS binary cache is used so the kernel is substituted instead of compiled.

## Consequences

**Positive:** The release gets the kernel that won both measured workloads,
including the priority singles case. The CachyOS cache keeps image builds fast.
The benchmark harness now stops the shell, so future comparisons measure the
emulator instead of the UI.

**Negative:** BORE is on a newer kernel branch than LTS, so it has a shorter
support window and more churn. The performance margin is small enough that
other hardware could reverse it. We accept that risk because the image ships a
tested kernel and updates are controlled; LTS is available as a fallback.

**Neutral:** The shell's idle CPU cost is a separate issue. It affects every
kernel and should be reduced for the release (frame cap or Godot
low-processor mode), but it does not change the kernel ranking.

## Alternatives considered

**CachyOS LTS 6.18.55** - For: longer support branch and effectively tied in
the clean comparison. Against: it lost both replays, including singles. Why not
chosen: BORE won the priority metric, and the stability difference is a
judgment call rather than a measured advantage.

**Stock NixOS 6.18.57** - For: no third-party kernel source, simplest update
story. Against: slower in the comparison, `madvise` THP, and no CachyOS gaming
patches. Why not chosen: both CachyOS kernels were ahead.

**Compile our own kernel** - For: full control. Against: slow builds, a large
maintenance burden, and no evidence of a gain over the cached CachyOS builds.
Why not chosen: the cache provides the tested kernel without the build cost.

## References

- [`flake.nix`](../../flake.nix)
- [`hosts/usb.nix`](../../hosts/usb.nix)
- [`modules/slippios/kernel.nix`](../../modules/slippios/kernel.nix)
- [`scripts/perf-test`](../../scripts/perf-test)
- [`scripts/perf-replay-device`](../../scripts/perf-replay-device)

Local benchmark results are in `.cache/perf-results` and are not committed.
