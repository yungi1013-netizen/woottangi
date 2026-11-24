package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
	"time"
	"unsafe"
)

func setupClimb(t *testing.T) {
	setupMove(t)
	for _, n := range []string{"walk", "idle", "blink", "sit", "sitblink", "sleep", "laugh", "wave", "st_lounge", "st_surprise", "st_me", "st_bow", "shy"} {
		if seqs[n] == nil {
			loadSeq(n)
		}
	}
	cursorOn, stay, autoMove = false, false, true
	q, dirY, airPerch, wantClimb, wantCeil, seatJump = 0, -1, false, false, false, false
	velX, velY, thrown = 0, 0, false
	posX, posY = 900, ground
	setMode("idle", time.Second)
	t.Cleanup(func() {
		cursorOn = true
		q, airPerch, wantClimb, wantCeil, seatJump = 0, false, false, false, false
	})
}

// 시간을 흘려 보내며 움직이기 (mode 시간도 같이 당김). check 가 false 면 멈춤
func simMove(sec float64, check func() bool) {
	dt := 1.0 / 30
	step := time.Duration(dt * float64(time.Second))
	for i := 0; i < int(sec/dt); i++ {
		modeUntil = modeUntil.Add(-step)
		modeStart = modeStart.Add(-step)
		moveTick(time.Now(), dt)
		if check != nil && !check() {
			return
		}
	}
}

// 화면 안에 있고, 벽·천장에서는 딱 붙어 있는지
func surfaceOK(t *testing.T) func() bool {
	c := float64(canvasW)
	return func() bool {
		bad := ""
		switch {
		case posX < workL-c*0.15-1.01 || posX > workR-c*0.85+1.01:
			bad = "가로로 화면 밖"
		case posY < workT-c*0.25-1.01 || posY > ground+0.01:
			bad = "세로로 화면 밖"
		case q == 1 && posX != workL-1, q == 3 && posX != workR-c+1:
			bad = "벽에서 떨어져 있음"
		case q == 2 && posY != workT-1:
			bad = "천장에서 떨어져 있음"
		case q != 0 && mode != "climb" && mode != "cling":
			bad = "벽인데 다른 동작"
		case q == 0 && !airPerch && perch == 0 && mode != "fall" && mode != "drag" && posY < ground-1:
			bad = "공중에 떠 있음"
		}
		if bad != "" {
			t.Fatalf("%s: q=%d mode=%s pos=(%.1f, %.1f)", bad, q, mode, posX, posY)
		}
		return true
	}
}

// 메뉴 '천장 매달리기': 가까운 벽으로 걸어가서 → 벽 타고 → 천장까지
func TestClimbToCeiling(t *testing.T) {
	setupClimb(t)
	goClimb(true)
	if !wantClimb || dir != -1 {
		t.Fatalf("왼쪽 벽으로 안 감: want=%v dir=%.0f", wantClimb, dir)
	}
	ok := surfaceOK(t)
	saw := map[int]bool{}
	simMove(120, func() bool { ok(); saw[q] = true; return q != 2 })
	if !saw[1] || q != 2 {
		t.Fatalf("천장까지 못 감: 벽 %v, 지금 q=%d mode=%s", saw[1], q, mode)
	}
	// 천장에서 놀다가 결국 바닥으로 돌아옴
	back := false
	simMove(300, func() bool { ok(); back = q == 0 && posY == ground && mode != "fall"; return !back })
	if !back {
		t.Fatalf("천장에서 안 내려옴: q=%d mode=%s", q, mode)
	}
}

// 오래 혼자 놀아도 화면 밖으로 안 나가고 공중에 안 뜸, 벽·천장도 가 봄
func TestClimbLongRun(t *testing.T) {
	setupClimb(t)
	ok := surfaceOK(t)
	saw := map[int]bool{}
	for round := 0; round < 16; round++ { // 최대 4시간
		simMove(15*60, func() bool { ok(); saw[q] = true; return true })
		if saw[1] && saw[2] && saw[3] {
			break
		}
	}
	if !saw[1] || !saw[3] {
		t.Fatalf("4시간 동안 양쪽 벽을 다 안 탐: %v", saw)
	}
}

// 벽에서 톡 → 손 놓고 떨어져서 바닥에 착지
func TestTapOnWall(t *testing.T) {
	setupClimb(t)
	toWall(3, -1)
	posY = 400
	letGo()
	if q != 0 || mode != "fall" || velX >= 0 {
		t.Fatalf("손을 안 놓음: q=%d mode=%s vx=%.0f", q, mode, velX)
	}
	simMove(5, func() bool { return mode == "fall" })
	if posY != ground || mode == "fall" {
		t.Fatalf("착지 못 함: y=%.0f mode=%s", posY, mode)
	}
	// 벽에서 간식 같은 다른 동작을 시키면 내려옴
	toWall(1, -1)
	posY = 500
	setMode("eat", 3*time.Second)
	if q != 0 {
		t.Fatal("벽에서 다른 동작을 해도 그대로")
	}
	simMove(5, func() bool { return posY < ground })
	if posY != ground {
		t.Fatalf("바닥으로 안 내려옴: y=%.0f", posY)
	}
}

// 살살 놓기: 벽 옆 → 벽, 꼭대기 → 천장, 공중 → 그 자리에 앉았다가 폴짝
func TestGentleDrop(t *testing.T) {
	setupClimb(t)
	gently := func(x, y float64) {
		startDrag()
		posX, posY = x, y
		dragN = 0
		onDragRelease()
	}
	gently(workL+5, 500)
	if q != 1 || mode != "cling" {
		t.Fatalf("왼쪽 벽에 안 붙음: q=%d mode=%s", q, mode)
	}
	gently(workR-float64(canvasW)-5, 300)
	if q != 3 || mode != "cling" {
		t.Fatalf("오른쪽 벽에 안 붙음: q=%d mode=%s", q, mode)
	}
	gently(900, workT+5)
	if q != 2 || posY != workT-1 {
		t.Fatalf("천장에 안 붙음: q=%d y=%.0f", q, posY)
	}
	gently(900, 400)
	if !airPerch || q != 0 || posY != 400 {
		t.Fatalf("공중에 안 앉음: perch=%v q=%d y=%.0f", airPerch, q, posY)
	}
	simMove(20, surfaceOK(t))
	if !airPerch || posY != 400 {
		t.Fatalf("금방 떨어짐: y=%.0f mode=%s", posY, mode)
	}
	simMove(60, func() bool { return airPerch || mode == "fall" })
	simMove(5, func() bool { return mode == "fall" })
	if airPerch || posY != ground {
		t.Fatalf("시간이 지나도 안 내려옴: y=%.0f", posY)
	}
	// 바닥 근처에서 살살 놓으면 그냥 떨어짐
	gently(900, ground-10)
	if airPerch || q != 0 || mode != "fall" {
		t.Fatalf("바닥 근처인데 붙음: perch=%v q=%d mode=%s", airPerch, q, mode)
	}
}

// 내 자리: 정하고 → 저장되고 → 바닥에서 폴짝 뛰어 정확히 그 자리로
func TestSeatJump(t *testing.T) {
	setupClimb(t)
	posX = 1300
	perchAt(330, 40)
	setSeat()
	if !S.HasSeat || S.SeatX != 1300 || S.SeatY != 330 {
		t.Fatalf("자리 저장 안 됨: %+v", S)
	}
	comeDown()
	simMove(5, func() bool { return mode == "fall" })
	posX = 300
	setMode("idle", time.Second)
	if !jumpToSeat() {
		t.Fatal("자리로 못 뜀")
	}
	minY := posY
	simMove(5, func() bool { minY = math.Min(minY, posY); return mode == "fall" })
	if !airPerch || math.Abs(posX-1300) > 0.01 || posY != 330 || minY > 330-float64(canvasH)*0.1 {
		t.Fatalf("자리에 정확히 안 앉음: perch=%v pos=(%.1f, %.1f) 최고 %.0f", airPerch, posX, posY, minY)
	}
	simMove(30, surfaceOK(t))
	if !airPerch {
		t.Fatal("자리에서 금방 내려옴")
	}
	// 앉아 있을 때 고르는 동작은 앉기·쉬기만
	for i := 0; i < 50; i++ {
		perchedChoice()
		switch mode {
		case "sleep", "sit", "st_lounge", "wave", "idle":
		default:
			t.Fatalf("자리 위에서 %s", mode)
		}
	}
	// 숨바꼭질을 하면 자리에서 내려옴
	detachSurface()
	if airPerch {
		t.Fatal("놀이 시작했는데 그대로")
	}
	clearSeat()
	if S.HasSeat || jumpToSeat() {
		t.Fatal("자리를 지웠는데 남아 있음")
	}
}

// 위로 세게 던지면 천장을 탁 잡음
func TestThrowGrabCeiling(t *testing.T) {
	setupClimb(t)
	ceilGrabChance = 1
	posX, posY = 900, 300
	velX, velY, thrown = 300, -2500, true
	setMode("fall", time.Hour)
	simMove(2, func() bool { return mode == "fall" })
	if q != 2 || mode != "cling" || posY != workT-1 {
		t.Fatalf("천장을 안 잡음: q=%d mode=%s y=%.0f", q, mode, posY)
	}
}

// 벽·천장 그림: 발이 벽(창 가장자리)에 붙어 있는지 (test_climb.png 로 저장해서 눈으로 확인)
func TestClimbRender(t *testing.T) {
	setupClimb(t)
	buf := make([]byte, canvasW*canvasH*4)
	bits = unsafe.Pointer(&buf[0])
	defer func() { bits = nil }()
	out := image.NewNRGBA(image.Rect(0, 0, canvasW*4, canvasH))
	edge := func() (l, r, t, b int) { // 그림이 있는 범위
		l, t = canvasW, canvasH
		for y := 0; y < canvasH; y++ {
			for x := 0; x < canvasW; x++ {
				if buf[(y*canvasW+x)*4+3] > 128 {
					l, r, t, b = min(l, x), max(r, x), min(t, y), max(b, y)
				}
			}
		}
		return
	}
	for k := 0; k < 4; k++ {
		q = k
		if k == 0 {
			setMode("walk", time.Second)
		} else {
			setMode("climb", time.Second)
		}
		render()
		l, r, tp, b := edge()
		limit := canvasW / 12
		switch k {
		case 0:
			if b < canvasH-limit {
				t.Errorf("바닥: 발이 아래에 없음 (b=%d)", b)
			}
		case 1:
			if l > limit {
				t.Errorf("왼쪽 벽: 발이 왼쪽에 없음 (l=%d)", l)
			}
		case 2:
			if tp > limit {
				t.Errorf("천장: 발이 위에 없음 (t=%d)", tp)
			}
		case 3:
			if r < canvasW-limit {
				t.Errorf("오른쪽 벽: 발이 오른쪽에 없음 (r=%d)", r)
			}
		}
		for y := 0; y < canvasH; y++ {
			for x := 0; x < canvasW; x++ {
				o := (y*canvasW + x) * 4
				a := float32(buf[o+3]) / 255
				px := func(v byte) uint8 { return uint8(float32(v) + 235*(1-a) + 0.5) }
				out.SetNRGBA(k*canvasW+x, y, color.NRGBA{px(buf[o+2]), px(buf[o+1]), px(buf[o]), 255})
			}
		}
	}
	q = 0
	if f, err := os.Create("test_climb.png"); err == nil {
		png.Encode(f, out)
		f.Close()
	}
}
