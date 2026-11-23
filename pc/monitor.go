package main

// 여러 모니터: 설정 → 다닐 모니터 (여러 개 체크 가능)
//   - 옆에 붙어 있는 켜진 모니터로는 걸어서·날아서 넘어감 (바닥 높이가 다르면 폴짝 내려가거나 올라섬)
//   - 꺼 둔 모니터로 끌어다 놓거나, 있던 모니터를 끄면 가까운 켜진 모니터 위에서 떨어짐
//   - 트레이 아이콘을 누르면 마우스가 있는 모니터로 불러옴 (켜 둔 모니터일 때)

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

type monInfo struct {
	dev       string // \\.\DISPLAY1 처럼 윈도우가 붙인 이름 (저장용)
	mon, work RECT   // 모니터 전체 · 작업 표시줄을 뺀 영역
	primary   bool
}

type MONITORINFOEX struct {
	MONITORINFO
	Device [32]uint16
}

var (
	pEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	pMonitorFromPoint    = user32.NewProc("MonitorFromPoint")
	monCB                = syscall.NewCallback(monEnumProc)
	monScan              []monInfo
	mons                 []monInfo // 왼쪽 모니터부터
	curMon               = -1
	monOverride          []monInfo // 테스트용 가짜 모니터 (nil 이면 실제 화면)
)

const idMonBase = 1300 // + 모니터 번호

func monEnumProc(h, _, _, _ uintptr) uintptr {
	var mi MONITORINFOEX
	mi.Size = uint32(unsafe.Sizeof(mi))
	if r, _, _ := pGetMonitorInfoW.Call(h, uintptr(unsafe.Pointer(&mi))); r != 0 {
		monScan = append(monScan, monInfo{syscall.UTF16ToString(mi.Device[:]), mi.Monitor, mi.Work, mi.Flags&1 != 0})
	}
	return 1
}

func scanMonitors() {
	if monOverride != nil {
		mons = monOverride
		return
	}
	monScan = monScan[:0]
	pEnumDisplayMonitors.Call(0, 0, monCB, 0)
	if len(monScan) == 0 { // 못 읽으면 주 모니터 작업 영역만
		var r RECT
		pSystemParametersInfoW.Call(SPI_GETWORKAREA, 0, uintptr(unsafe.Pointer(&r)), 0)
		if r.Right <= r.Left {
			w, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
			h, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
			r = RECT{0, 0, int32(w), int32(h)}
		}
		monScan = append(monScan, monInfo{"main", r, r, true})
	}
	sort.SliceStable(monScan, func(a, b int) bool {
		if monScan[a].mon.Left != monScan[b].mon.Left {
			return monScan[a].mon.Left < monScan[b].mon.Left
		}
		return monScan[a].mon.Top < monScan[b].mon.Top
	})
	mons = append(mons[:0], monScan...)
}

// 우땅이가 다녀도 되는 모니터인지 (전부 꺼져 있으면 주 모니터는 켠 걸로 침)
func monAllowed(i int) bool {
	if i < 0 || i >= len(mons) {
		return false
	}
	if !slices.Contains(S.MonOff, mons[i].dev) {
		return true
	}
	if !mons[i].primary {
		return false
	}
	for j := range mons {
		if !slices.Contains(S.MonOff, mons[j].dev) {
			return false
		}
	}
	return true
}

func inRect(r RECT, x, y float64) bool {
	return x >= float64(r.Left) && x < float64(r.Right) && y >= float64(r.Top) && y < float64(r.Bottom)
}

func rectDist(r RECT, x, y float64) float64 {
	dx := math.Max(0, math.Max(float64(r.Left)-x, x-float64(r.Right-1)))
	dy := math.Max(0, math.Max(float64(r.Top)-y, y-float64(r.Bottom-1)))
	return math.Hypot(dx, dy)
}

func inX(r RECT, x float64) bool { return x >= float64(r.Left) && x < float64(r.Right) }

// (x, y) 가 있는 켜진 모니터 → 없으면 x 가 걸친 켜진 모니터(바닥 높이가 달라도 걸어서 넘어가게)
// → 그래도 없으면 가장 가까운 켜진 모니터
func pickMonitor(x, y float64) int {
	best, bd := -1, math.MaxFloat64
	for i := range mons {
		if monAllowed(i) && inRect(mons[i].mon, x, y) {
			return i
		}
	}
	for i := range mons {
		if !monAllowed(i) {
			continue
		}
		d := rectDist(mons[i].mon, x, y)
		if inX(mons[i].mon, x) {
			d -= 1e7 // x 가 걸친 모니터 먼저
		}
		if d < bd {
			best, bd = i, d
		}
	}
	if best < 0 {
		best = 0
	}
	return best
}

func petCenter() (float64, float64) { return posX + float64(canvasW)/2, posY + float64(canvasH)*0.6 }

// 지금 있는 모니터에 맞춰 바닥·벽 정하기
func applyMonitor() {
	if len(mons) == 0 {
		scanMonitors()
	}
	var i int
	if curMon < 0 { // 처음: 주 모니터
		i = pickMonitor(-1e9, -1e9)
		for j := range mons {
			if mons[j].primary && monAllowed(j) {
				i = j
			}
		}
	} else {
		i = pickMonitor(petCenter())
	}
	curMon = i
	w := mons[i].work
	workL, workR, workT = float64(w.Left), float64(w.Right), float64(w.Top)
	ground = float64(w.Bottom) - float64(canvasH) + 2
}

// 모니터 경계를 넘었거나, 다니면 안 되는 모니터에 있으면 바로잡기 (매 프레임)
func followMonitor() {
	if len(mons) == 0 || curMon < 0 || mode == "drag" || hiding {
		return
	}
	cx, cy := petCenter()
	if curMon < len(mons) && monAllowed(curMon) && pickMonitor(cx, cy) == curMon {
		return
	}
	prev := curMon
	applyMonitor()
	if inX(mons[curMon].mon, cx) {
		if curMon != prev && perch == 0 && posY < ground-1 && mode != "fall" { // 더 낮은 모니터로: 폴짝
			maxFallSpeed, bounces = 0, 0
			setMode("fall", time.Hour)
		}
		return
	}
	// 꺼 둔 모니터 위 → 켜진 모니터 위에서 떨어짐
	perch = 0
	posX = math.Max(workL, math.Min(workR-float64(canvasW), cx-float64(canvasW)/2))
	if math.Abs(cx-(posX+float64(canvasW)/2)) > float64(canvasW) {
		posX = (workL+workR)/2 - float64(canvasW)/2
	}
	posY = workT
	velX, velY, thrown, maxFallSpeed, bounces = 0, 0, false, 0, 0
	setMode("fall", time.Hour)
}

// 지금 모니터 옆(왼쪽 -1 / 오른쪽 +1)에 붙은 켜진 모니터가 있는지
func monNeighbor(side int) bool {
	if curMon < 0 || curMon >= len(mons) {
		return false
	}
	c := mons[curMon].mon
	for j := range mons {
		if j == curMon || !monAllowed(j) {
			continue
		}
		m := mons[j].mon
		if m.Top >= c.Bottom || m.Bottom <= c.Top {
			continue
		}
		if side < 0 && abs32(m.Right-c.Left) <= 2 {
			return true
		}
		if side > 0 && abs32(m.Left-c.Right) <= 2 {
			return true
		}
	}
	return false
}

// 바닥에서 걷거나 날 때 벽 (옆 모니터로 넘어갈 수 있으면 벽 없음)
func floorBounds() (lo, hi float64) {
	lo, hi = workL-float64(canvasW)*0.15, workR-float64(canvasW)*0.85
	if monNeighbor(-1) {
		lo = math.Inf(-1)
	}
	if monNeighbor(1) {
		hi = math.Inf(1)
	}
	return
}

// 트레이에서 부를 때: 마우스가 있는 켜진 모니터로
func cursorMonitor() int {
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	for i := range mons {
		if monAllowed(i) && inRect(mons[i].mon, float64(c.X), float64(c.Y)) {
			return i
		}
	}
	return -1
}

func moveToMonitor(i int) {
	w := mons[i].work
	posX = float64(w.Left+w.Right)/2 - float64(canvasW)/2
	posY = float64(w.Bottom) - float64(canvasH) + 2
	perch = 0
	applyMonitor()
}

func monLabel(i int) string {
	m := mons[i]
	s := fmt.Sprintf("모니터 %d", i+1)
	if m.primary {
		s += " · 주 모니터"
	}
	s += fmt.Sprintf("  (%d×%d)", m.mon.Right-m.mon.Left, m.mon.Bottom-m.mon.Top)
	if i == curMon {
		s += "  ← 지금 여기"
	}
	return s
}

func appendMonitorMenu(st uintptr) {
	mm, _, _ := pCreatePopupMenu.Call()
	if len(mons) <= 1 {
		appendMenu(mm, MF_GRAYED, 0, "모니터가 하나예요")
	} else {
		for i := range mons {
			f := uintptr(MF_STRING)
			if monAllowed(i) {
				f |= MF_CHECKED
			}
			appendMenu(mm, f, uintptr(idMonBase+i), monLabel(i))
		}
		appendMenu(mm, MF_SEPARATOR, 0, "")
		appendMenu(mm, MF_GRAYED, 0, "체크한 모니터끼리 옆에 붙어 있으면 걸어서 넘어가요")
	}
	appendMenu(st, MF_POPUP, mm, "다닐 모니터")
}

func onMonitorCmd(cmd uintptr) bool {
	if cmd < idMonBase || cmd >= idMonBase+uintptr(len(mons)) {
		return false
	}
	i := int(cmd - idMonBase)
	dev := mons[i].dev
	if monAllowed(i) {
		n := 0
		for j := range mons {
			if monAllowed(j) {
				n++
			}
		}
		if n <= 1 {
			say("모니터 하나는 남겨 줘야 해!", 2500*time.Millisecond)
			return true
		}
		S.MonOff = append(S.MonOff, dev)
	} else {
		S.MonOff = slices.DeleteFunc(S.MonOff, func(d string) bool { return d == dev })
		say(fmt.Sprintf("모니터 %d에도 놀러 갈게!", i+1), 2500*time.Millisecond)
	}
	saveState()
	followMonitor()
	return true
}
