package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	pRegisterClassExW      = user32.NewProc("RegisterClassExW")
	pCreateWindowExW       = user32.NewProc("CreateWindowExW")
	pDefWindowProcW        = user32.NewProc("DefWindowProcW")
	pDestroyWindow         = user32.NewProc("DestroyWindow")
	pShowWindow            = user32.NewProc("ShowWindow")
	pUpdateWindow          = user32.NewProc("UpdateWindow")
	pGetMessageW           = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessageW      = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW      = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pPostMessageW          = user32.NewProc("PostMessageW")
	pSendMessageW          = user32.NewProc("SendMessageW")
	pMessageBoxW           = user32.NewProc("MessageBoxW")
	pFindWindowW           = user32.NewProc("FindWindowW")
	pSetWindowTextW        = user32.NewProc("SetWindowTextW")
	pEnableWindow          = user32.NewProc("EnableWindow")
	pGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	pAdjustWindowRectEx    = user32.NewProc("AdjustWindowRectEx")
	pLoadCursorW           = user32.NewProc("LoadCursorW")
	pSetCursor             = user32.NewProc("SetCursor")
	pLoadImageW            = user32.NewProc("LoadImageW")
	pBeginPaint            = user32.NewProc("BeginPaint")
	pEndPaint              = user32.NewProc("EndPaint")
	pGetDC                 = user32.NewProc("GetDC")
	pReleaseDC             = user32.NewProc("ReleaseDC")
	pSetProcessDPIAware    = user32.NewProc("SetProcessDPIAware")
	pSetFocus              = user32.NewProc("SetFocus")
	pPrintWindow           = user32.NewProc("PrintWindow")
	pGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
	pCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	pCreateFontW           = gdi32.NewProc("CreateFontW")
	pSetBkMode             = gdi32.NewProc("SetBkMode")
	pSetTextColor          = gdi32.NewProc("SetTextColor")
	pStretchDIBits         = gdi32.NewProc("StretchDIBits")
	pSetStretchBltMode     = gdi32.NewProc("SetStretchBltMode")
	pGetDeviceCaps         = gdi32.NewProc("GetDeviceCaps")
	pCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	pSelectObject          = gdi32.NewProc("SelectObject")
	pDeleteObject          = gdi32.NewProc("DeleteObject")
	pDeleteDC              = gdi32.NewProc("DeleteDC")
	pRegCreateKeyExW       = advapi32.NewProc("RegCreateKeyExW")
	pRegOpenKeyExW         = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW        = advapi32.NewProc("RegSetValueExW")
	pRegQueryValueExW      = advapi32.NewProc("RegQueryValueExW")
	pRegDeleteValueW       = advapi32.NewProc("RegDeleteValueW")
	pRegDeleteTreeW        = advapi32.NewProc("RegDeleteTreeW")
	pRegCloseKey           = advapi32.NewProc("RegCloseKey")
	pSHGetKnownFolderPath  = shell32.NewProc("SHGetKnownFolderPath")
	pCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
)

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}
type WNDCLASSEX struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type PAINTSTRUCT struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}
type BITMAPINFOHEADER struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}

// ── 레지스트리 (HKCU) ──

const (
	hkcu          = 0x80000001
	keyAllAccess  = 0xF003F
	keyRead       = 0x20019
	regSZ         = 1
	regDWORD      = 4
	errorNotFound = 2
)

func regSetSZ(key, name, val string) error {
	var k uintptr
	if r, _, _ := pRegCreateKeyExW.Call(hkcu, uintptr(unsafe.Pointer(u16(key))), 0, 0, 0, keyAllAccess, 0, uintptr(unsafe.Pointer(&k)), 0); r != 0 {
		return syscall.Errno(r)
	}
	defer pRegCloseKey.Call(k)
	v, _ := syscall.UTF16FromString(val)
	if r, _, _ := pRegSetValueExW.Call(k, uintptr(unsafe.Pointer(u16(name))), 0, regSZ, uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2)); r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func regSetDWORD(key, name string, val uint32) error {
	var k uintptr
	if r, _, _ := pRegCreateKeyExW.Call(hkcu, uintptr(unsafe.Pointer(u16(key))), 0, 0, 0, keyAllAccess, 0, uintptr(unsafe.Pointer(&k)), 0); r != 0 {
		return syscall.Errno(r)
	}
	defer pRegCloseKey.Call(k)
	if r, _, _ := pRegSetValueExW.Call(k, uintptr(unsafe.Pointer(u16(name))), 0, regDWORD, uintptr(unsafe.Pointer(&val)), 4); r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func regGetSZ(key, name string) (string, bool) {
	var k uintptr
	if r, _, _ := pRegOpenKeyExW.Call(hkcu, uintptr(unsafe.Pointer(u16(key))), 0, keyRead, uintptr(unsafe.Pointer(&k))); r != 0 {
		return "", false
	}
	defer pRegCloseKey.Call(k)
	buf := make([]uint16, 2048)
	n := uint32(len(buf) * 2)
	var typ uint32
	if r, _, _ := pRegQueryValueExW.Call(k, uintptr(unsafe.Pointer(u16(name))), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r != 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func regDeleteValue(key, name string) {
	var k uintptr
	if r, _, _ := pRegOpenKeyExW.Call(hkcu, uintptr(unsafe.Pointer(u16(key))), 0, keyAllAccess, uintptr(unsafe.Pointer(&k))); r != 0 {
		return
	}
	defer pRegCloseKey.Call(k)
	pRegDeleteValueW.Call(k, uintptr(unsafe.Pointer(u16(name))))
}

func regDeleteTree(key string) {
	pRegDeleteTreeW.Call(hkcu, uintptr(unsafe.Pointer(u16(key))))
	// RegDeleteTree 는 하위만 지우는 경우가 있어 키 자체도 한 번 더
	advapi32.NewProc("RegDeleteKeyW").Call(hkcu, uintptr(unsafe.Pointer(u16(key))))
}

// ── 알려진 폴더 (OneDrive 로 옮긴 바탕화면도 정확히) ──

type GUID struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}

var (
	folderDesktop  = GUID{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
	folderPrograms = GUID{0xA77F5D77, 0x2E2B, 0x44C3, [8]byte{0xA6, 0xA2, 0xAB, 0xA6, 0x01, 0x05, 0x4A, 0x51}}
)

func knownFolder(id *GUID, fallback string) string {
	var p *uint16
	if r, _, _ := pSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(id)), 0, 0, uintptr(unsafe.Pointer(&p))); r != 0 || p == nil {
		return fallback
	}
	defer pCoTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), n*2)) != 0 {
		n++
	}
	s := syscall.UTF16ToString(unsafe.Slice(p, n))
	if s == "" {
		return fallback
	}
	return s
}

func msgBox(owner uintptr, text string, flags uintptr) int {
	r, _, _ := pMessageBoxW.Call(owner, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(appName+" 설치"))), flags)
	return int(r)
}

const (
	mbOK          = 0x0
	mbYesNo       = 0x4
	mbIconInfo    = 0x40
	mbIconQuest   = 0x20
	mbIconError   = 0x10
	mbDefButton2  = 0x100
	idYes         = 6
)
