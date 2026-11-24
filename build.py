"""우땅이 전체 빌드 스크립트

    python build.py            # 웹 결과물 + PC exe (지금 frames/ 그대로 사용)
    python build.py frames     # 프레임부터 다시 그림 (머리 표정 → 동작 → 스티커)
    python build.py web        # 휴대폰용 HTML + PC 게임 페이지만
    python build.py exe        # PC exe만 (Go 필요)
    python build.py setup      # 배포용: exe + 설치 프로그램 + 압축 파일
    python build.py all        # 처음부터 끝까지 전부

결과물
    web/woottangi_mobile.html   휴대폰용 우땅이 키우기 (한 파일)
    dist/Woottangi.exe          PC 바탕화면 우땅이 (버전은 아래 VERSION)
"""
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PY = sys.executable
VERSION = "1.2"   # 버전은 여기만 바꾸면 exe·설치 프로그램·파일 정보에 모두 들어감
os.environ["WOO_VERSION"] = VERSION


def run(cmd, **kw):
    print(">", " ".join(str(c) for c in cmd))
    subprocess.run([str(c) for c in cmd], check=True, **kw)


def step_frames():
    run([PY, ROOT / "tools" / "heads.py"])
    run([PY, ROOT / "tools" / "compose.py", ROOT / "frames"])
    run([PY, ROOT / "tools" / "stickers.py", ROOT / "frames"])
    run([PY, ROOT / "tools" / "accessories.py"])   # 꾸미기 아이템 그림 + 프레임별 머리 위치
    run([PY, ROOT / "tools" / "effects.py"])       # 계절 날씨 조각 · 거품 · 이불
    run([PY, ROOT / "tools" / "emo_gif.py"])       # 이모티콘 GIF (pc/emo, 폰 앱에도 들어감)


def step_web():
    run([PY, ROOT / "tools" / "build_web.py"])


def step_exe():
    pc = ROOT / "pc"
    # exe 에 넣을 프레임 복사 (go:embed frames/*.png)
    dst = pc / "frames"
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(ROOT / "frames", dst)
    step_web()   # game.html · match.html 을 늘 최신 템플릿으로
    run([PY, ROOT / "tools" / "make_res.py", ROOT / "frames" / "idle_0.png", pc / "rsrc_windows_amd64.syso"])
    go = shutil.which("go")
    if not go:
        print("! Go 가 없어서 exe 빌드는 건너뜀 (https://go.dev/dl 에서 설치)")
        return
    (ROOT / "dist").mkdir(exist_ok=True)
    env = dict(os.environ, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
    run([go, "build", "-ldflags", f"-H windowsgui -s -w -X main.appVersion={VERSION}", "-o", ROOT / "dist" / "Woottangi.exe", "."], cwd=pc, env=env)
    print("exe: dist/Woottangi.exe")




def setup_art(out):
    """설치 화면 왼쪽 그림: 왕관 쓰고 손 흔드는 우땅이 (312px 폭, 2배 해상도)"""
    from PIL import Image
    sys.path.insert(0, str(ROOT / "tools"))
    import accessories as A
    anc = A.anchors()
    f = Image.open(ROOT / "frames" / "wave_2.png").convert("RGBA")
    A.put(f, Image.open(A.OUT / "crown.png").convert("RGBA"), anc["wave"][2])
    f = f.crop(f.getbbox())
    f.resize((312, round(f.height * 312 / f.width)), Image.LANCZOS).save(out)


def step_setup():
    """배포용: 설치 프로그램 + 압축 파일 (dist/)"""
    import zipfile
    step_exe()
    inst = ROOT / "installer"
    (inst / "payload").mkdir(exist_ok=True)
    shutil.copy2(ROOT / "dist" / "Woottangi.exe", inst / "payload" / "Woottangi.exe")
    setup_art(inst / "art.png")
    run([PY, ROOT / "tools" / "make_res.py", ROOT / "frames" / "idle_0.png", inst / "rsrc_windows_amd64.syso",
         "우땅이 설치", f"Woottangi_Setup_v{VERSION}.exe"])
    go = shutil.which("go")
    if not go:
        print("! Go 가 없어서 설치 프로그램 빌드는 건너뜀")
        return
    env = dict(os.environ, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
    setup = ROOT / "dist" / f"Woottangi_Setup_v{VERSION}.exe"
    run([go, "build", "-ldflags", f"-H windowsgui -s -w -X main.version={VERSION}", "-o", setup, "."], cwd=inst, env=env)
    # 설치 없이 쓰는 사람용 압축 파일
    z = ROOT / "dist" / f"Woottangi_v{VERSION}_portable.zip"
    with zipfile.ZipFile(z, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.write(ROOT / "dist" / "Woottangi.exe", "Woottangi.exe")
    print("setup:", setup.relative_to(ROOT))
    print("zip:  ", z.relative_to(ROOT))


if __name__ == "__main__":
    what = sys.argv[1] if len(sys.argv) > 1 else "default"
    if what in ("frames", "all"):
        step_frames()
    if what in ("web", "all", "default"):
        step_web()
    if what in ("exe", "all", "default"):
        step_exe()
    if what in ("setup", "all"):
        step_setup()
