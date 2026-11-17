package main

// 설치 화면: 왼쪽에 우땅이 그림, 오른쪽에 설명 · 선택 3개 · [취소] [설치]

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"syscall"
	"unsafe"
)

const (
	wsOverlappedWin = 0x00C00000 | 0x00080000 | 0x00020000 // CAPTION | SYSMENU | MINIMIZEBOX
	wsChild         = 0x40000000
	wsVisible       = 0x10000000
	wsTabStop       = 0x00010000
	bsPush          = 0x0
	bsDefPush       = 0x1
	bsAutoCheck     = 0x3
	ssNoPrefix      = 0x80
	ssPathEllipsis  = 0x8000

	wmDestroy     = 0x0002
	wmPaint       = 0x000F
	wmClose       = 0x0010
	wmSetFont     = 0x0030
	wmCommand     = 0x0111
	wmCtlColorSt  = 0x0138
	wmCtlColorBtn = 0x0135
	bmGetCheck    = 0x00F0
	bmSetCheck    = 0x00F1

	idOK      = 1
	idCancel  = 2
	idDesktop = 101
	idAuto    = 102
	idRun     = 103

	bgColor   = 0x00D6F4FF // RGB(255,244,214) 크림색 (COLORREF 는 BGR)
	inkColor  = 0x00161D2B
	greyColor = 0x00566D7D
)

var g struct {
	p                          paths
	hwnd                       uintptr
	title, desc, pathLbl       uintptr
	cbDesk, cbAuto, cbRun      uintptr
	btnOK, btnCancel           uintptr
	bg, fontN, fontB, fontS    uintptr
	dpi                        float64
	art                        []byte // BGRA, 배경색과 미리 섞음
	aw, ah                     int
	done, busy                 bool
}

func sc(v int) int32 { return int32(float64(v)*g.dpi + 0.5) }

func loadArt() {
	img, err := png.Decode(bytes.NewReader(artPNG))
	if err != nil {
		return
	}
	b := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	g.aw, g.ah = b.Dx(), b.Dy()
	g.art = make([]byte, g.aw*g.ah*4)
	bgR, bgG, bgB := 255.0, 244.0, 214.0
	for i := 0; i < g.aw*g.ah; i++ {
		a := float64(rgba.Pix[i*4+3]) / 255
		mix := func(c uint8, bg float64) byte { return byte(float64(c)*a + bg*(1-a) + 0.5) }
		g.art[i*4] = mix(rgba.Pix[i*4+2], bgB)
		g.art[i*4+1] = mix(rgba.Pix[i*4+1], bgG)
		g.art[i*4+2] = mix(rgba.Pix[i*4], bgR)
		g.art[i*4+3] = 255
	}
}

func mkFont(px int, weight int) uintptr {
	f, _, _ := pCreateFontW.Call(uintptr(-sc(px)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16("Malgun Gothic"))))
	return f
}

func mk(cls, text string, style uintptr, x, y, w, h int, id uintptr, font uintptr) uintptr {
	c, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16(cls))), uintptr(unsafe.Pointer(u16(text))),
		wsChild|wsVisible|style, uintptr(sc(x)), uintptr(sc(y)), uintptr(sc(w)), uintptr(sc(h)), g.hwnd, id, 0, 0)
	pSendMessageW.Call(c, wmSetFont, font, 1)
	return c
}

func setText(h uintptr, s string) { pSetWindowTextW.Call(h, uintptr(unsafe.Pointer(u16(s)))) }
func checked(h uintptr) bool     { r, _, _ := pSendMessageW.Call(h, bmGetCheck, 0, 0); return r == 1 }
func setCheck(h uintptr, on bool) {
	v := uintptr(0)
	if on {
		v = 1
	}
	pSendMessageW.Call(h, bmSetCheck, v, 0)
}
func show(h uintptr, on bool) {
	v := uintptr(0)
	if on {
		v = 5
	}
	pShowWindow.Call(h, v)
}

func introText() string {
	if g.p.installed() {
		return "이미 설치되어 있어요.\r\n새 버전으로 업데이트할게요.\r\n이름·바나나·레벨 기록은 그대로예요."
	}
	return "바탕화면을 돌아다니는 원숭이 우땅이를\r\n이 컴퓨터에 설치해요.\r\n관리자 권한은 필요 없어요."
}

// 창과 컨트롤 만들기 (보이기 전까지). 테스트에서도 사용
func createSetupWindow(p paths, x, y int) uintptr {
	g.p = p
	g.dpi = 1
	sdc, _, _ := pGetDC.Call(0)
	if v, _, _ := pGetDeviceCaps.Call(sdc, 90); v >= 96 {
		g.dpi = float64(v) / 96
	}
	pReleaseDC.Call(0, sdc)
	loadArt()
	g.bg, _, _ = pCreateSolidBrush.Call(bgColor)
	g.fontN, g.fontB, g.fontS = mkFont(15, 400), mkFont(26, 700), mkFont(12, 400)

	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("WoottangiSetup")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(wndProc), Instance: inst, ClassName: cls, Background: g.bg}
	wc.Cursor, _, _ = pLoadCursorW.Call(0, 32512)
	wc.Icon, _, _ = pLoadImageW.Call(inst, 1, 1, 0, 0, 0x40|0x8000)
	wc.IconSm = wc.Icon
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	r := RECT{0, 0, sc(480), sc(330)}
	pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), wsOverlappedWin, 0, 0)
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if x < -10000 { // 화면 가운데
		sw, _, _ := pGetSystemMetrics.Call(0)
		shh, _, _ := pGetSystemMetrics.Call(1)
		x, y = (int(sw)-w)/2, (int(shh)-h)/2
	}
	g.hwnd, _, _ = pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16(appName+" 설치"))),
		wsOverlappedWin, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, inst, 0)

	g.title = mk("STATIC", appName+" v"+version, ssNoPrefix, 200, 26, 260, 40, 0, g.fontB)
	g.desc = mk("STATIC", introText(), ssNoPrefix, 200, 72, 270, 70, 0, g.fontN)
	g.cbDesk = mk("BUTTON", "바탕화면에 바로가기 만들기", bsAutoCheck|wsTabStop, 200, 150, 270, 26, idDesktop, g.fontN)
	g.cbAuto = mk("BUTTON", "윈도우 켤 때 자동으로 실행", bsAutoCheck|wsTabStop, 200, 178, 270, 26, idAuto, g.fontN)
	g.cbRun = mk("BUTTON", "설치가 끝나면 바로 실행", bsAutoCheck|wsTabStop, 200, 206, 270, 26, idRun, g.fontN)
	g.pathLbl = mk("STATIC", "설치 위치: "+p.dir, ssNoPrefix|ssPathEllipsis, 24, 250, 436, 20, 0, g.fontS)
	label := "설치"
	if p.installed() {
		label = "업데이트"
	}
	g.btnCancel = mk("BUTTON", "취소", bsPush|wsTabStop, 256, 280, 100, 34, idCancel, g.fontN)
	g.btnOK = mk("BUTTON", label, bsDefPush|wsTabStop, 364, 280, 100, 34, idOK, g.fontN)
	setCheck(g.cbDesk, true)
	setCheck(g.cbAuto, p.autostartOn())
	setCheck(g.cbRun, true)
	return g.hwnd
}

func runSetupWindow(p paths) {
	createSetupWindow(p, -100000, 0)
	pShowWindow.Call(g.hwnd, 5)
	pUpdateWindow.Call(g.hwnd)
	pSetFocus.Call(g.btnOK)
	var m MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if r2, _, _ := pIsDialogMessageW.Call(g.hwnd, uintptr(unsafe.Pointer(&m))); r2 != 0 {
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func doInstall() {
	o := options{desktop: checked(g.cbDesk), autostart: checked(g.cbAuto), run: checked(g.cbRun)}
	g.busy = true
	pEnableWindow.Call(g.btnOK, 0)
	pEnableWindow.Call(g.btnCancel, 0)
	cur, _, _ := pLoadCursorW.Call(0, 32514) // 모래시계
	pSetCursor.Call(cur)
	setText(g.desc, "설치하는 중이에요…\r\n잠깐만 기다려 주세요.")
	pUpdateWindow.Call(g.desc)
	err := install(g.p, o, payload)
	g.busy = false
	pEnableWindow.Call(g.btnOK, 1)
	if err != nil {
		pEnableWindow.Call(g.btnCancel, 1)
		setText(g.desc, introText())
		msgBox(g.hwnd, "설치하지 못했어요.\n\n"+err.Error(), mbOK|mbIconError)
		return
	}
	g.done = true
	where := "시작 메뉴"
	if o.desktop {
		where = "바탕화면과 시작 메뉴"
	}
	setText(g.title, "설치 완료!")
	setText(g.desc, "우땅이가 설치됐어요.\r\n"+where+"에서\r\n\"우땅이\"를 찾을 수 있어요.")
	for _, h := range []uintptr{g.cbDesk, g.cbAuto, g.cbRun, g.btnCancel} {
		show(h, false)
	}
	setText(g.btnOK, "닫기")
	pSetFocus.Call(g.btnOK)
}

func wndProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case wmPaint:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		if g.art != nil {
			bi := BITMAPINFOHEADER{Size: 40, Width: int32(g.aw), Height: -int32(g.ah), Planes: 1, BitCount: 32}
			dw := 156
			dh := dw * g.ah / g.aw
			pSetStretchBltMode.Call(hdc, 4) // HALFTONE
			pStretchDIBits.Call(hdc, uintptr(sc(24)), uintptr(sc(22)), uintptr(sc(dw)), uintptr(sc(dh)), 0, 0, uintptr(g.aw), uintptr(g.ah),
				uintptr(unsafe.Pointer(&g.art[0])), uintptr(unsafe.Pointer(&bi)), 0, 0x00CC0020)
		}
		pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmCtlColorSt, wmCtlColorBtn:
		pSetBkMode.Call(w, 1) // TRANSPARENT
		col := uintptr(inkColor)
		if l == g.pathLbl {
			col = greyColor
		}
		pSetTextColor.Call(w, col)
		return g.bg
	case wmCommand:
		switch w & 0xffff {
		case idOK:
			if g.busy {
				return 0
			}
			if g.done {
				pDestroyWindow.Call(h)
			} else {
				doInstall()
			}
		case idCancel:
			if !g.busy {
				pDestroyWindow.Call(h)
			}
		}
		return 0
	case wmClose:
		if !g.busy {
			pDestroyWindow.Call(h)
		}
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), w, l)
	return r
}
