package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

//go:embed frames/*.png
var frameFS embed.FS

// ───────────────────────── Win32 ─────────────────────────
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	pRegisterClassExW      = user32.NewProc("RegisterClassExW")
	pCreateWindowExW       = user32.NewProc("CreateWindowExW")
	pDefWindowProcW        = user32.NewProc("DefWindowProcW")
	pShowWindow            = user32.NewProc("ShowWindow")
	pGetMessageW           = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessageW      = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW      = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pSetTimer              = user32.NewProc("SetTimer")
	pGetDC                 = user32.NewProc("GetDC")
	pReleaseDC             = user32.NewProc("ReleaseDC")
	pUpdateLayeredWindow   = user32.NewProc("UpdateLayeredWindow")
	pSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	pGetCursorPos          = user32.NewProc("GetCursorPos")
	pSetCapture            = user32.NewProc("SetCapture")
	pReleaseCapture        = user32.NewProc("ReleaseCapture")
	pCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	pAppendMenuW           = user32.NewProc("AppendMenuW")
	pTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	pDestroyMenu           = user32.NewProc("DestroyMenu")
	pSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	pPostMessageW          = user32.NewProc("PostMessageW")
	pLoadCursorW           = user32.NewProc("LoadCursorW")
	pSetProcessDPIAware    = user32.NewProc("SetProcessDPIAware")
	pSendMessageW          = user32.NewProc("SendMessageW")
	pGetWindowTextW        = user32.NewProc("GetWindowTextW")
	pSetWindowTextW        = user32.NewProc("SetWindowTextW")
	pSetFocus              = user32.NewProc("SetFocus")
	pMoveWindow            = user32.NewProc("MoveWindow")
	pGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	pMessageBoxW           = user32.NewProc("MessageBoxW")
	pLoadImageW            = user32.NewProc("LoadImageW")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pCreateFontW        = gdi32.NewProc("CreateFontW")

	pGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW       = kernel32.NewProc("CreateMutexW")
	pGetModuleFileNameW = kernel32.NewProc("GetModuleFileNameW")

	pRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	pRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	pRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

const (
	WS_EX_LAYERED       = 0x00080000
	WS_EX_TOPMOST       = 0x00000008
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_DLGMODALFRAME = 0x00000001
	WS_POPUP            = 0x80000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	ES_AUTOHSCROLL      = 0x0080
	BS_DEFPUSHBUTTON    = 0x0001
	SW_SHOW             = 5
	SW_HIDE             = 0

	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_TIMER         = 0x0113
	WM_COMMAND       = 0x0111
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_MOUSEMOVE     = 0x0200
	WM_SETFONT       = 0x0030
	WM_KEYDOWN       = 0x0100
	EM_SETSEL        = 0x00B1

	MF_STRING       = 0x0000
	MF_POPUP        = 0x0010
	MF_SEPARATOR    = 0x0800
	MF_GRAYED       = 0x0001
	MF_CHECKED      = 0x0008
	TPM_RETURNCMD   = 0x0100
	TPM_RIGHTBUTTON = 0x0002
	TPM_BOTTOMALIGN = 0x0020

	ULW_ALPHA         = 0x02
	AC_SRC_OVER       = 0x00
	AC_SRC_ALPHA      = 0x01
	SPI_GETWORKAREA   = 0x0030
	IDC_HAND          = 32649
	IDC_ARROW         = 32512
	HKEY_CURRENT_USER = 0x80000001
	KEY_ALL_ACCESS    = 0xF003F
	REG_SZ            = 1
	SM_CXSCREEN       = 0
	SM_CYSCREEN       = 1
	IMAGE_ICON        = 1
	LR_DEFAULTSIZE    = 0x40
	LR_SHARED         = 0x8000
)

type WNDCLASSEX struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type POINT struct{ X, Y int32 }
type SIZE struct{ CX, CY int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}
type BITMAPINFOHEADER struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}
type BLENDFUNCTION struct{ Op, Flags, Alpha, Format byte }

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

var dpi = 1.0
var pGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")

func eff() float64 { return scale * dpi }

func initDPI() {
	sdc, _, _ := pGetDC.Call(0)
	v, _, _ := pGetDeviceCaps.Call(sdc, 90) // LOGPIXELSY
	pReleaseDC.Call(0, sdc)
	if v >= 96 {
		dpi = float64(v) / 96
	}
}

// ───────────────────────── 프레임 ─────────────────────────
type Sprite struct {
	W, H int
	Pix  []uint8   // premultiplied BGRA (1바이트씩: 프레임 128장 ≈ 150MB)
	Seq  string    // 동작 이름 (프레임일 때)
	Anc  *anchorRT // 머리 위치 (꾸미기 아이템용, 없으면 nil)
}

var seqs = map[string][]*Sprite{}

func decodeSprite(b []byte) *Sprite {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	r := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, r.Min, draw.Src)
	sp := &Sprite{W: r.Dx(), H: r.Dy(), Pix: make([]uint8, r.Dx()*r.Dy()*4)}
	for p := 0; p < r.Dx()*r.Dy(); p++ {
		R, G, B, A := float32(rgba.Pix[p*4]), float32(rgba.Pix[p*4+1]), float32(rgba.Pix[p*4+2]), rgba.Pix[p*4+3]
		k := float32(A) / 255
		sp.Pix[p*4], sp.Pix[p*4+1], sp.Pix[p*4+2], sp.Pix[p*4+3] = uint8(B*k+0.5), uint8(G*k+0.5), uint8(R*k+0.5), A
	}
	return sp
}

func loadSeq(name string) {
	seqs[name] = nil
	for k := 0; ; k++ {
		b, err := frameFS.ReadFile(fmt.Sprintf("frames/%s_%d.png", name, k))
		if err != nil {
			break
		}
		seqs[name] = append(seqs[name], decodeSprite(b))
	}
}

// 쌍선형 샘플 (su, sv: 픽셀 중심이 정수인 좌표)
func sample(s *Sprite, su, sv float64) (acc [4]float32) {
	if su < -1 || sv < -1 || su >= float64(s.W) || sv >= float64(s.H) {
		return
	}
	ix, iy := int(math.Floor(su)), int(math.Floor(sv))
	ffx, ffy := float32(su-float64(ix)), float32(sv-float64(iy))
	for k := 0; k < 4; k++ {
		cx, cy := ix+(k&1), iy+(k>>1)
		if cx < 0 || cy < 0 || cx >= s.W || cy >= s.H {
			continue
		}
		var w float32
		switch k {
		case 0:
			w = (1 - ffx) * (1 - ffy)
		case 1:
			w = ffx * (1 - ffy)
		case 2:
			w = (1 - ffx) * ffy
		default:
			w = ffx * ffy
		}
		o := (cy*s.W + cx) * 4
		acc[0] += float32(s.Pix[o]) * w
		acc[1] += float32(s.Pix[o+1]) * w
		acc[2] += float32(s.Pix[o+2]) * w
		acc[3] += float32(s.Pix[o+3]) * w
	}
	return
}

// ───────────────────────── 상태 ─────────────────────────
var (
	hwnd             uintptr
	memDC, dib       uintptr
	bits             unsafe.Pointer
	canvasW, canvasH int

	scale        = 0.75
	posX, posY   float64
	dir          = 1.0
	velY         float64
	ground       float64
	workL, workR float64

	mode      = "fall" // walk idle laugh sit sleep drag fall
	modeUntil time.Time
	modeStart = time.Now()
	blinkAt   = time.Now().Add(3 * time.Second)
	blinkTill time.Time
	stay      = false // 가만히 있기
	autoMove  = true

	downPt, downWin   POINT
	pressed, dragging bool
	lastTick          = time.Now()
	lastArea          time.Time
	rng               = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// 버전: build.py 의 VERSION 이 -ldflags "-X main.appVersion=..." 로 들어옴
var appVersion = "1.2"

func setMode(m string, d time.Duration) {
	surfaceModeHook(m) // 벽·천장·자리에서 못 하는 동작이면 내려옴 (climb.go)
	mode = m
	modeStart = time.Now()
	modeUntil = time.Now().Add(d)
}

func rd(a, b float64) time.Duration {
	return time.Duration((a + rng.Float64()*(b-a)) * float64(time.Second))
}

func nextChoice() {
	if catching || hiding || !gachaAt.IsZero() { // 놀이가 끝낼 때까지
		modeUntil = time.Now().Add(time.Hour)
		return
	}
	if friendOn() { // 땅콩이가 정해 줌 (friend.go)
		setMode("idle", time.Hour)
		return
	}
	if toyActive() && petFree() { // 장난감 놀이: toyTick 이 정해 줌 (toy.go)
		if mode == "idle" {
			modeUntil = time.Now().Add(2 * time.Second)
		} else {
			setMode("idle", 2*time.Second)
		}
		return
	}
	if bathing { // 목욕은 bathTick 이 끝냄
		modeUntil = time.Now().Add(time.Hour)
		return
	}
	if mode == "sleep" && tucked { // 이불 덮고 푹 자고 일어남
		wakeTucked()
		return
	}
	if focusing() { // 집중 타이머: 옆에 조용히 앉아 있기 (반응 동작이 끝나면 다시 앉음)
		switch mode {
		case "sit", "sleep", "idle", "walk", "laugh", "shy", "wave", "st_me", "st_surprise", "st_lounge":
			setMode("sit", rd(20, 40))
			return
		}
		if isNewEmo(mode) {
			setMode("sit", rd(20, 40))
			return
		}
	}
	if isNewEmo(mode) { // 이모티콘 동작이 끝나면 잠깐 서 있기
		setMode("idle", rd(1.5, 3))
		return
	}
	switch mode {
	case "sleep":
		setMode("sit", rd(2, 4)) // 깨면 잠깐 앉아 있다가
		return
	case "sit":
		r := rng.Float64()
		if r < 0.35 {
			setMode("sleep", rd(12, 25))
		} else if r < 0.55 {
			setMode("sit", rd(4, 8))
		} else {
			setMode("idle", rd(1.5, 3))
		}
		return
	case "st_cry":
		setMode("punish", rd(4.5, 5.5))
		return
	case "st_bow":
		if !quitting {
			setMode("wave", rd(2.5, 3.2)) // 꾸벅 인사 후 손 흔들기
		}
		return
	case "eat", "eat_milk", "eat_cake":
		if rng.Float64() < 0.35 { // 다 먹고 가끔 "잘 먹었어요!"
			playEmo("yum", true)
		} else {
			setMode("idle", rd(1.5, 3))
		}
		return
	case "laugh", "shy", "st_surprise", "st_lounge", "st_me", "wave":
		setMode("idle", rd(1.5, 3))
		return
	case "punish":
		setMode("idle", rd(2, 3))
		if ownerName != "" {
			say("다시는 안 그럴게… "+voc()+"…", 3*time.Second)
		}
		return
	}
	if airPerch { // 공중 자리 위: 앉고 쉬고 졸기만 (climb.go)
		perchedChoice()
		return
	}
	if wantClimb { // 벽 타러 가는 중
		setMode("walk", 30*time.Second)
		return
	}
	if stay || !autoMove {
		setMode("idle", rd(3, 5))
		return
	}
	if hasSeat() && rng.Float64() < 0.05 && jumpToSeat() {
		return
	}
	if moodChoice() {
		return
	}
	r := rng.Float64()
	switch {
	case mode == "walk" && r < 0.45:
		setMode("idle", rd(2, 4))
	case r < 0.08:
		setMode("st_lounge", rd(5, 9))
	case r < 0.12:
		setMode("wave", rd(2.5, 3.5))
	case r < 0.18:
		setMode("laugh", rd(1.6, 2.6))
	case r < 0.21 && S.Mood >= 60: // 기분 좋으면 가끔 춤
		setMode("dance", rd(3.2, 4.2))
	case r < 0.23:
		setMode("huh", rd(2.4, 3))
	case r < 0.25: // 가끔 브이 · 노래 · 뭐해?
		setMode(pick([]string{"v", "sing", "what"}), rd(2.6, 3.4))
	case r < 0.32:
		setMode("sit", rd(6, 12))
	default:
		if rng.Float64() < 0.4 {
			dir = -dir
		}
		maybeFaceCursor()
		setMode("walk", rd(4, 9))
	}
}

// 모니터 목록을 다시 읽고 지금 모니터의 바닥·벽 정하기 (monitor.go)
func refreshWorkArea() {
	scanMonitors()
	applyMonitor()
}

func setupCanvas() {
	if dib != 0 {
		pDeleteObject.Call(dib)
	}
	if memDC == 0 {
		sdc, _, _ := pGetDC.Call(0)
		memDC, _, _ = pCreateCompatibleDC.Call(sdc)
		pReleaseDC.Call(0, sdc)
	}
	canvasW = int(150 * eff())
	canvasH = int(150 * eff())
	bi := BITMAPINFOHEADER{Size: 40, Width: int32(canvasW), Height: -int32(canvasH), Planes: 1, BitCount: 32}
	var p unsafe.Pointer
	dib, _, _ = pCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&p)), 0, 0)
	bits = p
	pSelectObject.Call(memDC, dib)
}

// 현재 보여줄 프레임
func currentFrame() (*Sprite, float64, bool) {
	t := time.Since(modeStart).Seconds()
	now := time.Now()
	pick := func(name string, fps float64) *Sprite {
		f := seqs[name]
		return f[int(t*fps)%len(f)]
	}
	blinking := now.Before(blinkTill)
	if now.After(blinkAt) {
		blinkTill = now.Add(140 * time.Millisecond)
		blinkAt = now.Add(rd(2.5, 6))
	}
	switch mode {
	case "walk":
		return pick("walk", 11), 0, dir < 0
	case "climb": // 벽·천장 (render 가 q 만큼 돌림)
		return pick("walk", 9), 0, climbFlip()
	case "cling":
		if blinking {
			return seqs["blink"][0], 0, climbFlip()
		}
		return pick("idle", 5), 0, climbFlip()
	case "laugh":
		return pick("laugh", 12), 0, false
	case "eat", "eat_milk", "eat_cake":
		return pick(mode, 6), 0, false
	case "wave":
		return pick("wave", 8), 0, false
	case "st_surprise", "st_lounge", "st_cry", "st_me", "st_bow":
		fps := 9.0
		if mode == "st_lounge" {
			fps = 4
		}
		if mode == "st_bow" {
			fps = 5
		}
		return pick(mode, fps), 0, false
	case "shy":
		return pick("shy", 7), 0, false
	case "clap", "cheer", "heart", "angry", "dance", "huh", "v", "thumb", "sing", "no", "flower", "yum", "what", "cute":
		return pick(mode, emoOf(mode).fps), 0, false
	case "punish":
		return pick("punish", 8), 0, false
	case "sit":
		if blinking {
			return seqs["sitblink"][0], 0, false
		}
		return pick("sit", 5), 0, false
	case "sleep":
		return pick("sleep", 6), 0, false
	case "bath": // 문질러 주면 헤헤, 아니면 가만히
		if time.Since(bathLastRub) < 500*time.Millisecond {
			return pick("shy", 7), 0, false
		}
		return pick("idle", 5), 0, false
	case "catch": // 바탕화면 바나나 받기
		n, fps := catchFrameSeq()
		return pick(n, fps), 0, n == "walk" && dir < 0
	case "hide": // 숨바꼭질 (빼꼼)
		if blinking {
			return seqs["blink"][0], 0, dir < 0
		}
		return pick("idle", 4), 0, dir < 0
	case "drag":
		return seqs["idle"][0], math.Sin(t*6) * 0.22, false
	case "fall":
		return seqs["idle"][0], math.Max(-0.5, math.Min(0.5, velX/(2600*dpi))), false // 날아갈 땐 기울어짐
	}
	if blinking {
		return seqs["blink"][0], 0, false
	}
	return pick("idle", 5), 0, false
}

func render() {
	n := canvasW * canvasH
	buf := unsafe.Slice((*byte)(bits), n*4)
	for i := range buf {
		buf[i] = 0
	}
	s, rot, flip := currentFrame()
	dressed := s.Anc != nil && (worn[slotHead] != nil || worn[slotNeck] != nil || ((wornFace || dirtyShown()) && !noFaceSeq[s.Seq]))
	blanket := tucked && mode == "sleep"
	base := eff() * float64(canvasH) * 0.98 / float64(s.H) / eff()
	sx, sy := base, base
	ax := float64(canvasW) / 2
	ay := float64(canvasH) - 1
	if mode == "drag" {
		ay -= 6 * eff()
	}
	cs, sn := math.Cos(rot), math.Sin(rot)
	sw, sh := float64(s.W), float64(s.H)
	hx, hy := float64(canvasW)/2, float64(canvasH)/2
	for py := 0; py < canvasH; py++ {
		for px := 0; px < canvasW; px++ {
			// 벽·천장: 화면 픽셀을 거꾸로 돌려서 세워진 그림 좌표로
			dx, dy := float64(px)+0.5-hx, float64(py)+0.5-hy
			switch q {
			case 1:
				dx, dy = dy, -dx
			case 2:
				dx, dy = -dx, -dy
			case 3:
				dx, dy = -dy, dx
			}
			fx, fy := dx+hx-ax, dy+hy-ay
			u := (fx*cs + fy*sn) / sx
			v := (-fx*sn + fy*cs) / sy
			if flip {
				u = -u
			}
			su := u + sw/2 - 0.5
			sv := v + sh - 0.5
			if su < -1 || sv < -1 || su >= sw || sv >= sh {
				continue
			}
			acc := sample(s, su, sv)
			if dressed {
				drawWorn(s, su+0.5, sv+0.5, &acc)
			}
			if blanket {
				drawBlanket(s, su, sv, &acc)
			}
			if acc[3] < 1 {
				continue
			}
			o := (py*canvasW + px) * 4
			buf[o], buf[o+1], buf[o+2], buf[o+3] = clampB(acc[0]), clampB(acc[1]), clampB(acc[2]), clampB(acc[3])
		}
	}
	drawMemoNote(buf)
	drawPetFx(buf)
	clipHidden(buf)
	present()
}

func tick() {
	now := time.Now()
	dt := now.Sub(lastTick).Seconds()
	if dt > 0.1 {
		dt = 0.1
	}
	lastTick = now
	if now.Sub(lastArea) > 2*time.Second {
		refreshWorkArea()
		lastArea = now
	}
	fsTick(now)
	followMonitor()
	moveTick(now, dt)
	toyTick(now, dt)
	giftTick(now)
	catchTick(now, dt)
	gachaTick(now)
	friendTick(now, dt)
	healthTick(now)
	batteryTick(now)
	memoTick(now)
	musicTick()
	aiTick()
	bathTick(now)
	tuckTick(dt)
	petFxTick(dt)
	if !hiddenFS {
		render()
	}
	screenFxTick(now, dt)
	seasonTick(now)
	focusTick(now)
	talkTick()
	stateTick()
	levelTick()
	remTick()
}

// 움직임 한 프레임 (테스트에서도 씀)
func moveTick(now time.Time, dt float64) {
	if hiding {
		hideTick(now)
		return
	}
	switch mode {
	case "drag":
		dragTick(now)
	case "fall":
		fallTick(dt)
	case "climb", "cling": // 벽·천장 (climb.go)
		climbTick(now, dt)
	default:
		perchTick()
		if airPerchTick(dt) {
			break
		}
		if g := groundY(); posY < g-1 {
			maxFallSpeed, bounces = 0, 0
			mode = "fall"
			break
		} else {
			posY = g
		}
		if mode == "walk" {
			walkTick(dt)
		}
		if mode == "catch" {
			catchMove(dt)
		}
		if mode != "climb" && now.After(modeUntil) {
			nextChoice()
		}
		if cursorOn {
			cursorTick(now)
		}
	}
}

// ───────────────────────── 메뉴 ─────────────────────────
const (
	idWalk   = 1001
	idSit    = 1002
	idSleep  = 1003
	idLaugh  = 1004
	idStay   = 1005
	idSmall  = 1010
	idMid    = 1011
	idBig    = 1012
	idStart  = 1020
	idName   = 1030
	idFeed   = 1040
	idPet    = 1041
	idScold  = 1042
	idSnack  = 1050 // +0,1,2
	idCook   = 1053
	idGame   = 1060 // +0 바나나 받기, +1 짝 맞추기, +2 바나나 탑, +3 두더지 잡기
	idLuck   = 1064
	idStamp  = 1065
	idInfo   = 1066
	idBook   = 1067
	idRemind = 1070
	idToast  = 1071
	idNeed   = 1072
	idQuit   = 1099
)

func appendMenu(m uintptr, flags uintptr, id uintptr, text string) {
	pAppendMenuW.Call(m, flags, id, uintptr(unsafe.Pointer(u16(text))))
}

func showMenu() {
	m, _, _ := pCreatePopupMenu.Call()
	appendMenu(m, MF_GRAYED, 0, levelMenuLine())
	appendMenu(m, MF_GRAYED, 0, expMenuLine())
	appendMenu(m, MF_GRAYED, 0, fmt.Sprintf("바나나 %d개", S.Bananas))
	appendMenu(m, MF_GRAYED, 0, fmt.Sprintf("배부름 %d · 기분 %d", int(S.Full+0.5), int(S.Mood+0.5)))
	appendMenu(m, MF_GRAYED, 0, fmt.Sprintf("청결 %d · 기운 %d", clean(), energy()))
	if b := batteryMenuLine(); b != "" {
		appendMenu(m, MF_GRAYED, 0, b)
	}
	appendMenu(m, MF_SEPARATOR, 0, "")
	appendPlayStop(m)
	sn, _, _ := pCreatePopupMenu.Call()
	for i, f := range snacks {
		fl := uintptr(MF_STRING)
		if S.Bananas < f.price {
			fl |= MF_GRAYED
		}
		appendMenu(sn, fl, uintptr(idSnack+i), fmt.Sprintf("%s  (바나나 %d개)", f.name, f.price))
	}
	appendMenu(sn, MF_SEPARATOR, 0, "")
	cf := uintptr(MF_STRING)
	if S.Bananas < cookCost {
		cf |= MF_GRAYED
	}
	appendMenu(sn, cf, idCook, fmt.Sprintf("바나나 요리하기  (바나나 %d개)", cookCost))
	appendMenu(m, MF_POPUP, sn, "간식 주기 · 요리")
	appendShopMenu(m)
	gm, _, _ := pCreatePopupMenu.Call()
	for i, g := range gameNames {
		appendMenu(gm, MF_STRING, uintptr(idGame+i), g)
	}
	appendPlayMenu(gm)
	appendToyMenu(gm)
	appendClimbPlayMenu(gm)
	appendMenu(m, MF_POPUP, gm, "놀아주기")
	appendFocusMenu(m)
	appendMenu(m, MF_STRING, idPet, "쓰다듬기")
	appendCareMenu(m)
	appendEmoMenu(m)
	appendMenu(m, MF_STRING, idScold, "혼내기")
	appendMenu(m, MF_STRING, idLuck, "오늘의 운세")
	appendMenu(m, MF_STRING, idStamp, "출석 도장")
	appendMenu(m, MF_STRING, idInfo, "우땅이 정보")
	appendMenu(m, MF_STRING, idChat, "우땅이랑 대화하기 (AI)…")
	appendMenu(m, MF_STRING, idBook, "우땅이 수첩 (업적·도감)")
	appendMenu(m, MF_STRING, idDiary, "우땅이 그림일기")
	appendMemoMenu(m)
	appendSeasonMenu(m)
	nt, _, _ := pCreatePopupMenu.Call()
	appendMenu(nt, MF_STRING, idRemind, fmt.Sprintf("시간 알림 설정… (%d개)", len(reminders)))
	appendMenu(nt, MF_SEPARATOR, 0, "")
	tf := uintptr(MF_STRING)
	if !S.NoToast {
		tf |= MF_CHECKED
	}
	appendMenu(nt, tf, idToast, "윈도우 알림 팝업")
	tf = MF_STRING
	if !S.NoNeedAlert {
		tf |= MF_CHECKED
	}
	appendMenu(nt, tf, idNeed, "배고픔·심심함 알림")
	appendHelperNotify(nt)
	appendMenu(m, MF_POPUP, nt, "알림")
	appendMenu(m, MF_SEPARATOR, 0, "")
	appendMenu(m, MF_STRING, idWalk, "걸어 다니기")
	appendMenu(m, MF_STRING, idSit, "앉기")
	appendMenu(m, MF_STRING, idSleep, "자기")
	appendMenu(m, MF_STRING, idLaugh, "웃기")
	appendSeatMenu(m)
	appendMenu(m, MF_STRING, idName, "이름 바꾸기")
	appendMenu(m, MF_SEPARATOR, 0, "")
	f := uintptr(MF_STRING)
	if stay {
		f |= MF_CHECKED
	}
	appendMenu(m, f, idStay, "제자리에 있기")
	appendSettingsMenu(m)
	appendMenu(m, MF_SEPARATOR, 0, "")
	appendMenu(m, MF_GRAYED, 0, "우땅이 v"+appVersion)
	appendMenu(m, MF_STRING, idQuit, "종료")

	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	pSetForegroundWindow.Call(hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, TPM_RETURNCMD|TPM_RIGHTBUTTON|TPM_BOTTOMALIGN, uintptr(c.X), uintptr(c.Y), 0, hwnd, 0)
	pPostMessageW.Call(hwnd, 0, 0, 0)
	pDestroyMenu.Call(m)

	switch cmd {
	case idWalk:
		stay = false
		setMode("walk", rd(6, 10))
	case idSit:
		setMode("sit", rd(10, 16))
	case idSleep:
		setMode("sleep", rd(20, 35))
	case idFeed:
		doFeed()
	case idSnack, idSnack + 1, idSnack + 2:
		doSnack(int(cmd - idSnack))
	case idCook:
		doCook()
	case idGame, idGame + 1, idGame + 2, idGame + 3:
		openGame(int(cmd - idGame))
	case idBook:
		openBook()
	case idChat:
		openChat()
	case idTakeOff:
		takeOffAll()
	case idLuck:
		doLuck()
	case idStamp:
		doAttend()
	case idInfo:
		doInfo()
	case idRemind:
		showReminders()
	case idToast:
		S.NoToast = !S.NoToast
		saveState()
		if !S.NoToast {
			toast("우땅이 알림", "이제 윈도우 알림으로도 알려줄게!")
		}
	case idNeed:
		S.NoNeedAlert = !S.NoNeedAlert
		needFullOn, needMoodOn = false, false
		saveState()
	case idPet:
		doPet()
	case idScold:
		doScold()
	case idName:
		askName()
	case idLaugh:
		setMode("laugh", rd(2, 3))
	case idStay:
		stay = !stay
		if stay && mode == "walk" {
			setMode("idle", rd(3, 5))
		}
	case idFocus25:
		startFocus(25)
	case idFocus50:
		startFocus(50)
	case idFocusStop:
		stopFocus()
	case idQuit:
		quitPet()
	default:
		if cmd >= idShopBase && cmd < idShopBase+uintptr(len(items)) {
			doItem(int(cmd - idShopBase))
		} else if !onOutfitCmd(cmd) && !onCareCmd(cmd) && !onSeasonCmd(cmd) && !onPlayCmd(cmd) && !onClimbCmd(cmd) && !onToyCmd(cmd) && !onEmoCmd(cmd) && !onHelperCmd(cmd) {
			onSettingsCmd(cmd)
		}
	}
}

func quitPet() {
	if catching {
		endDeskCatch(true)
	}
	if hiding {
		endHide(false)
	}
	if friendOn() {
		closeFriend()
	}
	gachaAt = time.Time{}
	quitting = true
	setMode("st_bow", 2*time.Second)
	say(pick([]string{"꾸벅! 다음에 또 만나~", "안녕! 꾸벅", "또 봐 " + voc() + "! 꾸벅"}), 2*time.Second)
	go func() {
		time.Sleep(1900 * time.Millisecond)
		pPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	}()
}

// ───────────────────────── 메인 창 ─────────────────────────
func wndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	if msgTaskbarCreated != 0 && uintptr(msg) == msgTaskbarCreated {
		addTray(trayIcon) // 탐색기가 다시 시작되면 트레이 아이콘 복구
		return 0
	}
	switch msg {
	case WM_TRAY:
		onTray(l)
		return 0
	case WM_TIMER:
		tick()
		return 0
	case WM_LBUTTONDOWN:
		if hiding { // 숨바꼭질: 찾았다!
			endHide(hideStage >= 2)
			return 0
		}
		if catching { // 바나나 받기 중엔 마우스만 따라감
			return 0
		}
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&downPt)))
		downWin = POINT{int32(posX), int32(posY)}
		pressed, dragging = true, false
		pSetCapture.Call(h)
		return 0
	case WM_MOUSEMOVE:
		if pressed && !dragging {
			var c POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
			if abs32(c.X-downPt.X) > 4 || abs32(c.Y-downPt.Y) > 4 {
				dragging = true
				startDrag()
			}
		}
		return 0
	case WM_LBUTTONUP:
		pReleaseCapture.Call()
		if pressed && dragging {
			onDragRelease()
		} else if pressed {
			if bathing { // 목욕 중 클릭 = 뽀득 문지르기
				var c POINT
				pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
				rub(float64(c.X)-posX, float64(c.Y)-posY)
			} else if q != 0 { // 벽·천장에서 톡 = 손 놓침
				letGo()
				say(pick([]string{"으앗! 떨어진다~", "깜짝이야! 으아아~"}), 2200*time.Millisecond)
			} else if tucked && mode == "sleep" {
				pokeTucked()
			} else if mode == "sleep" {
				setMode("sit", rd(2, 4)) // 깨우기
				blinkTill = time.Now().Add(300 * time.Millisecond)
			} else {
				doClick()
			}
		}
		pressed, dragging = false, false
		return 0
	case WM_RBUTTONUP:
		showMenu()
		return 0
	case WM_APP_SCORE:
		onGameScore(int(w), int(l))
		return 0
	case WM_APP_SCORE + 3:
		onTowerScore(int(w))
		return 0
	case WM_APP_SCORE + 4:
		onWhackScore(int(w))
		return 0
	case WM_APP_SCORE + 1:
		say(attendLine(int(w), true), 5*time.Second)
		return 0
	case WM_APP_SCORE + 2:
		if l := anniversaryLine(); l != "" {
			setMode("laugh", rd(2.5, 3.2))
			say(l, 6*time.Second)
		}
		return 0
	case WM_CLOSE, WM_DESTROY:
		removeTray()
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func main() {
	runtime.LockOSThread()
	_, _, e := pCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(u16("Local\\WoottangiWalkPet"))))
	if e == syscall.Errno(183) {
		pMessageBoxW.Call(0, uintptr(unsafe.Pointer(u16("우땅이가 이미 바탕화면에 있어요!"))), uintptr(unsafe.Pointer(u16("우땅이"))), 0x40)
		os.Exit(0)
	}
	pSetProcessDPIAware.Call()
	initDPI()
	startMusicWatch()
	for _, n := range []string{"walk", "idle", "blink", "laugh", "sit", "sitblink", "sleep", "eat", "eat_milk", "eat_cake", "shy", "punish", "st_surprise", "st_lounge", "st_cry", "st_me", "st_bow", "wave",
		"clap", "cheer", "heart", "angry", "dance", "huh", "v", "thumb", "sing", "no", "flower", "yum", "what", "cute"} {
		loadSeq(n)
	}
	linkAnchors()
	loadFX()
	loadCare()
	loadSounds()

	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiWalk")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(wndProc), Instance: inst, ClassName: cls}
	cur, _, _ := pLoadCursorW.Call(0, IDC_HAND)
	wc.Cursor = cur
	icon, _, _ := pLoadImageW.Call(inst, 1, IMAGE_ICON, 0, 0, LR_DEFAULTSIZE|LR_SHARED)
	wc.Icon, wc.IconSm = icon, icon
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	setupCanvas()
	refreshWorkArea()
	posX = workL + (workR-workL)*0.6
	posY = ground - 260
	setMode("fall", time.Hour)

	hwnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("우땅이"))),
		WS_POPUP, uintptr(int32(posX)), uintptr(int32(posY)), uintptr(canvasW), uintptr(canvasH), 0, 0, inst, 0)
	render()
	pShowWindow.Call(hwnd, SW_SHOW)
	createBubble(inst)
	createNameWindow(inst)
	loadName()
	loadState()
	applySavedSettings()
	initLevel()
	initAch()
	loadAccessories()
	loadReminders()
	createRemWindow(inst)
	createMemoWindow(inst)
	memoShowAt = time.Now().Add(20 * time.Second)
	msgTaskbarCreated, _, _ = pRegisterWindowMsgW.Call(uintptr(unsafe.Pointer(u16("TaskbarCreated"))))
	addTray(icon)
	pSetTimer.Call(hwnd, 1, 33, 0)
	if ownerName == "" {
		askName()
	} else {
		say("안녕 "+voc()+"! 우땅이 왔어!", 4*time.Second)
	}
	if g, isNew := checkAttendance(); isNew {
		go func() { // 인사 후 출석 도장
			time.Sleep(4500 * time.Millisecond)
			pPostMessageW.Call(hwnd, WM_APP_SCORE+1, uintptr(g), 0)
		}()
	}
	go func() { // 그 다음 기념일 인사 (해당하는 날만)
		time.Sleep(10 * time.Second)
		pPostMessageW.Call(hwnd, WM_APP_SCORE+2, 0, 0)
	}()

	var m MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if isRemMsg(m.Hwnd) {
			if r2, _, _ := pIsDialogMessageW.Call(remHwnd, uintptr(unsafe.Pointer(&m))); r2 != 0 {
				continue
			}
		}
		if isMemoMsg(m.Hwnd) {
			if r2, _, _ := pIsDialogMessageW.Call(memoHwnd, uintptr(unsafe.Pointer(&m))); r2 != 0 {
				continue
			}
		}
		if isNameMsg(m.Hwnd) {
			if r2, _, _ := pIsDialogMessageW.Call(nameHwnd, uintptr(unsafe.Pointer(&m))); r2 != 0 {
				continue
			}
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
func clampB(v float32) byte {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return byte(v + 0.5)
}

func present() {
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{int32(posX), int32(posY)}
	sz := SIZE{int32(canvasW), int32(canvasH)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, opacity(), AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(hwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		memDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
}

// ───────────────────────── 자동 실행 ─────────────────────────
const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func autostartEnabled() bool {
	var k uintptr
	if r, _, _ := pRegOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(u16(runKey))), 0, 0x20019, uintptr(unsafe.Pointer(&k))); r != 0 {
		return false
	}
	defer pRegCloseKey.Call(k)
	r, _, _ := pRegQueryValueExW.Call(k, uintptr(unsafe.Pointer(u16("Woottangi"))), 0, 0, 0, 0)
	return r == 0
}

func setAutostart(on bool) {
	var k uintptr
	if r, _, _ := pRegOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(u16(runKey))), 0, KEY_ALL_ACCESS, uintptr(unsafe.Pointer(&k))); r != 0 {
		return
	}
	defer pRegCloseKey.Call(k)
	if on {
		buf := make([]uint16, 1024)
		n, _, _ := pGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), 1024)
		path := `"` + syscall.UTF16ToString(buf[:n]) + `"`
		v, _ := syscall.UTF16FromString(path)
		pRegSetValueExW.Call(k, uintptr(unsafe.Pointer(u16("Woottangi"))), 0, REG_SZ, uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2))
	} else {
		pRegDeleteValueW.Call(k, uintptr(unsafe.Pointer(u16("Woottangi"))))
	}
}
