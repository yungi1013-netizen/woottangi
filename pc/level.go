package main

// 레벨 · 친밀도: 함께 지낼수록 경험치가 쌓이고, 레벨이 오르면 칭호·바나나·새 대사가 열림
//
// 경험치 얻는 곳 (expFor 표)
//   출석 도장 10 · 간식 3/5/8 · 바나나 받기 게임 3~20 · 쓰다듬기 2 (하루 10번까지)
//   오늘의 운세 3 (하루 한 번) · 꾸미기 아이템 사기 10 · 켜 두기 15분마다 1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxLevel = 50

const (
	expAttend = 10
	expPet    = 2
	expLuck   = 3
	expBuy    = 10
	expIdle   = 1 // 켜 두기 15분마다
)

var expSnack = []int{3, 5, 8}

// 레벨 L → L+1 에 필요한 경험치
func expNeed(l int) int { return 40 + 20*(l-1) }

// 누적 경험치 → (레벨, 이번 레벨에서 쌓은 양, 이번 레벨에 필요한 양)
func levelOf(exp int) (lv, cur, need int) {
	lv = 1
	for lv < maxLevel && exp >= expNeed(lv) {
		exp -= expNeed(lv)
		lv++
	}
	if lv == maxLevel {
		return lv, 0, 0
	}
	return lv, exp, expNeed(lv)
}

type title struct {
	from int
	name string
}

var titles = []title{
	{1, "아기 우땅이"},
	{3, "꼬마 우땅이"},
	{5, "장난꾸러기 우땅이"},
	{8, "단짝 우땅이"},
	{12, "베프 우땅이"},
	{16, "바나나 박사 우땅이"},
	{20, "전설의 우땅이"},
	{30, "우주 최강 우땅이"},
	{50, "영원한 가족 우땅이"},
}

func titleOf(lv int) string {
	t := titles[0].name
	for _, x := range titles {
		if lv >= x.from {
			t = x.name
		}
	}
	return t
}

// 칭호가 새로 바뀌는 레벨이면 true
func isTitleLevel(lv int) bool {
	for _, x := range titles {
		if x.from == lv && lv > 1 {
			return true
		}
	}
	return false
}

// ── 함께한 날 ──

func daysTogether() int { return daysTogetherOf(S.Since) }

// 처음 켠 날 기록. 레벨 기능 전부터 키우던 경우 이름 저장 날짜를 첫날로 보고 그동안의 경험치를 줌
func initLevel() {
	if S.Since != "" {
		return
	}
	S.Since = today()
	if fi, err := os.Stat(filepath.Join(filepath.Dir(statePath()), "name.txt")); err == nil {
		if d := fi.ModTime().Format("2006-01-02"); d < S.Since {
			S.Since = d
		}
	}
	if days := daysTogether(); days > 1 {
		S.Exp = (days - 1) * 30
	}
	S.LvShown, _, _ = levelOf(S.Exp)
	saveState()
}

// ── 경험치 ──

var (
	pendingLv   int // 말풍선이 비면 축하할 레벨 (0 = 없음)
	lastIdleExp = time.Now()
)

func addExp(n int) {
	if n <= 0 {
		return
	}
	S.Exp += n
	if lv, _, _ := levelOf(S.Exp); lv > S.LvShown {
		pendingLv = lv
	}
	saveState()
}

// 매 프레임: 켜 둔 시간 경험치 + 레벨 업 축하
func levelTick() {
	if time.Since(lastIdleExp) >= 15*time.Minute {
		lastIdleExp = time.Now()
		addExp(expIdle)
	}
	achTick()
	if (pendingLv == 0 && len(notices) == 0) || bubText != "" || quitting || ownerName == "" {
		return
	}
	switch mode {
	case "drag", "fall", "sleep", "eat", "eat_milk", "eat_cake", "st_cry", "punish":
		return
	}
	if pendingLv > 0 {
		celebrate(pendingLv)
		return
	}
	showNotice()
}

func celebrate(lv int) {
	from := S.LvShown
	pendingLv = 0
	S.LvShown = lv
	gift := 0
	newTitle := false
	for l := from + 1; l <= lv; l++ {
		gift += 2 * l
		newTitle = newTitle || isTitleLevel(l)
	}
	S.Bananas += gift
	S.Mood = clamp(S.Mood + 10)
	saveState()
	setMode("laugh", rd(3, 3.6))
	s := fmt.Sprintf("레벨 업! Lv.%d 됐어!\n바나나 %d개 선물!", lv, gift)
	if newTitle {
		s = fmt.Sprintf("레벨 업! Lv.%d!\n이제 나는 '%s'야!\n바나나 %d개 선물!", lv, titleOf(lv), gift)
	}
	say(s, 6*time.Second)
	toast("우땅이 레벨 업!", fmt.Sprintf("Lv.%d %s · 바나나 %d개 선물", lv, titleOf(lv), gift))
	nextTalk = time.Now().Add(rd(60, 120))
}

// ── 보여 주기 ──

func expBar(cur, need int) string {
	if need == 0 {
		return "■■■■■■■■■■"
	}
	n := cur * 10 / need
	return strings.Repeat("■", n) + strings.Repeat("□", 10-n)
}

func levelMenuLine() string {
	lv, _, _ := levelOf(S.Exp)
	return fmt.Sprintf("Lv.%d %s · 함께한 지 %d일째", lv, titleOf(lv), daysTogether())
}

func expMenuLine() string {
	lv, cur, need := levelOf(S.Exp)
	if lv == maxLevel {
		return "경험치 최고 레벨!"
	}
	return fmt.Sprintf("경험치 %s %d/%d", expBar(cur, need), cur, need)
}

// "우땅이 정보" 메뉴
func doInfo() {
	lv, cur, need := levelOf(S.Exp)
	next := "최고 레벨이야!"
	if lv < maxLevel {
		next = fmt.Sprintf("다음 레벨까지 경험치 %d", need-cur)
	}
	setMode("st_me", 2200*time.Millisecond)
	say(fmt.Sprintf("Lv.%d %s\n%s 함께한 지 %d일째!\n%s\n출석 %d일 연속 · 게임 최고 %d개\n꾸미기 아이템 %d/%d개",
		lv, titleOf(lv), nm("이랑", "랑"), daysTogether(), next, S.Streak, S.Best, len(S.Owned), len(items)), 8*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

func ownerOr(def string) string {
	if ownerName == "" {
		return def
	}
	return ownerName
}

// 이름 + 조사 (받침 있으면 a, 없으면 b)  예: nm("이랑", "랑") → 지수랑 / 민준이랑
func nm(a, b string) string { return josa(ownerOr("주인님"), a, b) }

// 낱말 + 조사 (받침 있으면 a, 없으면 b)
func josa(w, a, b string) string {
	rs := []rune(w)
	if len(rs) == 0 {
		return w
	}
	last := rs[len(rs)-1]
	if last >= 0xAC00 && last <= 0xD7A3 && (last-0xAC00)%28 != 0 {
		return w + a
	}
	return w + b
}

// 친해질수록 늘어나는 대사 (레벨에 따라 열림)
func affectionLine() string {
	lv, _, _ := levelOf(S.Exp)
	n := voc()
	lines := []string{n + " 오늘도 와 줘서 고마워!"}
	if lv >= 3 {
		lines = append(lines, n+" 목소리 들으면 기분 좋아~", "우리 벌써 "+fmt.Sprint(daysTogether())+"일이나 같이 있었어!")
	}
	if lv >= 5 {
		lines = append(lines, n+" 힘들면 나한테 말해!", "히히 "+n+" 옆이 제일 좋아")
	}
	if lv >= 8 {
		lines = append(lines, nm("은", "는")+" 내 단짝이야!", n+" 오늘 뭐 했는지 궁금해~")
	}
	if lv >= 12 {
		lines = append(lines, "세상에서 "+nm("이", "가")+" 제일 좋아 ♥", n+" 없으면 심심해서 못 살아~")
	}
	if lv >= 20 {
		lines = append(lines, n+" 우리 평생 같이 있자!", "전설의 우땅이가 된 건 다 "+ownerOr("주인님")+" 덕분이야!")
	}
	return pick(lines)
}

// 기념일 인사 (켤 때 한 번, 없으면 "")
func anniversaryLine() string {
	d := daysTogether()
	switch d {
	case 7, 30, 50, 100, 200, 365, 500, 1000:
		if S.AnnivDay == today() {
			return ""
		}
		S.AnnivDay = today()
		S.Bananas += 10
		saveState()
		return fmt.Sprintf("오늘은 우리 함께한 지 %d일째 되는 날!\n기념으로 바나나 10개 받았어 ♥", d)
	}
	return ""
}
