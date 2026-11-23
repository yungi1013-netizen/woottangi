package main

// PC 생활 도우미
//   - 건강 알림: 컴퓨터를 1시간 넘게 쓰면 스트레칭, 1시간 30분마다 물 마시기 (자리를 5분 넘게 비우면 다시 셈)
//   - 노트북 배터리 반응: 20%·10% 밑이면 기운 없어하고, 충전기를 꽂으면 기운 차림
//   - 포스트잇 메모: 메모를 맡기면 우땅이가 들고 다니다가 정한 시간에 보여 줌
//   - 효과음: 클릭 · 착지 · 바나나 받기 · 레벨 업 · 뽑기 (설정에서 끄기)

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	winmm                 = syscall.NewLazyDLL("winmm.dll")
	pPlaySoundW           = winmm.NewProc("PlaySoundW")
	pGetLastInputInfo     = user32.NewProc("GetLastInputInfo")
	pGetTickCount         = kernel32.NewProc("GetTickCount")
	pGetSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
)

// ───────── 건강 알림 ─────────

const (
	stretchEvery = 60 * time.Minute
	waterEvery   = 90 * time.Minute
	awayReset    = 5 * time.Minute
)

var (
	activeSince   = time.Now() // 이번에 쉬지 않고 쓰기 시작한 때
	lastStretch   = time.Now()
	lastWater     = time.Now()
	lastHealthChk time.Time
	idleOverride  = -1.0 // 테스트용 (초)
)

// 마지막 키보드·마우스 입력 뒤로 지난 시간
func idleTime() time.Duration {
	if idleOverride >= 0 {
		return time.Duration(idleOverride * float64(time.Second))
	}
	var li struct{ Size, Time uint32 }
	li.Size = 8
	if r, _, _ := pGetLastInputInfo.Call(uintptr(unsafe.Pointer(&li))); r == 0 {
		return 0
	}
	t, _, _ := pGetTickCount.Call()
	return time.Duration(uint32(t)-li.Time) * time.Millisecond
}

func healthTick(now time.Time) {
	if now.Sub(lastHealthChk) < 5*time.Second {
		return
	}
	lastHealthChk = now
	if idleTime() >= awayReset { // 자리를 비웠으면 쉰 걸로
		activeSince, lastStretch, lastWater = now, now, now
		return
	}
	if busy() || hiddenFS || bubText != "" || ownerName == "" || mode == "drag" || mode == "fall" {
		return
	}
	if !S.NoStretch && now.Sub(lastStretch) >= stretchEvery && now.Sub(activeSince) >= stretchEvery {
		lastStretch = now
		mins := int(now.Sub(activeSince).Minutes())
		setMode("wave", rd(3, 3.6))
		say(fmt.Sprintf("%s, %s 넘게 쉬지 않았어!\n같이 기지개 켜자~ 쭈욱~", voc(), hm(mins)), 7*time.Second)
		toast("우땅이 건강 알림", "잠깐 일어나서 스트레칭해요! 쭈욱~")
		S.StretchCnt++
		saveState()
		return
	}
	if !S.NoWater && now.Sub(lastWater) >= waterEvery {
		lastWater = now
		setMode("eat_milk", rd(3, 3.6))
		say("물 한 잔 마시고 와~\n나도 꿀꺽꿀꺽!", 6*time.Second)
		toast("우땅이 건강 알림", "물 한 잔 마실 시간이에요 💧")
		S.WaterCnt++
		saveState()
	}
}

func hm(mins int) string {
	if mins < 60 {
		return fmt.Sprintf("%d분", mins)
	}
	if mins%60 == 0 {
		return fmt.Sprintf("%d시간", mins/60)
	}
	return fmt.Sprintf("%d시간 %d분", mins/60, mins%60)
}

// ───────── 배터리 ─────────

type powerStatus struct {
	AC, Flag, Percent, Saver byte
	Life, FullLife           uint32
}

var (
	batPct      = -1 // -1 = 배터리 없음 (데스크톱)
	batAC       bool
	batWarned   int // 이미 말한 단계 (20, 10)
	lastBatChk  time.Time
	batOverride *powerStatus // 테스트용
)

func readPower() (powerStatus, bool) {
	if batOverride != nil {
		return *batOverride, true
	}
	var p powerStatus
	r, _, _ := pGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&p)))
	return p, r != 0
}

func batteryTick(now time.Time) {
	if now.Sub(lastBatChk) < 15*time.Second {
		return
	}
	lastBatChk = now
	p, ok := readPower()
	if !ok || p.Flag&128 != 0 || p.Flag == 255 || p.Percent > 100 {
		batPct = -1
		return
	}
	first := batPct < 0
	wasAC := batAC
	batPct, batAC = int(p.Percent), p.AC == 1
	if batAC || batPct > 25 {
		batWarned = 0
	}
	if first || S.NoBattery || busy() || hiddenFS || ownerName == "" || mode == "drag" || mode == "fall" {
		return
	}
	switch {
	case batAC && !wasAC && batPct < 90:
		setMode("laugh", rd(2.5, 3))
		burst("sparkle", 6)
		say(pick([]string{"충전 고마워! 힘이 난다~", "밥 먹는 중~ 냠냠 전기 맛있다!"}), 3500*time.Millisecond)
		sfx("gold")
	case !batAC && batPct <= 10 && batWarned != 10:
		batWarned = 10
		setMode("st_cry", 2500*time.Millisecond)
		say(fmt.Sprintf("배터리 %d%%야!\n곧 꺼질지도 몰라… 충전기!", batPct), 6*time.Second)
		toast("배터리가 거의 없어요", fmt.Sprintf("배터리 %d%% · 충전기를 꽂아 주세요", batPct))
	case !batAC && batPct <= 20 && batWarned == 0:
		batWarned = 20
		setMode("st_lounge", rd(4, 6))
		say(fmt.Sprintf("배터리가 %d%%야…\n나도 힘이 없어… 충전해 줘~", batPct), 5*time.Second)
	}
}

func batteryLow() bool { return batPct >= 0 && !batAC && batPct <= 20 && !S.NoBattery }

func batteryMenuLine() string {
	if batPct < 0 {
		return ""
	}
	if batAC {
		return fmt.Sprintf("배터리 %d%% · 충전 중", batPct)
	}
	return fmt.Sprintf("배터리 %d%%", batPct)
}

// ───────── 포스트잇 메모 ─────────

type memo struct {
	Text string `json:"t"`
	At   int64  `json:"at"` // 유닉스 초
}

var memoAfter = []struct {
	name string
	min  int
}{{"10분 뒤", 10}, {"30분 뒤", 30}, {"1시간 뒤", 60}, {"2시간 뒤", 120}, {"3시간 뒤", 180}, {"내일 이 시간", 24 * 60}}

var (
	memoHwnd, memoEdit, memoWhen uintptr
	lastMemoChk                  time.Time
	memoShowAt                   time.Time // 켜자마자 지난 메모를 보여 줄 때
)

func addMemo(text string, after time.Duration) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	S.Memos = append(S.Memos, memo{text, time.Now().Add(after).Unix()})
	sort.Slice(S.Memos, func(a, b int) bool { return S.Memos[a].At < S.Memos[b].At })
	S.MemoCnt++
	saveState()
}

func memoTick(now time.Time) {
	if len(S.Memos) == 0 || now.Sub(lastMemoChk) < time.Second {
		return
	}
	lastMemoChk = now
	if now.Before(memoShowAt) || hiddenFS || quitting || mode == "drag" || hiding || catching {
		return
	}
	if S.Memos[0].At > now.Unix() {
		return
	}
	m := S.Memos[0]
	S.Memos = S.Memos[1:]
	saveState()
	late := now.Unix()-m.At > 120
	head := "메모 배달 왔어요!"
	if late {
		head = "아까 맡긴 메모!"
	}
	if !tucked && !bathing {
		setMode("wave", rd(3, 3.6))
	}
	say(head+"\n「"+m.Text+"」", 12*time.Second)
	toast("우땅이 메모", m.Text)
	sfx("found")
	nextTalk = now.Add(rd(60, 120))
}

func memoLabel(m memo) string {
	t := time.Unix(m.At, 0)
	day := ""
	if t.Format("2006-01-02") != today() {
		day = "내일 "
	}
	txt := []rune(m.Text)
	if len(txt) > 18 {
		txt = append(txt[:17], '…')
	}
	return fmt.Sprintf("%s%s · %s", day, t.Format("15:04"), string(txt))
}

func memoProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		switch w & 0xffff {
		case 1:
			txt := getText(memoEdit)
			if strings.TrimSpace(txt) == "" {
				return 0
			}
			i, _, _ := pSendMessageW.Call(memoWhen, CB_GETCURSEL, 0, 0)
			if int(i) < 0 || int(i) >= len(memoAfter) {
				i = 0
			}
			a := memoAfter[i]
			addMemo(txt, time.Duration(a.min)*time.Minute)
			pShowWindow.Call(h, SW_HIDE)
			setMode("laugh", rd(2, 2.6))
			say(fmt.Sprintf("메모 받았어! %s에 꼭 알려 줄게!", a.name), 3500*time.Millisecond)
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

func createMemoWindow(inst uintptr) {
	cls := u16("WoottangiMemo")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(memoProc), Instance: inst, ClassName: cls, Background: 16}
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc.Cursor = cur
	icon, _, _ := pLoadImageW.Call(inst, 1, IMAGE_ICON, 0, 0, LR_DEFAULTSIZE|LR_SHARED)
	wc.Icon, wc.IconSm = icon, icon
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	k := func(v float64) uintptr { return uintptr(int32(v * dpi)) }
	sw, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
	ww, wh := int32(380*dpi), int32(200*dpi)
	memoHwnd, _, _ = pCreateWindowExW.Call(WS_EX_TOPMOST, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("우땅이에게 메모 맡기기"))),
		WS_POPUP|WS_CAPTION|WS_SYSMENU, uintptr((int32(sw)-ww)/2), uintptr((int32(sh)-wh)/2), uintptr(ww), uintptr(wh), 0, 0, inst, 0)
	font, _, _ := pCreateFontW.Call(uintptr(-int32(16*dpi)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16("맑은 고딕"))))
	lbl, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16("뭐라고 알려 줄까? (예: 택배 받기, 회의 준비)"))),
		WS_CHILD|WS_VISIBLE, k(16), k(12), k(350), k(24), memoHwnd, 101, inst, 0)
	memoEdit, _, _ = pCreateWindowExW.Call(0x200, uintptr(unsafe.Pointer(u16("EDIT"))), 0,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, k(16), k(42), k(340), k(30), memoHwnd, 100, inst, 0)
	pSendMessageW.Call(memoEdit, 0x00C5, 60, 0) // EM_LIMITTEXT
	lbl2, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16("언제?"))),
		WS_CHILD|WS_VISIBLE, k(16), k(90), k(50), k(24), memoHwnd, 102, inst, 0)
	memoWhen, _, _ = pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("COMBOBOX"))), 0,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, k(66), k(86), k(150), k(200), memoHwnd, 103, inst, 0)
	for _, a := range memoAfter {
		pSendMessageW.Call(memoWhen, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(a.name))))
	}
	pSendMessageW.Call(memoWhen, CB_SETCURSEL, 1, 0)
	ok, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16("맡기기"))),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON, k(170), k(130), k(92), k(32), memoHwnd, 1, inst, 0)
	cancel, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16("취소"))),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP, k(268), k(130), k(88), k(32), memoHwnd, 2, inst, 0)
	for _, h := range []uintptr{lbl, memoEdit, lbl2, memoWhen, ok, cancel} {
		pSendMessageW.Call(h, WM_SETFONT, font, 1)
	}
}

func askMemo() {
	pSetWindowTextW.Call(memoEdit, uintptr(unsafe.Pointer(u16(""))))
	pShowWindow.Call(memoHwnd, SW_SHOW)
	pSetForegroundWindow.Call(memoHwnd)
	pSetFocus.Call(memoEdit)
}

func isMemoMsg(h uintptr) bool {
	if memoHwnd == 0 {
		return false
	}
	for h != 0 {
		if h == memoHwnd {
			return true
		}
		h, _, _ = pGetParentN.Call(h)
	}
	return false
}

// 들고 다니는 메모 그리기 (우땅이 창 왼쪽 아래)
func drawMemoNote(buf []byte) {
	if len(S.Memos) == 0 || mode == "sleep" || hiding || catching {
		return
	}
	sz := 26 * eff() / 0.75
	x := float64(canvasW)*0.5 - dir*float64(canvasW)*0.28
	y := float64(canvasH) * 0.72
	blit(buf, canvasW, canvasH, fxSprites["memo"], x, y, sz, -dir*0.15, 1)
}

// ───────── 효과음 ─────────

var sounds = map[string][]byte{}

// 짧은 소리를 직접 만들기 (사인파 + 부드러운 끝)
type tone struct{ f0, f1, sec, vol float64 }

func synth(ts ...tone) []byte {
	const rate = 22050
	var pcm []int16
	for _, t := range ts {
		n := int(t.sec * rate)
		ph := 0.0
		for i := 0; i < n; i++ {
			p := float64(i) / float64(n)
			f := t.f0 + (t.f1-t.f0)*p
			ph += 2 * math.Pi * f / rate
			env := math.Min(1, float64(i)/(0.004*rate)) * math.Pow(1-p, 1.6)
			v := (math.Sin(ph) + 0.25*math.Sin(2*ph)) * env * t.vol
			if t.f0 == 0 { // 쉼
				v = 0
			}
			pcm = append(pcm, int16(v*12000))
		}
	}
	b := make([]byte, 44+len(pcm)*2)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+len(pcm)*2))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], rate)
	binary.LittleEndian.PutUint32(b[28:], rate*2)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(pcm)*2))
	for i, v := range pcm {
		binary.LittleEndian.PutUint16(b[44+i*2:], uint16(v))
	}
	return b
}

func loadSounds() {
	sounds["pop"] = synth(tone{520, 880, 0.07, 0.8})
	sounds["boing"] = synth(tone{420, 160, 0.18, 0.9})
	sounds["coin"] = synth(tone{988, 988, 0.05, 0.6}, tone{1319, 1319, 0.12, 0.6})
	sounds["gold"] = synth(tone{1047, 1047, 0.06, 0.6}, tone{1319, 1319, 0.06, 0.6}, tone{1568, 1568, 0.14, 0.6})
	sounds["bonk"] = synth(tone{200, 90, 0.14, 1})
	sounds["bubble"] = synth(tone{900, 1500, 0.05, 0.35})
	sounds["level"] = synth(tone{523, 523, 0.09, 0.6}, tone{659, 659, 0.09, 0.6}, tone{784, 784, 0.09, 0.6}, tone{1047, 1047, 0.25, 0.6})
	sounds["tada"] = synth(tone{784, 784, 0.08, 0.6}, tone{0, 0, 0.03, 0}, tone{784, 784, 0.08, 0.6}, tone{1047, 1047, 0.3, 0.6})
	sounds["found"] = synth(tone{660, 660, 0.07, 0.6}, tone{880, 880, 0.14, 0.6})
	sounds["start"] = synth(tone{440, 440, 0.1, 0.5}, tone{0, 0, 0.05, 0}, tone{440, 440, 0.1, 0.5}, tone{880, 880, 0.2, 0.5})
	sounds["drum"] = synth(tone{180, 200, 0.06, 0.5}, tone{180, 200, 0.06, 0.5}, tone{190, 210, 0.06, 0.5}, tone{200, 220, 0.06, 0.5}, tone{210, 230, 0.06, 0.5}, tone{220, 240, 0.06, 0.5})
}

// 소리 내기 (겹치면 앞 소리는 끊김)
func sfx(name string) {
	if S.NoSound || hiddenFS {
		return
	}
	b := sounds[name]
	if len(b) == 0 {
		return
	}
	const sndAsync, sndNoDefault, sndMemory = 0x1, 0x2, 0x4
	pPlaySoundW.Call(uintptr(unsafe.Pointer(&b[0])), 0, sndAsync|sndNoDefault|sndMemory)
}

// ───────── 메뉴 ─────────

const (
	idMemoNew  = 1400
	idStretch  = 1401
	idWater    = 1402
	idBattery  = 1403
	idSound    = 1404
	idMemoBase = 1410 // + 메모 번호
)

// 알림 메뉴에 덧붙이기
func appendHelperNotify(nt uintptr) {
	appendMenu(nt, MF_SEPARATOR, 0, "")
	for _, x := range []struct {
		id  uintptr
		off bool
		s   string
	}{{idStretch, S.NoStretch, "스트레칭 알림 (1시간 넘게 쓰면)"}, {idWater, S.NoWater, "물 마시기 알림 (1시간 30분마다)"}, {idBattery, S.NoBattery, "노트북 배터리 반응"}} {
		f := uintptr(MF_STRING)
		if !x.off {
			f |= MF_CHECKED
		}
		appendMenu(nt, f, x.id, x.s)
	}
}

func appendMemoMenu(m uintptr) {
	mm, _, _ := pCreatePopupMenu.Call()
	appendMenu(mm, MF_STRING, idMemoNew, "새 메모 맡기기…")
	if len(S.Memos) > 0 {
		appendMenu(mm, MF_SEPARATOR, 0, "")
		appendMenu(mm, MF_GRAYED, 0, "누르면 지금 보여 주고 지워요")
		for i, me := range S.Memos {
			if i >= 20 {
				break
			}
			appendMenu(mm, MF_STRING, uintptr(idMemoBase+i), memoLabel(me))
		}
	}
	t := "포스트잇 메모"
	if len(S.Memos) > 0 {
		t += fmt.Sprintf(" (%d)", len(S.Memos))
	}
	appendMenu(m, MF_POPUP, mm, t)
}

func appendSoundMenu(st uintptr) {
	f := uintptr(MF_STRING)
	if !S.NoSound {
		f |= MF_CHECKED
	}
	appendMenu(st, f, idSound, "효과음")
}

func onHelperCmd(cmd uintptr) bool {
	switch {
	case cmd == idMemoNew:
		askMemo()
		return true
	case cmd == idStretch:
		S.NoStretch = !S.NoStretch
	case cmd == idWater:
		S.NoWater = !S.NoWater
	case cmd == idBattery:
		S.NoBattery = !S.NoBattery
	case cmd == idSound:
		S.NoSound = !S.NoSound
		if !S.NoSound {
			sfx("pop")
		}
	case cmd >= idMemoBase && cmd < idMemoBase+uintptr(len(S.Memos)):
		i := int(cmd - idMemoBase)
		m := S.Memos[i]
		S.Memos = append(S.Memos[:i], S.Memos[i+1:]...)
		say("메모!\n「"+m.Text+"」", 8*time.Second)
	default:
		return false
	}
	saveState()
	return true
}
