package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
	"unsafe"
)

func setupCare(t *testing.T) {
	setupMove(t)
	for _, n := range []string{"idle", "sleep", "sit", "shy", "laugh", "wave", "st_me"} {
		if seqs[n] == nil {
			loadSeq(n)
		}
	}
	linkAnchors()
	if len(fxSprites) == 0 {
		loadFX()
	}
	loadCare()
	ownerName = "지수"
	bathing, tucked, petFx = false, false, nil
	t.Cleanup(func() { bathing, tucked = false, false })
}

// 씻기기: 문지를 때마다 거품 · 꼬질꼬질이 줄고, 다 씻으면 반짝 + 기록
func TestBath(t *testing.T) {
	setupCare(t)
	S.Dirt = 80
	startBath()
	if !bathing || mode != "bath" {
		t.Fatalf("목욕 시작 안 됨: %v %s", bathing, mode)
	}
	nextChoice() // 목욕 중엔 다른 동작으로 안 넘어감
	if mode != "bath" {
		t.Fatalf("목욕 중 동작이 바뀜: %s", mode)
	}
	for i := 0; i < 10; i++ {
		rub(50, 60)
	}
	if S.Dirt > 46 || len(petFx) < 10 {
		t.Fatalf("문지르기: dirt=%.1f 거품=%d", S.Dirt, len(petFx))
	}
	for i := 0; i < 20 && bathing; i++ {
		rub(50, 60)
		bathTick(time.Now())
	}
	if bathing || S.Dirt != 0 || S.BathCnt != 1 || dayLog().Bath != 1 || mode != "laugh" {
		t.Fatalf("목욕 끝: bathing=%v dirt=%.1f cnt=%d mode=%s", bathing, S.Dirt, S.BathCnt, mode)
	}
	if S.Exp != 6 {
		t.Fatalf("목욕 경험치 %d", S.Exp)
	}
	// 깨끗하면 안 씻음
	startBath()
	if bathing {
		t.Fatal("깨끗한데 목욕 시작됨")
	}
	// 중간에 끌어 버리면 그만
	S.Dirt = 60
	startBath()
	startDrag()
	bathTick(time.Now())
	if bathing {
		t.Fatal("끌었는데 목욕이 안 끝남")
	}
}

// 이불 덮어 재우기: 자는 동안 졸림이 풀리고, 일어나면 기록 · 기분
func TestTuck(t *testing.T) {
	setupCare(t)
	S.Tired = 90
	tuckIn()
	if !tucked || mode != "sleep" {
		t.Fatalf("재우기 안 됨: %v %s", tucked, mode)
	}
	pokeTucked() // 막 잠들었을 땐 5분만 더
	if !tucked || mode != "sleep" {
		t.Fatal("바로 깨 버림")
	}
	for i := 0; i < 30; i++ {
		tuckTick(1)
	}
	if S.Tired > 16 {
		t.Fatalf("졸림이 안 풀림: %.1f", S.Tired)
	}
	nextChoice()
	if tucked || S.Tired != 0 || S.TuckCnt != 1 || dayLog().Tuck != 1 || mode != "wave" {
		t.Fatalf("일어나기: tucked=%v tired=%.1f cnt=%d mode=%s", tucked, S.Tired, S.TuckCnt, mode)
	}
	// 낮에 하나도 안 졸리면 안 잠 (밤엔 잘 수 있음)
	S.Tired = 0
	tuckIn()
	if h := time.Now().Hour(); h >= 7 && h < 21 && tucked {
		t.Fatal("안 졸린데 잠")
	}
}

// 꼬질꼬질 · 졸림이 쌓이고, 말·속도에 반영
func TestCareStats(t *testing.T) {
	setupCare(t)
	mode = "walk"
	for i := 0; i < 300; i++ {
		careMinute()
	}
	if S.Dirt < 50 || S.Tired < 60 {
		t.Fatalf("5시간 뒤: dirt=%.1f tired=%.1f", S.Dirt, S.Tired)
	}
	S.Tired = 80
	if !sleepy() || speedMul() >= 1 {
		t.Fatal("졸린데 그대로 걸음")
	}
	careAway(8) // 밤새 꺼져 있었음
	if S.Tired != 0 {
		t.Fatalf("자고 왔는데 졸림 %.1f", S.Tired)
	}
	S.Dirt = 90
	found := false
	for i := 0; i < 30; i++ {
		if strings.Contains(careLine(), "씻") || strings.Contains(careLine(), "냄새") || strings.Contains(careLine(), "목욕") {
			found = true
		}
	}
	if !found {
		t.Fatal("꼬질꼬질한데 씻겨 달라는 말이 없음")
	}
}

// 날짜 이벤트 · 계절
func TestEventsAndSeasons(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.ParseInLocation("2006-01-02", s, time.Local); return v }
	for date, want := range map[string]string{
		"2026-12-25": "xmas", "2026-12-24": "xmaseve", "2027-01-01": "newyear", "2027-02-06": "seollal",
		"2027-09-15": "chuseok", "2026-10-31": "halloween", "2026-11-11": "pepero", "2027-05-05": "children",
	} {
		if e, ok := eventOn(d(date)); !ok || e.id != want {
			t.Errorf("%s: %v %v (기대 %s)", date, e.id, ok, want)
		}
	}
	if _, ok := eventOn(d("2026-11-23")); ok {
		t.Error("평범한 날에 이벤트")
	}
	if e, n, ok := nextEvent(d("2026-11-23")); !ok || e.id != "xmaseve" || n != 31 {
		t.Errorf("다음 이벤트 %v %d", e.id, n)
	}
	for date, want := range map[string]string{"2026-01-10": "snow", "2026-04-05": "petal", "2026-07-20": "bubble", "2026-10-10": "leaf", "2026-12-22": "xmas"} {
		if _, fx := seasonOf(d(date)); fx != want {
			t.Errorf("%s 계절 쇼 %s (기대 %s)", date, fx, want)
		}
	}
	for _, e := range events {
		if _, ok := weathers[e.ev.fx]; !ok {
			t.Errorf("%s: 없는 날씨 %s", e.ev.id, e.ev.fx)
		}
		if e.ev.gift != "" {
			if _, ok := findItem(e.ev.gift); !ok || giftFrom[e.ev.gift] == "" {
				t.Errorf("%s: 선물 %s 이 아이템 목록에 없음", e.ev.id, e.ev.gift)
			}
		}
	}
}

// 이벤트 인사: 바나나 · 선물 아이템 · 하루 한 번
func TestCelebrateEvent(t *testing.T) {
	setupCare(t)
	loadAccessories()
	S.NoWeather = true
	e, _ := eventOn(time.Date(2026, 12, 25, 10, 0, 0, 0, time.Local))
	b := S.Bananas
	celebrateEvent(e)
	if S.Bananas != b+15 || !owns("rudolph") || !wearing("rudolph") || S.EventDay != today() || S.EventCnt != 1 {
		t.Fatalf("크리스마스: 바나나 %d→%d, 루돌프 %v", b, S.Bananas, owns("rudolph"))
	}
	if dayLog().Event != "크리스마스" {
		t.Fatalf("일기에 이벤트 없음: %q", dayLog().Event)
	}
}

// 그림일기: 하루 기록 → 문장, 60일까지만
func TestDiary(t *testing.T) {
	setupCare(t)
	r := dayRec{Date: "2026-11-23", Min: 192, Pet: 12, Snack: 1, Play: 2, Bath: 1, Tuck: 1, Focus: 25, Throw: 3, Level: 7, Weather: "leaf"}
	txt := strings.Join(diaryText(r, "지수"), " ")
	for _, w := range []string{"낙엽", "지수랑 3시간 12분", "12번", "게임을 2판", "목욕", "이불", "지수가 25분", "하늘을 3번", "Lv.7"} {
		if !strings.Contains(txt, w) {
			t.Errorf("일기에 %q 없음: %s", w, txt)
		}
	}
	if strings.Join(diaryText(r, "지수"), " ") != txt {
		t.Error("같은 날 일기가 열 때마다 바뀜")
	}
	if diaryDate("2026-11-23") != "11월 23일 (월)" {
		t.Errorf("날짜 %s", diaryDate("2026-11-23"))
	}
	empty := strings.Join(diaryText(dayRec{Date: "2026-11-22"}, "민준"), " ")
	if !strings.Contains(empty, "민준을 잠깐밖에") {
		t.Errorf("빈 날: %s", empty)
	}
	S.Diary = nil
	for i := 0; i < 70; i++ {
		S.Diary = append(S.Diary, dayRec{Date: time.Now().AddDate(0, 0, i-80).Format("2006-01-02"), Min: 10})
	}
	dayLog().Pet++
	if len(S.Diary) != diaryKeep || S.Diary[len(S.Diary)-1].Date != today() {
		t.Fatalf("일기 보관 %d일", len(S.Diary))
	}
	h := diaryHTML(S.Diary, "지수", func(s string) string { return s })
	if strings.Count(h, `class="dy"`) != 30 || !strings.Contains(h, `id="diary"`) {
		t.Fatalf("수첩 일기 칸 %d개", strings.Count(h, `class="dy"`))
	}
}

// 실제 그리기: 꼬질꼬질 얼룩 · 이불 · 거품 · 화면 날씨 조각 (test_care.png 로 저장해서 눈으로 확인)
func TestCareRender(t *testing.T) {
	setupCare(t)
	loadAccessories()
	canvasW, canvasH = 225, 225
	scale = 1.5
	defer func() { scale = 0.75 }()
	buf := make([]byte, canvasW*canvasH*4)
	bits = unsafe.Pointer(&buf[0])
	defer func() { bits = nil }()
	out := image.NewNRGBA(image.Rect(0, 0, canvasW*3+600, 400))
	for i := range out.Pix {
		out.Pix[i] = 235
	}
	put := func(ox int, b []byte, w, h int) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				o := (y*w + x) * 4
				a := float32(b[o+3]) / 255
				px := func(v byte) uint8 { return uint8(float32(v) + 235*(1-a) + 0.5) }
				out.SetNRGBA(ox+x, y, color.NRGBA{px(b[o+2]), px(b[o+1]), px(b[o]), 255})
			}
		}
	}
	S.Dirt = 80
	mode, modeStart = "idle", time.Now()
	render()
	put(0, buf, canvasW, canvasH)
	S.Dirt, tucked = 0, true
	mode = "sleep"
	render()
	put(canvasW, buf, canvasW, canvasH)
	tucked = false
	mode = "bath"
	for i := 0; i < 12; i++ {
		addBubbleAt(rf(40, 180), rf(60, 200))
	}
	burst("sparkle", 6)
	petFxTick(0.3)
	render()
	put(canvasW*2, buf, canvasW, canvasH)
	// 화면 날씨 조각 (창 없이 버퍼만)
	fxW, fxH = 600, 400
	fb := make([]byte, fxW*fxH*4)
	screenFx = nil
	for _, k := range []string{"snow", "petal", "leaf", "bubble", "xmas", "confetti"} {
		for i := 0; i < 6; i++ {
			spawnWeather(weathers[k])
			p := &screenFx[len(screenFx)-1]
			p.y, p.age = rf(20, 380), 1
		}
	}
	for i := range screenFx {
		p := &screenFx[i]
		blit(fb, fxW, fxH, p.sp, p.x, p.y, p.size, p.rot, p.alpha())
	}
	if n := countOpaque(fb); n < 3000 {
		t.Fatalf("날씨 조각이 거의 안 그려짐: %d", n)
	}
	put(canvasW*3, fb, fxW, fxH)
	screenFx = nil
	if f, err := os.Create("test_care.png"); err == nil {
		png.Encode(f, out)
		f.Close()
	}
}

func countOpaque(b []byte) int {
	n := 0
	for i := 3; i < len(b); i += 4 {
		if b[i] > 128 {
			n++
		}
	}
	return n
}

// 효과 창: 실제 투명 창을 열어 쇼를 잠깐 돌리고 닫음
func TestShowWindow(t *testing.T) {
	setupCare(t)
	workL, workR, workT = 0, 800, 0
	ground = 600 - float64(canvasH) + 2
	if !startShow("snow") {
		t.Fatal("효과 창을 못 염")
	}
	now := time.Now()
	for i := 0; i < 20; i++ {
		screenFxTick(now.Add(time.Duration(i)*33*time.Millisecond), 0.033)
	}
	if len(screenFx) == 0 || !showing() {
		t.Fatalf("눈이 안 내림 %d", len(screenFx))
	}
	showUntil = time.Now().Add(-time.Second)
	for i := 0; i < 600 && showing(); i++ {
		screenFxTick(time.Now(), 0.1)
	}
	if showing() || fxBits != nil {
		t.Fatal("쇼가 끝났는데 창이 안 닫힘")
	}
	if fxHwnd != 0 {
		pDestroyWindowT.Call(fxHwnd)
		fxHwnd = 0
	}
}
