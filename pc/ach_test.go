package main

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupAch(t *testing.T) {
	statePathOverride = t.TempDir()
	t.Cleanup(func() { statePathOverride = "" })
	if seqs["idle"] == nil {
		loadSeq("idle")
		linkAnchors()
	}
	notices = nil
	ownerName = "지수"
}

func TestAchievementsUnlock(t *testing.T) {
	setupAch(t)
	// 기존 사용자: 바나나 받기 최고 61, 연속 출석 1
	S = petState{Bananas: 146, Full: 80, Mood: 70, Best: 61, Day: "2026-11-17", Streak: 1}
	initLevel()
	initAch()
	loadAccessories()
	if S.AttendTotal != 1 {
		t.Fatalf("누적 출석 시작값 %d", S.AttendTotal)
	}
	got := checkAchievements()
	has := func(id string) bool {
		for _, g := range got {
			if g == id {
				return true
			}
		}
		return false
	}
	if !has("catch30") || has("snack1") || has("attend30") {
		t.Fatalf("처음 달성 목록 이상: %v", got)
	}
	if again := checkAchievements(); len(again) != 0 {
		t.Fatalf("같은 업적을 두 번 줌: %v", again)
	}
	// 누적 출석 30일 → 무지개 핀 선물 + 바로 착용
	b := S.Bananas
	S.AttendTotal = 30
	got = checkAchievements()
	if !has("attend30") || !has("attend7") || !owns("rainbow") || S.Wear[slotHead] != "rainbow" || worn[slotHead] == nil {
		t.Fatalf("30일 선물 이상: %v owned=%v wear=%v", got, S.Owned, S.Wear)
	}
	if S.Bananas != b+30+10 {
		t.Fatalf("업적 바나나 이상: +%d", S.Bananas-b)
	}
	if len(notices) < 3 || !strings.Contains(notices[len(notices)-1], "무지개 핀을 받았어") && !strings.Contains(strings.Join(notices, "|"), "무지개 핀을 받았어") {
		t.Fatalf("축하 문구 이상: %q", notices)
	}
	// 선물 아이템은 상점에서 살 수 없음
	b = S.Bananas
	for i, it := range items {
		if it.id == "bananaband" {
			doItem(i)
		}
	}
	if owns("bananaband") || S.Bananas != b {
		t.Fatal("선물 아이템이 팔림")
	}
}

func TestAttendGiftLine(t *testing.T) {
	S = petState{AttendTotal: 12}
	if l := attendGiftLine(); !strings.Contains(l, "30일 선물(무지개 핀)까지 18일") {
		t.Error(l)
	}
	S.AttendTotal = 40
	if l := attendGiftLine(); !strings.Contains(l, "100일 선물(바나나 머리띠)까지 60일") {
		t.Error(l)
	}
}

func TestMatchScore(t *testing.T) {
	setupAch(t)
	S = petState{Bananas: 0, Full: 80, Mood: 50, Best: 61}
	onGameScore(18, 8)
	if S.BestMatch != 8 || S.MatchCnt != 1 || S.PlayCnt != 1 || S.Best != 61 || S.Bananas != 18 {
		t.Fatalf("짝 맞추기 기록 이상: %+v", S)
	}
	onGameScore(10, 16)
	if S.BestMatch != 8 || S.MatchCnt != 2 {
		t.Fatal("최소 횟수가 나빠짐")
	}
}

func TestBookPage(t *testing.T) {
	setupAch(t)
	ownerName = "<지수>"
	S = petState{Bananas: 50, Exp: 210, Since: "2026-11-10", AttendTotal: 3, Owned: []string{"crown"}, Wear: []string{"crown", "", ""},
		Ach: map[string]string{"catch30": "2026-11-17"}}
	loadAccessories()
	saveState()
	mux := http.NewServeMux()
	addBookRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/book")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(res.Body)
	page := buf.String()
	for _, want := range []string{"Lv.4 꼬마 우땅이", "&lt;지수&gt;", "업적 <small>1 / ", "바나나 받기 달인", "왕관", "쓰는 중", "누적 출석 30일 선물", "???"} {
		if !strings.Contains(page, want) {
			t.Errorf("수첩에 %q 없음", want)
		}
	}
	if strings.Contains(page, "<지수>") {
		t.Error("이름이 HTML 이스케이프 안 됨")
	}
	ids := []string{"none"}
	for _, it := range items {
		ids = append(ids, it.id)
	}
	for _, id := range ids {
		for _, q := range []string{"", "?solo=1"} {
			src := "/item/" + id + ".png" + q
			r, err := http.Get(srv.URL + src)
			if err != nil || r.StatusCode != 200 {
				t.Fatalf("%s 실패", src)
			}
			var raw bytes.Buffer
			raw.ReadFrom(r.Body)
			img, err := png.Decode(bytes.NewReader(raw.Bytes()))
			if err != nil || img.Bounds().Dx() != 200 {
				t.Fatalf("%s 그림 이상: %v", src, err)
			}
			page = strings.ReplaceAll(page, `src="`+src+`"`, `src="data:image/png;base64,`+base64.StdEncoding.EncodeToString(raw.Bytes())+`"`)
		}
	}
	if os.Getenv("WOOT_PREVIEW") != "" { // 미리보기 파일: WOOT_PREVIEW=1 go test -run TestBookPage
		os.WriteFile("test_book.html", []byte(page), 0644)
	}
	if r, _ := http.Get(srv.URL + "/item/../state.png"); r.StatusCode != 404 {
		t.Error("없는 아이템이 404 아님")
	}
}
