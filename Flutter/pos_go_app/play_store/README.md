# POS.Go — Google Play release 1.0.0 (versionCode 3)

Everything needed to publish `com.xamltech.pos_go` to Google Play, plus the
exact commands that produced the artifacts in this directory.

## Artifacts

| What | Path | Size | Notes |
|---|---|---|---|
| Play App Bundle | `../build/app/outputs/bundle/release/app-release.aab` | 62.0 MB | **Upload this to Play** |
| Universal APK | `../build/app/outputs/flutter-apk/app-release.apk` | 64.8 MB | Sideload / manual distribution only |
| App icon | `graphics/icon-512.png` | 512×512 RGBA | Console → App content → App icon |
| Feature graphic | `graphics/feature-graphic-1024x500.png` | 1024×500 | Console → Main store listing |
| Phone screenshots | `screenshots/phone/01..08-*.png` | 1080×1920 RGB | Upload in numbered order |

Both binaries are signed with the release key (not the debug key) and point at
production:

- `API_BASE_URL=https://posgo.xamltech.com`
- `SAAS_API_BASE_URL=https://api.xamltech.com`

Verified from the built artifacts: package `com.xamltech.pos_go`, versionName
`1.0.0`, versionCode `3`, `minSdk` 24, `targetSdk` 36, `compileSdk` 36,
release-only cleartext disabled, permissions `INTERNET` + `WAKE_LOCK` plus
Flutter's app-private receiver permission.

## Signing

The key lives outside the repository and is gitignored:

```
/mnt/nvme0n1p5/android-apps/pos/.secrets/play/pos-go-release.jks   (PKCS12)
/mnt/nvme0n1p5/android-apps/pos/.secrets/play/credentials.txt      (passwords, mode 600)
Flutter/pos_go_app/android/key.properties                         (gitignored)
```

- alias `pos-go-upload`, RSA 4096, valid 30 years
- certificate SHA-256 `6B:3C:FE:55:CD:FB:1B:18:13:07:CE:4A:8E:02:40:03:78:FE:75:CE:95:9E:41:D8:CC:45:62:36:0A:1D:A5:EB`

**Back up `credentials.txt` and the `.jks` somewhere durable and private now.**
If this key is lost and the app is already on Play with Play App Signing, the
only recovery is Google's upload-key reset request. If the app is enrolled
without Play App Signing, a lost key means a new package name.

When `android/key.properties` is absent the release build falls back to the
debug key and prints a warning — that fallback artifact must never be uploaded.

## Rebuilding

```bash
cd Flutter/pos_go_app
flutter pub get
flutter analyze && flutter test
python3 tool/make_store_assets.py        # icons + feature graphic
python3 tool/make_store_screenshots.py   # 9:16 store screenshots
flutter build appbundle --release \
  --dart-define=API_BASE_URL=https://posgo.xamltech.com \
  --dart-define=SAAS_API_BASE_URL=https://api.xamltech.com
flutter build apk --release \
  --dart-define=API_BASE_URL=https://posgo.xamltech.com \
  --dart-define=SAAS_API_BASE_URL=https://api.xamltech.com
python3 tool/check_store_listing.py      # enforces every Play limit below
```

Bump the version before each upload: `version: X.Y.Z+N` in `pubspec.yaml` and
`Telemetry.buildVersion` in `lib/core/telemetry.dart` must match, and
`versionCode` must be **higher** than every version already on Play.

## What changed for the store, versus the app you were testing

- Version `0.2.0+2` → `1.0.0+3`.
- `compileSdk`/`targetSdk` pinned to 36, `minSdk` pinned to 24 (previously
  inherited from the Flutter toolchain, so this is now explicit).
- Real release signing replaces the debug key.
- Release builds forbid cleartext HTTP; debug and profile keep it so a local
  backend still works.
- `allowBackup=false`: the offline cache is a cache, so it is not uploaded to
  Google backup.
- `supportsRtl=true` and a branded `roundIcon`; adaptive + themed (monochrome)
  launcher icons replace the default Flutter icon.
- Dropped `ACCESS_NOTIFICATION_POLICY`, template boilerplate nothing used.

## Console order of operations

1. **App content**
   - App icon: `graphics/icon-512.png`
   - Feature graphic: `graphics/feature-graphic-1024x500.png`
   - Phone screenshots: `screenshots/phone/` in numeric order
   - Privacy policy URL: https://posgo.xamltech.com/private
2. **Store listing** — copy from `listing/store_listing_en.md` (en-US) and
   `listing/store_listing_ar.md` (ar-EG). The `field:` blocks are the exact
   text to paste; limits are already validated.
3. **App content → App access** — "All functionality is unrestricted"; the app
   creates a 15-day trial store in-app, so no reviewer credentials are needed.
4. **Ads / Content rating** — no ads; answers in `listing/content_rating.md`.
5. **Data safety** — answers in `listing/data_safety.md`. Every line marked
   **[verify]** must be confirmed by a human.
6. **Pricing** — free to download. The plan tiers (trial/standard/premium/
   enterprise) are sold on the web, not inside the app.
7. **Release → Upload** — `app-release.aab`. Internal testing first, then
   production, then hand-picked rollout.

## Blockers before this can actually ship

Resolved on 2026-09-27:

1. ~~**The three-domain backend is not deployed.**~~ `posgo.xamltech.com`
   resolves (Cloudflare A record, grey cloud) and terminates TLS in host nginx
   with a Let's Encrypt certificate (`posgo.xamltech.com`, renews via
   `certbot.timer`); the committed vhost is
   `Backend/go-pos-backend-implementation-ready/deployments/nginx/posgo.xamltech.conf`.
   The Android app signs in and walks every screen against it. `xamltech.com`
   itself is a separate Cloudflare Pages site and is **not** used by the app or
   the listing. Surface routing is still off
   (`SURFACE_ROUTING_ENABLED=false`), so the POS host also answers
   `/v1/saas/*` and `/admin` — harmless for the Play reviewer, but it is not the
   enforced split described in `docs/25_CLOUDFLARE_POSGO.md`.
2. ~~**The privacy policy must be live and public.**~~ Served at
   `https://posgo.xamltech.com/private` (200, no auth, en + ar) and the deletion
   request page at `https://posgo.xamltech.com/delete-account`, which is what
   Play's Data safety form needs for an app that can create an account. The
   privacy policy links to it. Both are generated by the Go app
   (`internal/transport/server/pages.go`), so a Go deploy keeps them in sync.

Still open — none of these are code problems:

3. **Back the upload key up off-box.** The live key and the credentials live in
   `.secrets/play/` (gitignored), with a verified-identical mirror in
   `.secrets/play/offsite/` — but both are on this one machine, so that mirror
   only protects against accidental deletion, not disk or theft loss. **No
   off-box copy exists yet** (checked Desktop, Documents, Downloads, backups,
   keys and every `/mnt/*` mount). One encrypted file to produce and move:

   ```bash
   cd /mnt/nvme0n1p5/android-apps/pos
   zip -j -r /tmp/pos-go-play-keys.zip \
     .secrets/play/pos-go-release.jks .secrets/play/credentials.txt
   gpg --symmetric --cipher-algo AES256 \
       -o /tmp/pos-go-play-keys.zip.gpg /tmp/pos-go-play-keys.zip   # prompts for a passphrase
   sha256sum /tmp/pos-go-play-keys.zip.gpg
   rm -f /tmp/pos-go-play-keys.zip
   ```

   (`gpg` and `openssl` are both present; `age` is not installed on this box, so
   the archive is `.gpg` and stays `.gpg`. To restore:
   `gpg --decrypt pos-go-play-keys.zip.gpg > keys.zip && unzip keys.zip`.)

   Then upload the `pos-go-play-keys.zip.gpg` — the encrypted file only, never
   the plaintext zip — to encrypted cloud storage or a password-manager
   attachment, and **verify the copy you put there** by decrypting it and
   re-running the checks in `verify_backup.sh`. A backup that has never been
   restored is not a backup.

   Three values to keep, and they are easy to confuse:

   - `6B:3C:FE:…:EB` — the **signing certificate** fingerprint. Public, and it
     is in every release APK. Verify with
     `keytool -printcert -jarfile app-release.aab | grep SHA256`. This is the
     only one of the three written down here.
   - The **SHA-256 of the keystore file itself** — a `sha256sum` of the `.jks`,
     *not* a certificate fingerprint. `keytool` will never print it, so do not
     go looking for it in `keytool` output and conclude the key is corrupt. It
     is recorded as `JKS_SHA256:` in `.secrets/play/credentials.txt`, which is
     gitignored, so the value derived from a secret never lands in git history.
   - The `sha256sum` of the `.gpg` you uploaded — a third value, and the only
     one that proves *that particular copy* arrived intact.

   `verify_backup.sh` compares all three for you, so you do not have to
   transcribe any of them by hand.

   Once Play App Signing is enrolled, a lost upload key means a Google
   key-reset request — hence do this before the first upload, not after.
4. **Confirm the Play listing state** for `com.xamltech.pos_go`: if the package
   already exists there, `versionCode` must exceed the uploaded one, and the
   upload key must be the one enrolled (or reset through Play). The artifacts
   are currently `1.0.0` code `3`, so anything already on Play means a bump.
5. **Resolve the [verify] items** in `listing/data_safety.md` and
   `listing/content_rating.md` — especially whether in-app scan-credit top-ups
   can ever spend real money (if yes, Play Billing becomes mandatory). The
   deletion page currently promises a thirty-day turnaround and that only
   legally required receipt/invoice records survive; confirm both.
6. **Re-test the release build on a device**: sign-in, one sale, one receipt
   print, one offline sale replayed after reconnecting. Reviewer-equivalent
   trial stores exist from the production smoke tests:
   `playreview+check@xamltech.com` / `ReviewCheck2026!` and
   `playreview+flow@xamltech.com` / `ReviewFlow2026!` — stop or delete both once
   the device test is done.

## Verified on 2026-09-27 (before the first upload)

The reviewer flow was driven end to end against production over
`https://posgo.xamltech.com`, with `curl` standing in for the app:

| Step | Endpoint | Result |
|---|---|---|
| Create a trial store | `POST /v1/auth/register` | 201 — 10 seeded products, owner user, `access_level: admin`, trial active for 13 days |
| Sign in | `POST /v1/auth/login` | 200 — the tenant id from sign-up is the login `tenant_id` |
| Open the till | `POST /v1/registers/open` | 201 — 500.00 EGP opening float |
| Ring up a sale | `POST /v1/sales` | 201 — 16.00 EGP cash, stock decremented |
| Receipt | `GET /v1/sales/:id/receipt` | 200 JSON |
| Thermal print | `GET /v1/sales/:id/receipt/print` | 200 `application/vnd.escpos`, 617 bytes of real ESC/POS |
| Close the till | `POST /v1/registers/:id/close` | 200 — expected 516.00, counted 500.00, difference −16.00 (the uncounted cash sale) |

So a Play reviewer who installs the app and signs up gets a working store
immediately, with no credentials to hand out — which is what the App access
answer claims.

The release artifacts were re-verified after that change: both signed with the
uploaded key `6B:3C:FE:…:EB`, `zipalign -P 16` clean on the APK **and** the AAB,
every `.so` `LOAD` segment aligned to 16 KB, permissions limited to `INTERNET`
+ `WAKE_LOCK` (plus Flutter's own receiver permission), and the three baked URLs
(`posgo.xamltech.com`, `posgo.xamltech.com/private`, `api.xamltech.com` for the
SaaS panel) identical in all three ABIs with no stale apex privacy URL.

## Limits enforced by `tool/check_store_listing.py`

| Asset | Requirement |
|---|---|
| App name | ≤ 30 characters |
| Short description | ≤ 80 characters |
| Full description | ≤ 4000 characters |
| Release notes | ≤ 500 characters |
| App icon | 512×512 PNG, 32-bit with alpha |
| Feature graphic | 1024×500 PNG or JPEG |
| Phone screenshots | 2–8 files, 24-bit RGB (no alpha), each side 320–3840 px, aspect ratio between 16:9 and 9:16 |
| New apps / updates | must target a recent API level — API 36 for submissions from 31 Aug 2026 |
