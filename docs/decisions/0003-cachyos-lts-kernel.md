# ADR-0003: Use the CachyOS LTS kernel

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-10-28 |
| **Deciders** | Ryan Wendell |

## Context

SlippiOS ships one kernel for every machine that boots the image. Melee is
latency-sensitive and CPU-bound at 1x native resolution, so the kernel choice
matters more than the GPU driver for 4-player frame pacing.

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
| CachyOS LTS 6.18.55 | 0.9940 | 0.8596 | 4.42 GHz |
| CachyOS BORE 7.3.0 | 0.9945 | 0.8633 | 4.40 GHz |
| Stock NixOS 6.18.57 | 0.9822 | 0.7992 | 3.06 GHz |

The stock result was measured with the shell running, so treat it as
directional rather than clean. It also uses `madvise` transparent hugepages
while the CachyOS kernels use `always`.

## Decision

Release images use the CachyOS LTS kernel (`linux-cachyos-lts`, 6.18 series).
Dev configurations may override the variant for testing. The CachyOS binary
cache is used so the kernel is substituted instead of compiled.

LTS and BORE are effectively tied on this hardware: the differences are
0.05% on singles and 0.4% on 4-player, which is within run-to-run noise. LTS is
the safer release choice because it stays on a long-term branch and is the
variant the CachyOS project supports for longer.

## Consequences

**Positive:** The release gets the kernel that was fastest or tied-fastest in
the clean comparison, on a maintained LTS branch. The CachyOS cache keeps image
builds fast. The benchmark harness now stops the shell, so future comparisons
measure the emulator instead of the UI.

**Negative:** The image is tied to the CachyOS kernel packaging and its cache.
BORE remains interesting for newer hardware but does not justify a non-LTS
kernel for a small, noisy difference. The stock-kernel result was not rerun
after the shell fix.

**Neutral:** The shell's idle CPU cost is a separate issue. It affects every
kernel and should be reduced for the release (frame cap or Godot
low-processor mode), but it does not change the kernel ranking.

## Alternatives considered

**Stock NixOS 6.18.57** - For: no third-party kernel source, simplest update
story. Against: slower in the comparison, `madvise` THP, and no CachyOS
gaming patches. Why not chosen: both CachyOS kernels were ahead.

**CachyOS BORE 7.3.0** - For: marginally faster in the clean run. Against: a
newer kernel branch, no meaningful gain outside noise, and more churn for a
fixed release. Why not chosen: the difference does not justify the support
surface.

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
