package main

// 움직임 더하기
//   - 던지기: 끌다가 휙 놓으면 날아가서 벽·천장에 튕기고, 바닥에 통통 튀다가 착지
//   - 창 위에 앉기: 떨어지다 창 제목줄에 닿으면 그 위에 서고, 창을 옮기면 같이 움직임
//   - 마우스 반응: 커서가 가까이 오래 있으면 쳐다보고 수줍어하거나 손 흔들기, 가끔 커서 쪽으로 걸어감
//   - 전체화면(게임·영상·발표)일 때 숨기
//   - 기분·배고픔·밤에 따라 걷는 속도와 행동이 달라짐

import (
	"math"
	"syscall"
	"time"
	"unsafe"
)

var (
	dwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	pDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
	pEnumWindows           = user32.NewProc("EnumWindows")
	pIsWindow              = user32.NewProc("IsWindow")
	pIsIconic              = user32.NewProc("IsIconic")
	pIsZoomed              = user32.NewProc("IsZoomed")
	pGetWindowRect         = user32.NewProc("GetWindowRect")
	pGetWindowTextLengthW  = user32.NewProc("GetWindowTextLengthW")
	pGetClassNameW         = user32.NewProc("GetClassNameW")
	pWindowFromPoint       = user32.NewProc("WindowFromPoint")
	pGetAncestor           = user32.NewProc("GetAncestor")
	pGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	pMonitorFromWindow     = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfoW       = user32.NewProc("GetMonitorInfoW")
	enumCB                 = syscall.NewCallback(enumProc)
	enumList               []uintptr
)

var (
	velX         float64
	workT        float64
	thrown       bool    // 던져서 날아가는 중
	maxFallSpeed float64 // 이번 낙하에서 가장 빨랐던 속도 (착지 반응용)
	bounces      int
	dragHist     [8]struct {
		x, y float64
		t    time.Time
	}
	dragN           int
	perch           uintptr // 올라앉은 창 (0 = 바닥)
	perchR          RECT
	hiddenFS        bool // 전체화면 때문에 숨은 상태
	lastFSCheck     time.Time
	cursorNearSince time.Time
	lastCursorReact = time.Now()
	perchEnabled    = true // 테스트에서는 꺼서 실제 화면의 창에 걸리지 않게
)

func feetOff() float64 { return float64(canvasH) - 2 } // posY + feetOff() = 발 위치

// 지금 서 있어야 할 높이
func groundY() float64 {
	if airPerch { // 공중 자리 (climb.go)
		return airY
	}
	if perch != 0 {
		return float64(perchR.Top) - feetOff()
	}
	return ground
}

// 걷는 속도 배율: 설정 × (기분이 낮거나 배고프면 터벅터벅)
func speedMul() float64 {
	m := []float64{1, 0.6, 1.5}[clampI(S.Speed, 0, 2)]
	if S.Mood < 30 || S.Full < 20 {
		m *= 0.6
	}
	if sleepy() {
		m *= 0.75
	}
	if batteryLow() { // 배터리가 없으면 기운도 없음
		m *= 0.7
	}
	return m
}

func clampI(v, lo, hi int) int { return max(lo, min(hi, v)) }

// ── 끌기 · 던지기 ──

func dragTick(now time.Time) {
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	posX = float64(c.X - downPt.X + downWin.X)
	posY = float64(c.Y - downPt.Y + downWin.Y)
	dragHist[dragN%len(dragHist)] = struct {
		x, y float64
		t    time.Time
	}{posX, posY, now}
	dragN++
}

// 놓기 직전 0.12초 동안의 움직임으로 던지는 속도 계산
func releaseVelocity() (vx, vy float64) {
	if dragN < 2 {
		return 0, 0
	}
	last := dragHist[(dragN-1)%len(dragHist)]
	first := last
	for i := 2; i <= min(dragN, len(dragHist)); i++ {
		h := dragHist[(dragN-i)%len(dragHist)]
		if last.t.Sub(h.t) > 120*time.Millisecond {
			break
		}
		first = h
	}
	dt := last.t.Sub(first.t).Seconds()
	if dt < 0.015 {
		return 0, 0
	}
	lim := 3200 * dpi
	return math.Max(-lim, math.Min(lim, (last.x-first.x)/dt)), math.Max(-lim, math.Min(lim, (last.y-first.y)/dt))
}

func startDrag() {
	perch = 0
	q, airPerch, wantClimb, wantCeil, seatJump = 0, false, false, false, false // 창은 정사각형이라 자리 그대로
	dragN = 0
	setMode("drag", time.Hour)
}

func onDragRelease() {
	velX, velY = releaseVelocity()
	if math.Hypot(velX, velY) < 160*dpi && placeGently() { // 살살 놓기 (climb.go)
		velX, velY = 0, 0
		return
	}
	thrown = math.Hypot(velX, velY) > 500*dpi
	if !thrown {
		velX, velY = 0, 0
	}
	maxFallSpeed, bounces = 0, 0
	setMode("fall", time.Hour)
}

// ── 떨어지기 · 날기 ──

func fallTick(dt float64) {
	prevFeet := posY + feetOff()
	vy0 := velY
	velY += 1800 * dpi * dt
	if !seatJump {
		velX *= 1 - 0.5*dt // 공기 저항
	}
	posX += velX * dt
	posY += (vy0 + velY) / 2 * dt // 중력 그대로 (프레임이 길어도 높이를 덜 잃음)
	maxFallSpeed = math.Max(maxFallSpeed, velY)

	lo, hi := floorBounds()
	if posX < lo {
		posX, velX = lo, -velX*0.6
		dir = 1
	}
	if posX > hi {
		posX, velX = hi, -velX*0.6
		dir = -1
	}
	if top := workT - float64(canvasH)*0.25; posY < top {
		if thrown && !seatJump && rng.Float64() < ceilGrabChance { // 천장 탁 잡기
			d := 1.0
			if velX < 0 {
				d = -1
			}
			thrown, maxFallSpeed, bounces = false, 0, 0
			toCeiling(posX, d)
			setMode("cling", rd(1.5, 3))
			say(pick([]string{"천장 잡았다!", "매달리기 성공!"}), 2500*time.Millisecond)
			return
		}
		posY, velY = top, -velY*0.4
	}
	if seatLand() { // 내 자리 도착
		return
	}
	if velY > 0 && perchEnabled && !seatJump {
		if w, r, ok := findPerch(prevFeet, posY+feetOff()); ok {
			perch, perchR = w, r
			posY = groundY()
			land()
			return
		}
	}
	if posY >= ground {
		posY = ground
		perch = 0
		seatJump = false
		land()
	}
}

func land() {
	if velY > 900*dpi && bounces < 2 { // 세게 떨어지면 통통
		velY = -velY * 0.35
		velX *= 0.7
		bounces++
		posY -= 1
		return
	}
	hard := thrown && maxFallSpeed > 1300*dpi
	velX, velY, bounces, thrown, maxFallSpeed = 0, 0, 0, false, 0
	switch {
	case !greeted:
		greeted = true
		setMode("st_bow", 2200*time.Millisecond)
	case greetAfterFall != "": // 숨바꼭질 끝
		setMode("laugh", rd(2.5, 3))
		say(greetAfterFall, 4*time.Second)
		greetAfterFall = ""
	case hard:
		sfx("boing")
		dayLog().Throw++
		addDirt(5)
		setMode("st_surprise", 1800*time.Millisecond)
		say(pick([]string{"어지러워~ 빙글빙글", "우와 날았다!", "살살 던져 줘~", "착지 성공! 히히"}), 2500*time.Millisecond)
	case perch != 0 && rng.Float64() < 0.5:
		setMode("wave", rd(2, 3))
		say(pick([]string{"여기 경치 좋다~", "창문 위에 올라왔어!", "높다 높아~"}), 2500*time.Millisecond)
	default:
		setMode("idle", rd(1, 2))
	}
}

// ── 창 위에 앉기 ──

func enumProc(h, _ uintptr) uintptr {
	enumList = append(enumList, h)
	return 1
}

var shellClasses = map[string]bool{"Progman": true, "WorkerW": true, "Shell_TrayWnd": true, "Shell_SecondaryTrayWnd": true}

func className(h uintptr) string {
	buf := make([]uint16, 128)
	pGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), 128)
	return syscall.UTF16ToString(buf)
}

func cloaked(h uintptr) bool {
	var v uint32
	r, _, _ := pDwmGetWindowAttribute.Call(h, 14, uintptr(unsafe.Pointer(&v)), 4) // DWMWA_CLOAKED
	return r == 0 && v != 0
}

// 보이는 창 테두리 (그림자·투명 테두리 제외)
func frameRect(h uintptr) (RECT, bool) {
	var r RECT
	if res, _, _ := pDwmGetWindowAttribute.Call(h, 9, uintptr(unsafe.Pointer(&r)), unsafe.Sizeof(r)); res == 0 { // DWMWA_EXTENDED_FRAME_BOUNDS
		return r, true
	}
	ok, _, _ := pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&r)))
	return r, ok != 0
}

func isOurs(h uintptr) bool {
	return h == hwnd || h == bubHwnd || h == nameHwnd || h == remHwnd || (fxHwnd != 0 && h == fxHwnd) || (frHwnd != 0 && h == frHwnd) || (memoHwnd != 0 && h == memoHwnd) || isThing(h)
}

func perchable(h uintptr) bool {
	if isOurs(h) {
		return false
	}
	if v, _, _ := pIsWindowVisible.Call(h); v == 0 {
		return false
	}
	if v, _, _ := pIsIconic.Call(h); v != 0 {
		return false
	}
	if v, _, _ := pIsZoomed.Call(h); v != 0 {
		return false
	}
	if n, _, _ := pGetWindowTextLengthW.Call(h); n == 0 {
		return false
	}
	return !cloaked(h) && !shellClasses[className(h)]
}

// 발이 prevFeet → feet 로 내려오는 사이에 지나친 창 윗변 (가장 위에 보이는 창)
func findPerch(prevFeet, feet float64) (uintptr, RECT, bool) {
	cx := posX + float64(canvasW)/2
	enumList = enumList[:0]
	pEnumWindows.Call(enumCB, 0) // 화면 위쪽(Z 순서) 창부터
	for _, h := range enumList {
		if !perchable(h) {
			continue
		}
		r, ok := frameRect(h)
		if !ok {
			continue
		}
		top := float64(r.Top)
		w := float64(r.Right - r.Left)
		if w < float64(canvasW)*1.3 || top < workT+float64(canvasH)*0.5 || top > ground+feetOff()-20 {
			continue
		}
		if cx < float64(r.Left)+float64(canvasW)*0.2 || cx > float64(r.Right)-float64(canvasW)*0.2 {
			continue
		}
		if prevFeet > top+1 || feet < top {
			continue
		}
		// 그 자리 제목줄이 다른 창에 가려져 있지 않은지
		at, _, _ := pWindowFromPoint.Call(uintptr(uint32(int32(cx))) | uintptr(uint32(int32(top+4)))<<32) // POINT 를 값으로
		if root, _, _ := pGetAncestor.Call(at, 2); root != h {                                            // GA_ROOT
			continue
		}
		return h, r, true
	}
	return 0, RECT{}, false
}

// 올라앉은 창이 움직이면 같이 움직이고, 사라지면 떨어짐
func perchTick() {
	if perch == 0 {
		return
	}
	if v, _, _ := pIsWindow.Call(perch); v == 0 || !perchable(perch) {
		perch = 0
		return
	}
	r, ok := frameRect(perch)
	if !ok {
		perch = 0
		return
	}
	if r != perchR {
		posX += float64(r.Left - perchR.Left)
		posY += float64(r.Top - perchR.Top)
		perchR = r
		if float64(r.Top) < workT+float64(canvasH)*0.3 || float64(r.Top) > ground+feetOff() {
			perch = 0 // 화면 밖으로 밀려남
		}
	}
}

// 걷기 범위: 바닥이면 화면 끝, 창 위면 창 끝 (끝에 닿으면 가끔 뛰어내림)
func walkBounds() (lo, hi float64) {
	if perch != 0 {
		return float64(perchR.Left) - float64(canvasW)*0.3, float64(perchR.Right) - float64(canvasW)*0.7
	}
	return floorBounds()
}

func walkTick(dt float64) {
	posX += dir * 42 * eff() * dt * speedMul() * walkBoost()
	lo, hi := walkBounds()
	if posX >= lo && posX <= hi {
		return
	}
	if perch == 0 && tryClimbAtEdge(posX < lo) { // 화면 끝 벽 타기 (climb.go)
		return
	}
	if perch != 0 && rng.Float64() < 0.35 { // 창 끝에서 폴짝
		perch = 0
		velX, velY = dir*140*dpi, -380*dpi
		maxFallSpeed, bounces = 0, 0
		setMode("fall", time.Hour)
		return
	}
	if posX < lo {
		posX, dir = lo, 1
	} else {
		posX, dir = hi, -1
	}
}

// ── 마우스 반응 ──

func cursorTick(now time.Time) {
	switch mode {
	case "idle", "walk", "sit":
	default:
		cursorNearSince = time.Time{}
		return
	}
	if focusing() || hiddenFS {
		return
	}
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	cx, cy := posX+float64(canvasW)/2, posY+float64(canvasH)*0.45
	if math.Hypot(float64(c.X)-cx, float64(c.Y)-cy) > float64(canvasW)*0.9 {
		cursorNearSince = time.Time{}
		return
	}
	if cursorNearSince.IsZero() {
		cursorNearSince = now
		return
	}
	if now.Sub(cursorNearSince) < 700*time.Millisecond || now.Sub(lastCursorReact) < 25*time.Second {
		return
	}
	lastCursorReact = now
	if float64(c.X) < cx {
		dir = -1
	} else {
		dir = 1
	}
	switch r := rng.Float64(); {
	case r < 0.5:
		setMode("shy", rd(2.5, 3.5))
		if rng.Float64() < 0.6 {
			say(pick([]string{"헤헤 뭐야~", "간지러워~", "나 보고 있었어?"}), 2500*time.Millisecond)
		}
	case r < 0.8:
		setMode("wave", rd(2, 3))
		if rng.Float64() < 0.6 {
			say("안녕 "+voc()+"!", 2500*time.Millisecond)
		}
	default:
		setMode("st_surprise", 1800*time.Millisecond)
		say("앗 깜짝이야!", 2*time.Second)
	}
}

// 걷기 시작할 때 가끔 커서 쪽으로 (호기심)
func maybeFaceCursor() {
	if !cursorOn || rng.Float64() > 0.25 {
		return
	}
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	cx := posX + float64(canvasW)/2
	if math.Abs(float64(c.X)-cx) < 200*dpi || float64(c.Y) < workT+(ground-workT)*0.5 {
		return
	}
	if float64(c.X) < cx {
		dir = -1
	} else {
		dir = 1
	}
}

// ── 전체화면일 때 숨기 ──

type MONITORINFO struct {
	Size          uint32
	Monitor, Work RECT
	Flags         uint32
}

func fullscreenNow() bool {
	fg, _, _ := pGetForegroundWindow.Call()
	if fg == 0 || isOurs(fg) || shellClasses[className(fg)] {
		return false
	}
	mon, _, _ := pMonitorFromWindow.Call(fg, 2) // MONITOR_DEFAULTTONEAREST
	if mine, _, _ := pMonitorFromWindow.Call(hwnd, 2); mine != mon {
		return false
	}
	mi := MONITORINFO{Size: uint32(unsafe.Sizeof(MONITORINFO{}))}
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	var r RECT
	pGetWindowRect.Call(fg, uintptr(unsafe.Pointer(&r)))
	return r.Left <= mi.Monitor.Left && r.Top <= mi.Monitor.Top && r.Right >= mi.Monitor.Right && r.Bottom >= mi.Monitor.Bottom
}

func fsTick(now time.Time) {
	if now.Sub(lastFSCheck) < time.Second {
		return
	}
	lastFSCheck = now
	on := !S.NoAutoHide && fullscreenNow()
	if on == hiddenFS {
		return
	}
	hiddenFS = on
	if on {
		pShowWindow.Call(hwnd, SW_HIDE)
		pShowWindow.Call(bubHwnd, SW_HIDE)
		bubText = ""
	} else {
		pShowWindow.Call(hwnd, 4) // SW_SHOWNOACTIVATE
	}
}

// ── 기분 · 밤 ──

// nextChoice 의 기본 선택 전에: 밤엔 졸리고, 시무룩하면 뒹굴 (바꿨으면 true)
func moodChoice() bool {
	if h := time.Now().Hour(); h < 6 && rng.Float64() < 0.45 {
		if rng.Float64() < 0.6 {
			setMode("sleep", rd(15, 30))
		} else {
			setMode("sit", rd(8, 14))
		}
		return true
	}
	if sleepy() && rng.Float64() < 0.35 { // 졸리면 꾸벅꾸벅
		setMode(pick([]string{"sleep", "sit"}), rd(10, 18))
		return true
	}
	if (S.Mood < 30 || S.Full < 20) && rng.Float64() < 0.35 {
		setMode(pick([]string{"st_lounge", "sit"}), rd(6, 10))
		return true
	}
	return false
}
