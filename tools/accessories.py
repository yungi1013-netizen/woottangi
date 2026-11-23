"""우땅이 꾸미기 아이템(모자·안경·나비넥타이) 그림 + 프레임별 머리 위치 표 만들기

    python tools/accessories.py            # pc/acc/*.png + pc/anchors.go 생성
    python tools/accessories.py preview    # preview_acc.png (전체 동작에 아이템을 씌운 미리보기)

[머리 좌표계]
  기준 머리(head_full, 폭 300)의 "아래 가운데"가 원점(0,0), 위쪽이 -y.
    정수리   (0, -235)
    두 눈    (±53, -108)
    턱 끝    (0, 0)
  아이템 그림은 이 좌표계 x -200~200, y -330~70 (400x400 PNG, 원점 = (200, 330)).

[프레임별 머리 위치]  anchors.go
  각 프레임마다 (px, py, s, ang):  프레임 좌표 = (px,py) + s · R(ang) · 머리 좌표
  ang 은 PIL rotate 와 같은 방향(화면에서 반시계 +).
  - 일반 동작(compose.py)은 pose() 값에서 바로 계산
  - 스티커 동작은 0번 프레임에서 눈으로 맞춘 값 + stickers.motion() 의 움직임
"""
import math
import sys
import types
from pathlib import Path
from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
TOOLS = ROOT / "tools"
OUT = ROOT / "pc" / "acc"
SS = 4                     # 슈퍼샘플링 배율
CW, CH = 400, 400          # 아이템 캔버스
OX, OY = 200, 330          # 캔버스 안의 머리 원점
INK = (11, 11, 11, 255)
OW = 4.5                   # 외곽선 굵기 (몸 외곽선과 맞춤)


# ───────────────────────── 그림 도구 ─────────────────────────
class Pen:
    def __init__(self):
        self.im = Image.new("RGBA", (CW * SS, CH * SS))
        self.d = ImageDraw.Draw(self.im)

    @staticmethod
    def P(x, y):
        return ((OX + x) * SS, (OY + y) * SS)

    def poly(self, pts, fill, ow=OW):
        q = [self.P(*p) for p in pts]
        self.d.polygon(q, fill=fill)
        if ow:
            self.d.line(q + [q[0]], fill=INK, width=round(ow * SS), joint="curve")
            r = ow * SS / 2
            for x, y in q:
                self.d.ellipse([x - r, y - r, x + r, y + r], fill=INK)

    def ell(self, cx, cy, rx, ry, fill, ow=OW):
        x0, y0 = self.P(cx - rx, cy - ry)
        x1, y1 = self.P(cx + rx, cy + ry)
        if ow:
            o = ow * SS / 2
            self.d.ellipse([x0 - o, y0 - o, x1 + o, y1 + o], fill=INK)
            self.d.ellipse([x0 + o, y0 + o, x1 - o, y1 - o], fill=fill)
        else:
            self.d.ellipse([x0, y0, x1, y1], fill=fill)

    def ring(self, cx, cy, r, w, fill=None):
        x0, y0 = self.P(cx - r, cy - r)
        x1, y1 = self.P(cx + r, cy + r)
        if fill:
            self.d.ellipse([x0, y0, x1, y1], fill=fill)
        self.d.ellipse([x0, y0, x1, y1], outline=INK, width=round(w * SS))

    def line(self, pts, w, col=INK):
        q = [self.P(*p) for p in pts]
        self.d.line(q, fill=col, width=round(w * SS), joint="curve")
        r = w * SS / 2
        for x, y in (q[0], q[-1]):
            self.d.ellipse([x - r, y - r, x + r, y + r], fill=col)

    def done(self):
        return self.im.resize((CW, CH), Image.LANCZOS)


def arc_pts(cx, cy, rx, ry, a0, a1, n=40):
    """각도는 도 단위, 0 = 오른쪽, 90 = 아래 (화면 좌표)"""
    return [(cx + rx * math.cos(math.radians(a0 + (a1 - a0) * i / n)),
             cy + ry * math.sin(math.radians(a0 + (a1 - a0) * i / n))) for i in range(n + 1)]


def rot(pts, cx, cy, deg):
    c, s = math.cos(math.radians(deg)), math.sin(math.radians(deg))
    return [(cx + (x - cx) * c - (y - cy) * s, cy + (x - cx) * s + (y - cy) * c) for x, y in pts]


# ───────────────────────── 아이템 ─────────────────────────
def ribbon():
    p = Pen()
    cx, cy, a = 62, -212, 22
    red, dark = (255, 92, 108, 255), (214, 58, 80, 255)
    L = rot([(cx, cy), (cx - 52, cy - 30), (cx - 58, cy + 4), (cx - 46, cy + 30)], cx, cy, a)
    R = rot([(cx, cy), (cx + 52, cy - 30), (cx + 58, cy + 4), (cx + 46, cy + 30)], cx, cy, a)
    tl = rot([(cx - 6, cy + 6), (cx - 22, cy + 46), (cx - 6, cy + 40)], cx, cy, a)
    tr = rot([(cx + 6, cy + 6), (cx + 24, cy + 44), (cx + 8, cy + 40)], cx, cy, a)
    p.poly(tl, dark); p.poly(tr, dark)
    p.poly(L, red); p.poly(R, red)
    p.ell(cx, cy, 13, 13, dark)
    return p.done()


def flower():
    p = Pen()
    cx, cy = -74, -196
    pink = (255, 158, 196, 255)
    for k in range(5):
        a = math.radians(-90 + k * 72)
        p.ell(cx + math.cos(a) * 21, cy + math.sin(a) * 21, 17, 17, pink)
    p.ell(cx, cy, 14, 14, (255, 214, 77, 255))
    p.ell(cx - 4, cy - 4, 4, 4, (255, 245, 200, 255), 0)
    return p.done()


def sprout():
    p = Pen()
    g, g2 = (128, 206, 94, 255), (98, 178, 70, 255)
    p.line([(0, -228), (2, -252), (0, -270)], 9)
    p.line([(0, -228), (2, -252), (0, -270)], 3, g2)
    p.poly([(-1, -268), (-22, -298), (-52, -310), (-62, -290), (-36, -276)], g)
    p.poly([(1, -268), (22, -300), (52, -312), (62, -292), (36, -276)], g)
    return p.done()


def crown():
    p = Pen()
    gold, gold2 = (255, 210, 64, 255), (236, 176, 34, 255)
    base_y, top_y = -214, -292
    pts = [(-66, base_y), (-74, top_y + 18), (-38, top_y + 40), (0, top_y), (38, top_y + 40), (74, top_y + 18), (66, base_y)]
    p.poly(pts, gold)
    p.poly([(-68, base_y - 22), (68, base_y - 22), (66, base_y), (-66, base_y)], gold2)
    for x, y in ((-74, top_y + 14), (0, top_y - 6), (74, top_y + 14)):
        p.ell(x, y, 9, 9, (255, 246, 200, 255))
    p.ell(0, base_y - 11, 9, 8, (255, 84, 104, 255), 3.5)
    p.ell(-40, base_y - 11, 7, 7, (90, 170, 255, 255), 3.5)
    p.ell(40, base_y - 11, 7, 7, (90, 170, 255, 255), 3.5)
    return p.done()


def beanie():
    p = Pen()
    knit, band = (255, 214, 77, 255), (255, 120, 92, 255)
    dome = arc_pts(0, -168, 142, 90, 180, 360)
    p.poly(dome + [(142, -168), (-142, -168)], knit)
    for x in (-84, -42, 0, 42, 84):            # 뜨개 무늬
        h = math.sqrt(max(0.0, 1 - (x / 142) ** 2)) * 90
        p.line([(x, -168 - h * 0.86), (x * 1.04, -176)], 3, (226, 178, 52, 255))
    p.poly([(-148, -188), (148, -188), (150, -150), (-150, -150)], band)
    for x in range(-130, 140, 26):
        p.line([(x, -184), (x, -154)], 2.5, (220, 84, 64, 255))
    p.ell(0, -262, 24, 22, (255, 255, 255, 255))
    return p.done()


def santa():
    p = Pen()
    red, white = (232, 52, 60, 255), (255, 255, 255, 255)
    cone = [(-132, -176), (-96, -238), (-30, -270), (40, -276), (110, -262), (170, -224), (196, -170),
            (176, -176), (150, -214), (104, -228), (124, -176)]
    p.poly(cone, red)
    p.poly([(-150, -196), (150, -196), (154, -152), (-154, -152)], white)
    p.ell(190, -160, 24, 24, white)
    return p.done()


def glasses():
    p = Pen()
    tint = (180, 222, 255, 70)
    for x in (-55, 55):
        p.ring(x, -108, 37, 6.5, tint)
    p.line([(-20, -112), (0, -120), (20, -112)], 6)
    p.line([(-92, -112), (-128, -122)], 6)
    p.line([(92, -112), (128, -122)], 6)
    return p.done()


def sunglasses():
    p = Pen()
    lens = (24, 24, 32, 255)
    for x in (-56, 56):
        p.poly([(x - 44, -134), (x + 44, -134), (x + 40, -100), (x + 22, -80), (x - 22, -80), (x - 40, -100)], lens, 5)
        p.line([(x - 28, -122), (x - 14, -122)], 5, (150, 150, 170, 255))
    p.line([(-14, -128), (14, -128)], 7)
    p.line([(-100, -130), (-130, -136)], 7)
    p.line([(100, -130), (130, -136)], 7)
    return p.done()


def bowtie():
    p = Pen()
    red, dark = (255, 84, 104, 255), (210, 50, 74, 255)
    cy = 30
    p.poly([(0, cy), (-48, cy - 26), (-52, cy + 26)], red)
    p.poly([(0, cy), (48, cy - 26), (52, cy + 26)], red)
    p.ell(0, cy, 13, 13, dark)
    for x, y in ((-30, cy - 6), (-34, cy + 10), (30, cy - 6), (34, cy + 10)):
        p.ell(x, y, 3.5, 3.5, (255, 255, 255, 255), 0)
    return p.done()


def rainbow():
    """출석 30일 선물: 무지개 핀"""
    p = Pen()
    cx, cy = -66, -196
    cols = [(255, 92, 92, 255), (255, 170, 60, 255), (255, 226, 80, 255), (110, 205, 110, 255), (90, 160, 255, 255), (170, 110, 230, 255)]
    r0, bw = 62, 8
    outer = arc_pts(cx, cy, r0, r0, 180, 360)
    inner = arc_pts(cx, cy, r0 - bw * len(cols), r0 - bw * len(cols), 360, 180)
    p.poly(outer + inner, cols[0])
    for i, c in enumerate(cols):
        r = r0 - i * bw
        band = arc_pts(cx, cy, r, r, 180, 360) + arc_pts(cx, cy, r - bw, r - bw, 360, 180)
        p.poly(band, c, 0)
    p.line(arc_pts(cx, cy, r0, r0, 180, 360), OW)
    p.line(arc_pts(cx, cy, r0 - bw * len(cols), r0 - bw * len(cols), 180, 360), OW)
    for x in (cx - r0 + 22, cx + r0 - 22):      # 양 끝 구름
        p.ell(x - 12, cy + 2, 16, 13, (255, 255, 255, 255))
        p.ell(x + 10, cy + 2, 16, 13, (255, 255, 255, 255))
        p.ell(x, cy - 8, 15, 13, (255, 255, 255, 255))
        p.ell(x - 1, cy + 4, 22, 9, (255, 255, 255, 255), 0)
    return p.done()


def banana_band():
    """출석 100일 선물: 바나나 머리띠"""
    p = Pen()
    band = arc_pts(0, -112, 140, 124, 198, 342)
    p.line(band, 22)
    p.line(band, 12, (255, 214, 64, 255))
    # 먹기 동작의 바나나와 같은 초승달 모양 (compose.pose_eat)
    def q(a, c, b, n=16):
        return [((1 - t) ** 2 * a[0] + 2 * (1 - t) * t * c[0] + t * t * b[0],
                 (1 - t) ** 2 * a[1] + 2 * (1 - t) * t * c[1] + t * t * b[1]) for t in (i / n for i in range(n + 1))]
    k = 1.25
    B = lambda x, y: (x * k, y * k)
    ban = q(B(-46, 6), B(0, 34), B(46, -4)) + q(B(46, -4), B(50, -12), B(44, -14)) + q(B(44, -14), B(0, 14), B(-44, -6))
    ban = rot([(x, y - 256) for x, y in ban], 0, -256, -18)
    p.poly(ban, (255, 216, 77, 255))
    p.line(rot([(55, -273.5), (65, -283.5)], 0, -256, -18), 6)
    p.line(rot([(-55, -263.5), (-68, -268)], 0, -256, -18), 5, (120, 90, 40, 255))
    for k, x in enumerate((-60, 60)):
        p.ell(x, -226 + k * 0, 9, 9, (255, 255, 255, 255))
    return p.done()


def rudolph():
    """크리스마스 선물: 루돌프 뿔 머리띠"""
    p = Pen()
    brown, dark = (170, 112, 66, 255), (120, 74, 40, 255)
    band = arc_pts(0, -112, 140, 124, 198, 342)
    p.line(band, 20)
    p.line(band, 11, (232, 52, 60, 255))
    for sgn in (-1, 1):
        def X(x):
            return x * sgn
        main = [(X(58), -226), (X(70), -268), (X(92), -300), (X(108), -318)]
        p.line(main, 17)
        p.line(main, 9, brown)
        for a, b in (((X(72), -272), (X(46), -298)), ((X(90), -298), (X(126), -300)), ((X(66), -250), (X(96), -262))):
            p.line([a, b], 15)
            p.line([a, b], 7, brown)
        p.ell(X(58), -226, 12, 9, dark)
    for x in (-34, 34):                      # 방울
        p.ell(x, -238, 9, 9, (255, 214, 77, 255))
    return p.done()


def witch():
    """할로윈 선물: 마녀 모자"""
    p = Pen()
    hat, band = (96, 66, 150, 255), (255, 150, 52, 255)
    p.ell(0, -206, 172, 30, (78, 52, 124, 255))
    cone = [(-96, -214), (-62, -280), (-20, -340), (24, -372), (70, -380), (52, -356), (40, -318), (88, -214)]
    p.poly(cone, hat)
    p.poly([(-94, -218), (88, -218), (78, -246), (-82, -246)], band)
    star = [(52 + 13 * math.cos(math.radians(-90 + k * 36)) * (1 if k % 2 == 0 else 0.45),
             -296 + 13 * math.sin(math.radians(-90 + k * 36)) * (1 if k % 2 == 0 else 0.45)) for k in range(10)]
    p.poly(star, (255, 226, 80, 255), 3)
    p.ell(-36, -286, 9, 9, (255, 226, 80, 255), 3)
    return p.done()


def bokjumeoni():
    """설날·추석 선물: 복주머니 (목)"""
    p = Pen()
    red, pink, gold = (232, 60, 84, 255), (255, 140, 160, 255), (255, 206, 64, 255)
    cy = 34
    p.line([(-40, cy - 30), (0, cy - 18), (40, cy - 30)], 6, gold)
    body = arc_pts(0, cy + 22, 40, 30, -10, 190) + [(-20, cy - 2), (-26, cy - 14), (0, cy - 6), (26, cy - 14), (20, cy - 2)]
    p.poly(body, red)
    p.poly([(-24, cy - 14), (-14, cy - 30), (0, cy - 22), (14, cy - 30), (24, cy - 14), (0, cy - 6)], pink)
    p.ell(0, cy - 4, 7, 6, gold, 3)
    p.ell(0, cy + 26, 15, 15, gold, 3.5)
    p.ell(0, cy + 26, 7, 7, red, 0)
    return p.done()


def dirt():
    """꼬질꼬질할 때 얼굴에 묻은 얼룩 (외곽선 없음, 반투명)"""
    p = Pen()
    mud = (96, 66, 40, 165)
    for x, y, rx, ry in ((-84, -146, 16, 11), (-70, -156, 7, 5), (86, -80, 15, 10), (100, -70, 6, 4),
                         (-30, -64, 9, 6), (-66, -206, 18, 10), (60, -212, 9, 6), (-104, -40, 10, 7)):
        p.ell(x, y, rx, ry, mud, 0)
    for x, y in ((-96, -132), (40, -142), (74, -98)):   # 점 몇 개
        p.ell(x, y, 3.5, 3, mud, 0)
    return p.done()


def starpin():
    """뽑기 선물: 반짝 별 머리핀"""
    p = Pen()
    gold, light = (255, 212, 60, 255), (255, 240, 160, 255)
    for cx, cy, r, a in ((70, -214, 34, 12), (112, -186, 18, -8)):
        pts = [(cx + r * (1 if k % 2 == 0 else 0.46) * math.cos(math.radians(-90 + a + k * 36)),
                cy + r * (1 if k % 2 == 0 else 0.46) * math.sin(math.radians(-90 + a + k * 36))) for k in range(10)]
        p.poly(pts, gold)
        p.ell(cx - r * 0.18, cy - r * 0.2, r * 0.16, r * 0.12, light, 0)
    return p.done()


def halo():
    """뽑기 선물: 천사 링 (머리 위에 동동)"""
    p = Pen()
    x0, y0 = p.P(-92, -300)
    x1, y1 = p.P(92, -258)
    w = round(OW * SS)
    p.d.ellipse([x0 - w, y0 - w, x1 + w, y1 + w], outline=INK, width=round(22 * SS))
    p.d.ellipse([x0, y0, x1, y1], outline=(255, 226, 90, 255), width=round(13 * SS))
    for x, y in ((-104, -312), (112, -262)):
        p.line([(x - 9, y), (x + 9, y)], 3.5, (255, 226, 90, 255))
        p.line([(x, y - 9), (x, y + 9)], 3.5, (255, 226, 90, 255))
    return p.done()


def heartglasses():
    """뽑기 선물: 하트 선글라스 (얼굴)"""
    p = Pen()
    pink = (255, 86, 140, 235)
    for x in (-56, 56):
        heart = [(x + 40 * (16 * math.sin(t) ** 3) / 16,
                  -104 - 40 * (13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t)) / 16)
                 for t in [i * 2 * math.pi / 48 for i in range(48)]]
        p.poly(heart, pink, 5)
        p.ell(x - 16, -120, 8, 6, (255, 200, 222, 255), 0)
    p.line([(-16, -122), (16, -122)], 7)
    p.line([(-96, -126), (-130, -134)], 7)
    p.line([(96, -126), (130, -134)], 7)
    return p.done()


def scarf():
    """뽑기 선물: 줄무늬 목도리 (목)"""
    p = Pen()
    red, cream = (232, 70, 76, 255), (255, 244, 220, 255)
    band = [(-96, 6), (-48, 22), (0, 26), (48, 22), (96, 6), (102, 34), (50, 52), (0, 56), (-50, 52), (-102, 34)]
    p.poly(band, red)
    for x in (-62, -20, 22, 64):
        p.poly([(x - 9, 14 + abs(x) * -0.05), (x + 9, 16), (x + 9, 50), (x - 9, 48)], cream, 0)
    p.poly([(36, 40), (70, 44), (78, 118), (44, 116)], red)
    for y in (64, 92):
        p.poly([(40, y), (74, y + 2), (76, y + 12), (42, y + 10)], cream, 0)
    p.line([(46, 118), (44, 132)], 4); p.line([(60, 118), (60, 133)], 4); p.line([(74, 118), (76, 132)], 4)
    return p.done()


# id: (그림 함수)  — 이름·가격·자리는 pc/shop.go 에 있음
ITEMS = {
    "ribbon": ribbon, "flower": flower, "sprout": sprout, "crown": crown,
    "beanie": beanie, "santa": santa, "glasses": glasses, "sunglasses": sunglasses, "bowtie": bowtie,
    "rainbow": rainbow, "bananaband": banana_band,
    "rudolph": rudolph, "witch": witch, "bokjumeoni": bokjumeoni, "dirt": dirt,
    "starpin": starpin, "halo": halo, "heartglasses": heartglasses, "scarf": scarf,
}
FACE_ITEMS = {"glasses", "sunglasses", "heartglasses"}


# ───────────────────────── 프레임별 머리 위치 ─────────────────────────
def load_compose():
    # compose.py 의 pose() 만 쓰면 되므로 무거운 모듈은 가짜로 채움
    for m in ("playwright", "playwright.sync_api", "cv2"):
        mod = types.ModuleType(m)
        mod.sync_playwright = None
        sys.modules.setdefault(m, mod)
    sys.path.insert(0, str(TOOLS))
    import compose
    return compose


# 스티커 0번 프레임에서 기준 머리를 겹쳐 맞춘 값 (px, py, s, ang)
STICKER0 = {
    "me": (252, 398, 0.50, 0),
    "lounge": (185, 420, 0.60, 70),
    "cry": (250, 435, 0.95, 0),
    "surprise": (200, 452, 1.03, -12),
    "bow": (225, 485, 0.75, -12),
}
# 얼굴 모양이 기준 머리와 많이 달라서 안경을 잠깐 벗는 동작
NO_FACE = {"st_me", "st_lounge", "st_surprise"}


def rot_ccw(x, y, deg):
    a = math.radians(deg)
    return x * math.cos(a) + y * math.sin(a), -x * math.sin(a) + y * math.cos(a)


def anchors():
    cp = load_compose()
    sys.path.insert(0, str(TOOLS))
    import stickers
    res = {}
    for name, n in cp.SEQ.items():
        lst = []
        for k in range(n):
            _, _, _, (hk, hb, hrot, hdx) = cp.pose(name, k / n)
            head = cp.HEADS[hk]
            pad = 60
            x0 = int(cp.W / 2 - head.width / 2 - pad + hdx)
            y0 = int(hb - head.height - pad)
            lst.append((x0 + pad + head.width / 2, y0 + pad + head.height, 1.0, hrot))
        res[name] = lst
    W, G = stickers.W, stickers.GROUND
    for name, (px, py, s, ang) in STICKER0.items():
        dx0, dy0, sx0, sy0, r0 = stickers.motion(name, 0)
        rx, ry = rot_ccw(px - (W / 2 + dx0), py - (G + dy0), -r0)
        bx, by = rx / sx0, ry / sy0
        lst = []
        for k in range(stickers.N):
            dx, dy, sx, sy, r = stickers.motion(name, k)
            qx, qy = rot_ccw(bx * sx, by * sy, r)
            lst.append((W / 2 + dx + qx, G + dy + qy, s * math.sqrt(sx * sy / (sx0 * sy0)), ang - r0 + r))
        res["st_" + name] = lst
    return res


def write_go(res):
    out = ["// 자동 생성: python tools/accessories.py  (손으로 고치지 마세요)", "",
           "package main", "", "// 프레임별 머리 위치: 프레임 좌표 = (X,Y) + S · R(Ang) · 머리 좌표", "var headAnchors = map[string][]anchor{"]
    for name in sorted(res):
        out.append(f'\t"{name}": {{')
        for x, y, s, a in res[name]:
            out.append(f"\t\t{{{x:.2f}, {y:.2f}, {s:.4f}, {a:.3f}}},")
        out.append("\t},")
    out.append("}")
    out += ["", "// 안경(얼굴 자리)을 잠깐 벗는 동작", "var noFaceSeq = map[string]bool{"]
    out += [f'\t"{n}": true,' for n in sorted(NO_FACE)]
    out.append("}")
    (ROOT / "pc" / "anchors.go").write_text("\n".join(out) + "\n", encoding="utf-8")


# ───────────────────────── 미리보기 (Go 와 같은 계산) ─────────────────────────
def put(frame, acc, a):
    px, py, s, ang = a
    big = acc.resize((round(CW * s), round(CH * s)), Image.LANCZOS)
    pad = 300
    c = Image.new("RGBA", (big.width + pad * 2, big.height + pad * 2))
    c.alpha_composite(big, (pad, pad))
    cx, cy = pad + OX * s, pad + OY * s
    c = c.rotate(ang, resample=Image.BICUBIC, center=(cx, cy))
    frame.alpha_composite(c, (round(px - cx), round(py - cy)))


def preview(res, wear, out):
    fr_dir = ROOT / "frames"
    accs = [(w, Image.open(OUT / f"{w}.png").convert("RGBA")) for w in wear]
    names = [(n, k) for n in sorted(res) for k in range(0, len(res[n]), max(1, len(res[n]) // 2))]
    tw, th = 160, 200
    cols = 10
    sheet = Image.new("RGBA", (tw * cols, th * ((len(names) + cols - 1) // cols)), (205, 225, 245, 255))
    for i, (n, k) in enumerate(names):
        f = Image.open(fr_dir / f"{n}_{k}.png").convert("RGBA")
        for w, acc in accs:
            if w in FACE_ITEMS and n in NO_FACE:
                continue
            put(f, acc, res[n][k])
        sheet.alpha_composite(f.resize((tw, th), Image.LANCZOS), ((i % cols) * tw, (i // cols) * th))
    sheet.save(out)
    print("preview ->", out)


if __name__ == "__main__":
    OUT.mkdir(parents=True, exist_ok=True)
    for k, fn in ITEMS.items():
        fn().save(OUT / f"{k}.png")
    res = anchors()
    write_go(res)
    print("accessories: pc/acc/*.png, pc/anchors.go")
    if len(sys.argv) > 1 and sys.argv[1] == "preview":
        out = Path(sys.argv[2]) if len(sys.argv) > 2 else ROOT / "preview_acc.png"
        wear = sys.argv[3].split(",") if len(sys.argv) > 3 else ["crown", "glasses", "bowtie"]
        preview(res, wear, out)
