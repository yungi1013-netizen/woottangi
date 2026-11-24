"""안드로이드 우땅이(화면 위 펫) APK 만들기 — Gradle 없이 Android SDK 도구만 사용

    python tools/build_android.py          # dist/Woottangi_Android_v{버전}.apk
    python tools/build_android.py --test   # 우땅이 두뇌 테스트만 (PC 의 java 로)

필요한 것: JDK 17, Android SDK (platforms;android-34, build-tools;34.0.0)
SDK 위치: ANDROID_HOME 환경변수, 없으면 E:/Project/android-sdk 또는 %LOCALAPPDATA%/Android/Sdk
서명: ~/.android/debug.keystore (없으면 keytool 로 만듦). 잠금·라이선스 같은 보안 기능은 넣지 않음.

폰에 이미 깔린 우땅이 위에 "업데이트"로 깔리게 (기록 유지)
  - 패키지 이름(com.woottangi.pet)과 서명 키가 같아야 함 → 서명 인증서 지문을 tools/android_cert.txt 에 적어 두고,
    키가 바뀌면 빌드를 멈춤 (다른 키로 서명하면 폰에서 지우고 새로 깔아야 해서 기록이 날아감)
  - 버전 코드는 빌드할 때마다 커짐 (버전 이름은 build.py 의 VERSION)
  - 기록은 앱 저장소(SharedPreferences "woottangi")에 있어서 업데이트해도 그대로 남음
"""
import os
import re
import shutil
import subprocess
import sys
import zipfile
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent.parent
AND = ROOT / "android"
OUT = AND / "build"
API = "34"
BT = "34.0.0"
FRAME_W, FRAME_H = 240, 300


def version():
    m = re.search(r'^VERSION\s*=\s*"([^"]+)"', (ROOT / "build.py").read_text(encoding="utf-8"), re.M)
    return m.group(1) if m else "1.2"


def sdk_dir():
    cands = [os.environ.get("ANDROID_HOME"), os.environ.get("ANDROID_SDK_ROOT"),
             str(ROOT.parent.parent / "android-sdk"),
             os.path.join(os.environ.get("LOCALAPPDATA", ""), "Android", "Sdk")]
    for c in cands:
        if c and (Path(c) / "platforms" / f"android-{API}" / "android.jar").exists():
            return Path(c)
    sys.exit("Android SDK 를 찾지 못했어요. ANDROID_HOME 을 정해 주세요 (platforms;android-%s, build-tools;%s 필요)" % (API, BT))


def run(cmd, **kw):
    cmd = [str(c) for c in cmd]
    print(">", " ".join(Path(cmd[0]).name if i == 0 else c for i, c in enumerate(cmd))[:200])
    subprocess.run(cmd, check=True, **kw)


def tool(sdk, name):
    d = sdk / "build-tools" / BT
    for ext in ("", ".exe", ".bat", ".cmd"):
        if (d / (name + ext)).exists():
            return d / (name + ext)
    sys.exit(f"build-tools/{BT}/{name} 이 없어요")


def java_bin(name):
    jh = os.environ.get("JAVA_HOME")
    if jh and (Path(jh) / "bin" / (name + ".exe")).exists():
        return Path(jh) / "bin" / (name + ".exe")
    w = shutil.which(name)
    if w:
        return Path(w)
    # PATH 에 java 만 있을 때: java 가 알려 주는 JDK 폴더에서 찾기
    try:
        out = subprocess.run([shutil.which("java") or "java", "-XshowSettings:properties", "-version"],
                             capture_output=True, text=True, errors="replace").stderr
        home = re.search(r"java\.home = (.+)", out).group(1).strip()
        p = Path(home) / "bin" / (name + ".exe")
        if p.exists():
            return p
    except Exception:
        pass
    sys.exit(f"{name} not found (JDK 17 needed)")


# ───────── 테스트 ─────────
def test():
    cls = OUT / "test-classes"
    shutil.rmtree(cls, ignore_errors=True)
    cls.mkdir(parents=True)
    run([java_bin("javac"), "-encoding", "UTF-8", "-nowarn", "-d", cls,
         AND / "src/com/woottangi/pet/Pet.java", AND / "test/PetTest.java"])
    run([java_bin("java"), "-Dfile.encoding=UTF-8", "-Dstdout.encoding=UTF-8", "-cp", cls, "PetTest"])


# ───────── 그림 ─────────
def make_art(res, assets):
    frames = ROOT / "frames"
    pngs = sorted(frames.glob("*_*.png"))
    if not pngs:
        sys.exit("frames/ 가 비어 있어요. 먼저 python build.py frames")
    fd = assets / "frames"
    fd.mkdir(parents=True)
    for p in pngs:
        im = Image.open(p).convert("RGBA").resize((FRAME_W, FRAME_H), Image.LANCZOS)
        im.save(fd / (p.stem + ".webp"), "WEBP", quality=88, method=4)

    make_play_art(assets)

    idle = Image.open(frames / "idle_0.png").convert("RGBA")
    bbox = idle.getbbox()
    pet = idle.crop(bbox)
    # 앱 아이콘: 크림색 동그라미 + 우땅이 얼굴 쪽
    for d, s in {"mdpi": 48, "hdpi": 72, "xhdpi": 96, "xxhdpi": 144, "xxxhdpi": 192}.items():
        S = s * 4
        ic = Image.new("RGBA", (S, S), (0, 0, 0, 0))
        dr = ImageDraw.Draw(ic)
        dr.ellipse((S * 0.03, S * 0.03, S * 0.97, S * 0.97), fill=(255, 226, 120, 255), outline=(224, 184, 96, 255), width=S // 40)
        pw = int(S * 0.78)
        ph = int(pet.height * pw / pet.width)
        sm = pet.resize((pw, ph), Image.LANCZOS)
        mask = Image.new("L", (S, S), 0)
        ImageDraw.Draw(mask).ellipse((S * 0.03, S * 0.03, S * 0.97, S * 0.97), fill=255)
        layer = Image.new("RGBA", (S, S), (0, 0, 0, 0))
        layer.paste(sm, ((S - pw) // 2, int(S * 0.12)), sm)
        layer.putalpha(Image.composite(layer.getchannel("A"), Image.new("L", (S, S), 0), mask))
        ic.alpha_composite(layer)
        out = res / f"mipmap-{d}"
        out.mkdir(parents=True)
        ic.resize((s, s), Image.LANCZOS).save(out / "ic_launcher.png")

    # 알림 아이콘: 흰색 실루엣
    a = pet.getchannel("A")
    for d, s in {"mdpi": 24, "hdpi": 36, "xhdpi": 48, "xxhdpi": 72, "xxxhdpi": 96}.items():
        hh = s
        ww = max(1, int(pet.width * hh / pet.height))
        al = a.resize((ww, hh), Image.LANCZOS)
        im = Image.new("RGBA", (s, s), (255, 255, 255, 0))
        white = Image.new("RGBA", (ww, hh), (255, 255, 255, 255))
        white.putalpha(al)
        im.paste(white, ((s - ww) // 2, 0), white)
        out = res / f"drawable-{d}"
        out.mkdir(parents=True, exist_ok=True)
        im.save(out / "ic_noti.png")


# 앱 꾸미기 그림: 뽑기 4개 + 업적 보상 (Pet.ITEM_IDS 와 같음)
GACHA = ["starpin", "halo", "heartglasses", "scarf", "rainbow", "bananaband", "medal", "headphones", "flowercrown", "chefhat"]
FRIEND_SEQ = ["idle", "walk", "laugh", "wave", "sit", "shy"]

# 게임 페이지가 PC 에서는 작은 서버에 fetch 하는데, 앱에서는 Woo(자바)로 바로 전달
BRIDGE = ('<script>window.fetch=function(u){var r=String(Woo.call(String(u)));'
          'return Promise.resolve({ok:r!=="",text:function(){return Promise.resolve(r)}})};'
          'window.close=function(){Woo.close()};</script>')
CHAT_BRIDGE = ('<script>window.fetch=function(u,o){u=String(u);var b=o&&o.body!=null?String(o.body):"";'
               'function J(v){return {ok:true,json:function(){return Promise.resolve(v)}}}'
               'if(u==="/ai/chat")return new Promise(function(res){window.__wooAi=function(r){res(J(r))};Woo.ask(b)});'
               'return Promise.resolve(J(JSON.parse(String(Woo.ai(u,b)))))};</script>')


def friend_tint(im):
    """땅콩이 털색 (pc/friend.go friendTint 와 같은 계산): 밝은 땅콩색, 외곽선·흰 얼굴은 그대로"""
    import numpy as np
    a = np.asarray(im, dtype=np.float32).copy()
    r, g, b, al = a[..., 0], a[..., 1], a[..., 2], a[..., 3]
    lum = 0.3 * r + 0.59 * g + 0.11 * b
    w = np.clip((lum - 55) / 70, 0, 1) * np.clip((r - b - 25) / 40, 0, 1) * (al >= 1)
    for ch, t in ((0, 255), (1, 208), (2, 140)):
        c = a[..., ch]
        a[..., ch] = c + ((c * 0.64 + t * 0.36) - c) * w
    return Image.fromarray(np.clip(a, 0, 255).astype("uint8"), "RGBA")


FX = ["ball", "balloon", "bubble", "blanket", "banana", "goldbanana", "petal", "maple", "ginkgo", "acorn",
      "clover", "shell", "star", "candy"]


def make_play_art(assets):
    """놀아주기용: 게임 페이지 2개 · 땅콩이 그림 · 꾸미기 아이템 + 프레임별 머리 위치"""
    import json
    sys.path.insert(0, str(ROOT / "tools"))
    import accessories as A
    import build_web as W

    F = W.load_frames()
    g = (ROOT / "pc" / "game_template.html").read_text(encoding="utf-8")
    g = g.replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
    poses = [("idle", 0), ("laugh", 2), ("sit", 0), ("sleep", 3), ("eat", 2), ("shy", 2), ("wave", 2), ("punish", 0)]
    m = (ROOT / "pc" / "match_template.html").read_text(encoding="utf-8")
    m = (m.replace("__CARDS__", json.dumps([F[k][i] for k, i in poses]))
         .replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2]))
    t = (ROOT / "pc" / "tower_template.html").read_text(encoding="utf-8")
    t = t.replace("__WAVE__", F["wave"][2]).replace("__LAUGH__", F["laugh"][2])
    wh = W.whack_html(F)
    for name, html in (("game.html", g), ("match.html", m), ("tower.html", t), ("whack.html", wh)):
        assert "<head>" in html and 'fetch("/start")' in html
        (assets / name).write_text(html.replace("<head>", "<head>" + BRIDGE, 1), encoding="utf-8")

    # AI 대화 (pc/chat.html 그대로 + 폰용 연결: /ai/chat 은 인터넷이라 비동기)
    c = (ROOT / "pc" / "chat.html").read_text(encoding="utf-8")
    assert "/item/none.png" in c and "이 PC에만" in c
    c = c.replace("/item/none.png", F["idle"][0]).replace("이 PC에만", "이 폰에만")
    (assets / "chat.html").write_text(c.replace("<head>", "<head>" + CHAT_BRIDGE, 1), encoding="utf-8")

    anchors = A.anchors()
    (assets / "anchors.json").write_text(json.dumps(
        {n: [[round(v, 2) for v in a] for a in lst] for n, lst in anchors.items()}), encoding="utf-8")
    ad = assets / "acc"
    ad.mkdir()
    for k in GACHA:
        Image.open(A.OUT / f"{k}.png").convert("RGBA").save(ad / f"{k}.webp", "WEBP", quality=90, method=4)
    Image.open(ROOT / "pc" / "acc" / "dirt.png").convert("RGBA").save(ad / "dirt.webp", "WEBP", quality=90, method=4)

    # 장난감 · 선물 · 이불 · 거품 (tools/effects.py 가 만든 pc/fx 그림)
    xd = assets / "fx"
    xd.mkdir()
    for k in FX:
        Image.open(ROOT / "pc" / "fx" / f"{k}.png").convert("RGBA").save(xd / f"{k}.webp", "WEBP", quality=90, method=4)

    # 이모티콘 GIF (tools/emo_gif.py 가 만든 pc/emo) + 저장할 때 쓸 한글 이름 목록
    import shutil
    import emo_gif as EG
    ed = assets / "emo"
    ed.mkdir()
    rows = []
    for seq, name, _ in EG.EMOS:
        src = ROOT / "pc" / "emo" / f"{seq}.gif"
        if src.exists():
            shutil.copyfile(src, ed / f"{seq}.gif")
            rows.append(f"{seq}\t{name}")
    if not rows:
        sys.exit("pc/emo 가 비어 있어요. 먼저 python tools/emo_gif.py")
    (ed / "names.txt").write_text("\n".join(rows) + "\n", encoding="utf-8")

    flower = Image.open(A.OUT / "flower.png").convert("RGBA")
    fd = assets / "friend"
    fd.mkdir()
    for seq in FRIEND_SEQ:
        for i, a in enumerate(anchors[seq]):
            p = ROOT / "frames" / f"{seq}_{i}.png"
            if not p.exists():
                continue
            im = friend_tint(Image.open(p).convert("RGBA"))
            A.put(im, flower, a)
            im.resize((FRAME_W, FRAME_H), Image.LANCZOS).save(fd / f"{seq}_{i}.webp", "WEBP", quality=88, method=4)


# ───────── 서명 키 (개발용, 폰에 설치할 때만 필요) ─────────
def keystore():
    ks = Path.home() / ".android" / "debug.keystore"
    if not ks.exists():
        ks.parent.mkdir(parents=True, exist_ok=True)
        run([java_bin("keytool"), "-genkeypair", "-keystore", ks, "-storepass", "android", "-keypass", "android",
             "-alias", "androiddebugkey", "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000",
             "-dname", "CN=Android Debug,O=Android,C=US"])
    return ks


CERT_FILE = ROOT / "tools" / "android_cert.txt"


def version_code():
    """빌드할 때마다 커지는 버전 코드 (10분 단위 시각). 예전 코드(101)보다 항상 큼"""
    import time
    return int(time.time() // 600)


def cert_digest(sdk, apk):
    out = subprocess.run([str(tool(sdk, "apksigner")), "verify", "--print-certs", str(apk)],
                         capture_output=True, text=True, check=True).stdout
    m = re.search(r"certificate SHA-256 digest:\s*([0-9a-f]+)", out)
    if not m:
        sys.exit("서명 인증서 지문을 못 읽었어요:\n" + out)
    return m.group(1)


def check_cert(sdk, apk):
    """처음 빌드할 때 지문을 적어 두고, 다음부터는 같은 키인지 확인"""
    d = cert_digest(sdk, apk)
    if not CERT_FILE.exists():
        CERT_FILE.write_text(d + "\n", encoding="utf-8")
        print("서명 지문 저장:", CERT_FILE)
        return
    want = CERT_FILE.read_text(encoding="utf-8").split()[0]
    if d != want:
        apk.unlink()
        sys.exit("서명 키가 예전과 달라요! 이 APK 는 폰에 업데이트로 안 깔리고, 지우고 새로 깔면 기록이 날아가요.\n"
                 f"  예전: {want}\n  지금: {d}\n"
                 "예전에 쓰던 ~/.android/debug.keystore 를 되돌려 놓고 다시 빌드해 주세요.")
    print("서명 키 확인: 예전과 같음 (폰에 업데이트로 깔려요)")


def build():
    ver = version()
    code = version_code()
    sdk = sdk_dir()
    jar = sdk / "platforms" / f"android-{API}" / "android.jar"

    shutil.rmtree(OUT, ignore_errors=True)
    res, assets, gen, cls, dex = (OUT / n for n in ("res", "assets", "gen", "classes", "dex"))
    for d in (res, assets, gen, cls, dex):
        d.mkdir(parents=True)
    make_art(res, assets)

    aapt2 = tool(sdk, "aapt2")
    run([aapt2, "compile", "--dir", res, "-o", OUT / "res.zip"])
    base = OUT / "base.apk"
    run([aapt2, "link", "-o", base, "-I", jar, "--manifest", AND / "AndroidManifest.xml",
         "--min-sdk-version", "26", "--target-sdk-version", API,
         "--version-code", str(code), "--version-name", ver,
         "-A", assets, "--java", gen, "-0", "webp", OUT / "res.zip"])

    srcs = [str(p) for p in (AND / "src").rglob("*.java")] + [str(p) for p in gen.rglob("*.java")]
    # java.* 는 JDK 8 기준, android.* 는 android.jar (android.jar 의 LambdaMetafactory 는 껍데기라 람다 컴파일 불가)
    run([java_bin("javac"), "-encoding", "UTF-8", "--release", "8", "-nowarn",
         "-classpath", jar, "-d", cls] + srcs)

    classes = [str(p) for p in cls.rglob("*.class")]
    d8 = tool(sdk, "d8")
    run([d8, "--release", "--min-api", "26", "--lib", jar, "--output", dex] + classes)

    unsigned = OUT / "unsigned.apk"
    # aapt2 결과를 새 zip 으로 다시 쓰면서 classes.dex 추가 (그냥 덧붙이면 zipalign 이 헤더 불일치라고 함)
    with zipfile.ZipFile(base) as src, zipfile.ZipFile(unsigned, "w") as z:
        for info in src.infolist():
            ni = zipfile.ZipInfo(info.filename, date_time=(2024, 1, 1, 0, 0, 0))
            ni.compress_type = info.compress_type
            ni.external_attr = info.external_attr
            z.writestr(ni, src.read(info))
        ni = zipfile.ZipInfo("classes.dex", date_time=(2024, 1, 1, 0, 0, 0))
        ni.compress_type = zipfile.ZIP_DEFLATED
        z.writestr(ni, (dex / "classes.dex").read_bytes())

    aligned = OUT / "aligned.apk"
    run([tool(sdk, "zipalign"), "-p", "-f", "4", unsigned, aligned])
    (ROOT / "dist").mkdir(exist_ok=True)
    final = ROOT / "dist" / f"Woottangi_Android_v{ver}.apk"
    run([tool(sdk, "apksigner"), "sign", "--ks", keystore(), "--ks-pass", "pass:android", "--key-pass", "pass:android",
         "--ks-key-alias", "androiddebugkey", "--out", final, aligned])
    run([tool(sdk, "apksigner"), "verify", final])
    check_cert(sdk, final)
    print(f"\n완성: {final}  ({final.stat().st_size / 1e6:.1f} MB, 버전 {ver}, 코드 {code})")


if __name__ == "__main__":
    test()
    if "--test" not in sys.argv:
        build()
