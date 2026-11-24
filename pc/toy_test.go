package main

import (
	"strings"
	"testing"
	"time"
)

func setupToy(t *testing.T) {
	setupClimb(t)
	if len(fxSprites) == 0 {
		loadFX()
	}
	thingWindows = false
	S.Full, S.Mood = 80, 50
	t.Cleanup(func() {
		putAwayToy(false)
		for _, g := range gifts {
			g.th.close()
		}
		gifts, nextGiftAt = nil, time.Time{}
		thingWindows = true
	})
}

// 시간을 흘려 보내며 우땅이 + 장난감 움직이기 (장난감 시각도 같이 당김)
func simToy(sec float64, check func() bool) {
	dt := 1.0 / 30
	step := time.Duration(dt * float64(time.Second))
	for i := 0; i < int(sec/dt); i++ {
		modeUntil = modeUntil.Add(-step)
		modeStart = modeStart.Add(-step)
		if toy != nil {
			toy.playUntil = toy.playUntil.Add(-step)
			toy.restUntil = toy.restUntil.Add(-step)
			toy.lastTouch = toy.lastTouch.Add(-step)
			toy.lastHit = toy.lastHit.Add(-step)
		}
		now := time.Now()
		moveTick(now, dt)
		toyTick(now, dt)
		if check != nil && !check() {
			return
		}
	}
}

func toyInBounds(t *testing.T) func() bool {
	return func() bool {
		if toy == nil {
			return true
		}
		th := toy.th
		r := th.r()
		if th.x < toy.L+r*0.8-0.01 || th.x > toy.R-r*0.8+0.01 || th.y+toy.bottomOff() > toy.F+0.01 || th.y < toy.T {
			t.Fatalf("장난감이 화면 밖: (%.0f, %.0f)", th.x, th.y)
		}
		if posX < workL-cw()*0.15-0.01 || posX > workR-cw()*0.85+0.01 || posY > ground+0.01 {
			t.Fatalf("우땅이가 화면 밖: (%.0f, %.0f)", posX, posY)
		}
		return true
	}
}

// 공: 쫓아가서 여러 번 차고, 공도 우땅이도 화면 안
func TestToyBall(t *testing.T) {
	setupToy(t)
	posX = 300
	startToy("ball")
	if toy == nil || !toyActive() || S.ToyCnt != 1 {
		t.Fatal("공을 못 꺼냄")
	}
	toy.th.x = 1500 // 멀리 있는 공
	simToy(40, toyInBounds(t))
	if S.ToyKick < 4 {
		t.Fatalf("공을 별로 안 참: %d번", S.ToyKick)
	}
	// 실컷 놀면 쉼
	toy.playUntil = time.Now().Add(-time.Second)
	simToy(0.1, nil)
	if toyActive() {
		t.Fatal("놀이가 안 끝남")
	}
	// 누르면 다시 놀기
	toy.poke()
	if !toyActive() || toy.vy >= 0 {
		t.Fatal("눌러도 안 튐")
	}
	// 오래 안 만지면 치움
	toy.playing = false
	toy.lastTouch = time.Now().Add(-9 * time.Minute)
	simToy(0.1, nil)
	if toy != nil {
		t.Fatal("안 치움")
	}
}

// 풍선: 내려오면 폴짝 뛰어 머리로 통통
func TestToyBalloon(t *testing.T) {
	setupToy(t)
	posX = 800
	startToy("balloon")
	hops := 0
	simToy(40, func() bool {
		toyInBounds(t)()
		if mode == "fall" && velY < 0 {
			hops++
		}
		return true
	})
	if S.ToyKick < 3 || hops == 0 {
		t.Fatalf("풍선을 별로 안 튕김: %d번 (뛴 프레임 %d)", S.ToyKick, hops)
	}
}

// 장난감 놀이 중엔 벽 타기·자리 점프를 안 함, 다른 놀이를 하면 쉼
func TestToyBlocksOthers(t *testing.T) {
	setupToy(t)
	startToy("ball")
	if canClimb() {
		t.Fatal("놀이 중에 벽 타기")
	}
	pauseToy()
	if toyActive() {
		t.Fatal("다른 놀이를 해도 계속 놂")
	}
	startToy("balloon") // 다른 장난감으로 바꾸기
	if toy == nil || toy.kind != "balloon" {
		t.Fatal("풍선으로 안 바뀜")
	}
}

// 선물: 발밑에 두고 → 누르면 모음에 쌓임 → 40분 지나면 사라짐
func TestGifts(t *testing.T) {
	setupToy(t)
	seen := map[string]bool{}
	for i := 0; i < 3000; i++ {
		seen[pickGift(1+i%12).id] = true
	}
	if len(seen) != len(giftKinds) {
		t.Fatalf("안 나오는 선물이 있음: %v", seen)
	}
	autumn := 0
	for i := 0; i < 2000; i++ {
		if g := pickGift(10); g.id == "maple" || g.id == "ginkgo" || g.id == "acorn" {
			autumn++
		}
	}
	if autumn < 600 {
		t.Fatalf("가을인데 낙엽이 적음: %d/2000", autumn)
	}
	setMode("walk", 5*time.Second)
	nextGiftAt = time.Now().Add(-time.Second)
	giftTick(time.Now())
	if len(gifts) != 1 {
		t.Fatal("선물을 안 둠")
	}
	g := gifts[0]
	if g.th.y+g.th.r() > ground+float64(canvasH) || g.th.y < ground {
		t.Fatalf("선물이 바닥에 없음: y=%.0f", g.th.y)
	}
	b := S.Bananas
	g.kind = &giftKinds[0] // 바나나
	collectGift(g)
	if len(gifts) != 0 || S.Gifts["banana"] != 1 || S.Bananas != b+2 || S.GiftCnt != 1 {
		t.Fatalf("주웠는데 이상: %v %d→%d", S.Gifts, b, S.Bananas)
	}
	// 오래된 선물은 사라짐
	dropGift(&giftKinds[2])
	gifts[0].born = time.Now().Add(-giftLife - time.Second)
	giftTick(time.Now())
	if len(gifts) != 0 {
		t.Fatal("오래된 선물이 남음")
	}
	// 바쁠 땐 안 두고 30초 뒤 다시
	hiding = true
	nextGiftAt = time.Now().Add(-time.Second)
	giftTick(time.Now())
	hiding = false
	if len(gifts) != 0 || time.Until(nextGiftAt) > time.Minute {
		t.Fatal("숨바꼭질 중에 선물을 둠")
	}
	// 수첩
	h := bookHTML(S, "지수")
	if !strings.Contains(h, "선물 모음") || !strings.Contains(h, `/fx/banana.png`) || !strings.Contains(h, "바나나 탑 최고") {
		t.Fatal("수첩에 선물 모음이 없음")
	}
}

func TestTowerScore(t *testing.T) {
	setupToy(t)
	b := S.Bananas
	onTowerScore(12)
	if S.Bananas != b+7 || S.TowerBest != 12 || S.TowerCnt != 1 || mode != "laugh" {
		t.Fatalf("탑 점수 이상: 바나나 %d→%d 최고 %d", b, S.Bananas, S.TowerBest)
	}
	onTowerScore(40)
	if S.Bananas != b+7+15 || S.TowerBest != 40 {
		t.Fatalf("바나나 15개 상한: %d", S.Bananas-b)
	}
	onTowerScore(0)
	if S.TowerBest != 40 || mode != "st_cry" {
		t.Fatal("0층 처리 이상")
	}
}
