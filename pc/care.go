package main

// 돌봄 더하기
//   - 꼬질꼬질(Dirt): 돌아다니고 놀고 던져지면 조금씩 더러워짐 → 얼굴에 얼룩이 보이고, 마우스로 문질러 씻겨 줌
//   - 졸림(Tired):    깨어 있으면 쌓이고 낮잠으로 조금, 이불을 덮어 재우면 푹 풀림
//     (저장값은 0 이 "깨끗함 · 쌩쌩" 이라서 예전 저장 파일도 그대로 시작)

import (
	"math"
	"time"
	"unsafe"
)

func clean() int  { return int(100 - S.Dirt + 0.5) }
func energy() int { return int(100 - S.Tired + 0.5) }

func dirtyShown() bool { return S.Dirt >= 50 }
func sleepy() bool     { return S.Tired >= 75 }

var dirtSprite *Sprite

func loadCare() {
	if b, err := accFS.ReadFile("acc/dirt.png"); err == nil {
		dirtSprite = decodeSprite(b)
	}
}

// 1분마다 (stateTick)
func careMinute() {
	switch {
	case tucked:
	case mode == "sleep":
		S.Tired = clamp(S.Tired - 3)
	default:
		S.Dirt = clamp(S.Dirt + 0.18)
		S.Tired = clamp(S.Tired + 0.22)
	}
}

// 꺼져 있던 동안 (loadState): 자고 왔다고 보고 졸림은 풀리고, 조금 꼬질꼬질
func careAway(h float64) {
	if h >= 1 {
		S.Tired = clamp(S.Tired - h*15)
	}
	S.Dirt = clamp(S.Dirt + math.Min(h, 12))
}

func addDirt(v float64) { S.Dirt = clamp(S.Dirt + v) }

// 돌봄이 필요할 때 하는 말 (없으면 "")
func careLine() string {
	n := voc()
	if S.Dirt >= 70 && rng.Float64() < 0.6 {
		return pick([]string{"몸이 근질근질해… " + n + " 씻겨 줘~", "킁킁… 나한테서 냄새나?", "목욕하고 싶다… 거품 목욕!"})
	}
	if sleepy() && rng.Float64() < 0.6 {
		return pick([]string{"하암… 졸려…", n + " 이불 덮어 줄래? 하암~", "눈이 자꾸 감겨…"})
	}
	return ""
}

// ── 씻기기: 마우스로 우땅이를 문지르면 거품이 퐁퐁, 다 씻으면 반짝 ──

var (
	bathing     bool
	bathStart   time.Time
	bathLastRub time.Time
	bathNeed    float64 // 남은 문지르기 양
	bathPrompt  bool
	rubAcc      float64
	lastCur     POINT
	lastAutoBub time.Time
)

const rubStep = 24.0 // 이만큼(px) 문지를 때마다 거품 하나

func startBath() {
	if focusing() {
		say("집중 끝나고 씻자! 쉿~", 2500*time.Millisecond)
		return
	}
	if S.Dirt < 5 {
		setMode("st_me", 2200*time.Millisecond)
		say("나 이미 깨끗해! 반짝반짝~", 2500*time.Millisecond)
		return
	}
	tucked = false
	bathing, bathPrompt = true, false
	bathStart = time.Now()
	bathNeed = math.Max(S.Dirt, 35)
	rubAcc = 0
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&lastCur)))
	setMode("bath", time.Hour)
	say("목욕하자! 🫧\n마우스로 나를 문질문질 해 줘~", 4*time.Second)
	nextTalk = time.Now().Add(rd(90, 150))
}

// 문지르기 한 번 (cx, cy: 우땅이 창 안 좌표)
func rub(cx, cy float64) {
	bathLastRub = time.Now()
	bathNeed -= 3.5
	S.Dirt = clamp(S.Dirt - 3.5)
	addBubbleAt(cx, cy)
	if time.Since(lastRubSfx) > 280*time.Millisecond {
		lastRubSfx = time.Now()
		sfx("bubble")
	}
}

var lastRubSfx time.Time

func bathTick(now time.Time) {
	if !bathing {
		return
	}
	if mode != "bath" { // 끌어서 옮기는 등으로 중간에 끝남
		bathing = false
		saveState()
		say("앗 아직 거품 묻었는데~", 2500*time.Millisecond)
		return
	}
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	in := float64(c.X) >= posX && float64(c.X) < posX+float64(canvasW) && float64(c.Y) >= posY && float64(c.Y) < posY+float64(canvasH)
	d := math.Hypot(float64(c.X-lastCur.X), float64(c.Y-lastCur.Y))
	lastCur = c
	if in && d < 300*dpi {
		rubAcc += d
		for rubAcc >= rubStep*dpi && bathNeed > 0 {
			rubAcc -= rubStep * dpi
			rub(float64(c.X)-posX, float64(c.Y)-posY)
		}
	}
	// 문지르는 중이면 몸 여기저기서도 거품
	if now.Sub(bathLastRub) < 600*time.Millisecond && now.Sub(lastAutoBub) > 250*time.Millisecond {
		lastAutoBub = now
		addBubbleAt(float64(canvasW)*rf(0.25, 0.75), float64(canvasH)*rf(0.3, 0.85))
	}
	switch {
	case bathNeed <= 0:
		finishBath()
	case now.Sub(bathStart) > 45*time.Second:
		bathing = false
		saveState()
		setMode("idle", rd(1.5, 2.5))
		say("에취! 다음에 마저 씻자~", 3*time.Second)
	case !bathPrompt && now.Sub(bathStart) > 7*time.Second && bathLastRub.Before(bathStart):
		bathPrompt = true
		say("마우스를 내 위에서\n왔다 갔다 해 봐! 🫧", 3500*time.Millisecond)
	}
}

func finishBath() {
	bathing = false
	S.Dirt = 0
	S.BathCnt++
	S.Mood = clamp(S.Mood + 12)
	dayLog().Bath++
	burst("sparkle", 10)
	setMode("laugh", rd(2.8, 3.4))
	say(pick([]string{"반짝반짝! 개운하다~\n고마워 " + voc() + "!", "뽀송뽀송! 향기 나지? 히히", "깨끗해졌다! 반짝 ✨"}), 4*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
	if S.BathDay != today() {
		S.BathDay = today()
		addExp(6) // 저장 포함
	} else {
		saveState()
	}
}

// ── 이불 덮어 재우기 ──

var (
	tucked    bool
	tuckStart time.Time
)

func tuckIn() {
	if focusing() {
		say("집중 끝나면 잘게~ 쉿!", 2500*time.Millisecond)
		return
	}
	h := time.Now().Hour()
	if S.Tired < 15 && h >= 7 && h < 21 {
		setMode("laugh", rd(2, 2.6))
		say("아직 하나도 안 졸려! 더 놀자~", 2500*time.Millisecond)
		return
	}
	bathing = false
	tucked = true
	tuckStart = time.Now()
	setMode("sleep", time.Duration(math.Max(40, math.Min(120, S.Tired/1.5)))*time.Second)
	say(pick([]string{"이불 덮어 줘서 고마워…\n잘 자 " + voc() + "… ♥", "포근하다… 쿨쿨…", "따뜻해… 좋은 꿈 꿀게…"}), 3500*time.Millisecond)
	nextTalk = time.Now().Add(rd(40, 80))
}

// 이불 덮고 자는 동안: 졸림이 빨리 풀림
func tuckTick(dt float64) {
	if !tucked {
		return
	}
	if mode != "sleep" { // 끌어서 깨움 등
		tucked = false
		return
	}
	S.Tired = clamp(S.Tired - dt*2.5)
}

// 다 자고 일어남 (nextChoice 의 sleep 끝)
func wakeTucked() {
	tucked = false
	S.Tired = 0
	S.TuckCnt++
	S.Mood = clamp(S.Mood + 8)
	dayLog().Tuck++
	setMode("wave", rd(2.6, 3.2))
	say(pick([]string{"잘 잤다! 기운이 펄펄!", "하암~ 푹 잤어! 고마워 " + voc() + "!", "개운해! 이제 놀자!"}), 3500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
	if S.TuckDay != today() {
		S.TuckDay = today()
		addExp(4) // 저장 포함
	} else {
		saveState()
	}
}

// 이불 덮고 자는데 클릭: 처음엔 더 잔다고 하고, 좀 지나면 일어남
func pokeTucked() {
	if time.Since(tuckStart) < 15*time.Second {
		say(pick([]string{"으음… 5분만 더…", "쿨… 조금만 더 잘래…"}), 2500*time.Millisecond)
		return
	}
	wakeTucked()
}

// ── 그리기 ──

// render() 에서 프레임 좌표마다: 이불 (덮고 잘 때)
func drawBlanket(fr *Sprite, su, sv float64, acc *[4]float32) {
	b := fxSprites["blanket"]
	if b == nil || fr.Seq != "sleep" {
		return
	}
	c := sample(b.full, su, sv)
	if c[3] < 1 {
		return
	}
	k := 1 - c[3]/255
	for i := 0; i < 4; i++ {
		acc[i] = c[i] + acc[i]*k
	}
}

// ── 메뉴 ──

const (
	idBath  = 1210
	idTuck  = 1211
	idDiary = 1212
)

func appendCareMenu(m uintptr) {
	appendMenu(m, MF_STRING, idBath, "씻기기 🫧")
	appendMenu(m, MF_STRING, idTuck, "이불 덮어 재우기")
}

func onCareCmd(cmd uintptr) bool {
	switch cmd {
	case idBath:
		startBath()
	case idTuck:
		tuckIn()
	case idDiary:
		openDiary()
	default:
		return false
	}
	return true
}
