# Secure Boot

SlippiOS can boot with Secure Boot enabled on most PCs. The first boot
requires a one-time MOK enrollment per PC. Later updates use the same key
and do not require another enrollment.

## Chain

1. Firmware trusts Microsoft's UEFI CA.
2. The ESP boots `EFI/BOOT/BOOTX64.EFI`, the Fedora 16.1-7 shim. It is
   signed by Microsoft UEFI CA 2011 and 2023.
3. The shim loads `grubx64.efi`, which is systemd-boot signed with the
   SlippiOS key. The shim trusts that key after MOK enrollment.
4. systemd-boot loads the SlippiOS-signed UKI from `EFI/Linux`.
5. The UKI boots the system.

The shim's embedded vendor certificate is Fedora's. SlippiOS does not
have a Fedora-signed bootloader; the SlippiOS key is trusted through MOK.

## First boot

See [ENROLL.txt](ENROLL.txt) for the enrollment steps. The certificate is
`slippios.cer` on the ESP. The enrollment is stored in the PC's firmware
NVRAM. It survives reboots and SlippiOS updates.

Some PCs need **Allow Microsoft 3rd Party UEFI CA** enabled; Intel T2 Macs
need **Allow booting from external or removable media**. If the shim cannot
boot, use the unsigned image with Secure Boot off.

## Updates

The updater writes a new UKI signed with the same key. It boots without
another enrollment. The release script extracts the signed UKI from the image,
so the update asset and the image always match.

The shim and MokManager are part of the image. They are not updated by
systemd-sysupdate. A shim update needs a new image.

## Signing

The public certificate is tracked at
[`secure-boot/slippios.crt`](slippios.crt) and embedded in every signed image
as `slippios.cer`. The private key is not in this repository.

Signing keys are resolved in this order:

1. `SLIPPIOS_SB_KEY` and `SLIPPIOS_SB_CERT` from the environment.
2. `~/.config/slippios/secureboot/slippios.key` and `.crt`.
3. The encrypted `~/.secrets/slippios-sb-key.age` and
   `~/.secrets/slippios-sb-cert.age` copies from chezmoi, decrypted on demand.
4. The private `SlippiOS/release-secrets` vault, decrypted into tmpfs for the
   duration of the command.

The `mise` tasks use `scripts/with-release-secrets`, so the vault is the normal
path for release managers. A developer without the vault can still build
unsigned images.

### Key generation

Generate a key and certificate only for the initial setup or a deliberate
rotation:

```sh
umask 077
openssl req -new -x509 -newkey rsa:4096 \
  -keyout slippios.key -out slippios.crt \
  -days 3650 -nodes -subj "/CN=SlippiOS Secure Boot"
```

Keep `slippios.key` offline and out of this repository. Back it up; losing every
copy means every console must enroll a new key. The public certificate can be
published and checked in.

The production certificate SHA-256 fingerprint is:

```text
B6:C7:CF:F2:FC:B0:0F:73:48:5F:49:67:C1:79:1A:01:B5:07:2A:68:F0:37:57:70:BE:E6:34:2F:8C:5F:72:F2
```

Check a certificate with
`openssl x509 -in secure-boot/slippios.crt -noout -fingerprint -sha256`.

### Release workflow

Build, sign, and create a draft release:

```sh
mise run release
```

Use `--dry-run` to build and sign without uploading, and `--publish` to publish
the draft:

```sh
mise run release -- --dry-run
mise run release -- --publish
```

Verify that the available key matches the public certificate:

```sh
scripts/with-release-secrets --check
scripts/with-release-secrets --vault --check
```

Sign an unsigned image without publishing:

```sh
mise run sign:image -- result-local-image/slippios_<version>.raw signed.raw
```

The tool downloads the pinned Fedora shim RPM, verifies its SHA-256 and both
Microsoft signatures, signs systemd-boot and the UKI, installs the shim and
MokManager, and writes the enrollment note and shim license. Signing is a
post-build step so the private key never enters the Nix store.

### Key custody

The private key and publish credentials live in the private
`SlippiOS/release-secrets` repository, age-encrypted to one recipient per
release manager. Each release manager keeps their own age identity; private
identities are never shared. See
[ADR-0008](../docs/decisions/0008-release-key-custody.md) for the rationale and
the onboarding and offboarding procedure.

On the author's personal machines, chezmoi also carries encrypted copies of the
key, certificate, and release age identity in `~/.secrets`. The wrapper uses
them when the local key or the vault clone is absent, so a second PC can sign
after `chezmoi update` without a separate key copy.

Only release managers need this access. Everyone else builds unsigned images.
The dev update channel uses the same key as stable, so publishing a dev build is
a release-manager action.

## Release checklist

1. Confirm the production key is available through the environment, the local
   key path, or the `SlippiOS/release-secrets` vault.
2. Run `scripts/with-release-secrets --check` and confirm the fingerprint
   matches the production certificate.
3. Confirm an offline backup of the private key exists.
4. Run `mise run release` to create a draft.
5. Confirm the release contains the signed image, unsigned image, signed UKI,
   store image, zsync file, manifest, and `SHA256SUMS`.
6. Boot the signed image on a real PC with Secure Boot on and complete the MOK
   enrollment.
7. Install an update and confirm it boots without re-enrollment.
8. Publish the draft with `gh release edit <tag> --draft=false --latest`.

The release script refuses to run if the key is missing or the working tree is
dirty. The first signed release should be treated as a release test until the
assets and hardware boot are verified.

## Testing

Use OVMF Secure Boot:

```sh
qemu-system-x86_64 ... \
  -drive if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE.secboot.fd \
  -drive if=pflash,format=raw,file=OVMF_VARS.secboot.fd
```

The first boot shows the shim verification error, then MokManager. Enroll
`slippios.cer`. After the reboot, `bootctl status` should report
`Secure Boot: enabled (user)` and the MOK list should contain the key.

For automated tests, set `MokTimeout` to 30 in the OVMF vars:

```sh
uvx --from virt-firmware virt-fw-vars -i OVMF_VARS.secboot.fd \
  --set-json secure-boot/moktimeout.json -o OVMF_VARS.secboot.fd
```

## Limits

- Some firmware requires `Allow Microsoft 3rd Party UEFI CA` to be
  enabled. If it is off, the shim does not boot.
- Some firmware only trusts the 2011 or 2023 CA. The Fedora shim is
  dual-signed, so it covers both, but older firmware may not have either
  CA updated.
- A firmware or dbx update can revoke the shim. The USB then needs a new
  image with a current shim, or Secure Boot off.
- Intel T2 Macs require `Allow booting from external or removable media`.
  Apple Silicon cannot run this x86_64 image.
- The unsigned image remains available for development and for machines
  where the shim cannot boot.

## Source

Fedora `shim-x64-16.1-7`:

`https://download.fedoraproject.org/pub/fedora/linux/updates/44/Everything/x86_64/Packages/s/shim-x64-16.1-7.x86_64.rpm`

SHA-256: `4badee0c00c4c41a53f41c8a2dd566dca815ca051f1d2fba92490d0a0194f5c3`

The shim and MokManager are BSD-2-Clause upstream; the Fedora package lists
BSD-3-Clause. See [LICENSE.shim](LICENSE.shim).

## Key rotation

Every image and update must be signed with the same key that users enrolled.
Rotating the key requires a new enrollment on every PC. Do not rotate it
without a migration plan.
