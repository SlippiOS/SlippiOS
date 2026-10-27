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

See [ENROLL.txt](ENROLL.txt). The certificate is `slippios.cer` on the
ESP. The enrollment is stored in the PC's firmware NVRAM. It survives
reboots and SlippiOS updates.

## Updates

The updater writes a new UKI signed with the same key. It boots without
another enrollment. The release manifest must be built from the signed
UKI, not the unsigned one.

The shim and MokManager are part of the image. They are not updated by
systemd-sysupdate. A shim update needs a new image.

## Signing

Generate a key and certificate once:

```sh
umask 077
openssl req -new -x509 -newkey rsa:4096 \
  -keyout slippios.key -out slippios.crt \
  -days 3650 -nodes -subj "/CN=SlippiOS Secure Boot"
```

Keep `slippios.key` out of the repository. For local signing, keep it offline.
For CI, store the PEM key and certificate as `SLIPPIOS_SB_KEY` and
`SLIPPIOS_SB_CERT` repository secrets. The release workflow signs the default
image and UKI on tags. The public certificate is embedded in every image as
`slippios.cer`.

Sign an unsigned image:

```sh
SLIPPIOS_SB_KEY=/path/to/slippios.key \
SLIPPIOS_SB_CERT=/path/to/slippios.crt \
nix shell --inputs-from . \
  nixpkgs#sbsigntool nixpkgs#mtools nixpkgs#rpm nixpkgs#cpio \
  nixpkgs#parted nixpkgs#openssl \
  -c ./scripts/sign-image result-local-image/slippios_<version>.raw signed.raw
```

The script downloads the pinned Fedora shim RPM, verifies its SHA-256 and
both Microsoft signatures, signs systemd-boot and the UKI, installs the
shim and MokManager, and writes the enrollment note and shim license.
Signing is a post-build step so the private key never enters the Nix store.

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
