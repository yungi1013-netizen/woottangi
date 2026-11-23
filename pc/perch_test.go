package main

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

// 진짜 창을 잠깐 띄워서: 떨어지다 제목줄에 올라앉고 → 창을 옮기면 따라가고 → 창이 닫히면 내려옴
var pDestroyWindowT = user32.NewProc("DestroyWindow")

func TestPerchOnWindow(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	setupMove(t)
	perchEnabled = true
	refreshWorkArea() // 실제 화면 크기
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiPerchTest")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(func(h uintptr, m uint32, w, l uintptr) uintptr {
		r, _, _ := pDefWindowProcW.Call(h, uintptr(m), w, l)
		return r
	}), Instance: inst, ClassName: cls}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	const x0, y0, w0, h0 = 500, 420, 700, 300
	win, _, _ := pCreateWindowExW.Call(WS_EX_TOPMOST|0x08000000 /*NOACTIVATE*/, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16("우땅이 테스트 창"))),
		0x00CF0000 /*OVERLAPPEDWINDOW*/, x0, y0, w0, h0, 0, 0, inst, 0)
	if win == 0 {
		t.Skip("테스트 창을 만들 수 없음")
	}
	defer pDestroyWindowT.Call(win)
	pShowWindow.Call(win, 4)
	setPos := user32.NewProc("SetWindowPos")
	r0, _ := frameRect(win)

	posX = float64(r0.Left+r0.Right)/2 - float64(canvasW)/2
	posY = float64(r0.Top) - 300
	velX, velY, mode = 0, 0, "fall"
	for i := 0; i < 300 && mode == "fall"; i++ {
		fallTick(1.0 / 60)
	}
	if perch != win || posY != float64(r0.Top)-feetOff() {
		t.Fatalf("창 위에 앉지 못함: perch=%x win=%x y=%.0f top=%d", perch, win, posY, r0.Top)
	}
	// 창을 오른쪽 아래로 옮기면 같이 움직임
	bx, by := posX, posY
	setPos.Call(win, 0, x0+120, y0+60, 0, 0, 0x0001|0x0004|0x0010) // NOSIZE|NOZORDER|NOACTIVATE
	perchTick()
	if perch != win || posX != bx+120 || posY != by+60 {
		t.Fatalf("창을 따라가지 않음: (%.0f,%.0f) → (%.0f,%.0f)", bx, by, posX, posY)
	}
	// 창 끝까지 걸어가면 범위 안에서 멈추거나 뛰어내림
	lo, hi := walkBounds()
	if hi-lo < 100 {
		t.Fatal("창 위 걷기 범위 이상")
	}
	// 창이 닫히면 내려옴
	pDestroyWindowT.Call(win)
	perchTick()
	if perch != 0 || groundY() != ground {
		t.Fatal("창이 닫혔는데 그대로 있음")
	}
}
