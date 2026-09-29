#!/usr/bin/env python3
"""Builds Parley's logo, icons and banners from SVG (needs cairosvg, Pillow).

Outputs to assets/logo/: parley-logo.svg, parley-icon-<size>.png,
parley.ico (multi-size, with a simplified mark for tiny sizes),
curseforge-400.png, github-social-1280x640.png, and the addon's
minimap icon (addon/Parley/Media/icon.tga).
Fonts: Cinzel (SIL OFL 1.1) for the wordmark, IPAGothic for the glyph.
"""
import io, os, struct
import cairosvg
from PIL import Image

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
OUT = os.path.join(ROOT, "assets", "logo")
os.makedirs(OUT, exist_ok=True)

DEFS = """
<defs>
  <linearGradient id="gold" x1="0" y1="0" x2="0.3" y2="1">
    <stop offset="0" stop-color="#fff4c2"/><stop offset="0.35" stop-color="#f2c75a"/>
    <stop offset="0.7" stop-color="#b9831f"/><stop offset="1" stop-color="#6e4a10"/>
  </linearGradient>
  <linearGradient id="goldEdge" x1="0" y1="1" x2="0" y2="0">
    <stop offset="0" stop-color="#fff0b0"/><stop offset="1" stop-color="#5a3a08"/>
  </linearGradient>
  <radialGradient id="disc" cx="0.42" cy="0.36" r="0.75">
    <stop offset="0" stop-color="#27406b"/><stop offset="0.6" stop-color="#132441"/><stop offset="1" stop-color="#070d1c"/>
  </radialGradient>
  <linearGradient id="parch" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="#fff0bd"/><stop offset="1" stop-color="#e2ad3c"/>
  </linearGradient>
  <linearGradient id="cyan" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="#9aeeff"/><stop offset="1" stop-color="#2b9fd6"/>
  </linearGradient>
  <filter id="shadow" x="-20%" y="-20%" width="140%" height="140%">
    <feDropShadow dx="0" dy="3" stdDeviation="3" flood-color="#000" flood-opacity="0.55"/>
  </filter>
</defs>"""

# Speech bubble: rounded rectangle plus a tail, as one path.
def bubble(x, y, w, h, r, tail):
    tx, ty, side = tail  # tail tip and which bottom corner it leaves from
    if side == "left":
        b1, b2 = x + r * 0.9, x + r * 0.9 + w * 0.22
    else:
        b1, b2 = x + w - r * 0.9 - w * 0.22, x + w - r * 0.9
    return (f"M{x+r},{y} H{x+w-r} Q{x+w},{y} {x+w},{y+r} V{y+h-r} Q{x+w},{y+h} {x+w-r},{y+h} "
            f"H{b2} L{tx},{ty} L{b1},{y+h} H{x+r} Q{x},{y+h} {x},{y+h-r} V{y+r} Q{x},{y} {x+r},{y} Z")

def bubbles(detail=True):
    back = bubble(46, 58, 112, 80, 28, (52, 162, "left"))
    front = bubble(98, 108, 114, 82, 30, (206, 214, "right"))
    s = f"""
  <g filter="url(#shadow)">
    <path d="{back}" fill="url(#parch)" stroke="#4a2e06" stroke-width="5" stroke-linejoin="round"/>
  </g>
  <g filter="url(#shadow)">
    <path d="{front}" fill="url(#cyan)" stroke="#06283e" stroke-width="5" stroke-linejoin="round"/>
  </g>"""
    if detail:
        s += """
  <text x="90" y="112" font-family="Cinzel" font-weight="900" font-size="50" fill="#4a2e06" text-anchor="middle">A</text>
  <text x="156" y="170" font-family="IPAGothic" font-weight="bold" font-size="52" fill="#06283e" text-anchor="middle">文</text>"""
    else:
        for cx in (130, 155, 180):
            s += f'<circle cx="{cx}" cy="150" r="9" fill="#06283e"/>'
    return s

def badge(detail=True):
    return f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">{DEFS}
  <circle cx="128" cy="128" r="126" fill="#050403"/>
  <circle cx="128" cy="128" r="116" fill="none" stroke="url(#gold)" stroke-width="16"/>
  <circle cx="128" cy="128" r="124" fill="none" stroke="url(#goldEdge)" stroke-width="2" opacity="0.8"/>
  <circle cx="128" cy="128" r="107" fill="url(#disc)" stroke="#000" stroke-width="3"/>
  <ellipse cx="112" cy="70" rx="70" ry="30" fill="#ffffff" opacity="0.06"/>
  {bubbles(detail)}
</svg>"""

# Tiny sizes (16-32 px): no ring detail, bubbles fill the square.
def small():
    return f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="38 50 182 178" width="256" height="256">{DEFS}
  {bubbles(False).replace('filter="url(#shadow)"', '').replace('stroke-width="5"', 'stroke-width="9"')}
</svg>"""

def render(svg, size):
    png = cairosvg.svg2png(bytestring=svg.encode(), output_width=size, output_height=size)
    return Image.open(io.BytesIO(png)).convert("RGBA")

def write_ico(path, images):
    # ICO with PNG-compressed entries (Vista+), one per size.
    entries, blobs = [], []
    for im in images:
        b = io.BytesIO(); im.save(b, "PNG"); blobs.append(b.getvalue())
    off = 6 + 16 * len(images)
    with open(path, "wb") as f:
        f.write(struct.pack("<HHH", 0, 1, len(images)))
        for im, blob in zip(images, blobs):
            w = im.width if im.width < 256 else 0
            f.write(struct.pack("<BBBBHHII", w, w, 0, 0, 1, 32, len(blob), off))
            off += len(blob)
        for blob in blobs:
            f.write(blob)

big = badge(True)
open(os.path.join(OUT, "parley-logo.svg"), "w").write(big)
icons = []
for s in (16, 20, 24, 32, 40, 48, 64, 128, 256):
    im = render(small(), s) if s <= 32 else render(badge(s >= 64), s)
    icons.append(im)
    if s in (32, 64, 128, 256):
        im.save(os.path.join(OUT, f"parley-icon-{s}.png"))
write_ico(os.path.join(OUT, "parley.ico"), icons)

# Bubbles without the badge, drawn inside the WoW skins' portrait ring.
def bubbles_only():
    return f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="36 46 184 184" width="256" height="256">{DEFS}
  {bubbles(True)}
</svg>"""
write_ico(os.path.join(OUT, "parley-bubbles.ico"),
          [render(small(), s) if s <= 32 else render(bubbles_only(), s) for s in (24, 32, 48, 64, 96, 128)])

# Addon minimap/AddOn-list icon: 64x64 uncompressed 32-bit TGA.
media = os.path.join(ROOT, "addon", "Parley", "Media")
os.makedirs(media, exist_ok=True)
render(badge(True), 64).save(os.path.join(media, "icon.tga"))
# Minimap button artwork: just the bubbles; the minimap ring is WoW's own.
render(f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="36 46 184 184" width="256" height="256">{DEFS}{bubbles(False)}</svg>""", 64).save(
    os.path.join(media, "minimap.tga"))

# CurseForge project avatar (square) and GitHub social preview.
render(big, 400).save(os.path.join(OUT, "curseforge-400.png"))
banner = f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 640" width="1280" height="640">{DEFS}
  <defs><radialGradient id="bg" cx="0.3" cy="0.4" r="0.9">
    <stop offset="0" stop-color="#1c2c4c"/><stop offset="1" stop-color="#05080f"/></radialGradient></defs>
  <rect width="1280" height="640" fill="url(#bg)"/>
  <rect x="14" y="14" width="1252" height="612" fill="none" stroke="url(#gold)" stroke-width="4" rx="6"/>
  <g transform="translate(110,150) scale(1.3)">{big[big.index('<circle'):big.rindex('</svg>')]}</g>
  <defs><linearGradient id="word" gradientUnits="userSpaceOnUse" x1="0" y1="190" x2="0" y2="300">
    <stop offset="0" stop-color="#fff3c0"/><stop offset="0.45" stop-color="#f3c24c"/><stop offset="1" stop-color="#a86d12"/></linearGradient></defs>
  <text x="520" y="300" font-family="Cinzel" font-weight="900" font-size="150" fill="url(#word)" stroke="#2a1a04" stroke-width="3">Parley</text>
  <text x="526" y="370" font-family="DejaVu Sans" font-size="34" fill="#dfe7f2">Live chat translation for WoW Classic Era</text>
  <text x="526" y="420" font-family="DejaVu Sans" font-size="24" fill="#9fb0c4">Offline on your PC · 30+ languages · reply in theirs</text>
</svg>"""
render_w = cairosvg.svg2png(bytestring=banner.encode(), output_width=1280, output_height=640)
open(os.path.join(OUT, "github-social-1280x640.png"), "wb").write(render_w)
print("ok")
