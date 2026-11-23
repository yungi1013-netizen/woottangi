"""우땅이 효과 그림: 계절 날씨 쇼(눈·꽃잎·낙엽·비눗방울…) 조각 + 목욕 거품 + 이불

    python tools/effects.py            # pc/fx/*.png 생성
    python tools/effects.py preview    # preview_fx.png (전부 한 장에)

조각 그림은 64x64 (가운데 = 중심), 이불은 프레임과 같은 480x600 (자는 동작에 그대로 덧그림).
"""
import math
import sys
from pathlib import Path
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "pc" / "fx"
SS = 4
INK = (11, 11, 11, 255)


class Pen:
    def __init__(self, w=64, h=64, ox=32, oy=32):
        self.w, self.h, self.ox, self.oy = w, h, ox, oy
        self.im = Image.new("RGBA", (w * SS, h * SS))
        self.d = ImageDraw.Draw(self.im)

    def P(self, x, y):
        return ((self.ox + x) * SS, (self.oy + y) * SS)

    def poly(self, pts, fill, ow=2.5, ink=INK):
        q = [self.P(*p) for p in pts]
        self.d.polygon(q, fill=fill)
        if ow:
            self.d.line(q + [q[0]], fill=ink, width=round(ow * SS), joint="curve")

    def ell(self, cx, cy, rx, ry, fill, ow=2.5, ink=INK):
        x0, y0 = self.P(cx - rx, cy - ry)
        x1, y1 = self.P(cx + rx, cy + ry)
        if ow:
            o = ow * SS / 2
            self.d.ellipse([x0 - o, y0 - o, x1 + o, y1 + o], fill=ink)
            self.d.ellipse([x0 + o, y0 + o, x1 - o, y1 - o], fill=fill)
        else:
            self.d.ellipse([x0, y0, x1, y1], fill=fill)

    def line(self, pts, w, col=INK):
        q = [self.P(*p) for p in pts]
        self.d.line(q, fill=col, width=round(w * SS), joint="curve")
        r = w * SS / 2
        for x, y in (q[0], q[-1]):
            self.d.ellipse([x - r, y - r, x + r, y + r], fill=col)

    def done(self):
        return self.im.resize((self.w, self.h), Image.LANCZOS)


def star_pts(cx, cy, r, inner=0.45, n=5, a0=-90):
    return [(cx + r * (1 if k % 2 == 0 else inner) * math.cos(math.radians(a0 + k * 180 / n)),
             cy + r * (1 if k % 2 == 0 else inner) * math.sin(math.radians(a0 + k * 180 / n))) for k in range(2 * n)]


def snow():
    p = Pen()
    edge = (120, 160, 210, 255)
    for k in range(3):
        a = math.radians(k * 60)
        dx, dy = math.cos(a) * 20, math.sin(a) * 20
        p.line([(-dx, -dy), (dx, dy)], 7, edge)
    for k in range(6):                      # 가지
        a = math.radians(k * 60)
        bx, by = math.cos(a) * 12, math.sin(a) * 12
        for s in (-1, 1):
            b = a + s * math.radians(45)
            p.line([(bx, by), (bx + math.cos(b) * 7, by + math.sin(b) * 7)], 5, edge)
    for k in range(3):
        a = math.radians(k * 60)
        dx, dy = math.cos(a) * 19, math.sin(a) * 19
        p.line([(-dx, -dy), (dx, dy)], 3.5, (255, 255, 255, 255))
    for k in range(6):
        a = math.radians(k * 60)
        bx, by = math.cos(a) * 12, math.sin(a) * 12
        for s in (-1, 1):
            b = a + s * math.radians(45)
            p.line([(bx, by), (bx + math.cos(b) * 6, by + math.sin(b) * 6)], 2.2, (255, 255, 255, 255))
    p.ell(0, 0, 5, 5, (255, 255, 255, 255), 2, edge)
    return p.done()


def petal():
    """벚꽃잎: 아래가 뾰족하고 위 가운데가 살짝 파인 모양"""
    p = Pen()
    pts = []
    for i in range(61):
        t = math.pi * 2 * i / 60
        x = math.sin(t) * 15
        y = -math.cos(t) * 22
        if y < 0:                       # 위쪽은 넓고 둥글게
            x *= 1.15
        else:                           # 아래쪽은 뾰족하게
            x *= (1 - y / 26)
        pts.append((x, y))
    pts = [(x, y + 7 * math.exp(-(x / 5) ** 2)) if y < -14 else (x, y) for x, y in pts]
    p.poly(pts, (255, 196, 218, 255), 2.2, (214, 110, 150, 255))
    p.line([(0, 14), (0, -4)], 1.6, (240, 140, 180, 255))
    return p.done()


def maple():
    p = Pen()
    pts = []
    for k in range(10):
        r = 24 if k % 2 == 0 else 10
        if k in (4, 6):
            r = 17 if k % 2 == 0 else 10
        a = math.radians(-90 + k * 36)
        pts.append((math.cos(a) * r, math.sin(a) * r - 2))
    p.poly(pts, (236, 92, 52, 255), 2.2, (150, 46, 24, 255))
    p.line([(0, -2), (0, 26)], 2.2, (150, 46, 24, 255))
    return p.done()


def ginkgo():
    p = Pen()
    fan = [(0, 10)] + [(math.cos(math.radians(a)) * 24, math.sin(math.radians(a)) * 24 + 10) for a in range(200, 341, 10)]
    fan = [(x, y) if not (abs(x) < 3) else (x, y + 7) for x, y in fan]
    p.poly(fan, (255, 210, 64, 255), 2.2, (176, 128, 20, 255))
    p.line([(0, 10), (2, 26)], 2.2, (176, 128, 20, 255))
    return p.done()


def bubble():
    p = Pen()
    p.ell(0, 0, 22, 22, (200, 230, 255, 60), 2.4, (120, 170, 230, 230))
    p.d.arc([*p.P(-16, -16), *p.P(16, 16)], 200, 260, fill=(255, 255, 255, 255), width=3 * SS)
    p.ell(-9, -11, 3.5, 3.5, (255, 255, 255, 255), 0)
    p.d.arc([*p.P(-19, -19), *p.P(19, 19)], 20, 70, fill=(255, 170, 220, 200), width=2 * SS)
    return p.done()


def star():
    p = Pen()
    p.poly(star_pts(0, 1, 22), (255, 222, 72, 255), 2.6)
    p.ell(-5, -3, 3, 3, (255, 250, 210, 255), 0)
    return p.done()


def heart():
    p = Pen()
    pts = []
    for i in range(60):
        t = 2 * math.pi * i / 60
        x = 16 * math.sin(t) ** 3
        y = -(13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t))
        pts.append((x * 1.25, y * 1.25 + 1))
    p.poly(pts, (255, 110, 150, 255), 2.6)
    p.ell(-8, -6, 3.5, 3.5, (255, 220, 232, 255), 0)
    return p.done()


def sparkle():
    p = Pen()
    pts = star_pts(0, 0, 24, 0.22, 4)
    p.poly(pts, (255, 246, 170, 255), 2.2, (230, 170, 40, 255))
    p.ell(0, 0, 4, 4, (255, 255, 255, 255), 0)
    return p.done()


def candy():
    p = Pen()
    p.poly([(-12, 0), (-24, -10), (-24, 10)], (255, 120, 170, 255), 2.2)
    p.poly([(12, 0), (24, -10), (24, 10)], (255, 120, 170, 255), 2.2)
    p.ell(0, 0, 13, 11, (255, 160, 50, 255), 2.4)
    p.line([(-6, -6), (4, 8)], 2.5, (255, 240, 200, 255))
    return p.done()


def confetti(col):
    def f():
        p = Pen()
        p.poly([(-10, -6), (10, -6), (10, 6), (-10, 6)], col, 1.6)
        return p.done()
    return f


def blanket():
    """이불: 자는 동작(sleep) 프레임과 같은 480x600 에 그려서 그대로 덧그림"""
    p = Pen(480, 600, 0, 0)
    blue, blue2 = (150, 196, 255, 255), (120, 170, 240, 255)
    top = [(x, 492 + 7 * math.sin(x / 26)) for x in range(96, 405, 6)]
    body = top + [(410, 600), (90, 600)]
    p.poly(body, blue, 4.5)
    for x in range(118, 400, 46):            # 물방울무늬
        for y in (530, 572):
            xx = x + (23 if y == 572 else 0)
            if 104 < xx < 396:
                p.ell(xx, y, 9, 9, (255, 255, 255, 230), 0)
    sheet = [(x, y - 2) for x, y in top] + [(x, y + 16) for x, y in reversed(top)]
    p.poly(sheet, (255, 255, 255, 255), 3.5)
    p.line([(x, y + 7) for x, y in top[2:-2]], 1.8, blue2)
    return p.done()


def banana(gold=False):
    """바탕화면 바나나 받기: 바나나 (금 바나나)"""
    p = Pen()
    body, dark = ((255, 214, 40, 255), (230, 170, 20, 255)) if not gold else ((255, 196, 30, 255), (214, 140, 10, 255))
    outer = [(-25 + 50 * i / 20, -8 + 22 * math.sin(math.pi * i / 20)) for i in range(21)]
    inner = [(25 - 50 * i / 20, -12 + 10 * math.sin(math.pi * i / 20)) for i in range(21)]
    p.poly(outer + inner, body)
    p.line([(-12, 6), (0, 8), (12, 6)], 2, dark)
    p.line([(24, -10), (29, -19)], 4.5, (110, 80, 30, 255))
    p.ell(-25, -10, 2.5, 2.5, (90, 60, 20, 255), 0)
    if gold:
        for x, y, r in ((-18, -22, 7), (16, -24, 5), (0, -26, 4)):
            p.poly(star_pts(x, y, r, 0.35, 4, -90), (255, 255, 230, 255), 1.5)
    return p.done()


def rock():
    """바탕화면 바나나 받기: 돌멩이 (받으면 어질어질)"""
    p = Pen()
    p.poly([(-20, 8), (-16, -10), (-2, -18), (14, -14), (22, 0), (18, 14), (0, 18), (-14, 16)], (150, 150, 160, 255))
    p.line([(-6, -6), (2, -10)], 2, (110, 110, 120, 255))
    p.ell(8, 4, 3, 2, (120, 120, 130, 255), 0)
    return p.done()


def memo():
    """포스트잇 메모 (우땅이가 들고 다님)"""
    p = Pen()
    p.poly(rot([(-20, -20), (20, -20), (20, 12), (12, 20), (-20, 20)], 0, 0, -8), (255, 236, 120, 255), 2.5)
    for y in (-10, -2, 6):
        p.line(rot([(-12, y), (12, y)], 0, 0, -8), 1.6, (200, 170, 60, 255))
    return p.done()


def rot(pts, cx, cy, deg):
    c, s = math.cos(math.radians(deg)), math.sin(math.radians(deg))
    return [(cx + (x - cx) * c - (y - cy) * s, cy + (x - cx) * s + (y - cy) * c) for x, y in pts]


SPRITES = {
    "snow": snow, "petal": petal, "maple": maple, "ginkgo": ginkgo, "bubble": bubble,
    "star": star, "heart": heart, "sparkle": sparkle, "candy": candy,
    "conf0": confetti((255, 92, 108, 255)), "conf1": confetti((255, 214, 64, 255)),
    "conf2": confetti((90, 190, 255, 255)), "conf3": confetti((120, 210, 110, 255)),
    "blanket": blanket,
    "banana": banana, "goldbanana": lambda: banana(True), "rock": rock, "memo": memo,
}


if __name__ == "__main__":
    OUT.mkdir(parents=True, exist_ok=True)
    ims = {}
    for k, fn in SPRITES.items():
        ims[k] = fn()
        ims[k].save(OUT / f"{k}.png")
    print("effects: pc/fx/*.png")
    if len(sys.argv) > 1 and sys.argv[1] == "preview":
        sheet = Image.new("RGBA", (128 * (len(ims) - 1), 64 * 2 + 300), (205, 225, 245, 255))
        x = 0
        for k, im in ims.items():
            if k == "blanket":
                continue
            sheet.alpha_composite(im.resize((128, 128), Image.LANCZOS), (x, 0))
            x += 128
        fr = Image.open(ROOT / "frames" / "sleep_0.png").convert("RGBA")
        fr.alpha_composite(ims["blanket"])
        sheet.alpha_composite(fr.resize((240, 300)), (0, 128))
        sheet.save(ROOT / "preview_fx.png")
        print("preview -> preview_fx.png")
