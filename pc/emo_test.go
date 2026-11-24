package main

import (
	"testing"
	"time"
)

// 새 이모티콘 동작: 그림 8장 + 머리 위치, 메뉴로 보여 주고 끝나면 서 있기
func TestEmoticons(t *testing.T) {
	setupMove(t)
	for _, e := range emos {
		loadSeq(e.seq)
		if len(seqs[e.seq]) == 0 {
			t.Fatalf("%s 그림이 없음", e.seq)
		}
		if isNewEmo(e.seq) && len(headAnchors[e.seq]) != len(seqs[e.seq]) {
			t.Fatalf("%s 머리 위치 %d개 / 그림 %d장", e.seq, len(headAnchors[e.seq]), len(seqs[e.seq]))
		}
	}
	posX, posY = 800, ground
	for i, e := range emos {
		setMode("idle", time.Second)
		if !onEmoCmd(uintptr(idEmo+i)) || mode != e.seq {
			t.Fatalf("메뉴로 %s 를 못 보여 줌 (mode=%s)", e.name, mode)
		}
		if sp, _, _ := currentFrame(); sp == nil {
			t.Fatalf("%s 그림이 안 나옴", e.seq)
		}
	}
	setMode("dance", time.Second)
	nextChoice()
	if mode != "idle" {
		t.Fatalf("춤 다음에 %s", mode)
	}
	// 마구 누르면 흥!
	clickTimes = nil
	for i := 0; i < 5; i++ {
		setMode("idle", time.Second)
		doClick()
	}
	if mode != "angry" {
		t.Fatalf("연타했는데 %s", mode)
	}
	if emoForLine("지수야 화이팅!") != "cheer" || emoForLine("사랑해 ♥") != "heart" || emoForLine("뭐 해?") != "" {
		t.Fatal("대사 → 이모티콘 이상")
	}
	// 바쁠 땐 안 함
	hiding = true
	setMode("idle", time.Second)
	if playEmo("clap", false) {
		t.Fatal("숨바꼭질 중에 박수")
	}
	hiding = false
}
