package main

// 설정 메뉴: 크기 · 투명도 · 걷는 속도 · 전체화면일 때 숨기 · 자동 실행 (크기·투명도·속도는 저장됨)

import "math"

var (
	scales       = []float64{0.75, 1.0, 1.35}
	scaleNames   = []string{"작게", "보통", "크게"}
	opacities    = []byte{255, 217, 178, 140}
	opacityNames = []string{"100% (불투명)", "85%", "70%", "55%"}
	speedNames   = []string{"보통", "느리게", "빠르게"}
)

const (
	idOpacity  = 1080 // +0..3
	idSpeed    = 1085 // +0..2
	idAutoHide = 1090
)

func opacity() byte { return opacities[clampI(S.Opacity, 0, len(opacities)-1)] }

// 크기 바꾸기 (발 위치·가운데를 그대로)
func applyScale(sc float64) {
	if sc <= 0 || math.Abs(sc-scale) < 0.001 {
		return
	}
	detachSurface() // 크기가 바뀌면 일단 내려옴
	bottom := posY + float64(canvasH)
	cx := posX + float64(canvasW)/2
	scale = sc
	setupCanvas()
	refreshWorkArea()
	posX = cx - float64(canvasW)/2
	posY = bottom - float64(canvasH)
}

// 저장된 크기로 시작 (loadState 뒤)
func applySavedSettings() {
	for _, s := range scales {
		if math.Abs(S.Scale-s) < 0.01 {
			applyScale(s)
		}
	}
}

func appendSettingsMenu(m uintptr) {
	st, _, _ := pCreatePopupMenu.Call()
	sz, _, _ := pCreatePopupMenu.Call()
	for i, sc := range scales {
		f := uintptr(MF_STRING)
		if math.Abs(scale-sc) < 0.01 {
			f |= MF_CHECKED
		}
		appendMenu(sz, f, uintptr(idSmall+i), scaleNames[i])
	}
	appendMenu(st, MF_POPUP, sz, "크기")
	op, _, _ := pCreatePopupMenu.Call()
	for i, n := range opacityNames {
		f := uintptr(MF_STRING)
		if clampI(S.Opacity, 0, len(opacities)-1) == i {
			f |= MF_CHECKED
		}
		appendMenu(op, f, uintptr(idOpacity+i), n)
	}
	appendMenu(st, MF_POPUP, op, "투명도")
	sp, _, _ := pCreatePopupMenu.Call()
	for i, n := range speedNames {
		f := uintptr(MF_STRING)
		if clampI(S.Speed, 0, 2) == i {
			f |= MF_CHECKED
		}
		appendMenu(sp, f, uintptr(idSpeed+i), n)
	}
	appendMenu(st, MF_POPUP, sp, "걷는 속도")
	appendMonitorMenu(st)
	appendSoundMenu(st)
	appendMenu(st, MF_SEPARATOR, 0, "")
	f := uintptr(MF_STRING)
	if !S.NoAutoHide {
		f |= MF_CHECKED
	}
	appendMenu(st, f, idAutoHide, "전체화면(게임·영상·발표)일 때 숨기")
	f = MF_STRING
	if autostartEnabled() {
		f |= MF_CHECKED
	}
	appendMenu(st, f, idStart, "윈도우 켤 때 자동 실행")
	appendMenu(m, MF_POPUP, st, "설정")
}

// 설정 메뉴 명령 처리 (처리했으면 true)
func onSettingsCmd(cmd uintptr) bool {
	switch {
	case cmd >= idSmall && cmd <= idBig:
		applyScale(scales[cmd-idSmall])
		S.Scale = scale
	case cmd >= idOpacity && cmd < idOpacity+uintptr(len(opacities)):
		S.Opacity = int(cmd - idOpacity)
	case cmd >= idSpeed && cmd < idSpeed+3:
		S.Speed = int(cmd - idSpeed)
	case cmd == idAutoHide:
		S.NoAutoHide = !S.NoAutoHide
	case cmd == idStart:
		setAutostart(!autostartEnabled())
		return true
	default:
		return onMonitorCmd(cmd)
	}
	saveState()
	return true
}
