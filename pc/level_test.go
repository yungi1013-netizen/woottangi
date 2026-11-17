package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLevelCurve(t *testing.T) {
	cases := []struct{ exp, lv, cur int }{
		{0, 1, 0}, {39, 1, 39}, {40, 2, 0}, {99, 2, 59}, {100, 3, 0}, {180, 4, 0},
	}
	for _, c := range cases {
		lv, cur, _ := levelOf(c.exp)
		if lv != c.lv || cur != c.cur {
			t.Errorf("exp %d → Lv.%d (%d), 기대 Lv.%d (%d)", c.exp, lv, cur, c.lv, c.cur)
		}
	}
	if lv, _, need := levelOf(1 << 30); lv != maxLevel || need != 0 {
		t.Errorf("최고 레벨 처리 이상: %d %d", lv, need)
	}
	// 레벨 10/20/30 까지 누적 경험치 (하루 약 80~100 기준 2주/2달/4달)
	sum := 0
	for l := 1; l < 30; l++ {
		sum += expNeed(l)
		switch l + 1 {
		case 10, 20, 30:
			t.Logf("Lv.%d 까지 누적 경험치 %d", l+1, sum)
		}
	}
}

func TestTitlesAndParticles(t *testing.T) {
	if titleOf(1) != "아기 우땅이" || titleOf(4) != "꼬마 우땅이" || titleOf(12) != "베프 우땅이" || titleOf(50) != "영원한 가족 우땅이" {
		t.Error("칭호 이상")
	}
	ownerName = "지수"
	if nm("이랑", "랑") != "지수랑" || nm("은", "는") != "지수는" {
		t.Error("받침 없는 이름 조사 이상:", nm("이랑", "랑"))
	}
	ownerName = "민준"
	if nm("이랑", "랑") != "민준이랑" || nm("이", "가") != "민준이" {
		t.Error("받침 있는 이름 조사 이상")
	}
	ownerName = ""
	if nm("은", "는") != "주인님은" {
		t.Error("이름 없을 때 이상")
	}
}

func TestLevelMigrationAndLevelUp(t *testing.T) {
	statePathOverride = t.TempDir()
	defer func() { statePathOverride = "" }()
	// 8일 전에 이름을 정한 기존 사용자
	np := filepath.Join(statePathOverride, "name.txt")
	os.WriteFile(np, []byte("지수"), 0644)
	old := time.Now().AddDate(0, 0, -7)
	os.Chtimes(np, old, old)
	S = petState{Bananas: 100, Full: 70, Mood: 50}
	initLevel()
	if daysTogether() != 8 || S.Exp != 210 {
		t.Fatalf("함께한 날/시작 경험치 이상: %d일, exp %d", daysTogether(), S.Exp)
	}
	if lv, _, _ := levelOf(S.Exp); S.LvShown != lv || lv != 4 {
		t.Fatalf("옮겨 온 레벨은 축하 없이 바로 표시: LvShown %d, lv %d", S.LvShown, lv)
	}
	initLevel() // 두 번 불러도 그대로
	if S.Exp != 210 {
		t.Fatal("initLevel 이 두 번 적용됨")
	}
	// 경험치 70 더 → Lv.4 (30/100) → Lv.5
	addExp(70)
	if pendingLv != 5 {
		t.Fatalf("레벨 업 대기 이상: %d", pendingLv)
	}
	bananas := S.Bananas
	celebrate(pendingLv)
	if S.LvShown != 5 || pendingLv != 0 || S.Bananas != bananas+10 {
		t.Fatalf("축하 처리 이상: LvShown %d, 바나나 +%d", S.LvShown, S.Bananas-bananas)
	}
	// 한 번에 두 레벨 → 보상 합산 (Lv.6 12 + Lv.7 14)
	addExp(expNeed(5) - 0 + expNeed(6))
	bananas = S.Bananas
	celebrate(pendingLv)
	if S.LvShown != 7 || S.Bananas != bananas+26 {
		t.Fatalf("여러 레벨 한꺼번에: LvShown %d, 바나나 +%d", S.LvShown, S.Bananas-bananas)
	}
}

func TestNewUserLevel(t *testing.T) {
	statePathOverride = t.TempDir()
	defer func() { statePathOverride = "" }()
	S = petState{Bananas: 10, Full: 70, Mood: 70}
	initLevel()
	if S.Since != today() || S.Exp != 0 || S.LvShown != 1 || daysTogether() != 1 {
		t.Fatalf("새 사용자 시작값 이상: %+v", S)
	}
	g, isNew := checkAttendance()
	if !isNew || g != 5 || S.Exp != expAttend {
		t.Fatalf("출석 경험치 이상: exp %d", S.Exp)
	}
}
