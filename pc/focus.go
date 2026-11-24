package main

// 집중 타이머 (뽀모도로): 25분/50분 동안 우땅이가 옆에 조용히 앉아 기다리다가, 끝나면 축하하고 바나나를 줌

import (
	"fmt"
	"math"
	"time"
)

var (
	focusUntil time.Time
	focusLen   int // 분
)

func focusing() bool { return !focusUntil.IsZero() }

func focusLeft() int {
	if !focusing() {
		return 0
	}
	return max(1, int(math.Ceil(time.Until(focusUntil).Minutes())))
}

// 집중 보상: 5분마다 바나나 1개 · 경험치 2
func focusReward(minutes int) (bananas, exp int) { return minutes / 5, minutes / 5 * 2 }

func startFocus(minutes int) {
	focusLen = minutes
	focusUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
	stay = false
	setMode("sit", rd(20, 40))
	b, _ := focusReward(minutes)
	say(fmt.Sprintf("좋아! %d분 집중 시작!\n옆에서 조용히 기다릴게.\n끝나면 바나나 %d개 줄게~", minutes, b), 4500*time.Millisecond)
	nextTalk = focusUntil.Add(time.Minute)
}

func stopFocus() {
	focusUntil = time.Time{}
	setMode("idle", rd(1.5, 2.5))
	say("집중 그만! 조금 쉬었다 해도 괜찮아~", 3*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

func focusTick(now time.Time) {
	if !focusing() || now.Before(focusUntil) {
		return
	}
	focusUntil = time.Time{}
	b, e := focusReward(focusLen)
	S.Bananas += b
	S.FocusCnt++
	S.FocusMin += focusLen
	dayLog().Focus += focusLen
	addExp(e)                      // 저장 포함
	setMode("clap", 3*time.Second) // 수고했어 짝짝

	say(fmt.Sprintf("%d분 집중 끝! 수고했어 %s!\n바나나 %d개 선물이야.\n5분만 쉬었다 하자~", focusLen, voc(), b), 7*time.Second)
	toast("집중 끝!", fmt.Sprintf("%d분 집중 완료 · 바나나 %d개 · 잠깐 쉬어 가요", focusLen, b))
	nextTalk = time.Now().Add(rd(60, 120))
}

const (
	idFocus25   = 1091
	idFocus50   = 1092
	idFocusStop = 1093
)

func appendFocusMenu(m uintptr) {
	fm, _, _ := pCreatePopupMenu.Call()
	if focusing() {
		appendMenu(fm, MF_GRAYED, 0, fmt.Sprintf("%d분 집중 중 · %d분 남음", focusLen, focusLeft()))
		appendMenu(fm, MF_STRING, idFocusStop, "그만하기")
	} else {
		appendMenu(fm, MF_STRING, idFocus25, "25분 집중 (바나나 5개)")
		appendMenu(fm, MF_STRING, idFocus50, "50분 집중 (바나나 10개)")
	}
	label := "집중 타이머"
	if focusing() {
		label = fmt.Sprintf("집중 타이머 (%d분 남음)", focusLeft())
	}
	appendMenu(m, MF_POPUP, fm, label)
}
