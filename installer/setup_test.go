package main

// go test .          설치 → 업데이트 → 삭제를 임시 폴더·테스트용 레지스트리 이름으로 확인 (실제 설치는 건드리지 않음)
// WOOT_SHOT=1 go test -run TestWindowShot .   설치 화면을 test_setup.png 로 저장

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

func testPaths(t *testing.T) paths {
	root := t.TempDir()
	p := paths{
		dir:       filepath.Join(root, "Programs", "Woottangi"),
		startMenu: filepath.Join(root, "StartMenu"),
		desktop:   filepath.Join(root, "Desktop 바탕화면"),
		data:      filepath.Join(root, "AppData", "Woottangi"),
		uninstKey: `Software\Microsoft\Windows\CurrentVersion\Uninstall\WoottangiSetupTest`,
		runValue:  "WoottangiSetupTest",
	}
	t.Cleanup(func() {
		regDeleteTree(p.uninstKey)
		regDeleteValue(runKey, p.runValue)
	})
	return p
}

func exists(f string) bool { _, err := os.Stat(f); return err == nil }

func TestInstallUpdateUninstall(t *testing.T) {
	if real := defaultPaths(); real.uninstKey == testPaths(t).uninstKey {
		t.Fatal("테스트가 실제 레지스트리 키를 씀")
	}
	p := testPaths(t)
	os.MkdirAll(p.data, 0755)
	os.WriteFile(filepath.Join(p.data, "state.json"), []byte(`{"bananas":146}`), 0644)

	if p.installed() {
		t.Fatal("설치 전인데 설치됨")
	}
	if err := install(p, options{desktop: true, autostart: true}, payload); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{p.exe(), p.uninst(), p.menuLnk(), p.deskLnk()} {
		if !exists(f) {
			t.Fatalf("없음: %s", f)
		}
	}
	if b, _ := os.ReadFile(p.exe()); len(b) != len(payload) {
		t.Fatal("exe 크기 다름")
	}
	if v, _ := regGetSZ(p.uninstKey, "DisplayName"); v != "우땅이" {
		t.Fatalf("제거 목록 이름: %q", v)
	}
	if v, _ := regGetSZ(p.uninstKey, "UninstallString"); v != `"`+p.uninst()+`" /uninstall` {
		t.Fatalf("UninstallString: %q", v)
	}
	if v, _ := regGetSZ(runKey, p.runValue); v != p.runCmd() || !p.autostartOn() {
		t.Fatalf("자동 실행 값: %q", v)
	}

	// 업데이트: 바탕화면 바로가기·자동 실행 끄기 → 지워지고, 저장 파일은 그대로
	if err := install(p, options{desktop: false, autostart: false}, payload); err != nil {
		t.Fatal(err)
	}
	if exists(p.deskLnk()) || p.autostartOn() || !exists(p.menuLnk()) {
		t.Fatal("업데이트 후 바로가기/자동 실행 상태 이상")
	}
	if !exists(filepath.Join(p.data, "state.json")) {
		t.Fatal("업데이트가 저장 파일을 지움")
	}

	// 삭제 (기록은 남김)
	install(p, options{desktop: true, autostart: true}, payload)
	if err := uninstall(p, false); err != nil {
		t.Fatal(err)
	}
	if exists(p.dir) || exists(p.menuLnk()) || exists(p.deskLnk()) || p.autostartOn() {
		t.Fatal("삭제 후 남은 것이 있음")
	}
	if _, ok := regGetSZ(p.uninstKey, "DisplayName"); ok {
		t.Fatal("제거 목록이 남음")
	}
	if !exists(filepath.Join(p.data, "state.json")) {
		t.Fatal("기록을 남기기로 했는데 지워짐")
	}

	// 다시 설치 후 기록까지 삭제
	install(p, options{desktop: true}, payload)
	if err := uninstall(p, true); err != nil {
		t.Fatal(err)
	}
	if exists(p.data) {
		t.Fatal("기록 삭제를 골랐는데 남음")
	}
}

func TestUninstallKeepsOtherAutostart(t *testing.T) {
	p := testPaths(t)
	install(p, options{}, payload)
	regSetSZ(runKey, p.runValue, `"D:\다른곳\Woottangi.exe"`) // 다른 위치의 우땅이 자동 실행은 건드리지 않음
	uninstall(p, false)
	if v, _ := regGetSZ(runKey, p.runValue); v == "" {
		t.Fatal("다른 위치의 자동 실행 값을 지움")
	}
}

func TestDefaultPaths(t *testing.T) {
	p := defaultPaths()
	if !filepath.IsAbs(p.dir) || !filepath.IsAbs(p.startMenu) || !filepath.IsAbs(p.desktop) || filepath.Base(p.data) != "Woottangi" {
		t.Fatalf("기본 경로 이상: %+v", p)
	}
	t.Logf("설치 %s\n시작메뉴 %s\n바탕화면 %s", p.dir, p.startMenu, p.desktop)
}

// 설치 화면 그림 확인용 (화면 밖에 창을 만들어 PrintWindow 로 찍음)
func TestWindowShot(t *testing.T) {
	if os.Getenv("WOOT_SHOT") == "" {
		t.Skip("WOOT_SHOT=1 일 때만")
	}
	runtime.LockOSThread() // 창을 만든 스레드에서 메시지를 처리해야 함
	defer runtime.UnlockOSThread()
	p := testPaths(t)
	h := createSetupWindow(p, -4000, 100)
	pShowWindow.Call(h, 4) // SW_SHOWNOACTIVATE
	pUpdateWindow.Call(h)
	shot := func(name string) {
		var r RECT
		user32.NewProc("GetWindowRect").Call(h, uintptr(unsafe.Pointer(&r)))
		w, hh := int(r.Right-r.Left), int(r.Bottom-r.Top)
		sdc, _, _ := pGetDC.Call(0)
		mdc, _, _ := pCreateCompatibleDC.Call(sdc)
		bi := BITMAPINFOHEADER{Size: 40, Width: int32(w), Height: -int32(hh), Planes: 1, BitCount: 32}
		var bits unsafe.Pointer
		bmp, _, _ := pCreateDIBSection.Call(mdc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		pSelectObject.Call(mdc, bmp)
		pPrintWindow.Call(h, mdc, 0)
		px := unsafe.Slice((*byte)(bits), w*hh*4)
		img := image.NewNRGBA(image.Rect(0, 0, w, hh))
		for i := 0; i < w*hh; i++ {
			img.SetNRGBA(i%w, i/w, color.NRGBA{px[i*4+2], px[i*4+1], px[i*4], 255})
		}
		f, _ := os.Create(name)
		png.Encode(f, img)
		f.Close()
		pDeleteObject.Call(bmp)
		pDeleteDC.Call(mdc)
		pReleaseDC.Call(0, sdc)
	}
	pump := func() {
		var m MSG
		peek := user32.NewProc("PeekMessageW")
		for {
			if r, _, _ := peek.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1); r == 0 {
				return
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	pump()
	t.Log("shot1")
	shot("test_setup.png")
	t.Log("install")
	// 설치 완료 화면 (테스트 경로로 실제 설치)
	setCheck(g.cbRun, false)
	setCheck(g.cbAuto, false)
	doInstall()
	t.Log("installed")
	pump()
	shot("test_setup_done.png")
	pDestroyWindow.Call(h)
	pump()
}
