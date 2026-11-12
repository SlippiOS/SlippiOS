# ADR-0008: Custody of the Secure Boot and release keys

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-11-12 |
| **Deciders** | Ryan Wendell |

## Context

The Secure Boot private key is the root of trust for every SlippiOS console
that has completed MOK enrollment. Losing it means existing consoles cannot
receive signed updates until they enroll a new key. Leaking it lets someone
sign a bootloader or UKI that every enrolled console will accept. The key must
survive a laptop failure and still be usable by more than one trusted person.

Today the private key lives only at
`~/.config/slippios/secureboot/slippios.key` on one machine. The R2
credentials live at `~/.config/slippios/r2.env`, encrypted in the author's
personal chezmoi source state. That is acceptable for one developer and one
laptop, but it is not an onboarding or recovery plan:

- Adding another developer to the personal chezmoi repository would expose
  unrelated personal secrets.
- Copying one shared private key has no revocation or audit trail. Removing
  someone requires re-encrypting the copy and trusting that they deleted it.
- Losing the laptop loses the only signing key unless it was backed up
  separately.

CI does not build or sign images. Releases are built locally because the image
is a large Nix build and the private key must never enter the Nix store. Any
solution must keep the key out of CI, the repository, and the build sandbox.

## Decision

Keep the public certificate in the main repository at
`secure-boot/slippios.crt`. It is already embedded in every signed image and
published with its SHA-256 fingerprint, so it is not secret. Checking it in
makes verification and local signing consistent.

Store the private key and publish credentials in a separate private repository,
`SlippiOS/release-secrets`, encrypted with age to one recipient per release
manager. Each release manager generates a dedicated age identity and keeps the
private half in their password manager. Private age identities are never shared
or committed.

`scripts/with-release-secrets` decrypts the needed files into
`$XDG_RUNTIME_DIR` or `/dev/shm` for the duration of a release, signing, or
publish command, then removes them. The `mise` tasks use this wrapper
automatically, so the normal commands keep working:

```sh
mise run release
mise run release:dev
mise run release:minimal
mise run sign:image -- <unsigned.raw> <signed.raw>
mise run publish:r2 -- <assets-dir>
```

The wrapper still honors `SLIPPIOS_SB_KEY` and `SLIPPIOS_SB_CERT`, and still
uses `~/.config/slippios/secureboot` when present. `--vault` forces the team
vault, which is useful for testing a fresh onboarding.

Only release managers get access to the private repository. Other developers
build unsigned images. The dev update channel uses the same signing key as
stable, so publishing a dev build is a release-manager action rather than a
normal development step.

## Consequences

- Onboarding a release manager is: generate an age identity, send the public
  recipient, get added to `recipients.txt` and the private repository, clone
  it, and run `scripts/with-release-secrets --vault --check`.
- Offboarding is: remove the recipient and repository access, re-encrypt, and
  rotate anything the person should no longer hold. Removing a recipient does
  not revoke ciphertext they already downloaded. If trust is lost, rotate the
  Secure Boot key and plan a console re-enrollment.
- Every release manager must back up their age identity. A lost identity only
  locks that person out; another recipient can still decrypt. The Secure Boot
  key itself also needs an offline backup, because losing every copy is worse
  than losing one recipient.
- Rotation is a console migration. A new key requires a new MOK enrollment on
  every console, or a transitional signed update that imports the new
  certificate and asks the user to confirm it in MokManager.
- No signing key goes into GitHub Actions, the Nix store, or the public
  repository. If signing ever moves to CI, this ADR must be revisited with a
  protected environment and a separate signing design.
- Personal chezmoi remains personal. It holds encrypted copies of the key,
  certificate, and release age identity in `~/.secrets` for the author's own
  machines, and the wrapper uses them when no local key or vault clone exists.
  It is not the team distribution mechanism.

## Alternatives considered

- **One shared age private key.** Simple to explain, but it gives every holder
  the same identity, has no revocation, and makes offboarding a matter of
  trusting that a copy was deleted.
- **GitHub SSH keys as age recipients.** Age supports encrypting to
  `https://github.com/<user>.keys`, but hardware-backed SSH keys cannot be used
  by age, authentication keys are rotated independently, and one person often
  has several keys. It couples release access to GitHub account security.
- **GitHub Actions secrets and CI signing.** This moves the root key into
  GitHub and requires a CI build of a large image. It is a valid future design
  with a protected environment and required reviewers, but it is not the
  current release architecture.
- **A shared password manager entry.** Workable for a very small team, but it
  is manual, has no per-person audit trail, and does not fit the existing CLI
  release flow. Age recipients in a private repository are scriptable and
  revocable at the recipient list.
- **Committing the encrypted key to the public repository.** Technically safe
  while age is unbroken, but it publishes ciphertext and metadata forever and
  removes the repository-access boundary. The private repository is a better
  operational fit.
