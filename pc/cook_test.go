package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

// 바나나 요리: 바나나 3개, 요리 도감, 요리사 모자 업적
func TestCook(t *testing.T) {
	setupMove(t)
	S.Ach = map[string]string{}
	S.Cooked = nil
	S.Full, S.Bananas, S.CookCnt = 40, 2, 0
	doCook()
	if S.Bananas != 2 || S.CookCnt != 0 {
		t.Fatal("바나나가 모자란데 요리함")
	}
	// 확률: 앞쪽 요리는 잘 나오고 마지막은 드묾
	if pickRecipe(0) != 0 || pickRecipe(0.9999) != len(recipes)-1 {
		t.Fatal("요리 고르기 범위")
	}
	seen := map[int]bool{}
	for i := 0; i < 1000; i++ {
		seen[pickRecipe(float64(i)/1000)] = true
	}
	if len(seen) != len(recipes) {
		t.Fatalf("나올 수 있는 요리 %d/%d", len(seen), len(recipes))
	}
	S.Bananas = 100
	for i := 0; i < 10; i++ {
		S.Full = 40
		setMode("idle", time.Second)
		doCook()
	}
	if S.CookCnt != 10 || S.Bananas != 100-10*cookCost || cookedKinds(S.Cooked) < 1 {
		t.Fatalf("요리 10번: cnt=%d 바나나=%d 도감=%d", S.CookCnt, S.Bananas, cookedKinds(S.Cooked))
	}
	got := checkAchievements()
	if S.Ach["cook1"] == "" || S.Ach["cook10"] == "" || !owns("chefhat") {
		t.Fatalf("요리 업적/요리사 모자: %v owned=%v", got, S.Owned)
	}
	S.Full = 99
	doCook()
	if S.CookCnt != 10 {
		t.Fatal("배부른데 요리함")
	}
	if h := cookBookHTML(S); len(h) < 100 {
		t.Fatal("요리 도감 페이지")
	}
}

// 업적 보상 아이템: 예전에 깬 업적도 업데이트 뒤 아이템을 받음
func TestAchRewardBackfill(t *testing.T) {
	setupMove(t)
	S.Owned = nil
	S.Ach = map[string]string{"play30": "2026-01-01", "pet100": "2026-01-02"}
	initAch()
	if !owns("medal") || !owns("flowercrown") || owns("headphones") {
		t.Fatalf("보상 아이템 챙기기: %v", S.Owned)
	}
	for _, a := range achievements {
		if a.item != "" {
			if _, ok := findItem(a.item); !ok {
				t.Fatalf("%s 보상 %s 가 아이템 목록에 없음", a.id, a.item)
			}
			if _, err := os.Stat(filepath.Join("acc", a.item+".png")); err != nil {
				t.Fatalf("%s 그림 없음", a.item)
			}
		}
	}
}

// 음악 춤: 소리가 5초 넘게 이어지고, 한가할 때만
func TestMusicDance(t *testing.T) {
	setupMove(t)
	now := time.Now()
	musicNext = time.Time{}
	setMode("idle", time.Second)
	if musicWantsDance(3, now) {
		t.Fatal("짧은 소리에 춤춤")
	}
	if !musicWantsDance(5, now) {
		t.Fatal("음악에 안 춤")
	}
	n := S.DanceCnt
	musicDance()
	if mode != "dance" || S.DanceCnt != n+1 {
		t.Fatal("춤 시작")
	}
	setMode("idle", time.Second)
	if musicWantsDance(9, time.Now()) {
		t.Fatal("쉬는 시간인데 또 춤")
	}
	// 실제 스피커 미터 (소리 장치가 없는 PC 면 건너뜀)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pCoInitializeEx.Call(0, 0)
	if m := openMeter(); m == nil {
		t.Log("스피커 미터 없음")
	} else {
		var v float32
		if r := comCall(m, 3, uintptr(unsafe.Pointer(&v))); r != 0 || v < 0 || v > 1 {
			t.Fatalf("피크 읽기 r=%x v=%v", r, v)
		}
		comRelease(m)
	}
}
