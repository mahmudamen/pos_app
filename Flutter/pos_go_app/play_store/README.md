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

Still open:

3. **Confirm the Play listing state** for `com.xamltech.pos_go`: if the package
   already exists there, `versionCode` must exceed the uploaded one, and the
   upload key must be the one enrolled (or reset through Play).
4. **Resolve the [verify] items** in `listing/data_safety.md` and
   `listing/content_rating.md` — especially whether in-app scan-credit top-ups
   can ever spend real money (if yes, Play Billing becomes mandatory). The
   deletion page currently promises a thirty-day turnaround and that only
   legally required receipt/invoice records survive; confirm both.
5. **Re-test the release build on a device**: sign-in, one sale, one receipt
   print, one offline sale replayed after reconnecting. A reviewer-equivalent
   trial store exists from the sign-up smoke test:
   `playreview+check@xamltech.com` / `ReviewCheck2026!` (tenant
   `1ad661b0-38ad-461e-a2ca-d25e321b528e`, plan `trial`) — stop or delete it
   once the device test is done.

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
