package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

// 가짜 모니터 두 대: 왼쪽 주 모니터(1920x1080) + 오른쪽 보조(1280x1024, 바닥이 더 높음)
func twoMonitors() {
	monOverride = []monInfo{
		{`\\.\DISPLAY1`, RECT{0, 0, 1920, 1080}, RECT{0, 0, 1920, 1040}, true},
		{`\\.\DISPLAY2`, RECT{1920, -200, 3200, 824}, RECT{1920, -200, 3200, 824}, false},
	}
	scanMonitors()
	curMon = -1
	applyMonitor()
}

func stepFloor(dt float64) {
	followMonitor()
	if mode == "walk" {
		walkTick(dt)
	}
	if mode == "fall" {
		fallTick(dt)
	} else if g := groundY(); posY < g-1 {
		setMode("fall", time.Hour)
	} else {
		posY = g
	}
}

func TestMonitors(t *testing.T) {
	setupCare(t)
	twoMonitors()
	if curMon != 0 || workR != 1920 || ground != 1040-float64(canvasH)+2 {
		t.Fatalf("처음엔 주 모니터: %d %v %v", curMon, workR, ground)
	}
	// 오른쪽으로 걸어가면 보조 모니터로 넘어감 (바닥이 더 높으면 올라섬)
	posX, posY, dir = 1900-float64(canvasW), ground, 1
	setMode("walk", time.Hour)
	for i := 0; i < 300 && curMon == 0; i++ {
		stepFloor(0.1)
	}
	if curMon != 1 || workL != 1920 {
		t.Fatalf("보조 모니터로 못 넘어감: %d", curMon)
	}
	stepFloor(0.1)
	if posY != 824-float64(canvasH)+2 {
		t.Fatalf("보조 모니터 바닥에 안 섬: %v", posY)
	}
	// 다시 왼쪽으로: 주 모니터 바닥이 더 낮으니 폴짝 떨어짐
	dir = -1
	setMode("walk", time.Hour)
	for i := 0; i < 400 && curMon == 1; i++ {
		stepFloor(0.1)
	}
	if curMon != 0 || mode != "fall" {
		t.Fatalf("주 모니터로 돌아오며 떨어져야 함: %d %s", curMon, mode)
	}
	for i := 0; i < 200 && mode == "fall"; i++ {
		stepFloor(0.03)
	}
	if posY != ground {
		t.Fatalf("주 모니터 바닥에 착지 못 함 %v %v", posY, ground)
	}
	// 보조 모니터를 끄면 그 경계에서 벽에 막힘
	S.MonOff = []string{`\\.\DISPLAY2`}
	posX, dir = 1700, 1
	setMode("walk", time.Hour)
	for i := 0; i < 200; i++ {
		stepFloor(0.1)
	}
	if curMon != 0 || posX > workR-float64(canvasW)*0.85+0.01 {
		t.Fatalf("꺼 둔 모니터로 넘어감: %d %v", curMon, posX)
	}
	// 보조 모니터에 있을 때 끄면 주 모니터 위에서 떨어짐
	S.MonOff = nil
	moveToMonitor(1)
	setMode("idle", time.Hour)
	if curMon != 1 {
		t.Fatal("보조 모니터로 못 옮김")
	}
	S.MonOff = []string{`\\.\DISPLAY2`}
	followMonitor()
	if curMon != 0 || mode != "fall" || posX+float64(canvasW)/2 > 1920 {
		t.Fatalf("꺼진 모니터에서 안 돌아옴: %d %s %v", curMon, mode, posX)
	}
	// 마지막 하나는 못 끔 (전부 꺼져 있으면 주 모니터는 켠 걸로)
	if onMonitorCmd(idMonBase); len(S.MonOff) != 1 {
		t.Fatalf("마지막 모니터가 꺼짐: %v", S.MonOff)
	}
	S.MonOff = []string{`\\.\DISPLAY1`, `\\.\DISPLAY2`}
	if !monAllowed(0) || monAllowed(1) {
		t.Fatal("다 꺼지면 주 모니터만 켜진 걸로")
	}
	S.MonOff = nil
	onMonitorCmd(idMonBase + 1)
	if len(S.MonOff) != 1 || S.MonOff[0] != `\\.\DISPLAY2` {
		t.Fatalf("체크 끄기 안 됨 %v", S.MonOff)
	}
}

func TestDeskCatch(t *testing.T) {
	runtime.LockOSThread() // 창은 만든 스레드에서만 다룰 수 있음
	defer runtime.UnlockOSThread()
	setupCare(t)
	for _, n := range []string{"walk", "st_surprise", "st_cry"} {
		if seqs[n] == nil {
			loadSeq(n)
		}
	}
	workL, workR, workT = 0, 900, 0
	ground = 700 - float64(canvasH) + 2
	twoMonitorsOff()
	startDeskCatch()
	if !catching || mode != "catch" || fxBits == nil {
		t.Fatalf("바나나 받기 시작 안 됨 %v %s", catching, mode)
	}
	defer func() {
		if fxHwnd != 0 {
			pDestroyWindowT.Call(fxHwnd)
			fxHwnd = 0
		}
	}()
	nextChoice()
	if mode != "catch" {
		t.Fatal("놀이 중에 다른 동작으로 바뀜")
	}
	catchUntil = time.Now().Add(catchLen) // 준비 시간 건너뛰기
	head := func(kind string) {
		x := posX + float64(canvasW)/2 - float64(fxX)
		y := posY + float64(canvasH)*0.4 - float64(fxY)
		catchItems = append(catchItems, catchItem{particle{sp: fxSprites[kind], x: x, y: y, size: 30, life: 9}, kind})
	}
	head("banana")
	head("goldbanana")
	catchTick(time.Now(), 0.001)
	if catchScore != 6 {
		t.Fatalf("받은 수가 이상함: %d", catchScore)
	}
	head("rock")
	catchTick(time.Now(), 0.001)
	if catchScore != 6 || !time.Now().Before(catchDizzy) {
		t.Fatal("돌멩이 맞으면 어질어질해야 함")
	}
	if s, _ := catchFrameSeq(); s != "st_surprise" {
		t.Fatal("어질어질 동작이 아님")
	}
	catchDizzy = time.Time{}
	// 바나나가 계속 떨어짐
	for i := 0; i < 60; i++ {
		catchTick(time.Now(), 0.05)
	}
	if len(catchItems) == 0 {
		t.Fatal("바나나가 안 떨어짐")
	}
	b := S.Bananas
	catchUntil = time.Now().Add(-time.Millisecond)
	catchTick(time.Now(), 0.03)
	if catching || fxBits != nil || S.Bananas < b+6 || S.DeskBest < 6 || S.PlayCnt != 1 {
		t.Fatalf("끝나고 보상·정리 안 됨: %v %v %d→%d best %d", catching, fxBits != nil, b, S.Bananas, S.DeskBest)
	}
}

// 모니터 목록이 비어 있으면 setupMove 의 작업 영역 그대로 쓰도록 (가짜 한 대)
func twoMonitorsOff() {
	r := RECT{int32(workL), int32(workT), int32(workR), int32(ground + float64(canvasH) - 2)}
	monOverride = []monInfo{{"one", r, r, true}}
	scanMonitors()
	curMon = 0
}

func TestHide(t *testing.T) {
	runtime.LockOSThread() // 창은 만든 스레드에서만 다룰 수 있음
	defer runtime.UnlockOSThread()
	setupCare(t)
	twoMonitorsOff()
	t.Cleanup(func() { hiding, hideStage = false, 0 }) // 실패해도 '숨는 중'이 다음 테스트로 안 남게
	startHide()
	if !hiding || !busy() {
		t.Fatal("숨바꼭질 시작 안 됨")
	}
	hideTick(time.Now().Add(3 * time.Second))
	hideTick(time.Now().Add(5 * time.Second))
	if hideStage != 2 || mode != "hide" {
		t.Fatalf("안 숨음: %d %s", hideStage, mode)
	}
	cx := posX + float64(canvasW)/2
	cy := posY + float64(canvasH)
	if cx > workL && cx < workR && cy <= ground+float64(canvasH) {
		t.Fatalf("다 보이는 곳에 숨음: %v %v", posX, posY)
	}
	// 숨은 쪽 바깥은 안 그림
	buf := make([]byte, canvasW*canvasH*4)
	bits = unsafe.Pointer(&buf[0])
	defer func() { bits = nil }()
	render()
	for py := 0; py < canvasH; py++ {
		for px := 0; px < canvasW; px++ {
			sx, sy := int32(posX)+int32(px), int32(posY)+int32(py)
			if (sx < hideClip.Left || sx >= hideClip.Right || sy >= hideClip.Bottom) && buf[(py*canvasW+px)*4+3] != 0 {
				t.Fatalf("숨은 곳 바깥이 보임 (%d,%d)", px, py)
			}
		}
	}
	if n := countOpaque(buf); n < canvasW*canvasH/40 { // 바닥 아래 빼꼼은 머리 윗부분만 (112px 창에서 약 470)
		t.Fatalf("빼꼼 내민 부분이 너무 작음: %d", n)
	}
	b := S.Bananas
	endHide(true)
	if hiding || S.Bananas != b+8 || S.HideCnt != 1 || greetAfterFall == "" || mode != "fall" {
		t.Fatalf("찾았을 때 보상 이상: %d→%d %d %q %s", b, S.Bananas, S.HideCnt, greetAfterFall, mode)
	}
	for i := 0; i < 300 && mode == "fall"; i++ {
		fallTick(0.03)
	}
	if greetAfterFall != "" || mode != "laugh" {
		t.Fatalf("착지하고 웃지 않음 %s", mode)
	}
	// 시간이 다 되면 스스로 나옴 (보상 없음)
	startHide()
	hideTick(time.Now().Add(3 * time.Second))
	hideTick(time.Now().Add(5 * time.Second))
	hideFound = time.Now().Add(-hideLimit - time.Second)
	b = S.Bananas
	hideTick(time.Now())
	if hiding || S.Bananas != b || S.HideCnt != 1 {
		t.Fatal("시간 초과 처리 이상")
	}
}

func TestGacha(t *testing.T) {
	setupCare(t)
	loadAccessories()
	S.Bananas = 5
	startGacha()
	if S.GachaCnt != 0 || !gachaAt.IsZero() {
		t.Fatal("바나나가 모자라도 뽑힘")
	}
	S.Bananas = 3000
	got := map[string]bool{}
	for i := 0; i < 200; i++ {
		before := S.Bananas
		startGacha()
		if S.Bananas != before-gachaCost {
			t.Fatal("뽑기 값이 안 빠짐")
		}
		gachaTick(time.Now().Add(2 * time.Second))
		if !gachaAt.IsZero() {
			t.Fatal("결과가 안 나옴")
		}
		for _, id := range gachaPool {
			if owns(id) {
				got[id] = true
			}
		}
	}
	if len(got) != len(gachaPool) || S.GachaCnt != 200 {
		t.Fatalf("200번 뽑아도 아이템이 다 안 나옴: %v", got)
	}
	for _, id := range gachaPool {
		if accSprites[id] == nil {
			t.Fatalf("뽑기 아이템 그림 없음: %s", id)
		}
	}
}

func TestFriend(t *testing.T) {
	runtime.LockOSThread() // 창은 만든 스레드에서만 다룰 수 있음
	defer runtime.UnlockOSThread()
	setupCare(t)
	for _, n := range []string{"walk", "eat", "st_surprise"} {
		if seqs[n] == nil {
			loadSeq(n)
		}
	}
	twoMonitorsOff()
	if startFriend(true) || friendOn() {
		t.Fatal("레벨이 낮은데 친구가 옴")
	}
	S.Exp = 5000
	posX, posY = 300, ground
	setMode("idle", time.Hour)
	if !startFriend(true) || !friendOn() || fr.x < workR-1 {
		t.Fatalf("친구가 안 옴 (먼 쪽 끝에서 들어와야 함) %+v", fr)
	}
	defer func() {
		if frHwnd != 0 {
			pDestroyWindowT.Call(frHwnd)
			frHwnd = 0
		}
	}()
	// 땅콩이 그림 (꽃 핀 · 밝은 털) 한 장
	renderFriend()
	shot := image.NewNRGBA(image.Rect(0, 0, frW, frH))
	fb := unsafe.Slice((*byte)(frBits), frW*frH*4)
	for i := 0; i < frW*frH; i++ {
		a := float32(fb[i*4+3]) / 255
		px := func(v byte) uint8 { return uint8(float32(v) + 235*(1-a) + 0.5) }
		shot.SetNRGBA(i%frW, i/frW, color.NRGBA{px(fb[i*4+2]), px(fb[i*4+1]), px(fb[i*4]), 255})
	}
	var pb bytes.Buffer
	png.Encode(&pb, shot)
	os.WriteFile("test_friend.png", pb.Bytes(), 0o644)
	if countOpaque(fb) < 2000 {
		t.Fatal("땅콩이가 안 그려짐")
	}
	now := time.Now()
	stages := map[string]bool{}
	for i := 0; i < 20000 && friendOn(); i++ {
		now = now.Add(50 * time.Millisecond)
		stages[fr.stage] = true
		if fr.stage == "play" {
			stages["act:"+fr.act] = true
		}
		friendTick(now, 0.05)
		if mode == "walk" {
			walkTick(0.05)
		}
	}
	if friendOn() {
		t.Fatalf("친구가 안 돌아감 %+v", fr)
	}
	for _, s := range []string{"enter", "greet", "play", "leave"} {
		if !stages[s] {
			t.Fatalf("%s 단계를 안 거침 %v", s, stages)
		}
	}
	if S.FriendCnt != 1 || dayLog().Friend != 1 || S.FriendDay != today() {
		t.Fatalf("방문 기록 안 됨 %d", S.FriendCnt)
	}
	nextChoice()
	if mode == "idle" && modeUntil.After(time.Now().Add(time.Minute)) {
		t.Fatal("친구가 간 뒤에도 혼자 놀지 않음")
	}
}

func TestHealthBatteryMemo(t *testing.T) {
	setupCare(t)
	idleOverride = 0
	defer func() { idleOverride, batOverride = -1, nil }()
	now := time.Now()
	activeSince, lastStretch, lastWater, lastHealthChk = now.Add(-61*time.Minute), now.Add(-61*time.Minute), now, time.Time{}
	setMode("idle", time.Hour)
	healthTick(now)
	if S.StretchCnt != 1 || mode != "wave" {
		t.Fatalf("스트레칭 알림 안 옴 %d %s", S.StretchCnt, mode)
	}
	lastWater, lastHealthChk = now.Add(-91*time.Minute), time.Time{}
	healthTick(now)
	if S.WaterCnt != 1 {
		t.Fatal("물 알림 안 옴")
	}
	idleOverride, lastHealthChk = 400, time.Time{}
	healthTick(now)
	if now.Sub(activeSince) > time.Second {
		t.Fatal("자리를 비우면 다시 세야 함")
	}
	// 배터리
	setMode("idle", time.Hour)
	bat := func(ac, pct byte) {
		batOverride = &powerStatus{AC: ac, Flag: 1, Percent: pct}
		lastBatChk = time.Time{}
		batteryTick(time.Now())
	}
	bat(0, 50)
	if batPct != 50 || batteryMenuLine() != "배터리 50%" {
		t.Fatalf("배터리 못 읽음 %d", batPct)
	}
	bat(0, 18)
	if mode != "st_lounge" || !batteryLow() {
		t.Fatal("배터리 20% 반응 없음")
	}
	m := speedMul()
	bat(0, 9)
	if mode != "st_cry" {
		t.Fatal("배터리 10% 반응 없음")
	}
	bat(1, 9)
	if mode != "laugh" || batteryLow() || speedMul() <= m {
		t.Fatal("충전기 꽂으면 기운 차려야 함")
	}
	batOverride = &powerStatus{Flag: 128}
	lastBatChk = time.Time{}
	batteryTick(time.Now())
	if batPct != -1 || batteryMenuLine() != "" {
		t.Fatal("데스크톱이면 배터리 없음")
	}
	// 메모
	addMemo("  택배 받기  ", -time.Minute)
	addMemo("회의 준비", time.Hour)
	if len(S.Memos) != 2 || S.Memos[0].Text != "택배 받기" || S.MemoCnt != 2 {
		t.Fatalf("메모 저장 이상 %+v", S.Memos)
	}
	memoShowAt, lastMemoChk = time.Time{}, time.Time{}
	memoTick(time.Now())
	if len(S.Memos) != 1 || S.Memos[0].Text != "회의 준비" {
		t.Fatalf("때가 된 메모가 안 나옴 %+v", S.Memos)
	}
	if l := memoLabel(S.Memos[0]); l == "" {
		t.Fatal("메모 이름 없음")
	}
	onHelperCmd(idMemoBase)
	if len(S.Memos) != 0 {
		t.Fatal("메뉴에서 메모 지우기 안 됨")
	}
	// 효과음 데이터
	loadSounds()
	for k, b := range sounds {
		if len(b) < 200 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
			t.Fatalf("효과음 %s 이상", k)
		}
	}
}
