package main

// 설치 · 업데이트 · 삭제 동작 (화면 없이). 관리자 권한 없이 현재 사용자에게만 설치합니다.
//
//   프로그램   %LOCALAPPDATA%\Programs\Woottangi\Woottangi.exe (+ uninstall.exe)
//   바로가기   시작 메뉴 "우땅이", (선택) 바탕화면 "우땅이"
//   제거 목록  HKCU\...\Uninstall\Woottangi  → 설정 > 앱 에서 지울 수 있음
//   자동 실행  HKCU\...\Run "Woottangi"     → 우땅이 메뉴의 "윈도우 켤 때 자동 실행" 과 같은 값
//   저장 파일  %AppData%\Woottangi\          → 업데이트·삭제해도 그대로 (삭제할 때 물어봄)

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var version = "1.2" // build.py 가 -ldflags "-X main.version=..." 로 넣음

const (
	appName    = "우땅이"
	exeName    = "Woottangi.exe"
	uninstName = "uninstall.exe"
	repoURL    = "https://github.com/yungi1013-netizen/woottangi"
)

// 설치에 쓰는 위치들 (테스트에서는 임시 폴더·테스트용 레지스트리 이름으로 바꿔 씀)
type paths struct {
	dir       string // 설치 폴더
	startMenu string // 시작 메뉴 프로그램 폴더
	desktop   string // 바탕화면
	data      string // 우땅이 저장 폴더 (%AppData%\Woottangi)
	uninstKey string // HKCU 아래 제거 목록 키
	runValue  string // HKCU Run 값 이름
	closePet  bool   // 실행 중인 우땅이를 닫을지 (테스트에서는 끔)
}

func defaultPaths() paths {
	la := os.Getenv("LOCALAPPDATA")
	cfg, _ := os.UserConfigDir()
	return paths{
		dir:       filepath.Join(la, "Programs", "Woottangi"),
		startMenu: knownFolder(&folderPrograms, filepath.Join(cfg, `Microsoft\Windows\Start Menu\Programs`)),
		desktop:   knownFolder(&folderDesktop, filepath.Join(os.Getenv("USERPROFILE"), "Desktop")),
		data:      filepath.Join(cfg, "Woottangi"),
		uninstKey: `Software\Microsoft\Windows\CurrentVersion\Uninstall\Woottangi`,
		runValue:  "Woottangi",
		closePet:  true,
	}
}

func (p paths) exe() string       { return filepath.Join(p.dir, exeName) }
func (p paths) uninst() string    { return filepath.Join(p.dir, uninstName) }
func (p paths) menuLnk() string   { return filepath.Join(p.startMenu, appName+".lnk") }
func (p paths) deskLnk() string   { return filepath.Join(p.desktop, appName+".lnk") }
func (p paths) installed() bool   { _, err := os.Stat(p.exe()); return err == nil }
func (p paths) runCmd() string    { return `"` + p.exe() + `"` }
func (p paths) autostartOn() bool { v, ok := regGetSZ(runKey, p.runValue); return ok && v != "" }

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

type options struct {
	desktop   bool // 바탕화면 바로가기
	autostart bool // 윈도우 켤 때 자동 실행
	run       bool // 설치 후 바로 실행
}

// 설치(또는 업데이트)
func install(p paths, o options, payload []byte) error {
	if p.closePet {
		closeRunningPet()
	}
	if err := os.MkdirAll(p.dir, 0755); err != nil {
		return fmt.Errorf("설치 폴더를 만들 수 없어요: %w", err)
	}
	if err := writeAtomic(p.exe(), payload); err != nil {
		return fmt.Errorf("우땅이를 복사하지 못했어요: %w", err)
	}
	// 지금 실행 중인 설치 프로그램을 제거 프로그램으로 복사
	if self, err := os.Executable(); err == nil && !samePath(self, p.uninst()) {
		if b, err := os.ReadFile(self); err == nil {
			writeAtomic(p.uninst(), b)
		}
	}
	lnks := []string{p.menuLnk()}
	if o.desktop {
		lnks = append(lnks, p.deskLnk())
	} else {
		os.Remove(p.deskLnk())
	}
	if err := makeShortcuts(lnks, p.exe(), p.dir); err != nil {
		return fmt.Errorf("바로가기를 만들지 못했어요: %w", err)
	}
	if err := writeUninstallEntry(p, len(payload)); err != nil {
		return fmt.Errorf("제거 정보를 기록하지 못했어요: %w", err)
	}
	if o.autostart {
		regSetSZ(runKey, p.runValue, p.runCmd())
	} else {
		regDeleteValue(runKey, p.runValue)
	}
	if o.run {
		c := exec.Command(p.exe())
		c.Dir = p.dir
		c.Start()
	}
	return nil
}

func writeUninstallEntry(p paths, size int) error {
	k := p.uninstKey
	for _, kv := range [][2]string{
		{"DisplayName", appName},
		{"DisplayVersion", version},
		{"Publisher", "K"},
		{"DisplayIcon", p.exe() + ",0"},
		{"InstallLocation", p.dir},
		{"UninstallString", `"` + p.uninst() + `" /uninstall`},
		{"QuietUninstallString", `"` + p.uninst() + `" /uninstall /S`},
		{"URLInfoAbout", repoURL},
		{"InstallDate", time.Now().Format("20060102")},
	} {
		if err := regSetSZ(k, kv[0], kv[1]); err != nil {
			return err
		}
	}
	regSetDWORD(k, "NoModify", 1)
	regSetDWORD(k, "NoRepair", 1)
	regSetDWORD(k, "EstimatedSize", uint32(size/1024))
	return nil
}

// 삭제. removeData 면 바나나·기록(저장 폴더)도 지움
func uninstall(p paths, removeData bool) error {
	if p.closePet {
		closeRunningPet()
	}
	os.Remove(p.menuLnk())
	os.Remove(p.deskLnk())
	if v, ok := regGetSZ(runKey, p.runValue); ok && strings.Contains(strings.ToLower(v), strings.ToLower(p.dir)) {
		regDeleteValue(runKey, p.runValue)
	}
	regDeleteTree(p.uninstKey)
	if err := os.Remove(p.exe()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("우땅이를 지우지 못했어요: %w", err)
	}
	if removeData && p.data != "" {
		os.RemoveAll(p.data)
	}
	// 지금 실행 중인 제거 프로그램이 설치 폴더 안에 있으면, 끝난 뒤 폴더를 지우도록 맡김
	if self, err := os.Executable(); err == nil && samePath(filepath.Dir(self), p.dir) {
		c := exec.Command("cmd")
		c.SysProcAttr = &syscall.SysProcAttr{
			CmdLine:       `cmd /c ping -n 3 127.0.0.1 >nul & rd /s /q "` + p.dir + `"`,
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		return c.Start()
	}
	return os.RemoveAll(p.dir)
}

// ── 도우미 ──

func writeAtomic(dst string, b []byte) error {
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, b, 0755); err != nil {
		return err
	}
	var err error
	for i := 0; i < 20; i++ { // 막 닫힌 exe 는 잠깐 잠겨 있을 수 있음
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	os.Remove(tmp)
	return err
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// 바로가기(.lnk) 만들기: WScript.Shell 사용. 경로는 환경 변수로 넘겨 한글·공백도 안전하게
func makeShortcuts(lnks []string, target, dir string) error {
	for _, l := range lnks {
		os.MkdirAll(filepath.Dir(l), 0755)
	}
	script := `$w=New-Object -ComObject WScript.Shell;` +
		`foreach($p in $env:WOOT_LNKS.Split('|')){$s=$w.CreateShortcut($p);$s.TargetPath=$env:WOOT_TARGET;` +
		`$s.WorkingDirectory=$env:WOOT_DIR;$s.IconLocation=$env:WOOT_TARGET+',0';$s.Description=$env:WOOT_DESC;$s.Save()}`
	c := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	c.Env = append(os.Environ(),
		"WOOT_LNKS="+strings.Join(lnks, "|"), "WOOT_TARGET="+target, "WOOT_DIR="+dir,
		"WOOT_DESC=바탕화면을 돌아다니는 우땅이")
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%v %s", err, strings.TrimSpace(string(out)))
	}
	for _, l := range lnks {
		if _, err := os.Stat(l); err != nil {
			return err
		}
	}
	return nil
}

// 실행 중인 우땅이를 정중하게 닫기 (꾸벅 없이 바로 종료) → 안 닫히면 강제로
func closeRunningPet() {
	cls := u16("WoottangiWalk")
	h, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0)
	if h == 0 {
		return
	}
	pPostMessageW.Call(h, 0x0010, 0, 0) // WM_CLOSE
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if h, _, _ = pFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0); h == 0 {
			time.Sleep(300 * time.Millisecond)
			return
		}
	}
	c := exec.Command("taskkill", "/F", "/IM", exeName)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	c.Run()
	time.Sleep(500 * time.Millisecond)
}
