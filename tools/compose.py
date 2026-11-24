"""우땅이 프레임 합성기

원본 머리(assets/head_*.png) + SVG로 그린 몸·팔·다리를 한 실루엣으로 합쳐
손그림 느낌의 끊긴 외곽선을 입힌 뒤 480x600 PNG 프레임으로 저장합니다.

    python tools/compose.py            # frames/ 에 전체 동작 저장
    python tools/compose.py 출력폴더
    python tools/compose.py 출력폴더 clap heart   # 그 동작만
"""
import math, os, io
from pathlib import Path
from PIL import Image
from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parent.parent
ASSETS = ROOT / "assets"

W, H = 480, 600          # 2배 해상도 캔버스
GROUND = 588
SK = "#c98f6f"; INK = "#0b0b0b"
OW = 4.2                 # 외곽선 굵기 (원본 머리 선 굵기에 맞춤)
HEAD_W = 300             # 머리 폭
HEADS = {k: Image.open(ASSETS / f"head_{k}.png").convert("RGBA") for k in ("full", "blink", "sleep", "eat0", "eat1", "shy", "sad")}
S = HEAD_W / HEADS["full"].width
HEADS = {k: v.resize((HEAD_W, round(v.height * S)), Image.LANCZOS) for k, v in HEADS.items()}
HH = HEADS["full"].height
HS = 0.27                 # 손 크기 배율
HANDS = {k: Image.open(ASSETS / f"{k}.png").convert("RGBA") for k in ("fistL", "fistR", "palmL", "palmR")}
HANDS = {k: v.resize((round(v.width * HS * (1.25 if k.startswith("palm") else 1)), round(v.height * HS * (1.25 if k.startswith("palm") else 1))), Image.LANCZOS) for k, v in HANDS.items()}

DEF = ('<defs><filter id="w" filterUnits="userSpaceOnUse" x="0" y="0" width="480" height="600"><feTurbulence type="fractalNoise" baseFrequency="0.03" numOctaves="2" seed="3"/>'
       '<feDisplacementMap in="SourceGraphic" scale="2.2" xChannelSelector="R" yChannelSelector="G"/></filter>'
       '<filter id="glow" filterUnits="userSpaceOnUse" x="0" y="0" width="480" height="600"><feGaussianBlur stdDeviation="9"/></filter></defs>')


def limb(pts, width):
    """굵은 팔/다리: 검은 외곽 + 살색 속"""
    d = "M" + " L".join(f"{x:.1f} {y:.1f}" for x, y in pts[:1]) + "".join(
        f" Q{pts[i][0]:.1f} {pts[i][1]:.1f} {pts[i+1][0]:.1f} {pts[i+1][1]:.1f}" for i in range(1, len(pts) - 1, 2))
    return (f'<path d="{d}" stroke="{INK}" stroke-width="{width+2*OW:.1f}" fill="none" stroke-linecap="round" stroke-linejoin="round"/>'
            f'<path d="{d}" stroke="{SK}" stroke-width="{width:.1f}" fill="none" stroke-linecap="round" stroke-linejoin="round"/>')


CREAM = "#fbefe6"


def foot(x, y, rot=0, rx=17, ry=10):
    # 이모티콘처럼 작고 하얀 발
    return (f'<ellipse cx="{x:.1f}" cy="{y:.1f}" rx="{rx}" ry="{ry}" transform="rotate({rot:.1f} {x:.1f} {y:.1f})" '
            f'fill="{CREAM}" stroke="{INK}" stroke-width="{OW}"/>')


def torso(cx, top, bottom, wtop, wbelly, tilt=0.0):
    # 목 없이 머리에서 바로 이어지는 배불뚝 서양배 모양 몸
    h = bottom - top
    t = math.radians(tilt)
    def P(x, y):  # 아래 중심 기준 기울이기
        dx, dy = x - cx, y - bottom
        return (cx + dx * math.cos(t) - dy * math.sin(t), bottom + dx * math.sin(t) + dy * math.cos(t))
    pts = [P(cx - wtop, top), P(cx - wbelly - 6, top + h * 0.45), P(cx - wbelly, top + h * 0.85), P(cx - wbelly * 0.55, bottom + 2),
           P(cx, bottom + 6), P(cx + wbelly * 0.55, bottom + 2), P(cx + wbelly, top + h * 0.85), P(cx + wbelly + 6, top + h * 0.45), P(cx + wtop, top)]
    d = (f"M{pts[0][0]:.1f} {pts[0][1]:.1f} C{pts[1][0]:.1f} {pts[1][1]:.1f} {pts[2][0]:.1f} {pts[2][1]:.1f} {pts[3][0]:.1f} {pts[3][1]:.1f} "
         f"Q{pts[4][0]:.1f} {pts[4][1]:.1f} {pts[5][0]:.1f} {pts[5][1]:.1f} "
         f"C{pts[6][0]:.1f} {pts[6][1]:.1f} {pts[7][0]:.1f} {pts[7][1]:.1f} {pts[8][0]:.1f} {pts[8][1]:.1f} Z")
    return "U:" + d


def svg(parts):
    # "U:" 로 시작하는 몸·팔·다리는 하나로 합쳐서 바깥 테두리만 그림 (이음새 선 없음)
    us = [p[2:] for p in parts if p.startswith("U:")]
    rest = [p for p in parts if not p.startswith("U:")]
    uni = ""
    if us:
        uni = ("".join(f'<path d="{d}" fill="{INK}" stroke="{INK}" stroke-width="{OW*2:.1f}" stroke-linejoin="round"/>' for d in us)
               + "".join(f'<path d="{d}" fill="{SK}"/>' for d in us))
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}">{DEF}<g filter="url(#w)">{uni}{"".join(rest)}</g></svg>'


def glow(cx, y):
    return f'<ellipse cx="{cx:.1f}" cy="{y:.1f}" rx="60" ry="15" fill="#fff6f2" filter="url(#glow)"/>'


ARM = 17   # 가는 팔
LEG = 20   # 가는 다리


def tarm(joints, w0=34, w1=22):
    """어깨→팔꿈치→손목: 끝으로 갈수록 가늘어지고 부드럽게 굽은 팔"""
    P = [joints[0]] + list(joints) + [joints[-1]]
    pts = []
    for k in range(1, len(P) - 2):
        p0, p1, p2, p3 = P[k - 1], P[k], P[k + 1], P[k + 2]
        for s_ in range(10):
            u = s_ / 10; u2 = u * u; u3 = u2 * u
            pts.append(tuple(0.5 * ((2 * p1[i]) + (-p0[i] + p2[i]) * u + (2 * p0[i] - 5 * p1[i] + 4 * p2[i] - p3[i]) * u2
                                    + (-p0[i] + 3 * p1[i] - 3 * p2[i] + p3[i]) * u3) for i in (0, 1)))
    pts.append(tuple(joints[-1]))
    n = len(pts); L, R = [], []
    for k, (x, y) in enumerate(pts):
        a = pts[max(k - 1, 0)]; b = pts[min(k + 1, n - 1)]
        dx, dy = b[0] - a[0], b[1] - a[1]; ln = math.hypot(dx, dy) or 1
        nx, ny = -dy / ln, dx / ln
        w = (w0 + (w1 - w0) * k / (n - 1)) / 2
        L.append((x + nx * w, y + ny * w)); R.append((x - nx * w, y - ny * w))
    poly = L + R[::-1]
    d = "M" + " L".join(f"{x:.1f} {y:.1f}" for x, y in poly) + "Z"
    return "U:" + d


def chain(sh, a1, l1, a2, l2):
    """각도: 아래쪽 0, +는 오른쪽"""
    el = (sh[0] + math.sin(a1) * l1, sh[1] + math.cos(a1) * l1)
    wr = (el[0] + math.sin(a2) * l2, el[1] + math.cos(a2) * l2)
    return el, wr


# ───────────────────────── 이모티콘 동작 (박수 · 화이팅 · 사랑해 · 뿔났어 · 신나 · 갸우뚱) ─────────────────────────
def _angry_head():
    """기본 머리 + 찌푸린 눈썹 + 빨간 핏대"""
    from PIL import ImageDraw
    h = HEADS["full"].copy()
    d = ImageDraw.Draw(h)
    for a, b in (((76, 100), (126, 120)), ((232, 100), (184, 120))):
        d.line([a, b], fill=(11, 11, 11, 255), width=9, joint="curve")
        for p in (a, b):
            d.ellipse([p[0] - 4, p[1] - 4, p[0] + 4, p[1] + 4], fill=(11, 11, 11, 255))
    cx, cy = 248, 46  # 💢
    for sx, sy in ((-1, -1), (1, -1), (-1, 1), (1, 1)):
        d.arc([cx + sx * 14 - 11, cy + sy * 14 - 11, cx + sx * 14 + 11, cy + sy * 14 + 11],
              start={(-1, -1): 0, (1, -1): 90, (-1, 1): 270, (1, 1): 180}[(sx, sy)], end={(-1, -1): 90, (1, -1): 180, (-1, 1): 360, (1, 1): 270}[(sx, sy)],
              fill=(230, 40, 50, 255), width=6)
    return h


HEADS["angry"] = _angry_head()

EMO = ("clap", "cheer", "heart", "angry", "dance", "huh")
FONT = "NanumBarunpen, 'Nanum Pen Script', 'Malgun Gothic', sans-serif"


def say_text(x, y, s, size=46, rot=0, op=1.0, col=INK):
    return (f'<text x="{x:.1f}" y="{y:.1f}" font-family="{FONT}" font-weight="bold" font-size="{size:.0f}" text-anchor="middle" '
            f'transform="rotate({rot:.1f} {x:.1f} {y:.1f})" fill="{col}" stroke="#ffffff" stroke-width="7" paint-order="stroke" '
            f'opacity="{op:.2f}">{s}</text>')


def heart_path(hx, hy, k=1.0, fill="#ff8fa3", sw=3, op=1.0):
    return (f'<path transform="translate({hx:.1f} {hy:.1f}) scale({k:.3f})" d="M0 12 C-22 -4 -12 -18 0 -6 C12 -18 22 -4 0 12Z" '
            f'fill="{fill}" stroke="{INK}" stroke-width="{sw / k:.2f}" stroke-linejoin="round" opacity="{op:.2f}"/>')


def stand_legs(cx, btm, gy, spread=54, lift=(0, 0)):
    legs = []
    for i, side in enumerate((-1, 1)):
        y = gy - lift[i]
        hip = (cx + side * 36, btm - 24)
        ft = (cx + side * spread, y)
        knee = (cx + side * (spread + 9), (hip[1] + y) / 2 + 2)
        legs.append(tarm([hip, knee, ft], 36, 24))
        legs.append(foot(ft[0] + side * 8, y + 1, side * 12, 15, 9))
    return legs


def chin_fist(cx, hb, side, nm, front, behind, hands):
    """기본 포즈 팔: 위팔은 몸 옆, 아래팔이 올라와 턱 아래 주먹"""
    sh = (cx + side * 72, hb + 22)
    el = (cx + side * 96, hb + 70)
    wr = (cx + side * 40, hb + 24)
    behind.append(tarm([sh, (cx + side * 92, hb + 46), el], 34, 30))
    front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
    front.append(tarm([el, (cx + side * 76, hb + 58), wr], 30, 22))
    hands.append((nm, wr[0] - side * 2, wr[1] - 6, side * 6))


def up_arm(cx, hb, side, nm, a1, a2, l1, l2, behind, hands, hand_rot=0.0):
    sh = (cx + side * 68, hb + 22)
    el, wr = chain(sh, a1, l1, a2, l2)
    behind.append(tarm([sh, el, wr]))
    hands.append((nm, wr[0] + math.sin(a2) * 9, wr[1] + math.cos(a2) * 12, hand_rot))


def pose_emo(kind, t):
    cx = W / 2
    ph = t * 2 * math.pi
    back, front, hands, behind = [], [], [], []
    head_kind, head_rot, head_dx = "full", 0.0, 0.0
    tilt, bob, spread, lift = 0.0, math.sin(ph) * 2, 54, (0, 0)

    if kind == "clap":
        # 가슴 앞에서 짝짝 (한 바퀴에 두 번)
        c = 0.5 + 0.5 * math.cos(ph * 2)
        gap = 16 + 46 * c
        bob = -abs(math.sin(ph * 2)) * 8
        hb = GROUND - 154 + bob
        for side, nm in ((-1, "palmL"), (1, "palmR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 100, hb + 72)
            wr = (cx + side * gap, hb + 50)
            behind.append(tarm([sh, (cx + side * 94, hb + 48), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * (gap + 36), hb + 70), wr], 30, 22))
            hands.append((nm, wr[0] + side * 2, wr[1] - 14, side * -(20 + 25 * (1 - c))))
        if c < 0.25:
            for k, a in enumerate((-0.9, -0.3, 0.3, 0.9)):
                sd = -1 if a < 0 else 1
                aa = sd * 1.3 + a * 0.6
                r0, r1 = 52, 74
                x0, y0 = cx + math.sin(aa) * r0, hb + 36 - math.cos(aa) * r0
                x1, y1 = cx + math.sin(aa) * r1, hb + 36 - math.cos(aa) * r1
                front.append(f'<path d="M{x0:.1f} {y0:.1f} L{x1:.1f} {y1:.1f}" stroke="{INK}" stroke-width="5" stroke-linecap="round"/>')
            front.append(say_text(cx + 150, hb - 190, "짝!", 52, 10))
        else:
            front.append(say_text(cx - 150, hb - 190, "짝짝", 44, -10, 0.85))

    elif kind == "cheer":
        # 주먹 불끈 쥐고 위로 쭉쭉 + "화이팅!"
        p = 0.5 + 0.5 * math.sin(ph * 2)
        bob = -p * 22
        hb = GROUND - 154 + bob
        lift = (p * 8, p * 8)
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        up_arm(cx, hb, 1, "fistR", 2.05 + p * 0.25, math.pi - 0.2 + p * 0.1, 84, 56 + p * 14, behind, hands, -8)
        front.append(say_text(cx - 118, hb - 200 - p * 10, "화이팅!", 50, -8))
        for k in range(3):  # 반짝
            q = (t * 2 + k / 3) % 1
            sx, sy = cx + 120 + k * 34, hb - 200 - k * 22
            r = 10 * math.sin(q * math.pi)
            front.append(f'<path d="M{sx:.1f} {sy - r * 1.6:.1f} L{sx + r * 0.4:.1f} {sy:.1f} L{sx:.1f} {sy + r * 1.6:.1f} L{sx - r * 0.4:.1f} {sy:.1f} Z '
                         f'M{sx - r * 1.6:.1f} {sy:.1f} L{sx:.1f} {sy - r * 0.4:.1f} L{sx + r * 1.6:.1f} {sy:.1f} L{sx:.1f} {sy + r * 0.4:.1f} Z" fill="#ffd84d" stroke="{INK}" stroke-width="2.5"/>')

    elif kind == "heart":
        # 큰 하트를 두 손으로 꼭 안고 두근두근
        sw = math.sin(ph)
        hb = GROUND - 154 + bob
        tilt = sw * 3
        beat = 1 + 0.09 * max(0.0, math.sin(ph * 2)) ** 2
        hx, hy = cx, hb + 46
        for side, nm in ((-1, "fistL"), (1, "fistR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 98, hb + 72)
            wr = (cx + side * 62, hb + 46)
            behind.append(tarm([sh, (cx + side * 94, hb + 48), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * 86, hb + 64), wr], 30, 22))
            hands.append((nm, wr[0] - side * 2, wr[1] - 2, side * 30))
        front.append(heart_path(hx, hy, 3.6 * beat, "#ff6f8a", 4.5))
        front.append(f'<path transform="translate({hx:.1f} {hy:.1f}) scale({3.6 * beat:.3f})" d="M-8 -6 Q-10 -10 -6 -11" stroke="#ffffff" stroke-width="{2.2 / (3.6 * beat):.2f}" fill="none" stroke-linecap="round"/>')
        for k in range(3):
            q = (t + k / 3) % 1
            front.append(heart_path(cx + (-140, 135, 160)[k] + math.sin(q * 6) * 6, hb - 130 - q * 80, 1.1 + q * 0.4, "#ff8fa3", 3, math.sin(q * math.pi)))
        front.append(say_text(cx + 150, hb - 222, "사랑해", 44, 8))
        head_kind, head_rot = "shy", sw * 5

    elif kind == "angry":
        # 허리에 손 올리고 발 쿵쿵 + 머리에서 김
        st = math.sin(ph * 2)
        hb = GROUND - 154 + abs(st) * -4
        lift = (max(0.0, st) * 16, max(0.0, -st) * 16)
        tilt = st * 2
        for side, nm in ((-1, "fistL"), (1, "fistR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 122, hb + 52)
            wr = (cx + side * 88, hb + 92)
            behind.append(tarm([sh, (cx + side * 104, hb + 32), el, wr], 34, 26))
            hands.append((nm, wr[0] + side * 4, wr[1] + 2, side * -40))
        for k, side in enumerate((-1, 1)):  # 김 푹푹
            q = (t * 2 + k * 0.5) % 1
            sx, sy = cx + side * (110 + q * 30), hb - 230 - q * 50
            r = 12 + q * 14
            front.append(f'<g opacity="{1 - q:.2f}"><circle cx="{sx:.1f}" cy="{sy:.1f}" r="{r:.1f}" fill="#ffffff" stroke="{INK}" stroke-width="3.5"/>'
                         f'<circle cx="{sx + side * r * 0.9:.1f}" cy="{sy + r * 0.5:.1f}" r="{r * 0.6:.1f}" fill="#ffffff" stroke="{INK}" stroke-width="3.5"/></g>')
        front.append(say_text(cx + 6, hb - 262, "흥!", 56, st * 4))
        head_kind, head_rot = "angry", st * 3

    elif kind == "dance":
        # 팔 번갈아 번쩍, 몸 흔들흔들 폴짝
        s = math.sin(ph)
        bob = -abs(math.sin(ph)) * 20
        hb = GROUND - 154 + bob
        tilt = s * 8
        lift = (max(0.0, s) * 14, max(0.0, -s) * 14)
        for side, nm, up in ((-1, "palmL", max(0.0, -s)), (1, "palmR", max(0.0, s))):
            a1 = side * (1.25 + up * 0.95)
            a2 = side * (1.75 + up * 0.95)
            up_arm(cx + tilt * 1.5, hb, side, nm, a1, a2, 58 + up * 8, 50 + up * 14, behind, hands, side * -10 - side * up * 30)
        for k in range(2):  # 음표
            q = (t + k / 2) % 1
            nx, ny = cx + (-150 if k else 150) + math.sin(q * 7) * 8, hb - 150 - q * 70
            front.append(say_text(nx, ny, "♪" if k else "♫", 54, math.sin(q * 6) * 15, math.sin(q * math.pi)))
        head_kind = "shy" if s > 0.6 else "full"
        head_rot, head_dx = -s * 9, s * 6

    elif kind == "huh":
        # 고개 갸우뚱 + 머리 긁적 + 물음표
        w = math.sin(ph)
        hb = GROUND - 154 + bob
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        sh = (cx + 70, hb + 22)
        el = (cx + 136, hb - 8)
        wr = (cx + 128 + w * 4, hb - 104 + abs(w) * 6)
        behind.append(tarm([sh, (cx + 120, hb + 22), el, wr], 34, 24))
        hands.append(("fistR", wr[0] + 2, wr[1] - 4, -150 + w * 10))
        front.append(say_text(cx - 132, hb - 196 + w * 4, "?", 78, -14 + w * 6))
        front.append(say_text(cx - 86, hb - 230 - w * 4, "?", 50, 12, 0.8))
        head_kind, head_rot, head_dx = "full", 12 + w * 2, -8

    gy = GROUND - 10
    legs = stand_legs(cx, hb + 108, gy, spread, lift)
    back = legs + behind + [torso(cx, hb - 20, hb + 108, 80, 92, tilt), glow(cx, hb - 4)]
    return back, front, hands, (head_kind, hb, head_rot, head_dx)


# ───────────────────────── 이모티콘 2탄 (브이 · 엄지척 · 노래 · 안 돼요 · 꽃 선물 · 잘 먹었어요 · 뭐해? · 꽃받침) ─────────────────────────
EMO2 = ("v", "thumb", "sing", "no", "flower", "yum", "what", "cute")


def finger(x, y, ang, ln=34, w=13):
    """손가락 하나 (ang: 위쪽 0도, +는 시계 방향). 뿌리는 주먹 그림에 가려짐"""
    return (f'<g transform="translate({x:.1f} {y:.1f}) rotate({ang:.1f})"><rect x="{-w / 2:.1f}" y="{-ln:.1f}" width="{w:.1f}" height="{ln + 8:.1f}" '
            f'rx="{w / 2:.1f}" fill="{SK}" stroke="{INK}" stroke-width="4"/></g>')


def sparkle(sx, sy, r, col="#ffd84d"):
    return (f'<path d="M{sx:.1f} {sy - r * 1.6:.1f} L{sx + r * 0.4:.1f} {sy:.1f} L{sx:.1f} {sy + r * 1.6:.1f} L{sx - r * 0.4:.1f} {sy:.1f} Z '
            f'M{sx - r * 1.6:.1f} {sy:.1f} L{sx:.1f} {sy - r * 0.4:.1f} L{sx + r * 1.6:.1f} {sy:.1f} L{sx:.1f} {sy + r * 0.4:.1f} Z" fill="{col}" stroke="{INK}" stroke-width="2.5"/>')


def raise_hand(cx, hb, side, a1, a2, l1, l2, behind):
    """팔을 올리고 손(주먹 자리) 중심을 돌려줌"""
    sh = (cx + side * 68, hb + 22)
    el, wr = chain(sh, a1, l1, a2, l2)
    behind.append(tarm([sh, el, wr]))
    return wr[0] + math.sin(a2) * 9, wr[1] + math.cos(a2) * 12


def bubble(x, y, s, rx=76, ry=42, size=40):
    """말풍선"""
    return (f'<path d="M{x - 18:.1f} {y + ry - 6:.1f} L{x + 6:.1f} {y + ry + 26:.1f} L{x + 14:.1f} {y + ry - 8:.1f} Z" fill="#fff" stroke="{INK}" stroke-width="4" stroke-linejoin="round"/>'
            f'<ellipse cx="{x:.1f}" cy="{y:.1f}" rx="{rx}" ry="{ry}" fill="#fff" stroke="{INK}" stroke-width="4"/>'
            f'<path d="M{x - 24:.1f} {y + ry - 4:.1f} L{x + 10:.1f} {y + ry - 4:.1f}" stroke="#fff" stroke-width="8"/>'
            + say_text(x, y + size * 0.35, s, size))


def hips(cx, hb, behind, hands, sides=(-1, 1)):
    """허리에 손"""
    for side in sides:
        nm = "fistL" if side < 0 else "fistR"
        sh = (cx + side * 72, hb + 22)
        el = (cx + side * 122, hb + 52)
        wr = (cx + side * 88, hb + 92)
        behind.append(tarm([sh, (cx + side * 104, hb + 32), el, wr], 34, 26))
        hands.append((nm, wr[0] + side * 4, wr[1] + 2, side * -40))


def pose_emo2(kind, t):
    cx = W / 2
    ph = t * 2 * math.pi
    front, hands, behind, bg = [], [], [], []
    head_kind, head_rot, head_dx = "full", 0.0, 0.0
    tilt, bob, lift = 0.0, math.sin(ph) * 2, (0, 0)
    s = math.sin(ph)

    if kind == "v":
        # 얼굴 옆에서 브이~ (손을 까딱까딱)
        bob = -abs(s) * 6
        hb = GROUND - 154 + bob
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        hx, hy = raise_hand(cx, hb, 1, 2.3, math.pi - 0.12, 80, 60, behind)
        w = s * 8
        front.append(finger(hx - 8, hy - 8, -16 + w, 46, 15))
        front.append(finger(hx + 9, hy - 8, 14 + w, 46, 15))
        hands.append(("fistR", hx, hy, -10 + w))
        for k in range(3):
            q = (t * 2 + k / 3) % 1
            front.append(sparkle(cx + 196 - k * 6, hy - 70 + k * 46, 10 * math.sin(q * math.pi)))
        front.append(say_text(cx - 140, hb - 206, "브이~", 50, -10))
        head_rot, head_dx = 8 + s * 2, 6

    elif kind == "thumb":
        # 엄지척! 최고!
        p = max(0.0, math.sin(ph))
        bob = -p * 10
        hb = GROUND - 154 + bob
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        hx, hy = raise_hand(cx, hb, 1, 1.85, 2.75 + p * 0.1, 84, 58, behind)
        front.append(finger(hx - 8, hy - 12, -8, 30, 19))
        hands.append(("fistR", hx, hy, -6))
        for k in range(3):
            q = (t * 2 + k / 3) % 1
            front.append(sparkle(hx + (-30, 34, 0)[k], hy - 70 - (0, 16, 46)[k], 10 * math.sin(q * math.pi)))
        front.append(say_text(cx - 124, hb - 204 - p * 6, "최고!", 54, -8))
        head_rot = -4

    elif kind == "sing":
        # 마이크 잡고 노래 ♪
        bob = math.sin(ph) * 4
        hb = GROUND - 154 + bob
        tilt = s * 4
        sh = (cx + 72, hb + 22)
        el = (cx + 98, hb + 70)
        wr = (cx + 46, hb + 18)
        behind.append(tarm([sh, (cx + 92, hb + 46), el], 34, 30))
        front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
        front.append(tarm([el, (cx + 78, hb + 52), wr], 30, 22))
        mx, my = wr[0] - 14, wr[1] - 44
        front.append(f'<path d="M{wr[0]:.1f} {wr[1]:.1f} L{mx:.1f} {my:.1f}" stroke="{INK}" stroke-width="15" stroke-linecap="round"/>'
                     f'<path d="M{wr[0]:.1f} {wr[1]:.1f} L{mx:.1f} {my:.1f}" stroke="#5b5b66" stroke-width="8" stroke-linecap="round"/>'
                     f'<circle cx="{mx:.1f}" cy="{my:.1f}" r="16" fill="#9a9aa6" stroke="{INK}" stroke-width="4"/>'
                     f'<path d="M{mx - 9:.1f} {my - 5:.1f} L{mx + 9:.1f} {my - 5:.1f} M{mx - 10:.1f} {my + 3:.1f} L{mx + 10:.1f} {my + 3:.1f}" stroke="{INK}" stroke-width="2"/>')
        hands.append(("fistR", wr[0], wr[1] - 2, 20))
        up = 0.5 + 0.5 * s
        hx, hy = raise_hand(cx, hb, -1, -1.75 - up * 0.35, -2.0 - up * 0.4, 70, 56, behind)
        hands.append(("palmL", hx, hy, 30 + up * 20))
        for k in range(3):
            q = (t + k / 3) % 1
            nx, ny = cx + (-160, 150, 120)[k] + math.sin(q * 7) * 8, hb - 140 - q * 80 - k * 10
            front.append(say_text(nx, ny, "♪♫♬"[k], 50, math.sin(q * 6) * 15, math.sin(q * math.pi)))
        head_kind = "shy" if s > 0.3 else "full"
        head_rot, head_dx = -s * 7, s * 4

    elif kind == "no":
        # 두 팔 쫙 펴고 도리도리 "안 돼요!"
        st = math.sin(ph * 2)
        hb = GROUND - 154 - abs(st) * 3
        for side, nm in ((-1, "palmL"), (1, "palmR")):
            hx, hy = raise_hand(cx, hb, side, side * 1.55, side * 1.95, 70, 56, behind)
            hands.append((nm, hx, hy, side * (-70 + st * 8)))
        front.append(say_text(cx, hb - 262, "안 돼요!", 56, st * 3, col="#d6263a"))
        head_kind, head_rot, head_dx = "sad", st * 7, st * 8

    elif kind == "flower":
        # 꽃 한 송이 선물
        sw = math.sin(ph)
        hb = GROUND - 154 + bob
        tilt = sw * 2
        for side, nm in ((-1, "fistL"), (1, "fistR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 98, hb + 72)
            wr = (cx + side * 22 + 4, hb + 58)
            behind.append(tarm([sh, (cx + side * 94, hb + 48), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * 66, hb + 74), wr], 30, 22))
            hands.append((nm, wr[0] - side * 2, wr[1] - 2, side * 30))
        fx, fy = cx + 96 + sw * 8, hb - 70
        front.append(f'<path d="M{cx + 4:.1f} {hb + 80:.1f} Q{cx + 40:.1f} {hb:.1f} {fx:.1f} {fy:.1f}" stroke="{INK}" stroke-width="11" fill="none" stroke-linecap="round"/>'
                     f'<path d="M{cx + 4:.1f} {hb + 80:.1f} Q{cx + 40:.1f} {hb:.1f} {fx:.1f} {fy:.1f}" stroke="#4caf50" stroke-width="5" fill="none" stroke-linecap="round"/>'
                     f'<path d="M{cx + 40:.1f} {hb + 8:.1f} q26 -18 44 -4 q-22 18 -44 4Z" fill="#6cc96f" stroke="{INK}" stroke-width="3.5"/>')
        rot = sw * 12
        petals = "".join(f'<ellipse cx="0" cy="-24" rx="15" ry="20" transform="rotate({a})" fill="#ffd84d" stroke="{INK}" stroke-width="3.5"/>' for a in range(0, 360, 60))
        front.append(f'<g transform="translate({fx:.1f} {fy:.1f}) rotate({rot:.1f})">{petals}<circle r="15" fill="#ff9c3c" stroke="{INK}" stroke-width="3.5"/></g>')
        for k in range(2):
            q = (t + k / 2) % 1
            front.append(heart_path(cx - 140 + k * 20, hb - 140 - q * 70 - k * 30, 1.0 + q * 0.4, "#ff8fa3", 3, math.sin(q * math.pi)))
        front.append(say_text(cx - 128, hb - 214, "선물이야", 44, -8))
        head_kind, head_rot = "shy", -6 + sw * 3

    elif kind == "yum":
        # 바나나 껍질 머리에 얹고 배 통통 "잘 먹었어요!"
        pat = max(0.0, math.sin(ph * 2))
        hb = GROUND - 154 + bob
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        sh = (cx + 72, hb + 22)
        el = (cx + 104, hb + 80)
        wr = (cx + 34, hb + 82 - pat * 10)
        behind.append(tarm([sh, (cx + 100, hb + 52), el], 34, 30))
        front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
        front.append(tarm([el, (cx + 72, hb + 92 - pat * 4), wr], 30, 22))
        hands.append(("palmR", wr[0] - 4, wr[1] - 6, -100))
        px, py = cx + 14, hb - 236  # 바나나 껍질
        peel = (f'<g transform="translate({px:.1f} {py:.1f}) rotate({s * 4:.1f})">'
                f'<path d="M-6 -2 Q-48 4 -62 46 Q-40 30 -14 12Z" fill="#ffe14d" stroke="{INK}" stroke-width="4" stroke-linejoin="round"/>'
                f'<path d="M6 -2 Q48 6 60 44 Q38 30 14 12Z" fill="#ffe14d" stroke="{INK}" stroke-width="4" stroke-linejoin="round"/>'
                f'<path d="M-4 4 Q-6 34 -18 54 Q4 40 8 6Z" fill="#ffd21f" stroke="{INK}" stroke-width="4" stroke-linejoin="round"/>'
                f'<path d="M-12 -2 Q-4 -30 0 -40 L10 -38 Q8 -20 12 -2Z" fill="#ffe14d" stroke="{INK}" stroke-width="4" stroke-linejoin="round"/>'
                f'<path d="M-2 -42 L10 -40 L9 -32 L-1 -34Z" fill="#6b4a2a" stroke="{INK}" stroke-width="3"/></g>')
        front.append(peel)
        if pat > 0.6:
            front.append(say_text(cx + 120, hb + 70, "통통", 34, 10, 0.9))
        front.append(say_text(cx - 104, hb - 254, "잘 먹었어요!", 48, -6))
        head_kind, head_rot = ("eat1" if s > 0 else "full"), s * 3

    elif kind == "what":
        # 빼꼼 "뭐해?" 말풍선 + 손 흔들
        w = math.sin(ph * 2)
        hb = GROUND - 154 + bob
        chin_fist(cx, hb, -1, "fistL", front, behind, hands)
        hx, hy = raise_hand(cx, hb, 1, 2.2 + w * 0.1, math.pi - 0.25 + w * 0.35, 78, 56, behind)
        hands.append(("palmR", hx, hy, -10 + w * 22))
        front.append(bubble(cx - 128, hb - 236 + s * 3, "뭐해?", 78, 42, 42))
        head_rot, head_dx = -10 + s * 2, 8

    elif kind == "cute":
        # 꽃받침 + 분홍 반짝 배경
        bob = -abs(s) * 8
        hb = GROUND - 154 + bob
        for side, nm in ((-1, "palmL"), (1, "palmR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 100, hb + 70)
            wr = (cx + side * 66, hb + 4)
            behind.append(tarm([sh, (cx + side * 96, hb + 46), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * 88, hb + 40), wr], 30, 22))
            hands.append((nm, wr[0] - side * 4, wr[1] - 20, side * -28))
        bg.append(f'<ellipse cx="{cx:.1f}" cy="{hb - 60:.1f}" rx="{196 + s * 8:.1f}" ry="{220 + s * 8:.1f}" fill="#ffc2d6" opacity="0.75" filter="url(#glow)"/>')
        for k in range(4):
            q = (t * 2 + k / 4) % 1
            a = k * 1.6 + 0.4
            front.append(sparkle(cx + math.cos(a) * 190, hb - 110 + math.sin(a) * 150, 11 * math.sin(q * math.pi), "#fff4a8"))
        head_kind, head_rot = ("shy" if s > -0.2 else "full"), s * 6

    gy = GROUND - 10
    legs = stand_legs(cx, hb + 108, gy, 54, lift)
    back = ["B:" + b for b in bg] + legs + behind + [torso(cx, hb - 20, hb + 108, 80, 92, tilt), glow(cx, hb - 4)]
    return back, front, hands, (head_kind, hb, head_rot, head_dx)


def pose(kind, t):
    cx = W / 2
    ph = t * 2 * math.pi
    back, front, hands, behind = [], [], [], []
    head_kind, head_rot, head_dx = "full", 0.0, 0.0
    if kind in EMO:
        return pose_emo(kind, t)
    if kind in EMO2:
        return pose_emo2(kind, t)
    if kind == "eat":
        return pose_eat(t)
    if kind == "eat_milk":
        return pose_eat(t, "milk")
    if kind == "eat_cake":
        return pose_eat(t, "cake")
    if kind in ("idle", "blink", "walk", "laugh", "shy", "punish", "wave"):
        if kind == "walk":
            bob = -abs(math.sin(ph)) * 13
        elif kind == "laugh":
            bob = -abs(math.sin(ph)) * 40
        else:
            bob = math.sin(ph) * 2
        if kind == "shy":
            bob = math.sin(ph) * 2
        if kind == "punish":
            bob = math.sin(ph * 3) * 1.5
        tilt = (5 + math.sin(ph) * 2.5) if kind == "walk" else (math.sin(ph) * 5 if kind == "shy" else 0)
        hb = GROUND - 50 - 104 + bob
        btm = hb + 108
        legs = []
        if kind == "walk":
            for side, sgn in ((-1, 1), (1, -1)):
                lift = max(0, sgn * math.sin(ph)) * 20
                swing = sgn * math.cos(ph) * 14
                hip = (cx + side * 38, btm - 24)
                ft = (cx + side * 50 + swing, GROUND - 9 - lift)
                knee = (cx + side * (60 + lift * 0.6) + swing * 0.5, (hip[1] + ft[1]) / 2 - lift * 0.35)
                legs.append(tarm([hip, knee, ft], 36, 24))
                legs.append(foot(ft[0] + side * 7, ft[1] + 1, side * 10 - swing * 0.8, 15, 9))
        else:
            gy = GROUND - 10 + (bob if kind == "laugh" else 0)
            spread = 64 if kind == "laugh" else (40 if kind == "shy" else 54)
            for side in (-1, 1):
                hip = (cx + side * 36, btm - 24)
                ft = (cx + side * spread, gy)
                knee = (cx + side * (spread + 9), (hip[1] + gy) / 2 + 2)
                legs.append(tarm([hip, knee, ft], 36, 24))
                legs.append(foot(ft[0] + side * 8, gy + 1, side * 12, 15, 9))
        if kind == "walk":
            # 어깨에서 흔들리고, 팔이 앞으로 올 때 팔꿈치가 굽음
            for side, sgn, nm in ((-1, 1, "fistL"), (1, -1, "fistR")):
                f = sgn * math.sin(ph)
                sh = (cx + side * 74, hb + 20)
                a1 = side * (0.62 + 0.18 * f)
                a2 = a1 - side * (0.22 + 0.5 * max(0.0, f))
                l1 = 52 * (1 - 0.10 * max(0.0, f))
                el, wr = chain(sh, a1, l1, a2, 48)
                behind.append(tarm([sh, el, wr]))
                hands.append((nm, wr[0] + math.sin(a2) * 9, wr[1] + math.cos(a2) * 9, -math.degrees(a2) * 0.7))
            # 영상처럼 몸을 앞으로 기울이고 머리가 통통 따라옴
            head_rot = -5 + math.sin(ph * 2 - 0.8) * 2.5
            head_dx = 10
        elif kind == "wave":
            # 한 손은 턱 아래, 한 손은 얼굴 옆에서 흔들흔들
            wv = math.sin(ph * 2)
            sh = (cx - 72, hb + 22); el = (cx - 96, hb + 70); wr = (cx - 40, hb + 24)
            behind.append(tarm([sh, (cx - 92, hb + 46), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx - 76, hb + 58), wr], 30, 22))
            hands.append(("fistL", wr[0] + 2, wr[1] - 6, -6))
            sh = (cx + 70, hb + 22)
            a1 = math.pi - 0.95
            a2 = math.pi - 0.25 + wv * 0.28
            el, wr = chain(sh, a1, 62, a2, 66)
            behind.append(tarm([sh, el, wr]))
            hands.append(("palmR", wr[0] + math.sin(a2) * 10, wr[1] - 12, -14 - wv * 16))
            head_rot = math.sin(ph) * 3
            head_kind = "shy" if (t * 8) % 8 >= 6 else "full"
        elif kind == "laugh":
            for side, nm in ((-1, "palmL"), (1, "palmR")):
                wv = math.sin(ph * 2 + (0 if side < 0 else math.pi)) * 0.12
                sh = (cx + side * 66, hb + 22)
                a1 = side * (math.pi - 0.85 + wv)
                a2 = side * (math.pi - 0.45 + wv * 1.6)
                el, wr = chain(sh, a1, 54, a2, 48)
                behind.append(tarm([sh, el, wr]))
                hands.append((nm, wr[0] + math.sin(a2) * 10, wr[1] + math.cos(a2) * 14, side * -10 + math.degrees(wv) * side))
        elif kind == "shy":
            # 두 주먹을 볼에 대고 몸을 배배 꼼
            sw = math.sin(ph)
            for side, nm in ((-1, "fistL"), (1, "fistR")):
                sh = (cx + side * 72, hb + 22)
                el = (cx + side * 100 + sw * 4, hb + 56)
                wr = (cx + side * 92 + sw * 8, hb - 52)
                behind.append(tarm([sh, (cx + side * 96, hb + 40), el], 34, 30))
                front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
                front.append(tarm([el, (cx + side * 106 + sw * 6, hb + 6), wr], 30, 22))
                hands.append((nm, wr[0] - side * 4, wr[1] - 6, side * -20))
            for k in range(2):  # 둥실 하트
                p = (t + k * 0.5) % 1
                hx, hy = cx + (130 if k else -135) + math.sin(p * 6) * 6, hb - 150 - p * 70
                a = math.sin(p * math.pi)
                front.append(f'<path d="M{hx:.1f} {hy+12:.1f} C{hx-22:.1f} {hy-4:.1f} {hx-12:.1f} {hy-18:.1f} {hx:.1f} {hy-6:.1f} C{hx+12:.1f} {hy-18:.1f} {hx+22:.1f} {hy-4:.1f} {hx:.1f} {hy+12:.1f}Z" fill="#ff8fa3" stroke="#0b0b0b" stroke-width="3" opacity="{a:.2f}"/>')
            head_kind, head_rot = "shy", sw * 6
        elif kind == "punish":
            # 두 팔 번쩍 들고 벌서기 (팔이 바들바들)
            tr = math.sin(ph * 6) * 0.03
            for side, nm in ((-1, "palmL"), (1, "palmR")):
                sh = (cx + side * 70, hb + 22)
                a1 = side * (math.pi - 0.5 + tr)
                a2 = side * (math.pi - 0.16 + tr * 1.5)
                el, wr = chain(sh, a1, 130, a2, 140)
                behind.append(tarm([sh, el, wr], 34, 24))
                hands.append((nm, wr[0] + math.sin(a2) * 8, wr[1] - 14, side * -4 + math.degrees(tr) * 4))
            for k, (dx, dy) in enumerate(((-150, -120), (150, -90))):  # 식은땀
                p = (t + k * 0.4) % 1
                sx, sy = cx + dx, hb + dy + p * 26
                front.append(f'<path d="M{sx:.1f} {sy-14:.1f} Q{sx+11:.1f} {sy+2:.1f} {sx:.1f} {sy+8:.1f} Q{sx-11:.1f} {sy+2:.1f} {sx:.1f} {sy-14:.1f}Z" fill="#8fd3ff" stroke="#0b0b0b" stroke-width="3" opacity="{1-p*0.6:.2f}"/>')
            head_kind, head_rot = "sad", math.sin(ph * 6) * 0.8
        else:
            # 원본 포즈: 위팔은 몸 옆으로(뒤), 아래팔이 배 앞으로 올라와 턱 아래 주먹
            for side, nm in ((-1, "fistL"), (1, "fistR")):
                sh = (cx + side * 72, hb + 22)
                el = (cx + side * 96, hb + 70)
                wr = (cx + side * 40, hb + 24)
                behind.append(tarm([sh, (cx + side * 92, hb + 46), el], 34, 30))
                front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
                front.append(tarm([el, (cx + side * 76, hb + 58), wr], 30, 22))
                hands.append((nm, wr[0] - side * 2, wr[1] - 6, side * 6))
            head_kind = "blink" if kind == "blink" else "full"
        back = legs + behind + [torso(cx, hb - 20, btm, 80, 92, tilt), glow(cx, hb - 4)]
        return back, front, hands, (head_kind, hb, head_rot, head_dx)
    # 앉기 / 자기
    br = math.sin(ph)
    hb = GROUND - 118 + br * (1.5 if kind != "sleep" else 2.5)
    btm = GROUND - 8
    legs = []
    for side in (-1, 1):
        hip = (cx + side * 52, btm - 14)
        ft = (cx + side * 104, GROUND - 14)
        legs.append(tarm([hip, (cx + side * 82, btm - 2), ft], 38, 26))
        legs.append(foot(ft[0] + side * 5, ft[1] - 3, side * -62, 15, 10))
    if kind == "sleep":
        for side, nm in ((-1, "fistL"), (1, "fistR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 100, hb + 66)
            wr = (cx + side * 42, hb + 80)
            behind.append(tarm([sh, (cx + side * 94, hb + 44), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * 78, hb + 80), wr], 30, 22))
            hands.append((nm, wr[0] - side * 4, wr[1], side * 20))
        head_kind, head_rot, head_dx = "sleep", -10 + br * 1.2, -6
    else:
        for side, nm in ((-1, "fistL"), (1, "fistR")):
            sh = (cx + side * 72, hb + 22)
            el = (cx + side * 98, hb + 68)
            wr = (cx + side * 40, hb + 24)
            behind.append(tarm([sh, (cx + side * 94, hb + 44), el], 34, 30))
            front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
            front.append(tarm([el, (cx + side * 76, hb + 58), wr], 30, 22))
            hands.append((nm, wr[0] - side * 2, wr[1] - 6, side * 6))
        head_kind = "blink" if kind == "sitblink" else "full"
    back = behind + [torso(cx, hb - 20, btm, 80, 100)] + legs + [glow(cx - (6 if kind == "sleep" else 0), hb - 4)]
    return back, front, hands, (head_kind, hb, head_rot, head_dx)


def pose_eat(t, food="banana"):
    """앉아서 바나나 냠냠"""
    cx = W / 2
    ph = t * 2 * math.pi
    back, front, hands, behind = [], [], [], []
    chew = math.sin(ph * 2)
    hb = GROUND - 118 + chew * 1.5
    btm = GROUND - 8
    legs = []
    for side in (-1, 1):
        hip = (cx + side * 52, btm - 14)
        ft = (cx + side * 104, GROUND - 14)
        legs.append(tarm([hip, (cx + side * 82, btm - 2), ft], 38, 26))
        legs.append(foot(ft[0] + side * 5, ft[1] - 3, side * -62 + chew * 4, 15, 10))
    lift = (math.sin(ph) + 1) / 2          # 바나나를 입으로 가져갔다 내렸다
    by = hb - 8 - lift * 14
    for side, nm in ((-1, "fistL"), (1, "fistR")):
        sh = (cx + side * 72, hb + 22)
        el = (cx + side * 98, hb + 64)
        wr = (cx + side * 56, by + 14 - side * 8)
        behind.append(tarm([sh, (cx + side * 94, hb + 44), el], 34, 30))
        front.append(f'E:{el[0]:.1f},{el[1]:.1f}')
        front.append(tarm([el, (cx + side * 74, hb + 40), wr], 30, 22))
    bx = cx
    if food == "banana":
        # 바나나 (손 사이)
        front.append(f'<g transform="rotate({-14 + chew * 3:.1f} {bx:.1f} {by:.1f}) translate({bx:.1f} {by:.1f}) scale(1.55) translate({-bx:.1f} {-by:.1f})">'
                     f'<path d="M{bx-46:.1f} {by+6:.1f} Q{bx:.1f} {by+34:.1f} {bx+46:.1f} {by-4:.1f} Q{bx+50:.1f} {by-12:.1f} {bx+44:.1f} {by-14:.1f} Q{bx:.1f} {by+14:.1f} {bx-44:.1f} {by-6:.1f}Z" '
                     f'fill="#ffd84d" stroke="#0b0b0b" stroke-width="4" stroke-linejoin="round"/>'
                     f'<path d="M{bx+44:.1f} {by-14:.1f} l8 -8" stroke="#0b0b0b" stroke-width="5" stroke-linecap="round"/>'
                     f'<path d="M{bx-44:.1f} {by-6:.1f} q-10 -6 -6 -14 M{bx-46:.1f} {by+6:.1f} q-14 4 -16 -6" stroke="#0b0b0b" stroke-width="4" fill="#fff1a8"/></g>')
    elif food == "milk":
        # 노란 우유팩 + 빨대 (빨대 끝이 입으로)
        top = by + 2
        w, h = 84, 96
        x0 = bx - w / 2
        front.append(f'<g transform="rotate({chew * 2:.1f} {bx:.1f} {top + h/2:.1f})">'
                     f'<path d="M{bx+6:.1f} {top-6:.1f} L{bx-4:.1f} {hb-46:.1f}" stroke="#0b0b0b" stroke-width="13" stroke-linecap="round"/>'
                     f'<path d="M{bx+6:.1f} {top-6:.1f} L{bx-4:.1f} {hb-46:.1f}" stroke="#ff7a8a" stroke-width="6" stroke-linecap="round"/>'
                     f'<path d="M{x0:.1f} {top+10:.1f} L{bx:.1f} {top-16:.1f} L{x0+w:.1f} {top+10:.1f} Z" fill="#fff6d6" stroke="#0b0b0b" stroke-width="4" stroke-linejoin="round"/>'
                     f'<rect x="{x0:.1f}" y="{top+10:.1f}" width="{w}" height="{h}" rx="6" fill="#ffe04d" stroke="#0b0b0b" stroke-width="4"/>'
                     f'<rect x="{x0+10:.1f}" y="{top+30:.1f}" width="{w-20}" height="{h-42}" rx="9" fill="#fff8d8" stroke="#0b0b0b" stroke-width="3"/>'
                     f'<path d="M{bx-22:.1f} {top+58:.1f} Q{bx:.1f} {top+84:.1f} {bx+22:.1f} {top+50:.1f} Q{bx+4:.1f} {top+68:.1f} {bx-21:.1f} {top+52:.1f}Z" fill="#ffd84d" stroke="#0b0b0b" stroke-width="3" stroke-linejoin="round"/>'
                     f'</g>')
    else:
        # 바나나 케이크 한 조각 (크림 층 + 바나나 슬라이스)
        y0 = by - 2
        L, R = bx - 58, bx + 58
        front.append(f'<g transform="rotate({-6 + chew * 3:.1f} {bx:.1f} {y0+20:.1f})">'
                     f'<path d="M{L:.1f} {y0+44:.1f} L{R:.1f} {y0+44:.1f} L{R:.1f} {y0+4:.1f} L{L+6:.1f} {y0-6:.1f} Z" fill="#f6c36b" stroke="#0b0b0b" stroke-width="4" stroke-linejoin="round"/>'
                     f'<path d="M{L+2:.1f} {y0+20:.1f} L{R:.1f} {y0+22:.1f}" stroke="#fff4dd" stroke-width="9"/>'
                     f'<path d="M{L+4:.1f} {y0-6:.1f} Q{bx-30:.1f} {y0-20:.1f} {bx:.1f} {y0-14:.1f} Q{bx+30:.1f} {y0-22:.1f} {R+2:.1f} {y0+2:.1f} L{R:.1f} {y0+8:.1f} Q{bx+20:.1f} {y0+2:.1f} {bx:.1f} {y0+6:.1f} Q{bx-26:.1f} {y0-2:.1f} {L+4:.1f} {y0+4:.1f} Z" fill="#fffaf0" stroke="#0b0b0b" stroke-width="4" stroke-linejoin="round"/>'
                     + "".join(f'<ellipse cx="{bx-28+k*28:.1f}" cy="{y0-16-(k%2)*4:.1f}" rx="12" ry="8" fill="#fff1a8" stroke="#0b0b0b" stroke-width="3"/>' for k in range(3))
                     + f'<circle cx="{bx+30:.1f}" cy="{y0-28:.1f}" r="7" fill="#ff5a6a" stroke="#0b0b0b" stroke-width="3"/>'
                     f'</g>')
    for side, nm in ((-1, "fistL"), (1, "fistR")):
        hands.append((nm, cx + side * 58, by + 10 - side * 8, side * 10))
    if food == "milk":
        # 꿀꺽꿀꺽 하트
        for k in range(2):
            p = (t * 2 + k / 2) % 1
            front.append(f'<path transform="translate({cx + 70 + k * 16:.1f} {hb - 40 - p * 60:.1f}) scale({0.8 + p * 0.4:.2f})" d="M0 6 C-12 -6 -6 -14 0 -6 C6 -14 12 -6 0 6Z" fill="#ff7a8a" stroke="#0b0b0b" stroke-width="2.5" opacity="{1 - p:.2f}"/>')
    else:
        col = "#ffe27a" if food == "banana" else "#f6c36b"
        for k in range(3):
            p = (t * 2 + k / 3) % 1
            front.append(f'<circle cx="{cx - 30 + k * 30 + math.sin(p * 9) * 6:.1f}" cy="{hb + 20 + p * 70:.1f}" r="4" fill="{col}" stroke="#0b0b0b" stroke-width="2" opacity="{1 - p:.2f}"/>')
    head_kind = "eat1" if chew > 0 else "eat0"
    back = behind + [torso(cx, hb - 20, btm, 80, 100)] + legs + [glow(cx, hb - 4)]
    return back, front, hands, (head_kind, hb, chew * 1.5, 0.0)


def zzz(img, t):
    from PIL import ImageDraw, ImageFont
    d = ImageDraw.Draw(img)
    try:
        f = lambda s: ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", int(s))
    except Exception:
        f = lambda s: ImageFont.load_default()
    for k in range(3):
        p = (t + k / 3) % 1
        x, y = 360 + p * 70, 150 - p * 110
        a = int(255 * math.sin(p * math.pi))
        d.text((x, y), "Z", font=f(30 + p * 30), fill=(70, 110, 210, a), stroke_width=4, stroke_fill=(255, 255, 255, a))


import numpy as np, cv2


def _shot(page, inner):
    page.set_content(f'<html><body style="margin:0;background:transparent"><svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}">{DEF}{inner}</svg></body></html>')
    return Image.open(io.BytesIO(page.locator("svg").screenshot(omit_background=True))).convert("RGBA")


def hand_line(mask, erase=()):
    """덩어리 바깥 테두리를 손그림처럼 굵기가 살짝 변하고 군데군데 끊긴 선으로"""
    cs, _ = cv2.findContours(mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_NONE)
    line = np.zeros(mask.shape, np.uint8)
    for c in cs:
        pts = c[:, 0, :]
        for k in range(0, len(pts), 2):
            x, y = int(pts[k][0]), int(pts[k][1])
            g = math.sin(x / 19.0) + math.cos(y / 27.0) + math.sin((x + y) / 41.0)
            if g > 2.1:          # 끊김
                continue
            if any((x - ex) ** 2 + (y - ey) ** 2 < er * er for ex, ey, er in erase):
                continue
            r = 3.9 + 0.6 * math.sin(x / 13.0 + y / 17.0)
            cv2.circle(line, (x, y), int(round(r)), 255, -1, cv2.LINE_AA)
    lay = np.zeros(mask.shape + (4,), np.uint8)
    lay[..., :3] = 11
    lay[..., 3] = line
    return Image.fromarray(lay, "RGBA")


def blob(page, parts, erase=()):
    us = [p[2:] for p in parts if p.startswith("U:")]
    rest = [p for p in parts if not p.startswith("U:")]
    img = Image.new("RGBA", (W, H))
    if us:
        fill = _shot(page, '<g filter="url(#w)">' + "".join(f'<path d="{d}" fill="{SK}"/>' for d in us) + "</g>")
        m = (np.asarray(fill)[..., 3] > 127).astype(np.uint8) * 255
        img.alpha_composite(fill)
        img.alpha_composite(hand_line(m, erase))
    if rest:
        img.alpha_composite(_shot(page, '<g filter="url(#w)">' + "".join(rest) + "</g>"))
    return img


def make(page, kind, t):
    back, front, hands, (hk, hb, hrot, hdx) = pose(kind, t)
    bgs = [p[2:] for p in back if p.startswith("B:")]  # 몸 뒤 배경 (분홍 반짝 등)
    img = blob(page, bgs) if bgs else Image.new("RGBA", (W, H))
    img.alpha_composite(blob(page, [p for p in back if not p.startswith("B:")]))
    head = HEADS[hk]
    pad = 60
    hc = Image.new("RGBA", (head.width + pad * 2, head.height + pad * 2))
    hc.alpha_composite(head, (pad, pad))
    hc = hc.rotate(hrot, resample=Image.BICUBIC, center=(pad + head.width / 2, pad + head.height))
    img.alpha_composite(hc, (int(W / 2 - head.width / 2 - pad + hdx), int(hb - head.height - pad)))
    if front:
        # 몸 앞으로 오는 팔: 팔꿈치 쪽 선은 비워서 위팔과 이어지게
        elbows = [tuple(map(float, p[2:].split(","))) for p in front if p.startswith("E:")]
        fr = [p for p in front if not p.startswith("E:")]
        img.alpha_composite(blob(page, fr, [(x, y, 22) for x, y in elbows]))
    for nm, x, y, rot in hands:
        hnd = HANDS[nm].rotate(rot, resample=Image.BICUBIC, expand=True)
        img.alpha_composite(hnd, (int(x - hnd.width / 2), int(y - hnd.height / 2)))
    if kind == "sleep":
        zzz(img, t)
    return img


SEQ = {"wave": 8, "eat": 8, "eat_milk": 8, "eat_cake": 8, "shy": 8, "punish": 6, "walk": 8, "idle": 6, "blink": 1, "laugh": 8, "sit": 6, "sitblink": 1, "sleep": 12,
       "clap": 8, "cheer": 8, "heart": 8, "angry": 8, "dance": 8, "huh": 8,
       "v": 8, "thumb": 8, "sing": 8, "no": 8, "flower": 8, "yum": 8, "what": 8, "cute": 8}

if __name__ == "__main__":
    import sys
    out = sys.argv[1] if len(sys.argv) > 1 else str(ROOT / "frames")
    os.makedirs(out, exist_ok=True)
    with sync_playwright() as p:
        b = p.chromium.launch()
        pg = b.new_page(viewport={"width": W, "height": H})
        only = sys.argv[2:]  # 이름을 주면 그 동작만 (예: python tools/compose.py frames clap heart)
        for name, n in SEQ.items():
            if only and name not in only:
                continue
            for k in range(n):
                make(pg, name, k / n).save(f"{out}/{name}_{k}.png")
    print("compose: done ->", out)
