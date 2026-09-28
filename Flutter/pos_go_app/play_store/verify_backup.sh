#!/usr/bin/env bash
# Verify a Play upload-key backup without needing the archive passphrase.
#
# The point of this script is that a backup nobody has restored is not a
# backup. Losing the POS.Go upload key means no further updates can ship to
# com.xamltech.pos_go, so the archive is worth checking before you need it —
# not after.
#
# Usage:
#   ./verify_backup.sh [path/to/pos-go-play-keys.zip.gpg]
#
# With no argument it checks the local keystore against the recorded checksums,
# which answers "is the live key still the one I think it is". Given the
# encrypted archive it additionally asks for the passphrase and proves the
# archive decrypts to a matching keystore.
#
# The passphrase is read from GPG_PASSPHRASE when set (handy for CI or a
# scripted check); otherwise gpg prompts on the terminal. It is never written to
# disk, never passed on the command line, and never echoed -- it goes in on a
# pipe via --passphrase-fd.

set -euo pipefail

# play_store/verify_backup.sh -> pos_go_app -> Flutter -> repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KEYSTORE="${REPO_ROOT}/.secrets/play/pos-go-release.jks"
CREDENTIALS="${REPO_ROOT}/.secrets/play/credentials.txt"

# The certificate fingerprint is public — it is printed by keytool and appears
# in every release APK — so it is safe to keep in a tracked file.
EXPECTED_CERT_SHA256="6B:3C:FE:55:CD:FB:1B:18:13:07:CE:4A:8E:02:40:03:78:FE:75:CE:95:9E:41:D8:CC:45:62:36:0A:1D:A5:EB"

# The keystore FILE checksum is derived from a secret and is deliberately not
# hardcoded here, even though it is only a sha256. It is read from the
# gitignored credentials file (JKS_SHA256:) so it never lands in git history.
# Note it is NOT a certificate fingerprint and keytool will never print it.
cred_get() { grep -m1 "^$1:" "$CREDENTIALS" 2>/dev/null | sed "s/^$1:[[:space:]]*//" | tr -d '[:space:]"'; }
EXPECTED_KEYSTORE_SHA256=""; [[ -f "$CREDENTIALS" ]] && EXPECTED_KEYSTORE_SHA256="$(cred_get JKS_SHA256)"

# Keep the passphrase off argv: --passphrase-fd 0 reads it from stdin, so it
# does not show up in `ps` for the life of the gpg process.
decrypt() {
  if [[ -n "${GPG_PASSPHRASE:-}" ]]; then
    printf '%s' "$GPG_PASSPHRASE" | gpg --quiet --batch --yes \
      --passphrase-fd 0 --decrypt "$1"
  else
    gpg --quiet --decrypt "$1"
  fi
}

fail=0
ok()   { printf '  \033[32mok\033[0m    %s\n' "$1"; }
bad()  { printf '  \033[31mFAIL\033[0m  %s\n' "$1"; fail=1; }
info() { printf '  ----  %s\n' "$1"; }

printf '\nPlay upload key — backup verification\n\n'

# 1. The live keystore still hashes to what was recorded at creation.
if [[ -f "$KEYSTORE" ]]; then
  actual="$(sha256sum "$KEYSTORE" | awk '{print $1}')"
  if [[ -z "$EXPECTED_KEYSTORE_SHA256" ]]; then
    info "keystore present but no JKS_SHA256 recorded in credentials.txt"
    info "add this line so drift is detectable: JKS_SHA256: $actual"
  elif [[ "$actual" == "$EXPECTED_KEYSTORE_SHA256" ]]; then
    ok "live keystore file checksum matches the recorded value"
  else
    bad "live keystore checksum drifted"
    info "expected $EXPECTED_KEYSTORE_SHA256"
    info "actual   $actual"
  fi
else
  bad "no keystore at $KEYSTORE"
fi

# 2. The recorded certificate fingerprint is the one the credentials file and
#    the release APK must agree on. This needs the store password, so it is
#    only attempted when the credentials file is readable.
if [[ -f "$CREDENTIALS" ]]; then
  recorded="$(cred_get SHA256)"
  if [[ -z "$recorded" ]]; then
    bad "credentials file has no SHA256 line"
  elif [[ "$(echo "$recorded" | tr '[:lower:]' '[:upper:]')" == "$EXPECTED_CERT_SHA256" ]]; then
    ok "credentials file records the expected certificate fingerprint"
  else
    bad "credentials file certificate fingerprint does not match"
    info "file     $recorded"
    info "expected $EXPECTED_CERT_SHA256"
  fi
else
  info "no credentials file; skipping the certificate check"
fi

# 3. Optional: prove the encrypted archive actually decrypts to this keystore.
archive="${1:-}"
if [[ -n "$archive" ]]; then
  if [[ ! -f "$archive" ]]; then
    bad "archive not found: $archive"
  else
    printf '\n  archive %s\n' "$archive"
    info "sha256 $(sha256sum "$archive" | awk '{print $1}')"
    tmp="$(mktemp -d)"
    # shellcheck disable=SC2064
    trap "rm -rf '$tmp'" EXIT
    if decrypt "$archive" > "$tmp/keys.zip" 2>/dev/null; then
      ok "archive decrypts"
      if unzip -o -q "$tmp/keys.zip" -d "$tmp" 2>/dev/null; then
        ok "archive unzips"
        found="$(find "$tmp" -name 'pos-go-release.jks' -print -quit)"
        if [[ -z "$found" ]]; then
          bad "archive contains no pos-go-release.jks"
        else
          got="$(sha256sum "$found" | awk '{print $1}')"
          if [[ "$got" == "$actual" ]]; then
            ok "keystore inside the archive matches the live key"
          else
            bad "archive holds a DIFFERENT keystore — do not trust this backup"
            info "archive $got"
            info "live    $actual"
          fi
        fi
      else
        bad "archive did not unzip"
      fi
    else
      bad "archive would not decrypt (wrong passphrase, or corrupt)"
    fi
  fi
fi

if [[ $fail -eq 0 ]]; then
  printf '\n\033[32mAll checks passed.\033[0m\n\n'
else
  printf '\n\033[31mSomething failed.\033[0m Resolve it before the first upload —\n'
  printf 'after the first upload a lost key becomes a Google key-reset request.\n\n'
fi
exit $fail
