package main

// 바탕화면 위 작은 물건 창 (장난감 공·풍선, 선물)
//   - 물건 하나 = 정사각형 레이어 창 하나. 누르면 onClick, 끌면 따라오고 놓으면 그 속도로 날아감(onThrow)
//   - 테스트에서는 창 없이(hwnd 0) 위치·물리만 씀

import (
	"math"
	"syscall"
	"time"
	"unsafe"
)

type thing struct {
	hwnd, dib uintptr
	bits      unsafe.Pointer
	size      int
	sp        *fxSprite
	x, y      float64 // 가운데 (화면 좌표)
	rot       float64
	alpha     float64
	scale     float64 // 그림 크기 (창 대비)
	held      bool
	onClick   func()
	onThrow   func(vx, vy float64)
	// 끌기
	pressed, dragging bool
	downX, downY      int32
	grabX, grabY      float64
	hist              [6]struct {
		x, y float64
		t    time.Time
	}
	histN int
}

var (
	things         = map[uintptr]*thing{}
	thingClassDone bool
	thingDC        uintptr
	thingWindows   = true // 테스트에서는 false (창 안 만듦)
)

func newThing(sprite string, size int) *thing {
	t := &thing{sp: fxSprites[sprite], size: size, alpha: 1, scale: 0.9}
	if !thingWindows {
		return t
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiThing")
	if !thingClassDone {
		wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(thingWndProc), Instance: inst, ClassName: cls}
		cur, _, _ := pLoadCursorW.Call(0, IDC_HAND)
		wc.Cursor = cur
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		thingClassDone = true
	}
	if thingDC == 0 {
		sdc, _, _ := pGetDC.Call(0)
		thingDC, _, _ = pCreateCompatibleDC.Call(sdc)
		pReleaseDC.Call(0, sdc)
	}
	const exNoActivate = 0x08000000
	h, _, _ := pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW|exNoActivate,
		uintptr(unsafe.Pointer(cls)), 0, WS_POPUP, 0, 0, uintptr(size), uintptr(size), 0, 0, inst, 0)
	if h == 0 {
		return t
	}
	bi := BITMAPINFOHEADER{Size: 40, Width: int32(size), Height: -int32(size), Planes: 1, BitCount: 32}
	var p unsafe.Pointer
	dib, _, _ := pCreateDIBSection.Call(thingDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&p)), 0, 0)
	if dib == 0 || p == nil {
		pDestroyWindowT2.Call(h)
		return t
	}
	t.hwnd, t.dib, t.bits = h, dib, p
	things[h] = t
	return t
}

var pDestroyWindowT2 = user32.NewProc("DestroyWindow")

func (t *thing) close() {
	if t == nil || t.hwnd == 0 {
		return
	}
	delete(things, t.hwnd)
	pDestroyWindowT2.Call(t.hwnd)
	pDeleteObject.Call(t.dib)
	t.hwnd, t.dib, t.bits = 0, 0, nil
}

func (t *thing) r() float64 { return float64(t.size) / 2 }

// 그리고 화면에 올리기
func (t *thing) draw() {
	if t.hwnd == 0 || t.bits == nil {
		return
	}
	if hiddenFS {
		pShowWindow.Call(t.hwnd, SW_HIDE)
		return
	}
	n := t.size
	buf := unsafe.Slice((*byte)(t.bits), n*n*4)
	clear(buf)
	blit(buf, n, n, t.sp, float64(n)/2, float64(n)/2, float64(n)*t.scale, t.rot, t.alpha)
	sdc, _, _ := pGetDC.Call(0)
	pt := POINT{int32(t.x - t.r()), int32(t.y - t.r())}
	sz := SIZE{int32(n), int32(n)}
	src := POINT{0, 0}
	bl := BLENDFUNCTION{AC_SRC_OVER, 0, 255, AC_SRC_ALPHA}
	pUpdateLayeredWindow.Call(t.hwnd, sdc, uintptr(unsafe.Pointer(&pt)), uintptr(unsafe.Pointer(&sz)),
		thingDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&bl)), ULW_ALPHA)
	pReleaseDC.Call(0, sdc)
	pShowWindow.Call(t.hwnd, 4) // SW_SHOWNOACTIVATE
}

// 끄는 중이면 커서 따라가기 (매 프레임)
func (t *thing) dragTick(now time.Time) {
	if !t.dragging {
		return
	}
	var c POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
	t.x, t.y = float64(c.X)-t.grabX, float64(c.Y)-t.grabY
	t.hist[t.histN%len(t.hist)] = struct {
		x, y float64
		t    time.Time
	}{t.x, t.y, now}
	t.histN++
}

func (t *thing) releaseVel() (vx, vy float64) {
	if t.histN < 2 {
		return 0, 0
	}
	last := t.hist[(t.histN-1)%len(t.hist)]
	first := last
	for i := 2; i <= min(t.histN, len(t.hist)); i++ {
		h := t.hist[(t.histN-i)%len(t.hist)]
		if last.t.Sub(h.t) > 120*time.Millisecond {
			break
		}
		first = h
	}
	dt := last.t.Sub(first.t).Seconds()
	if dt < 0.015 {
		return 0, 0
	}
	lim := 2600 * dpi
	return math.Max(-lim, math.Min(lim, (last.x-first.x)/dt)), math.Max(-lim, math.Min(lim, (last.y-first.y)/dt))
}

func thingWndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	t := things[h]
	if t != nil {
		switch msg {
		case WM_LBUTTONDOWN:
			var c POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
			t.pressed, t.dragging = true, false
			t.downX, t.downY = c.X, c.Y
			t.grabX, t.grabY = float64(c.X)-t.x, float64(c.Y)-t.y
			pSetCapture.Call(h)
			return 0
		case WM_MOUSEMOVE:
			if t.pressed && !t.dragging && t.onThrow != nil {
				var c POINT
				pGetCursorPos.Call(uintptr(unsafe.Pointer(&c)))
				if abs32(c.X-t.downX) > 4 || abs32(c.Y-t.downY) > 4 {
					t.dragging, t.held, t.histN = true, true, 0
				}
			}
			return 0
		case WM_LBUTTONUP:
			pReleaseCapture.Call()
			if t.pressed && t.dragging {
				t.dragging, t.held = false, false
				vx, vy := t.releaseVel()
				t.onThrow(vx, vy)
			} else if t.pressed && t.onClick != nil {
				t.onClick()
			}
			t.pressed = false
			return 0
		}
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}

// 우리 물건 창인지 (창 위에 앉기·전체화면 판단에서 뺌)
func isThing(h uintptr) bool { _, ok := things[h]; return ok }
