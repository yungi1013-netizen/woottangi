"""프레임 → 웹 결과물 만들기

1) web/template.html  + 프레임(WebP 240x300) → web/woottangi_mobile.html  (휴대폰용 한 파일)
2) pc/game_template.html + 프레임            → pc/game.html             (PC 바나나 받기 게임, exe에 내장)

    python tools/build_web.py
"""
import base64
import io
import json
import re
from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
FRAMES = ROOT / "frames"


def webp_uri(path, size=(240, 300), q=82):
    im = Image.open(path).convert("RGBA").resize(size, Image.LANCZOS)
    b = io.BytesIO()
    im.save(b, "WEBP", quality=q)
    return "data:image/webp;base64," + base64.b64encode(b.getvalue()).decode()


def load_frames():
    groups = {}
    for p in FRAMES.glob("*.png"):
        m = re.fullmatch(r"(.+)_(\d+)", p.stem)
        if m:
            groups.setdefault(m.group(1), []).append((int(m.group(2)), p))
    return {k: [webp_uri(p) for _, p in sorted(v)] for k, v in sorted(groups.items())}


def webp_img(im, q=85):
    b = io.BytesIO()
    im.save(b, "WEBP", quality=q)
    return "data:image/webp;base64," + base64.b64encode(b.getvalue()).decode()


def accessory_data():
    """휴대폰용 꾸미기: 아이템 그림(머리 좌표계 400x400) + 프레임별 머리 위치 + 도감용 미리보기"""
    import sys
    sys.path.insert(0, str(ROOT / "tools"))
    import accessories as A
    anchors = A.anchors()
    acc, prev = {}, {}
    base = Image.open(FRAMES / "idle_0.png").convert("RGBA")
    for k in A.ITEMS:
        im = Image.open(A.OUT / f"{k}.png").convert("RGBA")
        acc[k] = webp_img(im)
        f = base.copy()
        A.put(f, im, anchors["idle"][0])
        prev[k] = webp_img(f.crop((40, 60, 440, 600)).resize((200, 270), Image.LANCZOS))
    prev["none"] = webp_img(base.crop((40, 60, 440, 600)).resize((200, 270), Image.LANCZOS))
    anc = {n: [[round(v, 2) for v in a] for a in lst] for n, lst in anchors.items()}
    return acc, anc, sorted(A.NO_FACE), prev


def fx_data():
    """계절 날씨 조각 · 목욕 거품 · 이불 (tools/effects.py 가 만든 pc/fx/*.png)"""
    out = {}
    for f in sorted((ROOT / "pc" / "fx").glob("*.png")):
        im = Image.open(f).convert("RGBA")
        if f.stem == "blanket":
            im = im.resize((240, 300), Image.LANCZOS)
        out[f.stem] = webp_img(im)
    return out


def main():
    F = load_frames()
    acc, anc, noface, prev = accessory_data()
    tpl = (ROOT / "web" / "template.html").read_text(encoding="utf-8")
    out = ROOT / "web" / "woottangi_mobile.html"
    tpl = (tpl.replace("__ACC__", json.dumps(acc)).replace("__ANCHORS__", json.dumps(anc))
           .replace("__NOFACE__", json.dumps(noface)).replace("__ITEMPREV__", json.dumps(prev))
           .replace("__FX__", json.dumps(fx_data())))
    out.write_text(tpl.replace("__FRAMES__", json.dumps(F)), encoding="utf-8")
    print("web:", out, f"{out.stat().st_size // 1024} KB")

    g = (ROOT / "pc" / "game_template.html").read_text(encoding="utf-8")
    g = g.replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
    (ROOT / "pc" / "game.html").write_text(g, encoding="utf-8")
    print("game: pc/game.html")

    # 짝 맞추기: 동작 8가지를 카드로
    poses = [("idle", 0), ("laugh", 2), ("sit", 0), ("sleep", 3), ("eat", 2), ("shy", 2), ("wave", 2), ("punish", 0)]
    cards = [F[k][i] for k, i in poses]
    m = (ROOT / "pc" / "match_template.html").read_text(encoding="utf-8")
    m = m.replace("__CARDS__", json.dumps(cards)).replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
    (ROOT / "pc" / "match.html").write_text(m, encoding="utf-8")
    print("match: pc/match.html")

    t = (ROOT / "pc" / "tower_template.html").read_text(encoding="utf-8")
    t = t.replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
    (ROOT / "pc" / "tower.html").write_text(t, encoding="utf-8")
    print("tower: pc/tower.html")

    (ROOT / "pc" / "whack.html").write_text(whack_html(F), encoding="utf-8")
    print("whack: pc/whack.html")


def whack_html(F):
    """두더지 잡기: 쏙 나오는 그림(손 흔들기) · 맞은 그림(깜짝)"""
    t = (ROOT / "pc" / "whack_template.html").read_text(encoding="utf-8")
    return (t.replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
            .replace("__POP__", F["laugh"][2]).replace("__HIT__", F["st_surprise"][0]))


if __name__ == "__main__":
    main()
