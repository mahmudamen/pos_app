#!/usr/bin/env python3
"""Generate every Google Play / Android launcher asset for POS.Go.

One source of truth (this file) renders the brand mark once at 4x and resamples
it into each size the platform asks for:

  * legacy launcher icons      mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher.png
  * adaptive icon layers       ic_launcher_foreground.png + ic_launcher_background
  * themed (monochrome) icon   ic_launcher_monochrome.png  (Android 13+)
  * Play Console icon          play_store/graphics/icon-512.png
  * Play feature graphic       play_store/graphics/feature-graphic-1024x500.png
  * brand mark source art      play_store/graphics/pos-go-mark-512.png

The mark is the same one the marketing site ships: a dark navy tile with a
green-to-cyan "P". Everything is drawn at 4x and downsampled with LANCZOS so
the 48px launcher icon stays crisp.

Usage:  python3 tool/make_store_assets.py
"""

from __future__ import annotations

import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = Path(__file__).resolve().parent.parent
RES = ROOT / "android/app/src/main/res"
GRAPHICS = ROOT / "play_store/graphics"

SS = 4  # supersampling factor

INK_TOP = (10, 22, 40)      # #0a1628
INK_BOTTOM = (15, 34, 51)   # #0f2233
GREEN = (34, 197, 94)       # #22c55e
CYAN = (0, 173, 238)        # #00adee
LIGHT = (226, 245, 255)

DENSITIES = {
    "mdpi": 48,
    "hdpi": 72,
    "xhdpi": 96,
    "xxhdpi": 144,
    "xxxhdpi": 192,
}

# Adaptive icons are 108dp; the inner 72dp is the mask, and the guaranteed
# safe zone is the middle 66dp, so the mark is sized against that.
ADAPTIVE_DP = {
    "mdpi": 108,
    "hdpi": 162,
    "xhdpi": 216,
    "xxhdpi": 324,
    "xxxhdpi": 432,
}


def lerp(a: tuple[int, int, int], b: tuple[int, int, int], t: float) -> tuple[int, int, int]:
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def gradient(size: tuple[int, int], top: tuple[int, int, int], bottom: tuple[int, int, int],
             angle: float = 90.0) -> Image.Image:
    """Linear gradient at an arbitrary angle, drawn small and scaled for speed."""
    import math

    w, h = size
    diag = max(1, int(math.hypot(w, h)))
    strip = Image.new("RGB", (diag, 1))
    px = strip.load()
    rad = math.radians(angle)
    for x in range(diag):
        t = (x / diag)
        colour = lerp(top, bottom, t) if abs(math.cos(rad)) > abs(math.sin(rad)) else lerp(bottom, top, t)
        px[x, 0] = colour
    strip = strip.resize((diag, diag), Image.NEAREST)
    rot = strip.rotate(-angle + 90, resample=Image.BICUBIC, expand=False)
    left = (diag - w) // 2
    top_off = (diag - h) // 2
    return rot.crop((left, top_off, left + w, top_off + h))


def glow(size: tuple[int, int], centre: tuple[float, float], radius: float,
         colour: tuple[int, int, int], alpha: int) -> Image.Image:
    layer = Image.new("RGBA", size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(layer)
    cx, cy = centre
    draw.ellipse([cx - radius, cy - radius, cx + radius, cy + radius], fill=colour + (alpha,))
    return layer.filter(ImageFilter.GaussianBlur(radius * 0.55))


def mark_mask(s: int, scale: float = 1.0) -> Image.Image:
    """The "P" glyph as an alpha mask on an s x s canvas, centred.

    The glyph is authored in percent of the canvas: a rounded stem with a bowl
    that closes back onto it. `scale` grows it about the centre, which is how
    the adaptive layer fits the 66/108 launcher safe zone.
    """
    u = s / 100.0
    mask = Image.new("L", (s, s), 0)
    draw = ImageDraw.Draw(mask)

    def box(x0, y0, x1, y1):
        # Authoring box is (30.5, 24) - (65, 79); centre it, then scale.
        cx, cy = 47.75, 51.5
        return [
            (50 + (x0 - cx) * scale) * u,
            (50 + (y0 - cy) * scale) * u,
            (50 + (x1 - cx) * scale) * u,
            (50 + (y1 - cy) * scale) * u,
        ]

    draw.rounded_rectangle(box(30.5, 24.0, 43.0, 79.0), radius=6.2 * scale * u, fill=255)
    draw.ellipse(box(30.5, 24.0, 65.0, 57.0), fill=255)
    draw.ellipse(box(43.0, 33.0, 56.0, 48.0), fill=0)
    return mask


def draw_mark(size: int, scale: float = 1.0, monochrome: bool = False,
              glow_highlight: bool = True) -> Image.Image:
    """The finished glyph as an RGBA image of `size` px."""
    s = size * SS
    mask = mark_mask(s, scale)
    if monochrome:
        layer = Image.new("RGBA", (s, s), (0, 0, 0, 0))
        layer.paste((255, 255, 255, 255), (0, 0), mask)
        return layer.resize((size, size), Image.LANCZOS)
    body = gradient((s, s), GREEN, CYAN, angle=75.0).convert("RGBA")
    layer = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    layer.paste(body, (0, 0), mask)
    if glow_highlight:
        # Soft highlight from the top left keeps the mark from looking flat on
        # the tile. The adaptive layer skips it: a launcher crops that layer to
        # whatever mask it likes, and a wide halo would tint the whole shape.
        layer = Image.alpha_composite(
            layer, glow((s, s), (0.34 * s, 0.30 * s), 0.30 * s, LIGHT, 60)
        )
    return layer.resize((size, size), Image.LANCZOS)


def tile_at(s: int, radius_ratio: float = 0.225) -> Image.Image:
    """Draw the tile at an exact canvas size of s px (caller resamples)."""
    tile = gradient((s, s), INK_TOP, INK_BOTTOM, angle=68.0).convert("RGBA")
    tile = Image.alpha_composite(
        tile,
        glow((s, s), (0.22 * s, 0.16 * s), 0.46 * s, CYAN, 70),
    )
    tile = Image.alpha_composite(
        tile,
        glow((s, s), (0.86 * s, 0.92 * s), 0.42 * s, GREEN, 46),
    )

    mask = Image.new("L", (s, s), 0)
    ImageDraw.Draw(mask).rounded_rectangle(
        [0, 0, s - 1, s - 1], radius=int(radius_ratio * s), fill=255
    )
    tile.putalpha(mask)

    # Thin brand stroke, echoing the site's favicon.
    stroke = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    sd = ImageDraw.Draw(stroke)
    pad = int(0.035 * s)
    sd.rounded_rectangle(
        [pad, pad, s - 1 - pad, s - 1 - pad],
        radius=int(radius_ratio * s) - pad,
        outline=GREEN + (110,),
        width=max(1, int(0.018 * s)),
    )
    tile = Image.alpha_composite(tile, stroke)

    tile.alpha_composite(draw_mark(s, scale=1.18))
    return tile


def rounded_tile(size: int) -> Image.Image:
    return tile_at(size * SS).resize((size, size), Image.LANCZOS)


def round_tile(size: int) -> Image.Image:
    """Circular variant for launchers that ask for android:roundIcon."""
    s = size * SS
    tile = tile_at(s)
    mask = Image.new("L", (s, s), 0)
    ImageDraw.Draw(mask).ellipse([0, 0, s - 1, s - 1], fill=255)
    tile.putalpha(mask)
    return tile.resize((size, size), Image.LANCZOS)


def adaptive_foreground(size: int) -> Image.Image:
    """Glyph only, sized to fit inside the 66/108dp guaranteed safe circle."""
    s = size * SS
    layer = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    layer.alpha_composite(draw_mark(s, scale=0.92, glow_highlight=False))
    return layer.resize((size, size), Image.LANCZOS)


def adaptive_background(size: int) -> Image.Image:
    s = size * SS
    bg = gradient((s, s), INK_TOP, INK_BOTTOM, angle=68.0).convert("RGBA")
    bg = Image.alpha_composite(bg, glow((s, s), (0.30 * s, 0.20 * s), 0.5 * s, CYAN, 64))
    bg = Image.alpha_composite(bg, glow((s, s), (0.80 * s, 0.88 * s), 0.45 * s, GREEN, 40))
    return bg.resize((size, size), Image.LANCZOS)


def wordmark(draw: ImageDraw.ImageDraw, xy: tuple[int, int], scale: float,
             colour: tuple[int, int, int], weight: str = "bold") -> None:
    x, y = xy
    regular = ROOT / "assets/fonts/lato/Lato-Regular.ttf"
    heavy = list((ROOT / "assets/fonts/lato").glob("Lato-*"))
    heavy_path = next((p for p in heavy if "Heavy" in p.name or "Black" in p.name or "Bold" in p.name), None)
    font = ImageFont.truetype(str(heavy_path or regular), int(96 * scale))
    draw.text((x, y), "POS", font=font, fill=colour)
    pos_width = draw.textlength("POS", font=font)
    thin = ImageFont.truetype(str(regular), int(96 * scale))
    draw.text((x + pos_width + int(6 * scale), y), ".Go", font=thin, fill=CYAN)


def feature_graphic() -> Image.Image:
    w, h = 1024, 500
    img = gradient((w, h), (9, 20, 35), (13, 32, 48), angle=8.0).convert("RGBA")
    img = Image.alpha_composite(img, glow((w, h), (0.18 * w, 0.30 * h), 0.55 * w, CYAN, 58))
    img = Image.alpha_composite(img, glow((w, h), (0.92 * w, 0.80 * h), 0.45 * w, GREEN, 40))

    # Faint dot grid, so the banner does not read as a flat rectangle.
    dots = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    dd = ImageDraw.Draw(dots)
    for y in range(0, h, 26):
        for x in range(0, w, 26):
            dd.ellipse([x - 1, y - 1, x + 1, y + 1], fill=(255, 255, 255, 16))
    img = Image.alpha_composite(img, dots)

    img.alpha_composite(draw_mark(132), (72, 96))

    draw = ImageDraw.Draw(img)
    wordmark(draw, (232, 92), 1.05, (255, 255, 255))
    tagline = ImageFont.truetype(str(ROOT / "assets/fonts/lato/Lato-Regular.ttf"), 34)
    draw.text((236, 218), "نقطة بيع تعمل بدون إنترنت", font=tagline, fill=(190, 226, 245))
    draw.text((236, 266), "Offline-first POS for Egyptian businesses", font=tagline,
              fill=(190, 226, 245))
    small = ImageFont.truetype(str(ROOT / "assets/fonts/lato/Lato-Regular.ttf"), 24)
    draw.text((236, 336), "Cashiers  ·  Stock & lots  ·  Split payments  ·  Refunds  ·  Community",
              font=small, fill=(120, 190, 225))
    return img.convert("RGB")


def write_png(image: Image.Image, path: Path, rgb: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    out = image.convert("RGB") if rgb else image
    out.save(path, "PNG", optimize=True)
    print(f"  {path.relative_to(ROOT)}  {out.size[0]}x{out.size[1]}  "
          f"{path.stat().st_size // 1024} KB")


def main() -> None:
    print("Legacy launcher icons")
    tile = rounded_tile(512)
    round_tile(512)
    for density, size in DENSITIES.items():
        write_png(tile.resize((size, size), Image.LANCZOS),
                  RES / f"mipmap-{density}" / "ic_launcher.png")
        write_png(round_tile(size),
                  RES / f"mipmap-{density}" / "ic_launcher_round.png")

    print("Adaptive icon layers")
    for density, size in ADAPTIVE_DP.items():
        write_png(adaptive_foreground(size),
                  RES / f"mipmap-{density}" / "ic_launcher_foreground.png")
        write_png(adaptive_background(size),
                  RES / f"mipmap-{density}" / "ic_launcher_background.png")
        write_png(draw_mark(size, scale=0.92, monochrome=True),
                  RES / f"mipmap-{density}" / "ic_launcher_monochrome.png")

    print("Play Console icon + feature graphic")
    # Play masks the icon itself: full bleed square with an alpha channel that
    # is fully opaque (Play asks for a 32-bit PNG, the mask does the rounding).
    flat = tile.convert("RGBA")
    write_png(flat, GRAPHICS / "icon-512.png")
    write_png(feature_graphic(), GRAPHICS / "feature-graphic-1024x500.png", rgb=True)

    print("Brand mark source art (not bundled in the app)")
    write_png(tile, GRAPHICS / "pos-go-mark-512.png")
    write_png(draw_mark(512, scale=1.18), GRAPHICS / "pos-go-glyph-512.png")


if __name__ == "__main__":
    main()
