package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ───────────────────────── 이름 저장 ─────────────────────────
var ownerName string

func namePath() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Woottangi", "name.txt")
}

func loadName() {
	b, err := os.ReadFile(namePath())
	if err == nil {
		ownerName = strings.TrimSpace(string(b))
	}
}

func saveName(n string) {
	ownerName = n
	p := namePath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	_ = os.WriteFile(p, []byte(n), 0644)
}

// ───────────────────────── 말풍선 창 ─────────────────────────
var (
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pCreatePen        = gdi32.NewProc("CreatePen")
	pRoundRect        = gdi32.NewProc("RoundRect")
	pPolygon          = gdi32.NewProc("Polygon")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pDrawTextW        = user32.NewProc("DrawTextW")
	pFillRect         = user32.NewProc("FillRect")
	pMoveToEx         = gdi32.NewProc("MoveToEx")
	pLineTo           = gdi32.NewProc("LineTo")
	pGetParentT       = user32.NewProc("GetParent")

	bubHwnd, bubDC, bubDIB uintptr
	bubBits                unsafe.Pointer
	bubW, bubH             int
	bubText                string
	bubUntil               time.Time
	bubFont                uintptr
	nextTalk               = time.Now().Add(25 * time.Second)
)

const (
	DT_CENTER    = 0x1
	DT_WORDBREAK = 0x10
	DT_CALCRECT  = 0x400
	DT_NOPREFIX  = 0x800
	TRANSPARENT  = 1
	PS_SOLID     = 0
	magentaBGR   = 0x00FF00FF
)

func bubWndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	if msg == WM_LBUTTONUP {
		bubUntil = time.Now()
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

func createBubble(inst uintptr) {
	cls := u16("WoottangiBubble")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(bubWndProc), Instance: inst, ClassName: cls}
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc.Cursor = cur
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	bubHwnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(cls)), 0,
		WS_POPUP, 0, 0, 10, 10, 0, 0, inst, 0)
	sdc, _, _ := pGetDC.Call(0)
	bubDC, _, _ = pCreateCompatibleDC.Call(sdc)
	pReleaseDC.Call(0, sdc)
	h := -int32(15 * dpi)
	bubFont, _, _ = pCreateFontW.Call(uintptr(h), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 4 /*ANTIALIASED*/, 0, uintptr(unsafe.Pointer(u16("맑은 고딕"))))
}

func say(text string, d time.Duration) {
	if bubHwnd == 0 || text == "" {
		return
	}
	bubText = text
	bubUntil = time.Now().Add(d)
	drawBubble()
	pShowWindow.Call(bubHwnd, 4) // SW_SHOWNOACTIVATE
	placeBubble()
}

func drawBubble() {
	pad := int32(12 * dpi)
	tail := int32(10 * dpi)
	maxW := int32(210 * dpi)
	txt, _ := syscall.UTF16FromString(bubText)
	pSelectObject.Call(bubDC, bubFont)
	rc := RECT{0, 0, maxW, 0}
	pDrawTextW.Call(bubDC, uintptr(unsafe.Pointer(&txt[0])), uintptr(len(txt)-1), uintptr(unsafe.Pointer(&rc)), DT_CALCRECT|DT_WORDBREAK|DT_CENTER|DT_NOPREFIX)
	tw, th := rc.Right-rc.Left, rc.Bottom-rc.Top
	bubW = int(tw + pad*2 + 4)
	bubH = int(th + pad*2 + tail + 4)
	if bubDIB != 0 {
		pDeleteObject.Call(bubDIB)
	}
	bi := BITMAPINFOHEADER{Size: 40, Width: int32(bubW), Height: -int32(bubH), Planes: 1, BitCount: 32}
	var p unsafe.Pointer
	bubDIB, _, _ = pCreateDIBSection.Call(bubDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&p)), 0, 0)
	bubBits = p
	pSelectObject.Call(bubDC, bubDIB)
	pSelectObject.Call(bubDC, bubFont)
	// 배경을 마젠타로 채운 뒤, 나중에 투명 처리
	mag, _, _ := pCreateSolidBrush.Call(magentaBGR)
	all := RECT{0, 0, int32(bubW), int32(bubH)}
	pFillRect.Call(bubDC, uintptr(unsafe.Pointer(&all)), mag)
	pDeleteObject.Call(mag)
	white, _, _ := pCreateSolidBrush.Call(0x00FFFFFF)
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(int32(3*dpi)), 0x00101010)
	ob, _, _ := pSelectObject.Call(bubDC, white)
	op, _, _ := pSelectObject.Call(bubDC, pen)
	bx2, by2 := int32(bubW)-2, int32(bubH)-tail-2
	r := int32(18 * dpi)
	pRoundRect.Call(bubDC, 2, 2, uintptr(bx2), uintptr(by2), uintptr(r), uintptr(r))
	cx := int32(bubW) / 2
	pts := [3]POINT{{cx - tail, by2 - 2}, {cx + tail, by2 - 2}, {cx - tail/3, int32(bubH) - 2}}
	pPolygon.Call(bubDC, uintptr(unsafe.Pointer(&pts[0])), 3)
	// 꼬리와 상자 사이 선 지우기
	wpen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(int32(3*dpi)), 0x00FFFFFF)
	pSelectObject.Call(bubDC, wpen)
	pMoveToEx.Call(bubDC, uintptr(cx-tail+int32(3*dpi)), uintptr(by2-int32(2*dpi)), 0)
	pLineTo.Call(bubDC, uintptr(cx+tail-int32(3*dpi)), uintptr(by2-int32(2*dpi)))
	pSelectObject.Call(bubDC, op)
	pSelectObject.Call(bubDC, ob)
	pDeleteObject.Call(pen)
	pDeleteObject.Call(wpen)
	pDeleteObject.Call(white)
	pSetBkMode.Call(bubDC, TRANSPARENT)
	pSetTextColor.Call(bubDC, 0x00202020)
	trc := RECT{pad + 2, pad + 2, pad + 2 + tw, pad + 2 + th}
	pDrawTextW.Call(bubDC, uintptr(unsafe.Pointer(&txt[0])), uintptr(len(txt)-1), uintptr(unsafe.Pointer(&trc)), DT_WORDBREAK|DT_CENTER|DT_NOPREFIX)
	// 알파 채우기
	buf := unsafe.Slice((*byte)(bubBits), bubW*bubH*4)
	for i := 0; i < bubW*bubH; i++ {
		o := i * 4
		if buf[o] == 0xFF && buf[o+1] == 0x00 && buf[o+2] == 0xFF {
			buf[o], buf[o+1], buf[o+2], buf[o+3] = 0, 0, 0, 0
		} else {
			buf[o+3] = 255
		}
	}
}

func placeBubble() {
	if bubHwnd == 0 {
		return
	}
	x := int32(posX + float64(canvasW)/2 - float64(bubW)/2)
	y := int32(posY + float64(canvasH)*0.30 - float64(bubH))
	if y < 0 {
		y = 0
	}
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{x, y}
	sz := SIZE{int32(bubW), int32(bubH)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, 255, AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(bubHwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		bubDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
}

// 매 프레임
func talkTick() {
	now := time.Now()
	if bubText != "" {
		if now.After(bubUntil) {
			pShowWindow.Call(bubHwnd, SW_HIDE)
			bubText = ""
		} else {
			placeBubble()
		}
	}
	if ownerName == "" || mode == "drag" || mode == "fall" {
		return
	}
	slotTick()
	if now.After(nextTalk) {
		nextTalk = now.Add(rd(60, 150))
		if mode == "sleep" {
			say(pick([]string{"쿨쿨… " + voc() + "…", "음냐음냐… 바나나…", "Zzz…"}), 3*time.Second)
			return
		}
		say(randomLine(), 4500*time.Millisecond)
		if (mode == "idle" || mode == "walk") && rng.Float64() < 0.5 {
			setMode("wave", rd(2.5, 3.5))
		}
	}
}

func pick(a []string) string { return a[rng.Intn(len(a))] }

// 시간대 대사 (PC 시계 기준, 분 단위)
type slot struct {
	from, to int // 하루 중 분 (to 미포함)
	lines    func(n string) []string
}

var slots = []slot{
	{9 * 60, 10 * 60, func(n string) []string { return []string{n + " 좋은 아침!", n + " 아침 먹었어?"} }},
	{11*60 + 30, 12 * 60, func(n string) []string {
		return []string{n + " 점심 뭐 먹어?", "배고프다… " + n + " 밥 먹으러 가자!", n + " 나도 밥줘"}
	}},
	{13 * 60, 13*60 + 30, func(n string) []string { return []string{n + " 일하자", n + " 졸리면 기지개 쭉~"} }},
	{17*60 + 30, 18 * 60, func(n string) []string {
		return []string{n + " 오늘도 수고했어!", n + " 집에 갈 준비하자!"}
	}},
	{19 * 60, 20 * 60, func(n string) []string { return []string{n + " 퇴근 언제 해?"} }},
	{22 * 60, 24 * 60, func(n string) []string {
		return []string{n + " 너무 늦었어. 이제 자자~", "하암… " + n + " 졸리지?"}
	}},
}

var lastSlotKey string

func currentSlot() (int, bool) {
	now := time.Now()
	m := now.Hour()*60 + now.Minute()
	for i, s := range slots {
		if m >= s.from && m < s.to {
			return i, true
		}
	}
	return -1, false
}

// 시간대에 막 들어왔으면 바로 한 번 말하기
func slotTick() {
	i, ok := currentSlot()
	if !ok || ownerName == "" || mode == "drag" || mode == "fall" || mode == "sleep" || quitting {
		return
	}
	key := time.Now().Format("2006-01-02") + "#" + string(rune('0'+i))
	if key == lastSlotKey {
		return
	}
	lastSlotKey = key
	say(pick(slots[i].lines(voc())), 5*time.Second)
	if mode == "idle" || mode == "walk" {
		setMode("wave", rd(2.5, 3.5))
	}
	nextTalk = time.Now().Add(rd(60, 150))
}

func randomLine() string {
	if l := needLine(); l != "" {
		return l
	}
	if rng.Float64() < 0.15 {
		if l := wearLine(); l != "" {
			return l
		}
	}
	if rng.Float64() < 0.2 {
		return affectionLine()
	}
	n := voc()
	if i, ok := currentSlot(); ok && rng.Float64() < 0.6 {
		return pick(slots[i].lines(n))
	}
	general := []string{
		n + " 화이팅!", n + " 물 한 잔 마셔~", n + " 사랑해 ♥", n + " 뭐 해?",
		n + " 잠깐 쉬었다 해~", "우땅! " + ownerName + " 최고!", n + " 허리 쭉 펴!", n + " 나 여기 있어~",
	}
	return pick(general)
}

func clickLine() string {
	n := voc()
	return pick([]string{"히히 " + n + " 좋아!", "쓰다듬어줘서 고마워 " + n + "~", "간지러워 " + n + "~", "우땅우땅 ♥"})
}

// 반말 호칭: 받침 있으면 "아", 없으면 "야" (예: 지수야, 민준아)
func voc() string {
	n := ownerName
	if n == "" || n == "주인님" {
		return "주인님"
	}
	rs := []rune(n)
	last := rs[len(rs)-1]
	if last >= 0xAC00 && last <= 0xD7A3 {
		if (last-0xAC00)%28 == 0 {
			return n + "야"
		}
		return n + "아"
	}
	return n
}

// ───────────────────────── 이름 묻는 창 ─────────────────────────
var nameHwnd, nameEdit uintptr

func nameProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		if w&0xffff == 1 {
			submitName()
			return 0
		}
	case WM_CLOSE:
		if ownerName == "" {
			saveName("주인님")
			say("안녕! 나는 우땅이야!", 5*time.Second)
		}
		pShowWindow.Call(h, SW_HIDE)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

func submitName() {
	buf := make([]uint16, 64)
	pGetWindowTextW.Call(nameEdit, uintptr(unsafe.Pointer(&buf[0])), 64)
	n := strings.TrimSpace(syscall.UTF16ToString(buf))
	if n == "" {
		return
	}
	saveName(n)
	pShowWindow.Call(nameHwnd, SW_HIDE)
	setMode("laugh", rd(2, 3))
	say("반가워 "+voc()+"! 나는 우땅이야!", 5*time.Second)
	setMode("wave", rd(2.5, 3.2))
	nextTalk = time.Now().Add(rd(40, 80))
}

func createNameWindow(inst uintptr) {
	cls := u16("WoottangiName")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(nameProc), Instance: inst, ClassName: cls, Background: 16}
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc.Cursor = cur
	icon, _, _ := pLoadImageW.Call(inst, 1, IMAGE_ICON, 0, 0, LR_DEFAULTSIZE|LR_SHARED)
	wc.Icon, wc.IconSm = icon, icon
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	k := func(v float64) uintptr { return uintptr(int32(v * dpi)) }
	sw, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
	ww, wh := int32(340*dpi), int32(170*dpi)
	nameHwnd, _, _ = pCreateWindowExW.Call(WS_EX_TOPMOST, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("우땅이"))),
		WS_POPUP|WS_CAPTION|WS_SYSMENU, uintptr((int32(sw)-ww)/2), uintptr((int32(sh)-wh)/2), uintptr(ww), uintptr(wh), 0, 0, inst, 0)
	font, _, _ := pCreateFontW.Call(uintptr(-int32(17*dpi)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16("맑은 고딕"))))
	lbl, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16("안녕! 나는 우땅이야.\n이름이 뭐야?"))),
		WS_CHILD|WS_VISIBLE, k(16), k(12), k(300), k(48), nameHwnd, 101, inst, 0)
	nameEdit, _, _ = pCreateWindowExW.Call(0x200, uintptr(unsafe.Pointer(u16("EDIT"))), 0,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, k(16), k(68), k(200), k(30), nameHwnd, 100, inst, 0)
	btn, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16("확인"))),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON, k(224), k(67), k(92), k(32), nameHwnd, 1, inst, 0)
	for _, h := range []uintptr{lbl, nameEdit, btn} {
		pSendMessageW.Call(h, WM_SETFONT, font, 1)
	}
}

func askName() {
	if ownerName != "" && ownerName != "주인님" {
		pSetWindowTextW.Call(nameEdit, uintptr(unsafe.Pointer(u16(ownerName))))
	}
	pShowWindow.Call(nameHwnd, SW_SHOW)
	pSetForegroundWindow.Call(nameHwnd)
	pSetFocus.Call(nameEdit)
	pSendMessageW.Call(nameEdit, EM_SETSEL, 0, 0xFFFF)
}

func isNameMsg(h uintptr) bool {
	if nameHwnd == 0 {
		return false
	}
	if h == nameHwnd {
		return true
	}
	p, _, _ := pGetParentT.Call(h)
	return p == nameHwnd
}

// ───────────────────────── 돌보기 ─────────────────────────
func callName() string {
	if ownerName == "" {
		return ""
	}
	return ownerName + " "
}

func doFeed() { doSnack(0) }

func doPet() {
	petBonus()
	setMode("shy", rd(3.2, 4.2))
	say(pick([]string{"헤헤… 부끄러워…", "히히 간지러워~", voc() + " 좋아… ♥", "에헤헤…", "우땅우땅 ♥"}), 3*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

func doScold() {
	scoldPenalty()
	setMode("st_cry", 2500*time.Millisecond) // 먼저 엉엉 울고 → 벌서기
	say(pick([]string{"잘못했어… 흑", "으앙… 벌 서고 있을게…", "미안해 " + voc() + "…", "팔 아파… 흑흑"}), 3500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}

var greeted, quitting bool

func doClick() {
	if quitting {
		return
	}
	if rng.Float64() < 0.3 {
		setMode("st_surprise", 1800*time.Millisecond)
		say(pick([]string{"깜짝이야!", "앗! 놀랐잖아~", "헉!"}), 2*time.Second)
		return
	}
	setMode("st_me", 2200*time.Millisecond)
	if ownerName != "" && rng.Float64() < 0.4 {
		say(pick([]string{"응! " + voc() + "!", "불렀어?", "나 여기 있어!"}), 2200*time.Millisecond)
	}
}
