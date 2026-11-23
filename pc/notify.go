package main

// 알림: 트레이 아이콘 + 윈도우 알림 팝업 · 내가 정한 시간 알림 · 배고픔/심심함 알림

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	shell32                = syscall.NewLazyDLL("shell32.dll")
	pShellNotifyIconW      = shell32.NewProc("Shell_NotifyIconW")
	pRegisterWindowMsgW    = user32.NewProc("RegisterWindowMessageW")
	pIsWindowVisible       = user32.NewProc("IsWindowVisible")
	pEnableWindow          = user32.NewProc("EnableWindow")
	pGetParentN            = user32.NewProc("GetParent")
	msgTaskbarCreated      uintptr
	trayIcon               uintptr
	trayAdded              bool
	remHwnd, remTime       uintptr
	remRepeat, remMsg      uintptr
	remList                uintptr
	reminders              []reminder
	lastRemCheck           string
	needFullOn, needMoodOn bool // 알림을 이미 보냈는지 (회복되면 다시 보낼 수 있음)
)

const (
	WM_TRAY              = 0x8000 + 10
	NIM_ADD              = 0
	NIM_MODIFY           = 1
	NIM_DELETE           = 2
	NIF_MESSAGE          = 0x1
	NIF_ICON             = 0x2
	NIF_TIP              = 0x4
	NIF_INFO             = 0x10
	NIIF_USER            = 0x4
	NIIF_LARGE_ICON      = 0x20
	NIN_BALLOONUSERCLICK = 0x405
	LB_ADDSTRING         = 0x180
	LB_RESETCONTENT      = 0x184
	LB_GETCURSEL         = 0x188
	CB_ADDSTRING         = 0x143
	CB_GETCURSEL         = 0x147
	CB_SETCURSEL         = 0x14E
	CBS_DROPDOWNLIST     = 0x3
	LBS_NOTIFY           = 0x1
	WS_VSCROLL           = 0x00200000
	ES_CENTER            = 0x1
)

type NOTIFYICONDATA struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        [16]byte
	BalloonIcon     uintptr
}

func copyU16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}

func trayData() NOTIFYICONDATA {
	d := NOTIFYICONDATA{Wnd: hwnd, ID: 1}
	d.Size = uint32(unsafe.Sizeof(d))
	return d
}

func addTray(icon uintptr) {
	trayIcon = icon
	d := trayData()
	d.Flags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	d.CallbackMessage = WM_TRAY
	d.Icon = icon
	copyU16(d.Tip[:], "우땅이 v"+appVersion)
	r, _, _ := pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&d)))
	trayAdded = r != 0
}

func removeTray() {
	if !trayAdded {
		return
	}
	d := trayData()
	pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&d)))
	trayAdded = false
}

// 윈도우 알림 팝업 (설정에서 끌 수 있음)
func toast(title, text string) {
	if S.NoToast || !trayAdded {
		return
	}
	d := trayData()
	d.Flags = NIF_INFO
	copyU16(d.InfoTitle[:], title)
	copyU16(d.Info[:], text)
	d.InfoFlags = NIIF_USER | NIIF_LARGE_ICON
	d.BalloonIcon = trayIcon
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&d)))
}

// 트레이 아이콘 이벤트
func onTray(l uintptr) {
	switch uint32(l & 0xffff) {
	case WM_RBUTTONUP:
		showMenu()
	case WM_LBUTTONDBLCLK, WM_LBUTTONUP:
		callPet()
	case NIN_BALLOONUSERCLICK:
		callPet()
	}
}

// 우땅이를 화면 가운데 아래로 불러오기
func callPet() {
	if mode == "drag" || catching {
		return
	}
	if hiding {
		endHide(false)
		return
	}
	if i := cursorMonitor(); i >= 0 && i != curMon { // 마우스가 있는 모니터로
		moveToMonitor(i)
	}
	posX = (workL+workR)/2 - float64(canvasW)/2
	if mode == "sleep" {
		setMode("sit", rd(2, 4))
	} else {
		setMode("wave", rd(2.5, 3.2))
	}
	say(pick([]string{"나 여기 있어 " + voc() + "!", "불렀어? 우땅!"}), 2500*time.Millisecond)
}

// ───────── 배고픔 / 심심함 알림 ─────────
func needTick() {
	if S.NoNeedAlert || ownerName == "" || quitting {
		return
	}
	if S.Full >= 40 {
		needFullOn = false
	}
	if S.Mood >= 40 {
		needMoodOn = false
	}
	if S.Full < 20 && !needFullOn {
		needFullOn = true
		setMode("st_lounge", 4*time.Second)
		say(voc()+" 배고파… 간식 줘~", 6*time.Second)
		toast("우땅이가 배고파해요", fmt.Sprintf("%s 배고파… 간식 줘~ (배부름 %d)", voc(), int(S.Full)))
		nextTalk = time.Now().Add(rd(90, 150))
		return
	}
	if S.Mood < 20 && !needMoodOn {
		needMoodOn = true
		setMode("st_lounge", 4*time.Second)
		say(voc()+" 심심해… 나랑 놀아줘…", 6*time.Second)
		toast("우땅이가 심심해해요", fmt.Sprintf("%s 나랑 놀아줘… (기분 %d)", voc(), int(S.Mood)))
		nextTalk = time.Now().Add(rd(90, 150))
	}
}

// ───────── 내가 정한 시간 알림 ─────────
type reminder struct {
	HM     string `json:"hm"` // "15:00"
	Text   string `json:"text"`
	Repeat int    `json:"repeat"` // 0 한 번, 1 매일, 2 평일
	Date   string `json:"date"`   // 한 번 알림의 날짜
	Fired  string `json:"fired"`  // 마지막으로 울린 날짜
}

var repeatNames = []string{"한 번", "매일", "평일"}

func remPath() string { return filepath.Join(filepath.Dir(statePath()), "reminders.json") }

func loadReminders() {
	if b, err := os.ReadFile(remPath()); err == nil {
		json.Unmarshal(b, &reminders)
	}
}

func saveReminders() {
	sort.SliceStable(reminders, func(i, j int) bool { return reminders[i].HM < reminders[j].HM })
	if b, err := json.Marshal(reminders); err == nil {
		os.WriteFile(remPath(), b, 0644)
	}
}

// 1분에 한 번
func remTick() {
	now := time.Now()
	key := now.Format("2006-01-02 15:04")
	if key == lastRemCheck {
		return
	}
	lastRemCheck = key
	needTick()
	hm, day := now.Format("15:04"), now.Format("2006-01-02")
	wd := now.Weekday()
	changed := false
	var keep []reminder
	for _, r := range reminders {
		due := r.HM == hm && r.Fired != day
		switch r.Repeat {
		case 0:
			due = due && r.Date == day
		case 2:
			due = due && wd != time.Saturday && wd != time.Sunday
		}
		if due {
			fireReminder(r)
			r.Fired = day
			changed = true
			if r.Repeat == 0 {
				continue // 한 번 알림은 울리고 삭제
			}
		}
		keep = append(keep, r)
	}
	if changed {
		reminders = keep
		saveReminders()
		refreshRemList()
	}
}

func fireReminder(r reminder) {
	if mode == "drag" || mode == "fall" {
		// 그대로 두고 말만
	} else {
		setMode("wave", rd(3.5, 4.5))
	}
	text := r.Text
	if text == "" {
		text = "알림 시간"
	}
	say(fmt.Sprintf("%s! %s 알림이야\n%s", voc(), r.HM, text), 30*time.Second)
	toast("우땅이 알림 · "+r.HM, voc()+"! "+text)
	nextTalk = time.Now().Add(rd(90, 150))
}

// "15:00", "1500", "3:5", "15시", "15시 30분" → "15:00"
func parseHM(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("시", ":", "분", "", " ", "", ".", ":").Replace(s)
	var h, m int
	var err error
	if i := strings.Index(s, ":"); i >= 0 {
		h, err = strconv.Atoi(s[:i])
		if err != nil {
			return "", false
		}
		if rest := s[i+1:]; rest != "" {
			if m, err = strconv.Atoi(rest); err != nil {
				return "", false
			}
		}
	} else {
		n, err := strconv.Atoi(s)
		if err != nil {
			return "", false
		}
		if len(s) <= 2 {
			h = n
		} else {
			h, m = n/100, n%100
		}
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", h, m), true
}

func remLabel(r reminder) string {
	when := repeatNames[r.Repeat]
	if r.Repeat == 0 {
		if t, err := time.Parse("2006-01-02", r.Date); err == nil {
			when = t.Format("1/2")
		}
	}
	return fmt.Sprintf("%s   %s   %s", r.HM, when, r.Text)
}

func refreshRemList() {
	if remList == 0 {
		return
	}
	pSendMessageW.Call(remList, LB_RESETCONTENT, 0, 0)
	for _, r := range reminders {
		pSendMessageW.Call(remList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(remLabel(r)))))
	}
}

func getText(h uintptr) string {
	buf := make([]uint16, 200)
	pGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), 200)
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func addReminder() {
	hm, ok := parseHM(getText(remTime))
	if !ok {
		pMessageBoxW.Call(remHwnd, uintptr(unsafe.Pointer(u16("시간을 15:00 처럼 적어줘!"))), uintptr(unsafe.Pointer(u16("우땅이 알림"))), 0x30)
		pSetFocus.Call(remTime)
		return
	}
	text := getText(remMsg)
	rep, _, _ := pSendMessageW.Call(remRepeat, CB_GETCURSEL, 0, 0)
	if int(rep) < 0 || int(rep) > 2 {
		rep = 0
	}
	r := reminder{HM: hm, Text: text, Repeat: int(rep)}
	now := time.Now()
	if r.Repeat == 0 {
		r.Date = now.Format("2006-01-02")
		if hm <= now.Format("15:04") { // 이미 지난 시간이면 내일
			r.Date = now.AddDate(0, 0, 1).Format("2006-01-02")
		}
	} else if hm <= now.Format("15:04") {
		r.Fired = now.Format("2006-01-02") // 오늘 지난 시간은 건너뜀
	}
	reminders = append(reminders, r)
	saveReminders()
	refreshRemList()
	pSetWindowTextW.Call(remMsg, uintptr(unsafe.Pointer(u16(""))))
	pSetFocus.Call(remMsg)
	when := "오늘"
	if r.Repeat == 1 {
		when = "매일"
	} else if r.Repeat == 2 {
		when = "평일마다"
	} else if r.Date != now.Format("2006-01-02") {
		when = "내일"
	}
	setMode("st_me", 2200*time.Millisecond)
	say(fmt.Sprintf("알았어! %s %s에 알려줄게!", when, hm), 3500*time.Millisecond)
}

func delReminder() {
	i, _, _ := pSendMessageW.Call(remList, LB_GETCURSEL, 0, 0)
	if int32(i) < 0 || int(i) >= len(reminders) {
		return
	}
	reminders = append(reminders[:i], reminders[i+1:]...)
	saveReminders()
	refreshRemList()
}

func remProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		switch w & 0xffff {
		case 1:
			addReminder()
			return 0
		case 3:
			delReminder()
			return 0
		case 2:
			pShowWindow.Call(h, SW_HIDE)
			return 0
		}
	case WM_CLOSE:
		pShowWindow.Call(h, SW_HIDE)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

func createRemWindow(inst uintptr) {
	cls := u16("WoottangiRemind")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(remProc), Instance: inst, ClassName: cls, Background: 16}
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc.Cursor = cur
	icon, _, _ := pLoadImageW.Call(inst, 1, IMAGE_ICON, 0, 0, LR_DEFAULTSIZE|LR_SHARED)
	wc.Icon, wc.IconSm = icon, icon
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	k := func(v float64) uintptr { return uintptr(int32(v * dpi)) }
	sw, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
	ww, wh := int32(400*dpi), int32(400*dpi)
	remHwnd, _, _ = pCreateWindowExW.Call(WS_EX_TOPMOST, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("우땅이 알림"))),
		WS_POPUP|WS_CAPTION|WS_SYSMENU, uintptr((int32(sw)-ww)/2), uintptr((int32(sh)-wh)/2), uintptr(ww), uintptr(wh), 0, 0, inst, 0)
	font, _, _ := pCreateFontW.Call(uintptr(-int32(15*dpi)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16("맑은 고딕"))))
	mk := func(ex uintptr, class, text string, style uintptr, x, y, w, h float64, id uintptr) uintptr {
		var tp uintptr
		if text != "" {
			tp = uintptr(unsafe.Pointer(u16(text)))
		}
		c, _, _ := pCreateWindowExW.Call(ex, uintptr(unsafe.Pointer(u16(class))), tp, WS_CHILD|WS_VISIBLE|style,
			k(x), k(y), k(w), k(h), remHwnd, id, inst, 0)
		pSendMessageW.Call(c, WM_SETFONT, font, 1)
		return c
	}
	mk(0, "STATIC", "언제 (예: 15:00)", 0, 16, 12, 160, 20, 0)
	mk(0, "STATIC", "반복", 0, 108, 12, 80, 20, 0)
	remTime = mk(0x200, "EDIT", "", WS_TABSTOP|ES_CENTER|ES_AUTOHSCROLL, 16, 34, 84, 28, 10)
	remRepeat = mk(0, "COMBOBOX", "", WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL, 108, 34, 96, 200, 11)
	for _, n := range repeatNames {
		pSendMessageW.Call(remRepeat, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(n))))
	}
	pSendMessageW.Call(remRepeat, CB_SETCURSEL, 0, 0)
	mk(0, "STATIC", "할 일 (예: 회의, 약 먹기)", 0, 16, 72, 260, 20, 0)
	remMsg = mk(0x200, "EDIT", "", WS_TABSTOP|ES_AUTOHSCROLL, 16, 94, 254, 28, 12)
	mk(0, "BUTTON", "추가", WS_TABSTOP|BS_DEFPUSHBUTTON, 278, 93, 90, 30, 1)
	remList = mk(0x200, "LISTBOX", "", WS_TABSTOP|WS_VSCROLL|LBS_NOTIFY, 16, 134, 352, 170, 13)
	mk(0, "BUTTON", "선택 삭제", WS_TABSTOP, 16, 314, 100, 30, 3)
	mk(0, "BUTTON", "닫기", WS_TABSTOP, 278, 314, 90, 30, 2)
	refreshRemList()
}

func showReminders() {
	t := time.Now().Add(10 * time.Minute)
	t = t.Add(-time.Duration(t.Minute()%5) * time.Minute)
	pSetWindowTextW.Call(remTime, uintptr(unsafe.Pointer(u16(t.Format("15:04")))))
	refreshRemList()
	pShowWindow.Call(remHwnd, SW_SHOW)
	pSetForegroundWindow.Call(remHwnd)
	pSetFocus.Call(remMsg)
}

func isRemMsg(h uintptr) bool {
	if remHwnd == 0 {
		return false
	}
	if h == remHwnd {
		return true
	}
	p, _, _ := pGetParentN.Call(h)
	return p == remHwnd
}
