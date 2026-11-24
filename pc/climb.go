package main

// 벽 타기 · 천장 매달리기 · 내 자리 (안드로이드 우땅이와 같은 동작)
//   - 걷다가 화면 끝(진짜 벽)에 닿으면 가끔 벽을 타고 올라감 → 꼭대기에서 천장으로 → 반대쪽 벽으로 내려오거나 뚝
//   - 위로 세게 던지면 가끔 천장을 잡고 매달림
//   - 살살 놓기: 벽·천장 가까우면 붙고, 공중이면 그 자리에 앉아 있다가 폴짝 내려옴
//   - 공중에 앉아 있을 때 '여기를 내 자리로' → 가끔 폴짝 뛰어 그 자리로 올라감
//
// 우땅이 창은 정사각형이라 돌려도 창 크기는 그대로. 그림만 q 만큼 돌려서 그림 (render)

import (
	"math"
	"strings"
	"time"
)

var (
	q         int     // 0 바닥·공중 · 1 왼쪽 벽(시계 90°) · 2 천장(180°) · 3 오른쪽 벽(반시계 90°)
	dirY      = -1.0  // 벽에서 -1 위로 · 1 아래로
	wantClimb bool    // 메뉴에서 '벽 타기': 가까운 벽으로 걸어가는 중
	wantCeil  bool    // 천장까지 쭉
	seatJump  bool    // 내 자리로 폴짝 뛰는 중
	airPerch  bool    // 공중(내 자리 등)에 앉아 있음
	airY      float64 // 앉은 높이 (posY)
	airLeft   float64 // 앉아 있을 남은 시간 (초)
	cursorOn  = true  // 테스트에서는 꺼서 실제 마우스에 반응하지 않게
)

const climbChance = 0.4

var ceilGrabChance = 0.5 // 위로 세게 던졌을 때 천장을 잡을 확률

func cw() float64 { return float64(canvasW) }

func away() bool { return q != 0 || airPerch }

// 진짜 벽인지 (옆에 모니터가 이어져 있으면 벽이 아님)
func wallAt(side int) bool {
	lo, hi := floorBounds()
	if side == 1 {
		return !math.IsInf(lo, -1)
	}
	return !math.IsInf(hi, 1)
}

func ceilLo() float64  { return workL - cw()*0.15 }
func ceilHi() float64  { return workR - cw()*0.85 }
func wallTop() float64 { return workT - cw()*0.15 }

func climbSpeed() float64 { return 55 * eff() * speedMul() }

func canClimb() bool {
	return !stay && autoMove && !busy() && !hiddenFS && perch == 0 && !toyActive()
}

func toWall(side int, dy float64) {
	q, dirY = side, dy
	if side == 1 {
		posX = workL - 1
	} else {
		posX = workR - cw() + 1
	}
	if dy < 0 {
		posY = ground
	} else {
		posY = wallTop()
	}
	perch, airPerch = 0, false
	velX, velY = 0, 0
	wantClimb = false
	setMode("climb", rd(3, 7))
	if rng.Float64() < 0.35 {
		say(pick([]string{"영차영차~", "벽 타기 선수 우땅이!", "높이 올라가 볼까?"}), 2500*time.Millisecond)
	}
}

func toCeiling(x, d float64) {
	q, dir = 2, d
	if d == 0 {
		dir = 1
	}
	posX = math.Max(ceilLo(), math.Min(ceilHi(), x))
	posY = workT - 1
	perch, airPerch = 0, false
	velX, velY = 0, 0
	wantCeil, wantClimb = false, false
	setMode("climb", rd(4, 8))
}

func toFloor(from int) {
	q = 0
	posY = ground
	if from == 1 {
		posX, dir = ceilLo(), 1
	} else {
		posX, dir = ceilHi(), -1
	}
	setMode("idle", rd(1, 2))
}

// 벽·천장 그림 → 보통 그림 (창은 정사각형이라 자리는 그대로, 공중이면 tick 이 떨어뜨림)
func leaveSurface() {
	if q == 0 {
		return
	}
	q = 0
	lo, hi := floorBounds()
	posX = math.Max(lo, math.Min(hi, posX))
	posY = math.Min(posY, ground)
}

// 벽·천장에서 손 놓기
func letGo() {
	was := q
	leaveSurface()
	switch was {
	case 1:
		velX, velY = 220*dpi, -120*dpi
	case 3:
		velX, velY = -220*dpi, -120*dpi
	default:
		velX, velY = 0, 0
	}
	maxFallSpeed, bounces, thrown = 0, 0, false
	setMode("fall", time.Hour)
}

// 벽·천장·자리에 있으면 바닥으로
func comeDown() {
	if q != 0 {
		letGo()
	} else if airPerch {
		hopDown()
	}
}

// 다른 놀이를 시작할 때: 붙어 있던 곳에서 떨어지기만 (떨어지는 건 tick 이)
func detachSurface() {
	leaveSurface()
	airPerch, wantClimb, wantCeil, seatJump = false, false, false, false
}

// setMode 가 부름: 벽·천장·자리에서 할 수 없는 동작이면 내려옴
func surfaceModeHook(m string) {
	surf := m == "climb" || m == "cling"
	if q != 0 && !surf && m != "fall" && m != "drag" {
		leaveSurface()
	}
	if airPerch {
		switch m {
		case "fall", "drag", "walk", "catch", "hide", "climb", "cling", "bath":
			airPerch = false
		}
	}
}

func climbTick(now time.Time, dt float64) {
	if q == 0 { // 혹시 어긋나면 그냥 떨어짐
		setMode("fall", time.Hour)
		return
	}
	sp := climbSpeed()
	if mode == "climb" {
		if q == 1 || q == 3 {
			posY += dirY * sp * dt
			if dirY < 0 && posY <= wallTop() {
				posY = wallTop()
				if wantCeil || rng.Float64() < 0.65 {
					if q == 1 {
						toCeiling(ceilLo(), 1)
					} else {
						toCeiling(ceilHi(), -1)
					}
					return
				}
				dirY = 1
			}
			if dirY > 0 && posY >= ground {
				toFloor(q)
				return
			}
		} else {
			posX += dir * sp * dt
			if dir > 0 && posX >= ceilHi() {
				posX = ceilHi()
				if wallAt(3) && rng.Float64() < 0.6 {
					toWall(3, 1)
					return
				}
				dir = -1
			}
			if dir < 0 && posX <= ceilLo() {
				posX = ceilLo()
				if wallAt(1) && rng.Float64() < 0.6 {
					toWall(1, 1)
					return
				}
				dir = 1
			}
		}
	}
	if now.Before(modeUntil) {
		return
	}
	if mode == "cling" {
		setMode("climb", rd(3, 7))
		return
	}
	if wantCeil && q != 2 { // 천장 가는 중엔 쭉 올라감
		dirY = -1
		setMode("climb", rd(3, 7))
		return
	}
	switch r := rng.Float64(); {
	case r < 0.22:
		letGo()
		if rng.Float64() < 0.5 {
			say(pick([]string{"뿅! 착지 준비~", "에잇, 뛰어내리기!"}), 2*time.Second)
		}
	case r < 0.55:
		setMode("cling", rd(2, 4))
	case r < 0.7:
		if q == 2 {
			dir = -dir
		} else {
			dirY = -dirY
		}
		setMode("climb", rd(3, 6))
	default:
		setMode("climb", rd(3, 7))
	}
}

// 벽·천장에서의 좌우 뒤집기 (걷는 그림은 원래 오른쪽을 봄)
func climbFlip() bool {
	switch q {
	case 1:
		return dirY < 0
	case 3:
		return dirY > 0
	}
	return dir > 0
}

// 걷다가 화면 끝에 닿았을 때 (true = 벽에 붙음)
func tryClimbAtEdge(left bool) bool {
	side := 3
	if left {
		side = 1
	}
	if !wallAt(side) || !canClimb() || !(wantClimb || rng.Float64() < climbChance) {
		return false
	}
	toWall(side, -1)
	return true
}

// ── 살살 놓기 ──

// 살살 놓았을 때: 벽·천장 가까우면 붙고, 공중이면 그 자리에 앉음. 아니면 false (그냥 떨어짐)
func placeGently() bool {
	c := cw()
	cx, cy := posX+c/2, posY+c*0.55
	edge := c * 0.75 // 몸(창 가운데 절반)이 화면 끝에 닿을 만큼
	if cy > workT+c*0.6 && cy < ground+c*0.8 {
		if cx < workL+edge && wallAt(1) {
			attachWall(1, cy)
			return true
		}
		if cx > workR-edge && wallAt(3) {
			attachWall(3, cy)
			return true
		}
	}
	if posY < workT+c*0.12 {
		toCeiling(posX, dir)
		setMode("cling", rd(2, 4))
		say(pick([]string{"천장에 착!", "거꾸로 매달리기~"}), 2200*time.Millisecond)
		return true
	}
	if posY < ground-c*0.3 {
		// 바로 아래에 올라앉을 창이 있으면 원래대로 창 위로 떨어짐
		if perchEnabled {
			feet := posY + feetOff()
			if _, _, ok := findPerch(feet, feet+c*2); ok {
				return false
			}
		}
		perchAt(math.Max(posY, workT), rd(25, 60).Seconds())
		say(pick([]string{"여기 앉아 있을게!", "높다~ 경치 좋다!", "여기 좋은 자리네~"}), 2500*time.Millisecond)
		return true
	}
	return false
}

func attachWall(side int, cy float64) {
	toWall(side, -1)
	posY = math.Max(wallTop(), math.Min(ground, cy-cw()/2))
	setMode("cling", rd(2, 4))
	say(pick([]string{"벽에 착 붙었다!", "매달려 있기~"}), 2200*time.Millisecond)
}

// 공중에 앉기
func perchAt(y, sec float64) {
	q, perch = 0, 0
	airPerch, airY, airLeft = true, y, sec
	posY = y
	velX, velY = 0, 0
	setMode("sit", rd(4, 8))
}

func hopDown() {
	airPerch = false
	if rng.Float64() < 0.5 {
		velX = 120 * dpi
	} else {
		velX = -120 * dpi
	}
	velY = -260 * dpi
	maxFallSpeed, bounces, thrown = 0, 0, false
	setMode("fall", time.Hour)
	if rng.Float64() < 0.5 {
		say(pick([]string{"폴짝!", "이제 내려갈래~"}), 2*time.Second)
	}
}

// 공중에 앉아 있을 때 (tick 에서): 시간이 다 되면 폴짝
func airPerchTick(dt float64) bool {
	if !airPerch {
		return false
	}
	airLeft -= dt
	if airLeft <= 0 && !strings.HasPrefix(mode, "eat") {
		hopDown()
		return true
	}
	return false
}

// 공중 자리 위에서 고르는 동작: 앉고 쉬고 졸기만
func perchedChoice() {
	switch r := rng.Float64(); {
	case r < 0.2:
		setMode("sleep", rd(10, 20))
	case r < 0.55:
		setMode("sit", rd(5, 10))
	case r < 0.7:
		setMode("st_lounge", rd(5, 8))
	case r < 0.78:
		setMode("wave", rd(2.5, 3.2))
	default:
		setMode("idle", rd(2, 4))
	}
}

// ── 내 자리 ──

func hasSeat() bool { return S.HasSeat }

// 내 자리가 지금 모니터 안에 있으면 그 위치
func seatPos() (x, y float64, ok bool) {
	if !S.HasSeat {
		return 0, 0, false
	}
	c := cw()
	cx := S.SeatX + c/2
	if cx < workL || cx > workR || S.SeatY < workT-c || S.SeatY > ground {
		return 0, 0, false
	}
	return math.Max(ceilLo(), math.Min(ceilHi(), S.SeatX)), math.Max(workT, math.Min(ground-c*0.3, S.SeatY)), true
}

// 지금 앉아 있는 곳을 내 자리로
func setSeat() {
	if !airPerch {
		return
	}
	S.HasSeat, S.SeatX, S.SeatY = true, posX, posY
	saveState()
	airLeft = math.Max(airLeft, rd(60, 120).Seconds())
	setMode("laugh", 2500*time.Millisecond)
	say("여기를 내 자리로 할게!\n가끔 여기 올라와 있을게~", 3500*time.Millisecond)
}

func clearSeat() {
	S.HasSeat, S.SeatX, S.SeatY = false, 0, 0
	saveState()
	say("알았어! 자리는 다시 정해 줘~", 2500*time.Millisecond)
}

// 바닥에서 내 자리까지 폴짝 (포물선)
func jumpToSeat() bool {
	sx, sy, ok := seatPos()
	if !ok || away() || busy() || toyActive() || mode == "fall" || mode == "drag" {
		return false
	}
	c := cw()
	if posY-sy < c*0.3 {
		return false
	}
	g, hd := 1800*dpi, c*0.15
	vy0 := -math.Sqrt(2 * g * (posY - sy + hd))
	T := -vy0/g + math.Sqrt(2*hd/g)
	velX, velY = (sx-posX)/T, vy0
	perch = 0
	seatJump, thrown, bounces, maxFallSpeed = true, false, 0, 0
	if velX >= 0 {
		dir = 1
	} else {
		dir = -1
	}
	setMode("fall", time.Hour)
	say(pick([]string{"내 자리로 점프!", "영차~ 폴짝!"}), 2*time.Second)
	return true
}

// 떨어지는 중 내 자리에 닿았으면 앉음 (fallTick 에서)
func seatLand() bool {
	if !seatJump || velY <= 0 {
		return false
	}
	sx, sy, ok := seatPos()
	if !ok {
		seatJump = false
		return false
	}
	if posY < sy {
		return false
	}
	posX = sx
	seatJump, thrown, maxFallSpeed, bounces = false, false, 0, 0
	perchAt(sy, rd(60, 180).Seconds())
	if rng.Float64() < 0.6 {
		say(pick([]string{"내 자리 도착!", "여기가 제일 좋아~", "폴짝! 내 자리!"}), 2500*time.Millisecond)
	}
	return true
}

// 메뉴: 벽 타기 · 천장 매달리기
func goClimb(ceiling bool) {
	if busy() || hiddenFS || mode == "drag" || mode == "fall" {
		say("지금은 못 해~ 조금 이따!", 2500*time.Millisecond)
		return
	}
	if q != 0 {
		if ceiling && q != 2 {
			wantCeil = true
			dirY = -1
			setMode("climb", rd(3, 7))
		} else {
			say("벌써 매달려 있어~", 2*time.Second)
		}
		return
	}
	left, right := wallAt(1), wallAt(3)
	if !left && !right {
		say("여기엔 탈 벽이 없어~", 2500*time.Millisecond)
		return
	}
	stay = false
	cx := posX + cw()/2
	switch {
	case left && (!right || cx < (workL+workR)/2):
		dir = -1
	default:
		dir = 1
	}
	wantClimb, wantCeil = true, ceiling
	if airPerch {
		hopDown()
	} else {
		perch = 0
		setMode("walk", 30*time.Second)
	}
	if ceiling {
		say("천장까지 올라가 볼게!", 2500*time.Millisecond)
	} else {
		say("벽 타기 보여 줄게!", 2500*time.Millisecond)
	}
}

// ── 메뉴 ──

const (
	idClimb     = 1430
	idCeil      = 1431
	idSeatSet   = 1432
	idSeatGo    = 1433
	idSeatClear = 1434
	idComeDown  = 1435
)

// 놀아주기 메뉴에 덧붙이기
func appendClimbPlayMenu(gm uintptr) {
	appendMenu(gm, MF_SEPARATOR, 0, "")
	appendMenu(gm, MF_STRING, idClimb, "벽 타기")
	appendMenu(gm, MF_STRING, idCeil, "천장 매달리기")
}

// 메인 메뉴: 자리 · 내려오기
func appendSeatMenu(m uintptr) {
	if airPerch {
		appendMenu(m, MF_STRING, idSeatSet, "여기를 내 자리로 정하기")
	}
	if hasSeat() {
		if !away() {
			f := uintptr(MF_STRING)
			if _, _, ok := seatPos(); !ok {
				f |= MF_GRAYED
			}
			appendMenu(m, f, idSeatGo, "내 자리로 가기")
		}
		appendMenu(m, MF_STRING, idSeatClear, "내 자리 지우기")
	}
	if away() {
		appendMenu(m, MF_STRING, idComeDown, "내려오기")
	}
}

func onClimbCmd(cmd uintptr) bool {
	switch cmd {
	case idClimb:
		goClimb(false)
	case idCeil:
		goClimb(true)
	case idSeatSet:
		setSeat()
	case idSeatGo:
		if !jumpToSeat() {
			say("지금은 못 가~ 조금 이따!", 2500*time.Millisecond)
		}
	case idSeatClear:
		clearSeat()
	case idComeDown:
		comeDown()
	default:
		return false
	}
	return true
}
