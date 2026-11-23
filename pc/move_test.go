package main

import (
	"testing"
	"time"
)

func setupMove(t *testing.T) {
	statePathOverride = t.TempDir()
	t.Cleanup(func() { statePathOverride = "" })
	perchEnabled = false
	dpi, scale = 1, 0.75
	canvasW, canvasH = 112, 112
	workL, workR, workT = 0, 1920, 0
	ground = 1040 - float64(canvasH) + 2
	greeted, perch, hiddenFS = true, 0, false
	S = petState{Bananas: 10, Full: 80, Mood: 80, NoSound: true}
	mons, monOverride, curMon = nil, nil, -1
	focusUntil = time.Time{}
}

// 오른쪽 위로 휙 던지면: 날아가서 벽에 튕기고, 바닥에 통통 튄 뒤 착지
func TestThrowBounceLand(t *testing.T) {
	setupMove(t)
	posX, posY = 1500, 500
	t0 := time.Now()
	dragN = 0
	for i := 0; i < 6; i++ { // 16ms 마다 오른쪽 40px, 위로 25px
		dragHist[dragN%len(dragHist)] = struct {
			x, y float64
			t    time.Time
		}{1500 + float64(i*40), 500 - float64(i*25), t0.Add(time.Duration(i*16) * time.Millisecond)}
		dragN++
	}
	posX, posY = 1700, 375
	onDragRelease()
	if !thrown || velX < 2000 || velY > -1000 {
		t.Fatalf("던지기 속도 이상: thrown=%v v=(%.0f, %.0f)", thrown, velX, velY)
	}
	hitWall, minY := false, posY
	sawBounce := false
	for i := 0; i < 600 && mode == "fall"; i++ {
		fallTick(1.0 / 60)
		lo, hi := walkBounds()
		if posX < lo-0.01 || posX > hi+0.01 {
			t.Fatalf("화면 밖으로 나감: x=%.0f", posX)
		}
		if velX < 0 {
			hitWall = true
		}
		if bounces > 0 {
			sawBounce = true
		}
		minY = min(minY, posY)
	}
	if mode == "fall" || posY != ground {
		t.Fatalf("착지 못 함: mode=%s y=%.0f ground=%.0f", mode, posY, ground)
	}
	if !hitWall || !sawBounce || minY >= 375 {
		t.Fatalf("벽 튕김 %v, 바닥 통통 %v, 위로 날아감 %v", hitWall, sawBounce, minY < 375)
	}
	if velX != 0 || velY != 0 || thrown {
		t.Fatal("착지 후 속도가 남음")
	}
}

// 살짝 놓으면 그냥 떨어짐 (던지기 아님)
func TestDropNotThrow(t *testing.T) {
	setupMove(t)
	dragN = 1
	posX, posY = 800, 900
	onDragRelease()
	if thrown || velX != 0 {
		t.Fatal("가만히 놓았는데 던져짐")
	}
	for i := 0; i < 300 && mode == "fall"; i++ {
		fallTick(1.0 / 60)
	}
	if posY != ground || posX != 800 {
		t.Fatalf("제자리에 떨어지지 않음: (%.0f, %.0f)", posX, posY)
	}
}

func TestSpeedAndSettings(t *testing.T) {
	setupMove(t)
	if speedMul() != 1 {
		t.Fatal("기본 속도")
	}
	onSettingsCmd(idSpeed + 2) // 빠르게
	if speedMul() != 1.5 || S.Speed != 2 {
		t.Fatal("빠르게 설정 안 됨")
	}
	S.Mood = 20 // 시무룩하면 느릿느릿
	if d := speedMul() - 0.9; d > 1e-9 || d < -1e-9 {
		t.Fatalf("시무룩 속도 %.2f", speedMul())
	}
	onSettingsCmd(idOpacity + 2)
	if S.Opacity != 2 || opacity() != 178 {
		t.Fatal("투명도 설정 안 됨")
	}
	onSettingsCmd(idAutoHide)
	if !S.NoAutoHide {
		t.Fatal("전체화면 숨기 끄기 안 됨")
	}
	if onSettingsCmd(idQuit) {
		t.Fatal("설정이 아닌 명령을 처리함")
	}
}

func TestFocusTimer(t *testing.T) {
	setupMove(t)
	startFocus(25)
	if !focusing() || focusLeft() != 25 || mode != "sit" {
		t.Fatalf("집중 시작 이상: %v %d %s", focusing(), focusLeft(), mode)
	}
	nextChoice()
	if mode != "sit" {
		t.Fatal("집중 중인데 돌아다님")
	}
	focusUntil = time.Now().Add(-time.Second)
	focusTick(time.Now())
	if focusing() || S.Bananas != 15 || S.FocusCnt != 1 || S.FocusMin != 25 || S.Exp != 10 {
		t.Fatalf("집중 보상 이상: %+v", S)
	}
	startFocus(50)
	stopFocus()
	if focusing() || S.FocusCnt != 1 {
		t.Fatal("그만하기 이상")
	}
	if got := checkAchievements(); len(got) == 0 || S.Ach["focus1"] == "" {
		t.Fatalf("집중 업적 없음: %v", got)
	}
}
