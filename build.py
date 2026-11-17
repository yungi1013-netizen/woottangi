"""우땅이 전체 빌드 스크립트

    python build.py            # 웹 결과물 + PC exe (지금 frames/ 그대로 사용)
    python build.py frames     # 프레임부터 다시 그림 (머리 표정 → 동작 → 스티커)
    python build.py web        # 휴대폰용 HTML + PC 게임 페이지만
    python build.py exe        # PC exe만 (Go 필요)
    python build.py all        # 처음부터 끝까지 전부

결과물
    web/woottangi_mobile.html   휴대폰용 우땅이 키우기 (한 파일)
    dist/Woottangi.exe          PC 바탕화면 우땅이 v1.0
"""
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PY = sys.executable


def run(cmd, **kw):
    print(">", " ".join(str(c) for c in cmd))
    subprocess.run([str(c) for c in cmd], check=True, **kw)


def step_frames():
    run([PY, ROOT / "tools" / "heads.py"])
    run([PY, ROOT / "tools" / "compose.py", ROOT / "frames"])
    run([PY, ROOT / "tools" / "stickers.py", ROOT / "frames"])
    run([PY, ROOT / "tools" / "accessories.py"])   # 꾸미기 아이템 그림 + 프레임별 머리 위치


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
    run([go, "build", "-ldflags", "-H windowsgui -s -w", "-o", ROOT / "dist" / "Woottangi.exe", "."], cwd=pc, env=env)
    print("exe: dist/Woottangi.exe")


if __name__ == "__main__":
    what = sys.argv[1] if len(sys.argv) > 1 else "default"
    if what in ("frames", "all"):
        step_frames()
    if what in ("web", "all", "default"):
        step_web()
    if what in ("exe", "all", "default"):
        step_exe()
