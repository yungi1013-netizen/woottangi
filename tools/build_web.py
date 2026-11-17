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


def main():
    F = load_frames()
    tpl = (ROOT / "web" / "template.html").read_text(encoding="utf-8")
    out = ROOT / "web" / "woottangi_mobile.html"
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


if __name__ == "__main__":
    main()
