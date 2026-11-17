# 우땅이 v1.0 소스

바탕화면을 돌아다니는 우땅이(PC)와 휴대폰용 "우땅이 키우기" 웹앱의 전체 소스입니다.
제작: K

- **그림(프레임)은 Python**으로 만듭니다. 원본 머리 + 그린 몸 → 480×600 PNG
- **PC 프로그램은 Go**(Win32, cgo 없음)로 만들어 exe 하나로 나옵니다. 프레임은 exe 안에 들어갑니다.
- **휴대폰용은 HTML 한 파일**입니다. 프레임이 WebP로 안에 들어갑니다.

## 폴더 구조

```
Woottangi_source/
├─ build.py               전체 빌드 (아래 "빌드 방법")
├─ requirements.txt       파이썬 패키지 목록
├─ assets/                원본 그림
│  ├─ head_full.png        원본 머리 (정체성 — 이 얼굴은 그대로 유지)
│  ├─ head_*.png           표정 머리 (heads.py 가 만든 것: 먹기·수줍음·벌받기·깜빡·잠)
│  ├─ fistL/R, palmL/R     주먹·손바닥
│  └─ stickers/            이모티콘 스티커 5종 (고해상도)
├─ tools/                 파이썬 도구
│  ├─ heads.py             표정 머리 만들기 (눈·입 inpaint 후 새로 그림)
│  ├─ compose.py           동작 프레임 합성 (걷기·앉기·자기·웃기·먹기·벌서기 …)
│  ├─ stickers.py          스티커 동작 프레임 (놀람·뒹굴·울기·저요·인사)
│  ├─ accessories.py       꾸미기 아이템 그림(pc/acc) + 프레임별 머리 위치(pc/anchors.go)
│  ├─ build_web.py         휴대폰 HTML + PC 게임 페이지 만들기
│  └─ make_res.py          exe 아이콘·버전정보(.syso) 만들기
├─ frames/                완성 프레임 128장 (위 도구들의 결과물)
├─ pc/                    PC 프로그램 (Go)
│  ├─ main.go              창·애니메이션·이동·드래그·우클릭 메뉴
│  ├─ talk.go              말풍선·이름 묻기·시간대 대사·클릭/밥/쓰다듬기/혼내기
│  ├─ state.go             바나나·배부름·기분·간식·출석·운세·바나나 받기 게임
│  ├─ notify.go            트레이 아이콘·윈도우 알림·시간 알림·배고픔 알림
│  ├─ shop.go              꾸미기 상점 (바나나로 모자·안경·나비넥타이 사서 씌우기)
│  ├─ level.go             레벨·친밀도 (경험치·칭호·레벨 업 보상·함께한 날·기념일·친밀도 대사)
│  ├─ ach.go               업적 19개 · 누적 출석 선물(30·100일) · 우땅이 수첩(/book 업적·도감 페이지)
│  ├─ match_template.html  짝 맞추기 게임 화면 (build_web.py 가 match.html 로 만듦)
│  ├─ anchors.go           프레임별 머리 위치 (accessories.py 가 자동 생성)
│  ├─ acc/                 꾸미기 아이템 그림 (accessories.py 결과)
│  ├─ shop_test.go         go test → 상점 동작 확인 + test_worn.png 미리보기
│  └─ game_template.html   바나나 받기 게임 화면 (build_web.py 가 game.html 로 만듦)
├─ web/
│  ├─ template.html        휴대폰용 화면 원본
│  └─ woottangi_mobile.html  완성본 (build_web.py 결과)
└─ dist/Woottangi.exe     완성 exe (build.py 결과)
```

## 준비물

```
pip install -r requirements.txt
python -m playwright install chromium     # compose.py 가 SVG를 그릴 때 사용
```
PC exe 를 만들려면 Go 1.22 이상 (https://go.dev/dl). 윈도우·맥·리눅스 어디서든 윈도우용 exe 가 만들어집니다.

## 빌드 방법

| 명령 | 하는 일 |
|---|---|
| `python build.py` | 지금 frames/ 그대로 → 휴대폰 HTML + exe |
| `python build.py frames` | 프레임만 처음부터 다시 그림 (표정 → 동작 → 스티커) |
| `python build.py web` | 휴대폰 HTML + 게임 페이지만 |
| `python build.py exe` | exe 만 |
| `python build.py all` | 전부 처음부터 |

동작 하나만 다시 그려 보려면:
```python
import sys; sys.path.insert(0, "tools")
import compose
from playwright.sync_api import sync_playwright
with sync_playwright() as p:
    pg = p.chromium.launch().new_page(viewport={"width": 480, "height": 600})
    compose.make(pg, "eat_milk", 0.25).save("test.png")
```

## 자주 고칠 곳

| 바꾸고 싶은 것 | 위치 |
|---|---|
| 시간대 대사 (9시 좋은 아침 등) | `pc/talk.go` → `var slots` |
| 평소 혼잣말 / 클릭했을 때 말 | `pc/talk.go` → `randomLine()`, `clickLine()` |
| 간식 종류·가격·효과 | `pc/state.go` → `var snacks` / 휴대폰은 `web/template.html` → `SNACKS` |
| 경험치 양 · 레벨 곡선 · 칭호 · 친밀도 대사 | `pc/level.go` → `exp*`, `expNeed()`, `var titles`, `affectionLine()` |
| 업적 (조건·보상·선물 아이템) | `pc/ach.go` → `var achievements` |
| 짝 맞추기 카드 그림 / 보상 | `tools/build_web.py` → `poses` / `pc/state.go` 의 `26-moves` (페이지 `reward()` 도 같이) |
| 꾸미기 아이템 이름·가격·대사 | `pc/shop.go` → `var items` |
| 꾸미기 아이템 그림 / 새 아이템 | `tools/accessories.py` → 그림 함수 + `ITEMS` (그 뒤 `python tools/accessories.py`) |
| 아이템이 머리에서 어긋나는 스티커 동작 | `tools/accessories.py` → `STICKER0` (`preview` 로 확인) |
| 오늘의 운세 문구 | `pc/state.go` → `var luck` / 휴대폰은 `web/template.html` → `LUCK` |
| 기본 크기 (작게 0.75) | `pc/main.go` → `scale = 0.75` |
| 동작 프레임 수 | `tools/compose.py` → `SEQ` |
| 몸·팔·다리 모양 | `tools/compose.py` → `pose()`, `pose_eat()` |
| 버전 (지금 1.0 고정) | `pc/main.go` 메뉴 "우땅이 v1.0", `pc/notify.go` 트레이 이름, `tools/make_res.py` → `VER`, `FileVersion`, `ProductVersion` |

## 저장 위치 (PC)

`%AppData%\Woottangi\`
- `name.txt` 주인 이름
- `state.json` 바나나·배부름·기분·출석·설정
- `reminders.json` 시간 알림 목록

휴대폰용은 브라우저 저장소(localStorage)에 저장됩니다.

## 참고

- `stickers.py` 는 정리하면서 다시 작성한 것이라, 처음 만든 스티커 프레임과 움직임이 아주 조금 다를 수 있습니다. 지금 `frames/` 에 들어 있는 것은 실제 exe 에 쓰인 원본입니다.
- 나머지 도구(`heads.py`, `compose.py`)는 지금 `frames/` 와 똑같은 그림을 다시 만들어 냅니다.
