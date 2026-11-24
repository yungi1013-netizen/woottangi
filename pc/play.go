package main

// 바탕화면 놀이
//   - 바탕화면 바나나 받기: 화면 위에서 바나나가 떨어지고, 우땅이가 마우스를 따라 좌우로 움직여 받음 (30초)
//   - 숨바꼭질: 우땅이가 화면 가장자리·창 뒤에 빼꼼 숨고, 찾아서 누르면 바나나
//   - 바나나 뽑기: 바나나 10개로 뽑기 (바나나 · 경험치 · 뽑기에서만 나오는 아이템)

import (
	"fmt"
	"math"
	"time"
	"unsafe"
)

func busy() bool {
	return catching || hiding || bathing || tucked || friendOn() || focusing() || quitting || gachaAt != (time.Time{})
}

// ───────── 바탕화면 바나나 받기 ─────────

const catchLen = 30 * time.Second

var (
	catching     bool
	catchUntil   time.Time
	catchScore   int
	catchSpawn   float64
	catchHappy   time.Time // 받은 직후 웃는 얼굴
	catchDizzy   time.Time // 돌멩이 맞으면 어질어질
	catchLastSay time.Time
	catchItems   []catchItem
)

type catchItem struct {
	particle
	kind string // banana goldbanana rock
}

func startDeskCatch() {
	if S.Full < 15 {
		setMode("st_lounge", 4*time.Second)
		say("배고파서 못 놀겠어… 간식 먼저!", 3*time.Second)
		return
	}
	if busy() || hiddenFS {
		say("지금은 못 해~ 조금 이따 놀자!", 2500*time.Millisecond)
		return
	}
	if showing() { // 날씨 쇼 중이면 그만
		showKind = ""
		closeFxWindow()
	}
	if !openFxWindow() {
		say("놀이 창을 못 열었어… 흑", 3*time.Second)
		return
	}
	perch = 0
	posY = ground
	pauseToy()
	catching, catchScore, catchSpawn = true, 0, 0
	catchItems = catchItems[:0]
	catchUntil = time.Now().Add(catchLen + 3*time.Second) // 3초 준비
	setMode("catch", time.Hour)
	say("바나나 받기 시작!\n마우스로 나를 움직여 줘!", 3*time.Second)
	sfx("start")
}

func catchStarted() bool { return time.Until(catchUntil) <= catchLen }

// 우땅이: 커서 x 를 따라 바닥에서 좌우로
func catchMove(dt float64) {
	if time.Now().Before(catchDizzy) {
		return
	}
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	target := float64(c.X) - float64(canvasW)/2
	target = math.Max(workL-float64(canvasW)*0.15, math.Min(workR-float64(canvasW)*0.85, target))
	d := target - posX
	step := 1100 * dpi * dt
	if math.Abs(d) <= step {
		posX = target
	} else {
		posX += math.Copysign(step, d)
	}
	if math.Abs(d) > 2 {
		dir = math.Copysign(1, d)
	}
	catchWalking = math.Abs(d) > 4
}

var catchWalking bool

func catchFrameSeq() (string, float64) {
	now := time.Now()
	switch {
	case now.Before(catchDizzy):
		return "st_surprise", 9
	case now.Before(catchHappy):
		return "laugh", 12
	case catchWalking:
		return "walk", 14
	}
	return "idle", 5
}

func catchTick(now time.Time, dt float64) {
	if !catching {
		return
	}
	if hiddenFS || fxBits == nil {
		endDeskCatch(true)
		return
	}
	if catchStarted() {
		left := catchUntil.Sub(now).Seconds()
		if left <= 0 {
			endDeskCatch(false)
			return
		}
		// 시간이 갈수록 많이 · 빨리
		prog := 1 - left/catchLen.Seconds()
		catchSpawn += dt * (1.3 + 1.8*prog) * float64(fxW) / (1400 * dpi)
		for catchSpawn >= 1 {
			catchSpawn--
			spawnCatch(prog)
		}
		if now.Sub(catchLastSay) > time.Second {
			catchLastSay = now
			say(fmt.Sprintf("바나나 %d개 · %d초", catchScore, int(math.Ceil(left))), 1500*time.Millisecond)
		}
	}
	// 받기 판정: 우땅이 머리~가슴 (화면 좌표)
	hx0, hx1 := posX+float64(canvasW)*0.22, posX+float64(canvasW)*0.78
	hy0, hy1 := posY+float64(canvasH)*0.15, posY+float64(canvasH)*0.62
	alive := catchItems[:0]
	for _, it := range catchItems {
		it.step(dt)
		sx, sy := it.x+float64(fxX), it.y+float64(fxY)
		if sx > hx0 && sx < hx1 && sy > hy0 && sy < hy1 && now.After(catchDizzy) {
			onCatch(it.kind, it.x, it.y)
			continue
		}
		if it.y < float64(fxH)+it.size {
			alive = append(alive, it)
		}
	}
	catchItems = alive
	drawCatch()
}

func spawnCatch(prog float64) {
	kind := "banana"
	switch r := rng.Float64(); {
	case r < 0.07:
		kind = "goldbanana"
	case r < 0.07+0.08+0.1*prog:
		kind = "rock"
	}
	size := rf(30, 40) * dpi
	sp := rf(230, 330)*dpi + prog*200*dpi
	catchItems = append(catchItems, catchItem{particle{sp: fxSprites[kind], x: rf(size, float64(fxW)-size), y: -size,
		vy: sp, size: size, rot: rf(-0.5, 0.5), vr: rf(-1.5, 1.5), life: 60}, kind})
}

func onCatch(kind string, x, y float64) {
	switch kind {
	case "rock":
		catchDizzy = time.Now().Add(1100 * time.Millisecond)
		say("아야! 돌멩이…", 1200*time.Millisecond)
		sfx("bonk")
		return
	case "goldbanana":
		catchScore += 5
		burst("sparkle", 6)
		sfx("gold")
	default:
		catchScore++
		sfx("coin")
	}
	catchHappy = time.Now().Add(350 * time.Millisecond)
}

func drawCatch() {
	buf := unsafe.Slice((*byte)(fxBits), fxW*fxH*4)
	clear(buf)
	for i := range catchItems {
		p := &catchItems[i]
		blit(buf, fxW, fxH, p.sp, p.x, p.y, p.size, p.rot, 1)
	}
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{int32(fxX), int32(fxY)}
	sz := SIZE{int32(fxW), int32(fxH)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, 255, AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(fxHwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		fxDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
	pShowWindow.Call(fxHwnd, 4)
}

// stopped: 중간에 그만둠 (보상 없음)
func endDeskCatch(stopped bool) {
	catching = false
	catchItems = catchItems[:0]
	closeFxWindow()
	bubUntil = time.Now()
	if stopped {
		setMode("idle", rd(1, 2))
		say("다음에 또 하자!", 2*time.Second)
		return
	}
	n := catchScore
	best := n > S.DeskBest
	if best {
		S.DeskBest = n
	}
	onGameScore(n, 0) // 바나나 · 기분 · 경험치 · 기록 (짝 맞추기가 아닌 '받기' 기록)
	if best && n > 0 {
		say(fmt.Sprintf("바탕화면 바나나 %d개!\n바탕화면 최고 기록이야!", n), 4500*time.Millisecond)
	}
	if n > 0 {
		sfx("tada")
	}
}

// ───────── 숨바꼭질 ─────────

var (
	hiding    bool
	hideStage int // 0 숨는 중(안 보임) 1 숨어 있음
	hideAt    time.Time
	hideFound time.Time // 숨기 끝난 시각 (찾는 시간 재기)
	hideClip  RECT      // 이 영역 밖은 안 그림 (화면 좌표: 창 뒤·화면 밖에 숨은 것처럼)
	hideBaseX float64
	hideSide  float64 // 빼꼼 내미는 방향 (x 로 -1/+1, 아래에서 내밀면 0)
	hideHint  bool
)

const hideLimit = 60 * time.Second

func startHide() {
	if busy() || hiddenFS {
		say("지금은 못 해~ 조금 이따 놀자!", 2500*time.Millisecond)
		return
	}
	detachSurface() // 벽·천장·자리에서 내려와서 숨음
	pauseToy()
	hiding, hideStage, hideHint = true, 0, false
	hideAt = time.Now().Add(2500 * time.Millisecond)
	perch = 0
	setMode("st_me", time.Hour)
	say("숨바꼭질! 내가 숨을게~\n찾으면 나를 눌러 줘!", 2500*time.Millisecond)
}

func hideTick(now time.Time) {
	switch hideStage {
	case 0:
		if now.After(hideAt) {
			pShowWindow.Call(hwnd, SW_HIDE)
			pShowWindow.Call(bubHwnd, SW_HIDE)
			bubText = ""
			hideStage = 1
			hideAt = now.Add(1500 * time.Millisecond) // 꼭꼭 숨는 시간
		}
		return
	case 1:
		if now.After(hideAt) {
			pickHideSpot()
			hideStage = 2
			hideFound = now
			setMode("hide", time.Hour)
			pShowWindow.Call(hwnd, 4)
		}
		return
	}
	// 가끔 빼꼼 더 내밀었다 들어감
	t := now.Sub(hideFound).Seconds()
	peek := math.Max(0, math.Sin(t*1.3)) * 0.12 * float64(canvasW)
	if hideSide != 0 {
		posX = hideBaseX + hideSide*peek
	}
	if !hideHint && now.Sub(hideFound) > 35*time.Second {
		hideHint = true
		say("여기야 여기~", 2500*time.Millisecond)
	}
	if now.Sub(hideFound) > hideLimit {
		endHide(false)
	}
}

// 숨을 곳 고르기: 켜진 모니터의 왼쪽/오른쪽 끝, 바닥 아래, 열린 창 뒤
func pickHideSpot() {
	var cands []int
	for i := range mons {
		if monAllowed(i) {
			cands = append(cands, i)
		}
	}
	if len(cands) == 0 {
		cands = []int{curMon}
	}
	m := mons[cands[rng.Intn(len(cands))]]
	w := m.work
	cw, ch := float64(canvasW), float64(canvasH)
	gy := float64(w.Bottom) - ch + 2
	hideSide = 0
	r := rng.Float64()
	if r < 0.35 {
		if _, wr, ok := hideWindow(w); ok && perchEnabled {
			posX = rf(float64(wr.Left)+cw*0.1, float64(wr.Right)-cw*1.1)
			posY = float64(wr.Top) - ch*0.42 // 머리만 빼꼼
			hideClip = RECT{-1 << 30, -1 << 30, 1 << 30, wr.Top}
			hideBaseX = posX
			dir = []float64{-1, 1}[rng.Intn(2)]
			return
		}
	}
	switch {
	case r < 0.62: // 왼쪽 끝
		hideSide = 1
		posX = float64(w.Left) - cw*0.5
		posY = rf(float64(w.Top)+ch*0.2, gy)
		hideClip = RECT{w.Left, -1 << 30, 1 << 30, 1 << 30}
		dir = 1
	case r < 0.89: // 오른쪽 끝
		hideSide = -1
		posX = float64(w.Right) - cw*0.5
		posY = rf(float64(w.Top)+ch*0.2, gy)
		hideClip = RECT{-1 << 30, -1 << 30, w.Right, 1 << 30}
		dir = -1
	default: // 바닥(작업 표시줄) 아래에서 빼꼼
		posX = rf(float64(w.Left)+cw*0.2, float64(w.Right)-cw*1.2)
		posY = float64(w.Bottom) - ch*0.45
		hideClip = RECT{-1 << 30, -1 << 30, 1 << 30, w.Bottom}
	}
	hideBaseX = posX
}

// 그 모니터에서 머리를 숨길 만한 창 (윗변이 보이는 넓은 창)
func hideWindow(w RECT) (uintptr, RECT, bool) {
	enumList = enumList[:0]
	pEnumWindows.Call(enumCB, 0)
	var hs []uintptr
	var rs []RECT
	for _, h := range enumList {
		if !perchable(h) {
			continue
		}
		r, ok := frameRect(h)
		if !ok || r.Right-r.Left < int32(canvasW*2) || r.Bottom-r.Top < int32(canvasH) {
			continue
		}
		if r.Top < w.Top+int32(canvasH) || r.Top > w.Bottom-int32(canvasH) || r.Left < w.Left || r.Right > w.Right {
			continue
		}
		cx := (r.Left + r.Right) / 2
		at, _, _ := pWindowFromPoint.Call(uintptr(uint32(cx)) | uintptr(uint32(r.Top+4))<<32)
		if root, _, _ := pGetAncestor.Call(at, 2); root != h {
			continue
		}
		hs = append(hs, h)
		rs = append(rs, r)
		if len(hs) >= 4 {
			break
		}
	}
	if len(hs) == 0 {
		return 0, RECT{}, false
	}
	i := rng.Intn(len(hs))
	return hs[i], rs[i], true
}

// 숨은 곳 바깥(창 아래·화면 밖)은 지우기
func clipHidden(buf []byte) {
	if !hiding || hideStage < 2 {
		return
	}
	for py := 0; py < canvasH; py++ {
		sy := int32(posY) + int32(py)
		for px := 0; px < canvasW; px++ {
			sx := int32(posX) + int32(px)
			if sx < hideClip.Left || sx >= hideClip.Right || sy < hideClip.Top || sy >= hideClip.Bottom {
				o := (py*canvasW + px) * 4
				buf[o], buf[o+1], buf[o+2], buf[o+3] = 0, 0, 0, 0
			}
		}
	}
}

// found: 찾았음 / 아니면 시간이 다 돼서 나옴
func endHide(found bool) {
	if !hiding {
		return
	}
	stage := hideStage
	hiding = false
	pShowWindow.Call(hwnd, 4)
	if stage < 2 { // 숨기 전에 눌러서 그만
		setMode("idle", rd(1, 2))
		return
	}
	el := time.Since(hideFound)
	// 숨었던 자리에서 화면 안으로 폴짝
	posX = math.Max(workL, math.Min(workR-float64(canvasW), posX))
	applyMonitor()
	posX = math.Max(workL-float64(canvasW)*0.1, math.Min(workR-float64(canvasW)*0.9, posX))
	posY = math.Min(posY, ground)
	velX, velY, thrown, maxFallSpeed, bounces = 0, -300*dpi, false, 0, 0
	if !found {
		setMode("fall", time.Hour)
		greetAfterFall = "여기 있었지롱~ 다음엔 꼭 찾아 줘!"
		return
	}
	gain := 4
	switch {
	case el < 15*time.Second:
		gain = 8
	case el < 30*time.Second:
		gain = 6
	}
	S.Bananas += gain
	S.Mood = clamp(S.Mood + 15)
	S.HideCnt++
	S.PlayCnt++
	dayLog().Play++
	addExp(5)
	burst("sparkle", 8)
	sfx("found")
	setMode("fall", time.Hour)
	greetAfterFall = fmt.Sprintf("들켰다! 히히\n%d초 만에 찾았어! 바나나 %d개!", int(el.Seconds()), gain)
}

// 숨바꼭질이 끝나고 착지하면 할 말
var greetAfterFall string

// ───────── 바나나 뽑기 ─────────

const gachaCost = 10

var (
	gachaAt   time.Time
	gachaPool = []string{"starpin", "halo", "heartglasses", "scarf"}
)

func startGacha() {
	if S.Bananas < gachaCost {
		say(fmt.Sprintf("바나나가 %d개 있어야 해~", gachaCost), 2500*time.Millisecond)
		return
	}
	if busy() {
		say("지금은 못 해~ 조금 이따!", 2500*time.Millisecond)
		return
	}
	pauseToy()
	S.Bananas -= gachaCost
	S.GachaCnt++
	saveState()
	gachaAt = time.Now().Add(1800 * time.Millisecond)
	setMode("st_me", time.Hour)
	say("두구두구두구…", 1800*time.Millisecond)
	sfx("drum")
}

func gachaTick(now time.Time) {
	if gachaAt.IsZero() || now.Before(gachaAt) {
		return
	}
	gachaAt = time.Time{}
	var left []string
	for _, id := range gachaPool {
		if !owns(id) {
			left = append(left, id)
		}
	}
	r := rng.Float64()
	switch {
	case r < 0.15 && len(left) > 0:
		id := left[rng.Intn(len(left))]
		giveItem(id)
		it, _ := findItem(id)
		setMode("laugh", rd(3, 3.6))
		burst("star", 10)
		say(fmt.Sprintf("우와아! 희귀 아이템!\n'%s' 나왔다!\n바로 써 볼게!", it.name), 5*time.Second)
		toast("바나나 뽑기", it.name+" 당첨!")
		sfx("tada")
	case r < 0.20:
		S.Bananas += 50
		setMode("laugh", rd(3, 3.6))
		burst("star", 10)
		say("대박!!! 바나나 50개!", 4*time.Second)
		sfx("tada")
	case r < 0.35:
		addExp(15)
		setMode("laugh", rd(2, 2.6))
		burst("sparkle", 6)
		say("경험치 사탕! 경험치 +15", 3500*time.Millisecond)
		sfx("gold")
	case r < 0.60:
		n := 15 + rng.Intn(16)
		S.Bananas += n
		setMode("laugh", rd(2, 2.6))
		burst("sparkle", 5)
		say(fmt.Sprintf("바나나 %d개 나왔다!", n), 3*time.Second)
		sfx("coin")
	default:
		n := 5 + rng.Intn(8)
		S.Bananas += n
		setMode("shy", rd(2, 2.6))
		say(fmt.Sprintf("바나나 %d개… 다음엔 더 좋은 거!", n), 3*time.Second)
		sfx("pop")
	}
	saveState()
}

// ───────── 메뉴 ─────────

const (
	idDeskCatch = 1420
	idHide      = 1421
	idGacha     = 1422
	idFriend    = 1423
	idCatchStop = 1424
	idHideStop  = 1425
)

// 놀아주기 메뉴에 덧붙이기
func appendPlayMenu(gm uintptr) {
	appendMenu(gm, MF_STRING, idDeskCatch, "바탕화면 바나나 받기 (30초)")
	appendMenu(gm, MF_STRING, idHide, "숨바꼭질")
	f := uintptr(MF_STRING)
	if S.Bananas < gachaCost {
		f |= MF_GRAYED
	}
	appendMenu(gm, f, idGacha, fmt.Sprintf("바나나 뽑기  (바나나 %d개)", gachaCost))
	appendMenu(gm, MF_SEPARATOR, 0, "")
	if lvNow() >= friendLevel {
		f = MF_STRING
		if friendOn() {
			f |= MF_GRAYED
		}
		appendMenu(gm, f, idFriend, "친구 "+friendName+" 부르기")
	} else {
		appendMenu(gm, MF_GRAYED, 0, fmt.Sprintf("친구 %s 부르기 (Lv.%d부터)", friendName, friendLevel))
	}
}

// 놀이 중이면 메뉴 맨 위에 '그만하기'
func appendPlayStop(m uintptr) {
	if catching {
		appendMenu(m, MF_STRING, idCatchStop, "바나나 받기 그만하기")
		appendMenu(m, MF_SEPARATOR, 0, "")
	}
	if hiding {
		appendMenu(m, MF_STRING, idHideStop, "숨바꼭질 그만하기 (나와!)")
		appendMenu(m, MF_SEPARATOR, 0, "")
	}
}

func onPlayCmd(cmd uintptr) bool {
	switch cmd {
	case idDeskCatch:
		startDeskCatch()
	case idHide:
		startHide()
	case idGacha:
		startGacha()
	case idFriend:
		startFriend(true)
	case idCatchStop:
		if catching {
			endDeskCatch(true)
		}
	case idHideStop:
		endHide(false)
	default:
		return false
	}
	return true
}
