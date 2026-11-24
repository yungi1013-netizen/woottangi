"""우땅이 이모티콘 GIF 만들기

frames/ 의 이모티콘 동작을 메신저에 보내기 좋은 움직이는 GIF 로 저장합니다.
(흰 배경, 240x240, 반복 재생)

    python tools/emo_gif.py            # pc/emo/*.gif
"""
import sys
from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
FRAMES = ROOT / "frames"
OUT = ROOT / "pc" / "emo"
SIZE = 240

# (동작, 파일 이름, 초당 프레임) — pc/emo.go 의 emos 와 같은 순서
EMOS = [("clap", "박수", 10), ("cheer", "화이팅", 9), ("heart", "사랑해", 7), ("angry", "흥", 9), ("dance", "신나는춤", 9),
        ("huh", "갸우뚱", 6), ("st_surprise", "깜짝", 9), ("st_me", "저요", 9), ("st_lounge", "뒹굴뒹굴", 4), ("st_bow", "꾸벅", 5),
        ("laugh", "신나", 12), ("shy", "부끄", 7), ("wave", "안녕", 8), ("sleep", "쿨쿨", 6), ("st_cry", "엉엉", 9),
        ("v", "브이", 8), ("thumb", "엄지척", 8), ("sing", "노래", 8), ("no", "안돼요", 10), ("flower", "꽃선물", 7),
        ("yum", "잘먹었어요", 8), ("what", "뭐해", 8), ("cute", "꽃받침", 7)]


def frames_of(seq):
    out, k = [], 0
    while (FRAMES / f"{seq}_{k}.png").exists():
        out.append(Image.open(FRAMES / f"{seq}_{k}.png").convert("RGBA"))
        k += 1
    return out


def make(seq, fps):
    fr = frames_of(seq)
    if not fr:
        raise SystemExit(f"frames/{seq}_0.png 가 없어요")
    # 모든 프레임을 덮는 상자로 잘라 가운데 정사각형에 놓기 (움직여도 안 잘리게)
    box = None
    for f in fr:
        b = f.getbbox()
        box = b if box is None else (min(box[0], b[0]), min(box[1], b[1]), max(box[2], b[2]), max(box[3], b[3]))
    w, h = box[2] - box[0], box[3] - box[1]
    side = int(max(w, h) * 1.06)
    k = SIZE / side
    out = []
    for f in fr:
        c = Image.new("RGBA", (side, side), (255, 255, 255, 255))
        c.alpha_composite(f.crop(box), ((side - w) // 2, (side - h) // 2))
        out.append(c.convert("RGB").resize((SIZE, SIZE), Image.LANCZOS))
    return out, int(1000 / fps)


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    for seq, name, fps in EMOS:
        ims, dur = make(seq, fps)
        pal = [im.quantize(colors=128, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE) for im in ims]
        pal[0].save(OUT / f"{seq}.gif", save_all=True, append_images=pal[1:], duration=dur, loop=0, optimize=True, disposal=1)
    print("emo gif:", OUT, len(EMOS), "개")


if __name__ == "__main__":
    main()
