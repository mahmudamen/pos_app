#!/usr/bin/env python3
"""Fail the release if the store listing or graphics would be rejected.

Checks, against the rules Play Console actually enforces:

  * listing text   -- app name <= 30, short description <= 80,
                       full description and release notes <= 4000 / 500
  * app icon       -- 512x512, PNG, alpha present
  * feature graphic-- 1024x500, PNG or JPEG
  * phone shots    -- 2 to 8, 24-bit RGB (no alpha), each side 320..3840,
                       aspect ratio within 16:9 .. 9:16
  * version        -- the built artifact matches the listing and pubspec

Usage:  python3 tool/check_store_listing.py [--apk path] [--aab path]
Exit code 0 means the pack is uploadable as-is.
"""

from __future__ import annotations

import re
import sys
import zipfile
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
PLAY = ROOT / "play_store"
LISTING = PLAY / "listing"
SHOTS = PLAY / "screenshots/phone"
GRAPHICS = PLAY / "graphics"

LIMITS = {"title": 30, "short": 80, "full": 4000, "release_notes": 500,
          "review_notes": 4000}
LOCALES = {"store_listing_en.md": "en-US", "store_listing_ar.md": "ar-EG"}

failures: list[str] = []
notes: list[str] = []


def fail(msg: str) -> None:
    failures.append(msg)
    print(f"  FAIL  {msg}")


def ok(msg: str) -> None:
    print(f"  ok    {msg}")


def field(path: Path, name: str) -> str | None:
    match = re.search(
        rf"<!-- field: {name} -->\n(.*?)\n<!-- /field -->", path.read_text(), re.S
    )
    return match.group(1).strip() if match else None


def check_text() -> None:
    print("listing text")
    for filename, locale in LOCALES.items():
        path = LISTING / filename
        if not path.exists():
            fail(f"missing {path.relative_to(ROOT)}")
            continue
        for name, limit in LIMITS.items():
            value = field(path, name)
            if value is None:
                fail(f"{locale}: field {name!r} not found (markers intact?)")
                continue
            length = len(value)
            if length > limit:
                fail(f"{locale} {name}: {length} chars, limit {limit}")
            else:
                ok(f"{locale} {name}: {length}/{limit} chars")
        for value in (field(path, "title"), field(path, "short")):
            if value and value != value.strip():
                fail(f"{locale}: leading/trailing whitespace in a short field")
            if value and "\n" in value:
                fail(f"{locale}: single-line field contains a newline")


def check_images() -> None:
    print("graphics")
    icon = GRAPHICS / "icon-512.png"
    if not icon.exists():
        fail("missing graphics/icon-512.png")
    else:
        image = Image.open(icon)
        if image.size != (512, 512):
            fail(f"app icon must be 512x512, found {image.size}")
        elif image.format != "PNG":
            fail(f"app icon must be PNG, found {image.format}")
        elif image.mode not in ("RGBA", "LA"):
            fail(f"app icon must be 32-bit with an alpha channel, found {image.mode}")
        else:
            ok("app icon 512x512 PNG, 32-bit with alpha")

    feature = GRAPHICS / "feature-graphic-1024x500.png"
    if not feature.exists():
        fail("missing graphics/feature-graphic-1024x500.png")
    else:
        image = Image.open(feature)
        if image.size != (1024, 500):
            fail(f"feature graphic must be 1024x500, found {image.size}")
        elif image.format not in ("PNG", "JPEG"):
            fail(f"feature graphic must be PNG or JPEG, found {image.format}")
        else:
            ok("feature graphic 1024x500")

    print("phone screenshots")
    shots = sorted(SHOTS.glob("*.png")) + sorted(SHOTS.glob("*.jpg"))
    if not 2 <= len(shots) <= 8:
        fail(f"phone screenshots must number 2..8, found {len(shots)}")
    for shot in shots:
        image = Image.open(shot)
        width, height = image.size
        ratio = width / height
        problem = None
        if image.mode != "RGB":
            problem = f"must be 24-bit RGB without alpha, found {image.mode}"
        elif not (9 / 16 - 1e-6 <= ratio <= 16 / 9 + 1e-6):
            problem = f"aspect ratio {ratio:.3f} outside 16:9..9:16"
        elif min(width, height) < 320:
            problem = f"shortest side {min(width, height)} < 320"
        elif max(width, height) > 3840:
            problem = f"longest side {max(width, height)} > 3840"
        if problem:
            fail(f"{shot.name}: {problem}")
        else:
            ok(f"{shot.name}: {width}x{height} ratio {ratio:.4f}")
    if shots:
        print(f"  ({len(shots)} phone screenshots)")


AAPT2 = sorted(
    (Path.home() / "Android/Sdk/build-tools").glob("*/aapt2")
)
AAPT2 = AAPT2[-1] if AAPT2 else None


def apk_badging(apk: Path) -> dict[str, str]:
    """versionCode / versionName / targetSdk straight from aapt2."""
    if AAPT2 is None:
        return {}
    import subprocess

    out = subprocess.run(
        [str(AAPT2), "dump", "badging", str(apk)],
        capture_output=True, text=True,
    ).stdout
    found: dict[str, str] = {}
    package = re.search(r"package: name='([^']+)' versionCode='(\d+)' versionName='([^']+)'", out)
    if package:
        found.update(package="", versionCode=package.group(2), versionName=package.group(3))
        found["package"] = package.group(1)
    target = re.search(r"targetSdkVersion:'(\d+)'", out)
    if target:
        found["targetSdk"] = target.group(1)
    return found


def aab_smoke(aab: Path, version_name: str) -> str | None:
    """A Play bundle is a signed jar; check its shape and that it is signed."""
    with zipfile.ZipFile(aab) as zf:
        names = zf.namelist()
        for required in ("BundleConfig.pb", "base/manifest/AndroidManifest.xml",
                         "base/dex/classes.dex"):
            if required not in names:
                return f"{aab.name}: missing {required}"
        manifest = zf.read("base/manifest/AndroidManifest.xml")
    if version_name.encode() not in manifest:
        return f"{aab.name}: version name {version_name} not in base manifest"
    import subprocess

    verified = subprocess.run(["jarsigner", "-verify", str(aab)],
                              capture_output=True, text=True)
    if "jar verified" not in verified.stdout + verified.stderr:
        return f"{aab.name}: JAR signature did not verify"
    return None


def check_version(apk: Path | None, aab: Path | None) -> None:
    print("version")
    pubspec = (ROOT / "pubspec.yaml").read_text()
    match = re.search(r"^version:\s*(\S+)\+(\d+)", pubspec, re.M)
    if not match:
        fail("could not read version from pubspec.yaml")
        return
    name, code = match.group(1), match.group(2)
    listing = (LISTING / "store_listing_en.md").read_text()
    if name not in listing:
        fail(f"pubspec version {name} is not mentioned in the listing")
    if f"code {code}" not in listing:
        fail(f"pubspec versionCode {code} is not mentioned in the listing")
    else:
        ok(f"listing documents {name} (code {code})")

    if apk and apk.exists():
        found = apk_badging(apk)
        if not found:
            notes.append("aapt2 not found; skipped APK version check")
        else:
            for key, expected in (("versionCode", code), ("versionName", name),
                                  ("package", "com.xamltech.pos_go")):
                if found.get(key) != expected:
                    fail(f"{apk.name}: {key}={found.get(key)!r}, expected {expected!r}")
            if found.get("targetSdk") != "36":
                fail(f"{apk.name}: targetSdk={found.get('targetSdk')!r}, expected 36")
            if all(found.get(k) == v for k, v in (("versionCode", code),
                                                    ("versionName", name),
                                                    ("package", "com.xamltech.pos_go"),
                                                    ("targetSdk", "36"))):
                ok(f"{apk.name}: {found.get('versionName')} "
                   f"(code {found.get('versionCode')}), targetSdk "
                   f"{found.get('targetSdk')}, signed build")
    else:
        notes.append(f"{apk} not built; skipped APK check")

    if aab and aab.exists():
        problem = aab_smoke(aab, name)
        if problem:
            fail(problem)
        else:
            ok(f"{aab.name}: valid signed Play bundle for {name}")
    else:
        notes.append(f"{aab} not built; skipped AAB check")


def main() -> int:
    check_text()
    check_images()

    apk = Path(sys.argv[sys.argv.index("--apk") + 1]) if "--apk" in sys.argv else \
        ROOT / "build/app/outputs/flutter-apk/app-release.apk"
    aab = Path(sys.argv[sys.argv.index("--aab") + 1]) if "--aab" in sys.argv else \
        ROOT / "build/app/outputs/bundle/release/app-release.aab"
    check_version(apk, aab)

    print()
    for note in notes:
        print(f"  note  {note}")
    if failures:
        print(f"{len(failures)} problem(s) found.")
        return 1
    print("Store pack is consistent and within Play's limits.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
