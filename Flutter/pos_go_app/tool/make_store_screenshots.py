#!/usr/bin/env python3
"""Turn the raw emulator screenshots into Play-acceptable store screenshots.

The emulator captures are 1080x2400 (20:9). Play Console only accepts phone
screenshots whose aspect ratio sits between 16:9 and 9:16, so a 20:9 capture is
either rejected or silently cropped. Rather than crop the UI, each capture is
scaled to 864x1920 (the same 9:16 height at the original width) and centred on
a 1080x1920 canvas whose 108px side panels carry the brand, which keeps the
status bar and the bottom nav of every screen intact.

Output is 24-bit RGB PNG with no alpha channel, which is what Play asks for.

Usage:  python3 tool/make_store_screenshots.py
"""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "screenshots"
OUT = ROOT / "play_store/screenshots/phone"

CANVAS = (1080, 1920)      # 9:16, the narrowest ratio Play accepts
PANEL = (CANVAS[0] - 864) // 2

INK_TOP = (10, 22, 40)
INK_BOTTOM = (15, 34, 51)
CYAN = (0, 173, 238)
GREEN = (34, 197, 94)

# Play allows 2-8 phone screenshots. These eight walk the whole product:
# sign in, sell, check out, analyse, look back, loyalty, cash session, Copilot.
FEATURED = [
    "01-login",
    "02-pos-grid",
    "03-pos-cart",
    "04-dashboard",
    "05-sales-history",
    "06-customers",
    "07-session-reports",
    "08-community-hub",
]


def gradient(size: tuple[int, int], top: tuple[int, int, int], bottom: tuple[int, int, int],
             angle: float = 90.0) -> Image.Image:
    import math

    w, h = size
    diag = max(1, int(math.hypot(w, h)))
    strip = Image.new("RGB", (diag, 1))
    px = strip.load()
    horizontal = abs(math.cos(math.radians(angle))) > abs(math.sin(math.radians(angle)))
    for x in range(diag):
        t = x / (diag - 1)
        a, b = (top, bottom) if horizontal else (bottom, top)
        px[x, 0] = tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))
    strip = strip.resize((diag, diag), Image.NEAREST)
    rot = strip.rotate(-angle + 90, resample=Image.BICUBIC, expand=False)
    left, top_off = (diag - w) // 2, (diag - h) // 2
    return rot.crop((left, top_off, left + w, top_off + h))


def side_panel(width: int, height: int, inner_edge_left: bool) -> Image.Image:
    panel = gradient((width, height), INK_TOP, INK_BOTTOM, angle=72.0 if inner_edge_left else 108.0)

    glow = Image.new("RGBA", (width, height), (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    cx = width * (0.85 if inner_edge_left else 0.15)
    gd.ellipse([cx - width, height * 0.12 - width, cx + width, height * 0.12 + width],
               fill=CYAN + (54,))
    gd.ellipse([width * 0.1 - width, height * 0.88 - width, width * 0.1 + width, height * 0.88 + width],
               fill=GREEN + (36,))
    panel = Image.alpha_composite(panel.convert("RGBA"),
                                  glow.filter(ImageFilter.GaussianBlur(width * 0.5)))

    accent = Image.new("RGBA", (width, height), (0, 0, 0, 0))
    ad = ImageDraw.Draw(accent)
    x = width - 3 if inner_edge_left else 0
    ad.rectangle([x, 0, x + 3, height], fill=CYAN + (150,))
    panel = Image.alpha_composite(panel, accent)
    return panel.convert("RGB")


def build(source: Path, target: Path) -> None:
    shot = Image.open(source).convert("RGB")
    scale = CANVAS[1] / shot.height
    resized = shot.resize((max(1, round(shot.width * scale)), CANVAS[1]), Image.LANCZOS)

    canvas = Image.new("RGB", CANVAS, (0, 0, 0))
    canvas.paste(side_panel(PANEL, CANVAS[1], False), (0, 0))
    canvas.paste(side_panel(PANEL, CANVAS[1], True), (CANVAS[0] - PANEL, 0))
    canvas.paste(resized, (PANEL, 0))

    target.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(target, "PNG", optimize=True)
    print(f"  {target.relative_to(ROOT)}  {canvas.size[0]}x{canvas.size[1]}  "
          f"ar={canvas.size[0] / canvas.size[1]:.4f}  "
          f"{(target.stat().st_size) // 1024} KB")


def main() -> None:
    print("Play phone screenshots (9:16, 24-bit RGB, no alpha)")
    for name in FEATURED:
        source = SRC / f"{name}.png"
        if not source.exists():
            raise SystemExit(f"missing source screenshot: {source}")
        build(source, OUT / f"{name}.png")


if __name__ == "__main__":
    main()
