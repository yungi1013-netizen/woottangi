package main

// 친구 우땅이 '땅콩이': Lv.5 부터 가끔(1~2시간에 한 번) 놀러 와서 같이 놀다 감
//   - 화면 끝에서 걸어 들어와 인사 → 같이 웃기 · 나란히 걷기 · 술래잡기 · 같이 앉기 · 바나나 나눠 먹기 → 손 흔들고 돌아감
//   - 놀아주기 → 친구 부르기 로 바로 부를 수도 있음
//   - 땅콩이는 조금 작고 털이 밝고, 꽃 핀을 꽂고 있음. 누르면 수줍어함

import (
	"fmt"
	"math"
	"syscall"
	"time"
	"unsafe"
)

const (
	friendName  = "땅콩이"
	friendLevel = 5
	friendScale = 0.84
)

type friendT struct {
	x, y       float64
	dir        float64
	mode       string // walk idle laugh wave sit shy
	modeStart  time.Time
	modeUntil  time.Time
	stage      string // enter greet play leave
	stageUntil time.Time
	playUntil  time.Time
	act        string
	shared     bool
	L, R, gr   float64 // 땅콩이가 다니는 모니터
	speed      float64
}

var (
	fr                  *friendT
	frHwnd, frDC, frDib uintptr
	frBits              unsafe.Pointer
	frW, frH            int
	frClassDone         bool
	nextFriendAt        = time.Now().Add(rd(20*60, 40*60))
	friendFlower        *Sprite
)

func friendOn() bool { return fr != nil }

func friendWndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	if msg == WM_LBUTTONUP && fr != nil && (fr.stage == "play" || fr.stage == "greet") {
		fr.setMode("shy", 1800*time.Millisecond)
		say(pick([]string{friendName + "도 간지럼 타!", friendName + ": 헤헤 안녕~", friendName + "가 부끄럽대!"}), 2500*time.Millisecond)
		sfx("pop")
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

func (f *friendT) setMode(m string, d time.Duration) {
	f.mode, f.modeStart, f.modeUntil = m, time.Now(), time.Now().Add(d)
}

func openFriendWindow() bool {
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiFriend")
	if !frClassDone {
		wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(friendWndProc), Instance: inst, ClassName: cls}
		cur, _, _ := pLoadCursorW.Call(0, IDC_HAND)
		wc.Cursor = cur
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		frClassDone = true
	}
	frW, frH = int(float64(canvasW)*friendScale), int(float64(canvasH)*friendScale)
	if frHwnd == 0 {
		const exNoActivate = 0x08000000
		frHwnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW|exNoActivate,
			uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16(friendName))), WS_POPUP, 0, 0, uintptr(frW), uintptr(frH), 0, 0, inst, 0)
		if frHwnd == 0 {
			return false
		}
	}
	if frDC == 0 {
		sdc, _, _ := pGetDC.Call(0)
		frDC, _, _ = pCreateCompatibleDC.Call(sdc)
		pReleaseDC.Call(0, sdc)
	}
	bi := BITMAPINFOHEADER{Size: 40, Width: int32(frW), Height: -int32(frH), Planes: 1, BitCount: 32}
	var p unsafe.Pointer
	frDib, _, _ = pCreateDIBSection.Call(frDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&p)), 0, 0)
	if frDib == 0 || p == nil {
		return false
	}
	frBits = p
	pSelectObject.Call(frDC, frDib)
	if friendFlower == nil {
		if b, err := accFS.ReadFile("acc/flower.png"); err == nil {
			friendFlower = decodeSprite(b)
		}
	}
	return true
}

func closeFriend() {
	fr = nil
	if frHwnd != 0 {
		pShowWindow.Call(frHwnd, SW_HIDE)
	}
	if frDib != 0 {
		pDeleteObject.Call(frDib)
		frDib, frBits = 0, nil
	}
	modeUntil = time.Now() // 우땅이는 다시 혼자 놀기
}

// manual: 메뉴에서 불렀을 때 (안 되면 이유를 말함)
func startFriend(manual bool) bool {
	if lvNow() < friendLevel {
		if manual {
			say(fmt.Sprintf("Lv.%d가 되면 친구 %s가 놀러 올 거야!", friendLevel, friendName), 3*time.Second)
		}
		return false
	}
	if busy() || hiddenFS {
		if manual {
			say("지금은 못 불러~ 조금 이따!", 2500*time.Millisecond)
		}
		return false
	}
	if !openFriendWindow() {
		return false
	}
	detachSurface() // 바닥에서 같이 놀기
	pauseToy()
	f := &friendT{stage: "enter", speed: 1}
	f.L, f.R = workL, workR
	f.gr = ground + float64(canvasH) - float64(frH)
	f.y = f.gr
	cx := posX + float64(canvasW)/2
	if cx-workL > workR-cx { // 먼 쪽 끝에서 걸어 들어옴
		f.x, f.dir = workL-float64(frW), 1
	} else {
		f.x, f.dir = workR, -1
	}
	f.setMode("walk", time.Hour)
	fr = f
	nextFriendAt = time.Now().Add(rd(60*60, 150*60))
	if perch != 0 { // 창 위에 있으면 내려와서 맞이
		perch = 0
		maxFallSpeed, bounces = 0, 0
		setMode("fall", time.Hour)
	} else {
		setMode("st_surprise", 1800*time.Millisecond)
	}
	if manual {
		say(friendName+"야~ 놀자!", 2500*time.Millisecond)
	} else {
		say("어? 저기 누가 온다!", 2500*time.Millisecond)
	}
	return true
}

func friendTick(now time.Time, dt float64) {
	if fr == nil {
		if now.After(nextFriendAt) && lvNow() >= friendLevel && bubText == "" && ownerName != "" && !busy() && !hiddenFS &&
			(mode == "idle" || mode == "walk" || mode == "sit") {
			startFriend(false)
		}
		if now.After(nextFriendAt) && lvNow() < friendLevel {
			nextFriendAt = now.Add(30 * time.Minute)
		}
		return
	}
	if hiddenFS {
		closeFriend()
		return
	}
	f := fr
	if f.mode == "walk" {
		f.x += f.dir * friendWalk * eff() * speedMul() * f.speed * dt // 우땅이보다 조금 느긋하게
	}
	if f.mode != "walk" && now.After(f.modeUntil) {
		f.setMode("idle", time.Hour)
	}
	pc := posX + float64(canvasW)/2
	fc := f.x + float64(frW)/2
	gap := float64(canvasW) * 0.78
	switch f.stage {
	case "enter":
		if math.Abs(fc-pc) <= gap || (f.dir > 0 && fc > pc-gap) || (f.dir < 0 && fc < pc+gap) {
			f.stage, f.stageUntil = "greet", now.Add(3500*time.Millisecond)
			f.setMode("wave", 3500*time.Millisecond)
			if mode != "fall" && mode != "drag" {
				setMode("wave", 3500*time.Millisecond)
			}
			say(pick([]string{"안녕 " + friendName + "! 놀러 왔구나!", friendName + "다! 반가워~", "우와 " + friendName + "! 보고 싶었어!"}), 3500*time.Millisecond)
			sfx("found")
			S.FriendCnt++
			S.Mood = clamp(S.Mood + 15)
			dayLog().Friend++
			if S.FriendDay != today() {
				S.FriendDay = today()
				addExp(5)
			} else {
				saveState()
			}
		}
	case "greet":
		if now.After(f.stageUntil) {
			f.stage = "play"
			f.playUntil = now.Add(rd(50, 80))
			nextActivity(f, now)
		}
	case "play":
		f.keepIn()
		if f.act == "chase" { // 우땅이가 땅콩이 쫓아가기
			if mode == "walk" {
				dir = math.Copysign(1, fc-pc)
			}
		}
		if now.After(f.stageUntil) {
			if now.After(f.playUntil) {
				f.stage, f.stageUntil = "leave", now.Add(2800*time.Millisecond)
				f.setMode("wave", 2800*time.Millisecond)
				if mode != "fall" && mode != "drag" {
					setMode("wave", 2800*time.Millisecond)
				}
				say(pick([]string{"잘 가 " + friendName + "! 또 놀러 와~", "벌써 가? 다음에 또 놀자!", friendName + " 안녕~ 재밌었어!"}), 3500*time.Millisecond)
			} else {
				nextActivity(f, now)
			}
		}
	case "leave":
		if now.After(f.stageUntil) && f.mode != "walk" {
			if fc-f.L < f.R-fc {
				f.dir = -1
			} else {
				f.dir = 1
			}
			f.speed = 1.1
			f.setMode("walk", time.Hour)
			setMode("idle", rd(2, 4))
		}
		if f.mode == "walk" && (f.x < f.L-float64(frW) || f.x > f.R) {
			closeFriend()
			return
		}
	}
	renderFriend()
}

const friendWalk = 28.0 // px/s (우땅이는 42)

func speedMulBase() float64 { return []float64{1, 0.6, 1.5}[clampI(S.Speed, 0, 2)] }

// 땅콩이는 화면 안에서 (끝에 닿으면 돌아섬)
func (f *friendT) keepIn() {
	lo, hi := f.L-float64(frW)*0.1, f.R-float64(frW)*0.9
	if f.x < lo {
		f.x, f.dir = lo, 1
	}
	if f.x > hi {
		f.x, f.dir = hi, -1
	}
}

// 다음 같이 놀기
func nextActivity(f *friendT, now time.Time) {
	acts := []string{"laugh", "walk", "chase", "sit", "laugh", "walk"}
	if !f.shared {
		acts = append(acts, "share", "share")
	}
	a := pick(acts)
	for a == f.act {
		a = pick(acts)
	}
	f.act = a
	d := rd(4, 7)
	f.stageUntil = now.Add(d)
	f.speed = 1
	petOK := mode != "fall" && mode != "drag"
	switch a {
	case "laugh":
		f.setMode("laugh", d)
		if petOK {
			setMode("laugh", d)
		}
		if rng.Float64() < 0.5 {
			say(pick([]string{friendName + " 너무 웃겨~ 히히", "깔깔깔!", "우리 둘이 웃으니까 더 재밌다!"}), 2500*time.Millisecond)
		}
	case "walk":
		d2 := []float64{-1, 1}[rng.Intn(2)]
		f.dir = d2
		f.setMode("walk", time.Hour)
		if petOK {
			dir = d2
			setMode("walk", d)
		}
		if rng.Float64() < 0.4 {
			say("같이 산책하자~", 2*time.Second)
		}
	case "chase":
		pc := posX + float64(canvasW)/2
		fc := f.x + float64(frW)/2
		f.dir = math.Copysign(1, fc-pc)
		f.speed = 1.15
		f.setMode("walk", time.Hour)
		if petOK {
			dir = f.dir
			setMode("walk", d)
		}
		say(pick([]string{"잡아라~!", friendName + " 거기 서!", "술래잡기다!"}), 2*time.Second)
	case "sit":
		f.setMode("sit", d)
		if petOK {
			setMode("sit", d)
		}
		if rng.Float64() < 0.6 {
			say(pick([]string{"같이 앉아 있으니까 좋다~", friendName + "랑 멍때리기~", "날씨 좋다 그치?"}), 3*time.Second)
		}
	case "share":
		f.shared = true
		f.setMode("idle", d)
		S.Bananas += 3
		S.Full = clamp(S.Full + 10)
		saveState()
		if petOK {
			setMode("eat", d)
		}
		say(friendName+"가 바나나 나눠 줬어!\n냠냠 맛있다~ (바나나 +3)", 3500*time.Millisecond)
		sfx("coin")
	}
}

func friendFrame(f *friendT) (*Sprite, bool) {
	t := time.Since(f.modeStart).Seconds()
	pickF := func(name string, fps float64) *Sprite {
		s := seqs[name]
		return s[int(t*fps)%len(s)]
	}
	switch f.mode {
	case "walk":
		return pickF("walk", 8*f.speed), f.dir < 0 // 느린 걸음에 맞춘 발걸음
	case "laugh":
		return pickF("laugh", 12), false
	case "wave":
		return pickF("wave", 8), false
	case "sit":
		return pickF("sit", 5), false
	case "shy":
		return pickF("shy", 7), false
	}
	return pickF("idle", 5), false
}

// 땅콩이 털 색: 밝은 땅콩색 (검은 외곽선·흰 얼굴은 그대로)
func friendTint(c *[4]float32) {
	a := c[3]
	if a < 1 {
		return
	}
	b, g, r := c[0]/a*255, c[1]/a*255, c[2]/a*255
	lum := 0.3*r + 0.59*g + 0.11*b
	sat := r - b
	w := float32(math.Max(0, math.Min(1, float64((lum-55)/70)))) * float32(math.Max(0, math.Min(1, float64(sat-25)/40)))
	if w <= 0 {
		return
	}
	tr, tg, tb := r*0.64+255*0.36, g*0.64+208*0.36, b*0.64+140*0.36
	r, g, b = r+(tr-r)*w, g+(tg-g)*w, b+(tb-b)*w
	k := a / 255
	c[0], c[1], c[2] = b*k, g*k, r*k
}

func renderFriend() {
	if fr == nil || frBits == nil {
		return
	}
	buf := unsafe.Slice((*byte)(frBits), frW*frH*4)
	clear(buf)
	s, flip := friendFrame(fr)
	base := float64(frH) * 0.98 / float64(s.H)
	ax, ay := float64(frW)/2, float64(frH)-1
	sw, sh := float64(s.W), float64(s.H)
	for py := 0; py < frH; py++ {
		v := (float64(py) + 0.5 - ay) / base
		sv := v + sh - 0.5
		if sv < -1 || sv >= sh {
			continue
		}
		for px := 0; px < frW; px++ {
			u := (float64(px) + 0.5 - ax) / base
			if flip {
				u = -u
			}
			su := u + sw/2 - 0.5
			if su < -1 || su >= sw {
				continue
			}
			acc := sample(s, su, sv)
			friendTint(&acc)
			if friendFlower != nil && s.Anc != nil {
				drawLayer(s, friendFlower, su+0.5, sv+0.5, &acc)
			}
			if acc[3] < 1 {
				continue
			}
			o := (py*frW + px) * 4
			buf[o], buf[o+1], buf[o+2], buf[o+3] = clampB(acc[0]), clampB(acc[1]), clampB(acc[2]), clampB(acc[3])
		}
	}
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{int32(fr.x), int32(fr.y)}
	sz := SIZE{int32(frW), int32(frH)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, opacity(), AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(frHwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		frDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
	pShowWindow.Call(frHwnd, 4)
}
