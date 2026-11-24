package main

// 장난감 놀이: 공 · 풍선
//   - 공: 우땅이가 쫓아가서 뻥 차고, 떨어지는 공은 머리로 통
//   - 풍선: 둥실 내려오면 폴짝 뛰어 머리로 튕겨 올림
//   - 장난감을 누르면 톡 튀고, 끌어서 던지면 날아감. 실컷 놀면 잠깐 쉬었다가 또 놂
//   - 한동안(8분) 아무도 안 만지면 알아서 치움

import (
	"math"
	"time"
)

type toyT struct {
	th                   *thing
	kind                 string // ball · balloon
	vx, vy               float64
	L, R, T, F           float64 // 놀이 범위 (꺼낸 모니터의 작업 영역, F = 바닥)
	playUntil, restUntil time.Time
	lastTouch, lastHit   time.Time
	playing              bool
}

var toy *toyT

func toyOn() bool     { return toy != nil }
func toyActive() bool { return toy != nil && toy.playing }

// 우땅이가 놀 수 있는 상태 (바닥에 서 있고 다른 놀이 중이 아님)
func petFree() bool {
	return q == 0 && !airPerch && perch == 0 && !hiding && !catching && gachaAt.IsZero() &&
		!bathing && !tucked && !friendOn() && !focusing() && !quitting
}

func startToy(kind string) {
	if toy != nil && toy.kind == kind {
		say("벌써 꺼내 놨어~ 같이 놀자!", 2500*time.Millisecond)
		toy.wake(time.Now())
		return
	}
	if busy() || hiddenFS {
		say("지금은 못 해~ 조금 이따 놀자!", 2500*time.Millisecond)
		return
	}
	if toy != nil {
		putAwayToy(false)
	}
	detachSurface()
	perch = 0
	c := cw()
	size := int(c * 0.36)
	if kind == "balloon" {
		size = int(c * 0.5)
	}
	t := &toyT{kind: kind, th: newThing(kind, size), L: workL, R: workR, T: workT, F: ground + float64(canvasH) - 2}
	r := t.th.r()
	t.th.x = math.Max(t.L+r, math.Min(t.R-r, posX+c/2+dir*c*1.3))
	if kind == "ball" {
		t.th.y = posY + c*0.2
		t.vy = -200 * dpi
	} else {
		t.th.y = t.F - c*1.6
		t.th.scale = 1
	}
	t.th.onClick = func() { t.poke() }
	t.th.onThrow = func(vx, vy float64) { t.vx, t.vy = vx, vy; t.wake(time.Now()) }
	toy = t
	now := time.Now()
	t.lastTouch = now
	t.wake(now)
	S.ToyCnt++
	S.Mood = clamp(S.Mood + 8)
	if S.ToyDay != today() {
		S.ToyDay = today()
		addExp(3)
	} else {
		saveState()
	}
	setMode("laugh", 1500*time.Millisecond)
	if kind == "ball" {
		say(pick([]string{"공이다! 같이 놀자~", "뻥 차 볼게!", "공놀이 좋아!"}), 2500*time.Millisecond)
	} else {
		say(pick([]string{"풍선이다! 안 떨어뜨릴 거야~", "둥실둥실 풍선!", "풍선 통통 놀이!"}), 2500*time.Millisecond)
	}
	sfx("pop")
}

func putAwayToy(talk bool) {
	if toy == nil {
		return
	}
	toy.th.close()
	toy = nil
	if talk {
		say("장난감은 치워 둘게~ 또 놀자!", 2500*time.Millisecond)
	}
}

// 다른 놀이를 시작할 때: 잠깐 쉼
func pauseToy() {
	if toy != nil && toy.playing {
		toy.playing = false
		toy.restUntil = time.Now().Add(2 * time.Minute)
	}
}

// 다시 놀기 시작
func (t *toyT) wake(now time.Time) {
	t.lastTouch = now
	if !t.playing {
		t.playing = true
		t.playUntil = now.Add(rd(70, 110))
	}
}

// 장난감을 톡 누름
func (t *toyT) poke() {
	if t.kind == "ball" {
		t.vy = -rf(600, 800) * dpi
		t.vx = rf(-220, 220) * dpi
		sfx("boing")
	} else {
		t.vy = -rf(260, 340) * dpi
		t.vx = rf(-60, 60) * dpi
		sfx("pop")
	}
	t.wake(time.Now())
}

// 공·풍선의 몸통 (가운데, 반지름)
func (t *toyT) body() (x, y, r float64) {
	s := float64(t.th.size)
	if t.kind == "balloon" {
		return t.th.x, t.th.y - s*0.1, s * 0.3
	}
	return t.th.x, t.th.y, s * 0.4
}

// 장난감 바닥 (그림 아래 끝에서 가운데까지)
func (t *toyT) bottomOff() float64 {
	if t.kind == "balloon" {
		return float64(t.th.size) * 0.45
	}
	return float64(t.th.size) * 0.4
}

func (t *toyT) physics(now time.Time, dt float64) {
	th := t.th
	if th.held {
		t.vx, t.vy = 0, 0
		return
	}
	r := th.r() * 0.8
	bo := t.bottomOff()
	if t.kind == "ball" {
		t.vy += 1500 * dpi * dt
		t.vx *= 1 - 0.3*dt
	} else {
		t.vy += 70 * dpi * dt
		t.vx *= 1 - 1.2*dt
		t.vy *= 1 - 0.8*dt
	}
	th.x += t.vx * dt
	th.y += t.vy * dt
	if th.y+bo > t.F {
		th.y = t.F - bo
		if t.kind == "ball" && t.vy > 140*dpi {
			t.vy = -t.vy * 0.55
			t.vx *= 0.85
		} else if t.kind == "balloon" && t.vy > 20*dpi {
			t.vy = -t.vy * 0.3
		} else {
			t.vy = 0
		}
	}
	if th.y+bo >= t.F-0.5 && t.kind == "ball" { // 굴러가다 멈춤
		t.vx *= 1 - 1.4*dt
		if math.Abs(t.vx) < 6*dpi {
			t.vx = 0
		}
	}
	if th.x < t.L+r {
		th.x, t.vx = t.L+r, math.Abs(t.vx)*0.7
	}
	if th.x > t.R-r {
		th.x, t.vx = t.R-r, -math.Abs(t.vx)*0.7
	}
	if th.y-r < t.T {
		th.y, t.vy = t.T+r, math.Abs(t.vy)*0.5
	}
	if t.kind == "ball" {
		th.rot += t.vx / math.Max(1, r) * dt
	} else {
		th.rot = math.Sin(float64(now.UnixMilli())/1000*1.3)*0.12 + math.Max(-0.3, math.Min(0.3, t.vx*0.0006/dpi))
	}
}

// 우땅이 머리에 닿으면 통 튕김
func (t *toyT) headBump(now time.Time) {
	if !t.playing || t.th.held || now.Sub(t.lastHit) < 250*time.Millisecond || mode == "drag" || q != 0 {
		return
	}
	c := cw()
	bx, by, br := t.body()
	hx0, hx1 := posX+c*0.28, posX+c*0.72
	hy0, hy1 := posY+c*0.12, posY+c*0.5
	nx, ny := math.Max(hx0, math.Min(hx1, bx)), math.Max(hy0, math.Min(hy1, by))
	if math.Hypot(bx-nx, by-ny) > br {
		return
	}
	rising := mode == "fall" && velY < 0
	if t.vy < 0 && !rising { // 이미 위로 날아가는 중
		return
	}
	pc := posX + c/2
	if t.kind == "ball" {
		t.vy = -rf(520, 700) * dpi
		t.vx = (bx-pc)*4 + rf(-60, 60)*dpi
		sfx("boing")
	} else {
		t.vy = -rf(300, 420) * dpi
		t.vx = (bx-pc)*2.5 + rf(-50, 50)*dpi
		sfx("pop")
	}
	t.hit(now)
	if rng.Float64() < 0.25 {
		say(pick([]string{"통!", "헤딩!", "머리로 통~"}), 1200*time.Millisecond)
	}
}

func (t *toyT) hit(now time.Time) {
	t.lastHit = now
	S.ToyKick++
	S.Mood = clamp(S.Mood + 0.5)
}

// 우땅이가 장난감 쪽으로 (toyActive 이고 놀 수 있을 때)
func (t *toyT) petAI(now time.Time) {
	if !petFree() || mode == "fall" || mode == "drag" {
		return
	}
	if mode != "walk" && mode != "idle" && now.Before(modeUntil) {
		return // 웃기 같은 반응이 끝날 때까지
	}
	c := cw()
	pc := posX + c/2
	bx, by, _ := t.body()
	d := bx - pc
	face := math.Copysign(1, d)
	if t.th.held { // 들고 있으면 쳐다보기만
		dir = face
		waitIdle()
		return
	}
	floorTop := t.F - c*0.6
	if t.kind == "ball" {
		low := by > floorTop
		if math.Abs(d) > c*0.3 || !low {
			if !low && math.Abs(d) < c*0.25 { // 공이 위에 있으면 밑에서 기다림
				dir = face
				waitIdle()
				return
			}
			chaseTo(face)
			return
		}
		speed := math.Hypot(t.vx, t.vy)
		if speed < 260*dpi && now.Sub(t.lastHit) > 700*time.Millisecond {
			dir = face
			t.vx = face * rf(220, 420) * dpi
			t.vy = -rf(200, 520) * dpi
			t.hit(now)
			sfx("boing")
			setMode("laugh", 700*time.Millisecond)
			if rng.Float64() < 0.3 {
				say(pick([]string{"뻥!", "슛~!", "간다~!", "받아라!"}), 1500*time.Millisecond)
			}
			return
		}
		dir = face
		waitIdle()
		return
	}
	// 풍선: 바닥까지 내려왔으면 발로 톡, 머리 높이로 내려오면 밑에서 폴짝
	if math.Abs(d) < c*0.3 && by > posY+c*0.55 && now.Sub(t.lastHit) > 500*time.Millisecond {
		dir = face
		t.vy = -rf(420, 520) * dpi
		t.vx = face * rf(20, 80) * dpi
		t.hit(now)
		sfx("pop")
		setMode("laugh", 600*time.Millisecond)
		return
	}
	if math.Abs(d) < c*0.28 && by > posY-c*0.35 && t.vy >= 0 {
		dir = face
		velX, velY = 0, -540*dpi
		maxFallSpeed, bounces, thrown = 0, 0, false
		setMode("fall", time.Hour)
		return
	}
	if math.Abs(d) > c*0.18 {
		chaseTo(face)
		return
	}
	dir = face
	waitIdle()
}

// 제자리에서 지켜보기 (이미 서 있으면 그대로 두어 숨쉬기 그림이 이어지게)
func waitIdle() {
	if mode != "idle" {
		setMode("idle", 2*time.Second)
	}
}

func chaseTo(face float64) {
	dir = face
	if mode != "walk" {
		setMode("walk", 5*time.Second)
	}
}

// 걷는 속도 배율: 장난감 쫓을 땐 빨리
func walkBoost() float64 {
	if toyActive() {
		return 2.4
	}
	return 1
}

// 매 프레임
func toyTick(now time.Time, dt float64) {
	t := toy
	if t == nil {
		return
	}
	t.th.dragTick(now)
	t.physics(now, dt)
	if t.th.held || t.th.dragging {
		t.lastTouch = now
	}
	if t.playing && now.After(t.playUntil) { // 실컷 놀았음
		t.playing = false
		t.restUntil = now.Add(rd(40, 90))
		if petFree() && mode != "fall" {
			setMode("sit", rd(6, 10))
			say(pick([]string{"헉헉 재밌었다~ 잠깐 쉴래!", "후~ 신난다! 조금만 쉬고 또 하자", "놀다 보니 숨차~"}), 3*time.Second)
		}
	}
	if !t.playing && now.After(t.restUntil) && petFree() && (mode == "idle" || mode == "walk") && !busy() && rng.Float64() < 0.01 {
		t.wake(now)
		say(pick([]string{"또 놀자!", "장난감 어디 있지? 저기다!"}), 2*time.Second)
	}
	if t.playing {
		t.headBump(now)
		t.petAI(now)
	}
	if now.Sub(t.lastTouch) > 8*time.Minute && !t.playing {
		putAwayToy(true)
		return
	}
	t.th.draw()
}

// ── 메뉴 ──

const (
	idToyBall    = 1440
	idToyBalloon = 1441
	idToyAway    = 1442
)

func appendToyMenu(gm uintptr) {
	tm, _, _ := pCreatePopupMenu.Call()
	appendMenu(tm, MF_STRING, idToyBall, "공 꺼내기")
	appendMenu(tm, MF_STRING, idToyBalloon, "풍선 꺼내기")
	f := uintptr(MF_STRING)
	if toy == nil {
		f |= MF_GRAYED
	}
	appendMenu(tm, f, idToyAway, "장난감 치우기")
	appendMenu(gm, MF_POPUP, tm, "장난감")
}

func onToyCmd(cmd uintptr) bool {
	switch cmd {
	case idToyBall:
		startToy("ball")
	case idToyBalloon:
		startToy("balloon")
	case idToyAway:
		putAwayToy(true)
	default:
		return false
	}
	return true
}
