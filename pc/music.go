package main

// 음악이 나오면 춤추기
//   - 스피커로 나가는 소리 크기(Windows Core Audio 의 피크 미터)만 1초마다 봄. 소리 내용은 듣지 않음
//   - 소리가 몇 초 넘게 이어지면(알림음·짧은 효과음은 무시) 우땅이가 신나는 춤
//   - 한 번 추면 한동안 쉼. 집중 타이머 · 숨바꼭질 · 게임 중엔 안 춤

import (
	"math"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	ole32                = syscall.NewLazyDLL("ole32.dll")
	pCoInitializeEx      = ole32.NewProc("CoInitializeEx")
	pCoCreateInstance    = ole32.NewProc("CoCreateInstance")
	clsidMMDeviceEnum    = syscall.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnum     = syscall.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	iidIAudioMeterInfo   = syscall.GUID{Data1: 0xC02216F6, Data2: 0x8C67, Data3: 0x4B5B, Data4: [8]byte{0x9D, 0x00, 0xD0, 0x08, 0xE7, 0x3E, 0x00, 0x64}}
	musicPeak            atomic.Uint32 // math.Float32bits(최근 피크)
	musicOn              int           // 소리가 이어진 초
	musicNext            time.Time     // 다음에 춤출 수 있는 때
	lastMusicCheck       time.Time
	musicDisabledForTest bool
)

// COM 객체: 첫 칸이 메서드 표(vtable)
type comObj struct{ vt *[8]uintptr }

// COM 메서드 호출: obj 의 vtable[idx]
func comCall(obj *comObj, idx int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(obj.vt[idx], append([]uintptr{uintptr(unsafe.Pointer(obj))}, args...)...)
	return r
}

func comRelease(obj *comObj) {
	if obj != nil {
		comCall(obj, 2)
	}
}

// 기본 스피커의 피크 미터 (실패하면 0)
func openMeter() *comObj {
	var en *comObj
	if r, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidMMDeviceEnum)), 0, 0x17,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnum)), uintptr(unsafe.Pointer(&en))); r != 0 || en == nil {
		return nil
	}
	defer comRelease(en)
	var dev *comObj
	if comCall(en, 4, 0 /*eRender*/, 1 /*eMultimedia*/, uintptr(unsafe.Pointer(&dev))) != 0 || dev == nil {
		return nil
	}
	defer comRelease(dev)
	var meter *comObj
	if comCall(dev, 3, uintptr(unsafe.Pointer(&iidIAudioMeterInfo)), 0x17, 0, uintptr(unsafe.Pointer(&meter))) != 0 {
		return nil
	}
	return meter
}

// 따로 도는 스레드에서 1초마다 소리 크기를 잼 (스피커가 바뀌어도 30초마다 다시 잡음)
func startMusicWatch() {
	if musicDisabledForTest {
		return
	}
	go func() {
		runtime.LockOSThread()
		if r, _, _ := pCoInitializeEx.Call(0, 0 /*COINIT_MULTITHREADED*/); int32(r) < 0 {
			return
		}
		var meter *comObj
		var opened time.Time
		for {
			if meter == nil || time.Since(opened) > 30*time.Second {
				comRelease(meter)
				meter, opened = openMeter(), time.Now()
			}
			var v float32
			if meter != nil && comCall(meter, 3, uintptr(unsafe.Pointer(&v))) != 0 {
				comRelease(meter)
				meter, v = nil, 0
			}
			musicPeak.Store(math.Float32bits(v))
			time.Sleep(time.Second)
		}
	}()
}

// tick 에서 부름: 소리가 5초 넘게 이어지면 춤
func musicTick() {
	if time.Since(lastMusicCheck) < time.Second {
		return
	}
	lastMusicCheck = time.Now()
	if math.Float32frombits(musicPeak.Load()) > 0.03 {
		musicOn++
	} else {
		musicOn = 0
	}
	if musicWantsDance(musicOn, time.Now()) {
		musicDance()
	}
}

func musicWantsDance(on int, now time.Time) bool {
	return on >= 5 && now.After(musicNext) && !focusing() && !busy() && q == 0 && !hiddenFS &&
		(mode == "idle" || mode == "walk" || mode == "sit")
}

func musicDance() {
	musicNext = time.Now().Add(rd(50, 90))
	S.DanceCnt++
	setMode("dance", rd(6, 8))
	say(pick([]string{"음악이다! 둠칫둠칫~", "이 노래 좋아! 같이 춤춰~", "♪ 우땅우땅 신난다!", "흥이 난다 흥이 나~"}), 3*time.Second)
	saveState()
}
