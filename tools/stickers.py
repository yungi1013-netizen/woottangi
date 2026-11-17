"""우땅이 스티커 동작 프레임 만들기

assets/stickers/*.png (고해상도로 키운 이모티콘 스티커)를
480x600 캔버스 바닥(GROUND)에 세우고 8프레임 짜리 짧은 움직임을 줍니다.

  surprise  놀람   : 움찔했다가 펄쩍 뛰고 착지   (클릭 시 30%)
  lounge    뒹굴   : 숨쉬듯 들썩                  (가끔 랜덤)
  cry       울기   : 흑흑 위아래로 들썩           (혼낼 때 → 벌서기)
  me        저요   : 손 들고 좌우로 흔들          (클릭 시)
  bow       인사   : 앞으로 꾸벅                  (시작·종료 시)

    python tools/stickers.py            # frames/st_{이름}_{0..7}.png
"""
import math
import sys
from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "assets" / "stickers"
W, H, GROUND = 480, 600, 588
N = 8

# 스티커별 크기 배율 (원본 머리 크기와 맞춤)
SCALE = {"surprise": 0.944, "lounge": 0.90, "cry": 0.87, "me": 0.84, "bow": 0.79}


def motion(name, k):
    """k번째 프레임의 (dx, dy, 가로배율, 세로배율, 회전각) — 회전 중심은 발 아래"""
    t = k / N
    if name == "surprise":
        jump = [0, 40, 53, 40, 0, 0, 0, 0][k]
        sx, sy = (1.06, 0.94) if k in (0, 7) else (1, 1)
        return 0, -jump, sx, sy, 0
    if name == "lounge":
        b = math.sin(t * 2 * math.pi)
        return 0, 0, 1 - 0.01 * b, 1 + 0.015 * b, 0
    if name == "cry":
        return 0, -5 * (k % 2), 1, 1, 0
    if name == "me":
        sway = math.sin(t * 2 * math.pi)
        hop = abs(math.sin(t * 2 * math.pi)) * 18
        return 0, -hop, 1, 1, 4 * sway
    if name == "bow":
        return 0, 0, 1, 1, -12 * math.sin(t * math.pi)   # 0 → 꾸벅 → 0
    return 0, 0, 1, 1, 0


def frame(src, name, k):
    s = SCALE.get(name, 0.85)
    dx, dy, sx, sy, rot = motion(name, k)
    w, h = round(src.width * s * sx), round(src.height * s * sy)
    im = src.resize((w, h), Image.LANCZOS)
    # 발 아래를 중심으로 회전하도록 여백을 둔 판에 올려서 돌림
    pad = Image.new("RGBA", (w * 2, h * 2), (0, 0, 0, 0))
    pad.alpha_composite(im, (w // 2, 0))
    if rot:
        pad = pad.rotate(rot, resample=Image.BICUBIC, center=(w, h))
    out = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    out.alpha_composite(pad, (round(W / 2 - w + dx), round(GROUND - h + dy)))
    return out


def main(out_dir):
    out_dir = Path(out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    for p in sorted(SRC.glob("*.png")):
        src = Image.open(p).convert("RGBA")
        for k in range(N):
            frame(src, p.stem, k).save(out_dir / f"st_{p.stem}_{k}.png")
    print("stickers: done ->", out_dir)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else ROOT / "frames")
