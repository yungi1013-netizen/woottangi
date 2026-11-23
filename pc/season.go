package main

// 계절 · 날짜 이벤트
//   - 계절 날씨 쇼: 겨울 눈 · 봄 벚꽃잎 · 여름 비눗방울 · 가을 낙엽이 가끔 화면 가득 내림 (날짜만 보고, 인터넷 안 씀)
//   - 날짜 이벤트: 새해 · 설날 · 발렌타인 · 어린이날 · 추석 · 할로윈 · 빼빼로데이 · 크리스마스 · 마지막 날
//     그날 처음 만나면 특별 인사 + 바나나, 몇몇 날은 선물 아이템 (루돌프 뿔 · 마녀 모자 · 복주머니)

import (
	"fmt"
	"time"
)

// 설날·추석 (음력이라 해마다 다름)
var lunarDays = map[string]map[int]string{
	"seollal": {2026: "02-17", 2027: "02-06", 2028: "01-26", 2029: "02-13", 2030: "02-03"},
	"chuseok": {2026: "09-25", 2027: "09-15", 2028: "10-03", 2029: "09-22", 2030: "09-12"},
}

type event struct {
	id, name, icon string
	fx             string // 날씨 쇼 종류
	gift           string // 선물 아이템 (없으면 "")
	bananas        int
	anim           string
	lines          func(n string) []string
}

var events = []struct {
	md string // "MM-DD" 또는 lunarDays 키
	ev event
}{
	{"01-01", event{"newyear", "새해 첫날", "🎍", "confetti", "", 10, "wave", func(n string) []string {
		return []string{"새해 복 많이 받아 " + n + "!\n올해도 잘 부탁해!", "새해다! " + n + " 올해는 꼭 좋은 일만 있을 거야!"}
	}}},
	{"seollal", event{"seollal", "설날", "🧧", "confetti", "bokjumeoni", 10, "st_bow", func(n string) []string {
		return []string{"설날이다! 새해 복 많이 받아~\n세배 꾸벅! 세뱃돈은 바나나로 줘~", n + " 떡국 먹었어?\n새해 복 많이 받아!"}
	}}},
	{"02-14", event{"valentine", "발렌타인데이", "💝", "heart", "", 5, "shy", func(n string) []string {
		return []string{"오늘 발렌타인데이래!\n초콜릿 대신 내 마음 줄게 ♥", n + " 사랑해! 헤헤 ♥"}
	}}},
	{"05-05", event{"children", "어린이날", "🎈", "bubble", "", 10, "laugh", func(n string) []string {
		return []string{"어린이날이다!\n오늘은 내가 주인공이야~", "어린이날 선물은… 바나나 케이크! 헤헤"}
	}}},
	{"chuseok", event{"chuseok", "추석", "🌕", "leaf", "bokjumeoni", 10, "st_bow", func(n string) []string {
		return []string{"추석이다! 보름달 보면서 소원 빌자~", n + " 송편 먹었어?\n나는 바나나 송편!"}
	}}},
	{"10-31", event{"halloween", "할로윈", "🎃", "spooky", "witch", 5, "st_surprise", func(n string) []string {
		return []string{"트릭 오어 트릿!\n바나나 안 주면 장난칠 거야~", "으스스~ 할로윈이다!\n마녀 우땅이 등장!"}
	}}},
	{"11-11", event{"pepero", "빼빼로데이", "🍫", "heart", "", 5, "shy", func(n string) []string {
		return []string{"빼빼로데이다!\n빼빼로 대신 바나나 하나 줄래?", n + " 빼빼로 받았어? 나는 " + n + " 줄래~"}
	}}},
	{"12-24", event{"xmaseve", "크리스마스 이브", "🎄", "xmas", "rudolph", 10, "wave", func(n string) []string {
		return []string{"내일은 크리스마스!\n산타 할아버지 오시려나?", n + " 크리스마스 이브야! 설렌다~"}
	}}},
	{"12-25", event{"xmas", "크리스마스", "🎁", "xmas", "rudolph", 15, "laugh", func(n string) []string {
		return []string{"메리 크리스마스 " + n + "!\n산타 할아버지 다녀가셨나 봐!", "메리 크리스마스!\n" + n + " 선물은 나야~ 헤헤"}
	}}},
	{"12-31", event{"yearend", "한 해 마지막 날", "🔔", "confetti", "", 5, "wave", func(n string) []string {
		return []string{"올해 마지막 날이야!\n" + n + " 올해도 정말 고마웠어!", "한 해 동안 수고했어 " + n + "!"}
	}}},
}

// 그날의 이벤트
func eventOn(t time.Time) (event, bool) {
	md := t.Format("01-02")
	for _, e := range events {
		if e.md == md {
			return e.ev, true
		}
		if days, ok := lunarDays[e.md]; ok && days[t.Year()] == md {
			return e.ev, true
		}
	}
	return event{}, false
}

// 다음 이벤트까지 며칠 (오늘 제외, 60일 안만)
func nextEvent(t time.Time) (event, int, bool) {
	for d := 1; d <= 60; d++ {
		if e, ok := eventOn(t.AddDate(0, 0, d)); ok {
			return e, d, true
		}
	}
	return event{}, 0, false
}

func seasonOf(t time.Time) (name, fx string) {
	switch t.Month() {
	case 12, 1, 2:
		if t.Month() == 12 && t.Day() >= 20 && t.Day() <= 25 {
			return "겨울", "xmas"
		}
		return "겨울", "snow"
	case 3, 4, 5:
		return "봄", "petal"
	case 6, 7, 8:
		return "여름", "bubble"
	}
	return "가을", "leaf"
}

var fxNames = map[string]string{
	"snow": "눈", "petal": "벚꽃잎", "bubble": "비눗방울", "leaf": "낙엽", "xmas": "크리스마스 눈",
	"confetti": "꽃가루", "heart": "하트", "spooky": "할로윈 사탕",
}

// 쇼가 시작될 때 하는 말
func showLine(kind string) string {
	n := voc()
	m := map[string][]string{
		"snow":     {"와 눈이다! 눈사람 만들자~", "펑펑 내린다! " + n + " 손 시렵지?", "첫눈일까? 소원 빌자!"},
		"petal":    {"벚꽃이 날린다~ 예쁘다!", "꽃비다! " + n + " 같이 꽃구경 가자", "봄바람 살랑살랑~"},
		"bubble":   {"비눗방울이다! 퐁퐁~", "여름이다! 시원한 바나나 셰이크 먹고 싶다", "퐁! 퐁! 다 터뜨릴 거야~"},
		"leaf":     {"낙엽이 바스락바스락~", "단풍 예쁘다! 가을이네", "은행잎이 노랗다~"},
		"xmas":     {"화이트 크리스마스다!", "메리 크리스마스 " + n + "!", "산타 할아버지 언제 오지?"},
		"confetti": {"빵빠레~ 축하해!", "꽃가루 펑펑! 신난다~"},
		"heart":    {"사랑이 퐁퐁~ ♥", n + " 좋아 ♥"},
		"spooky":   {"으스스~ 사탕 줘!", "트릭 오어 트릿!"},
	}[kind]
	if len(m) == 0 {
		return ""
	}
	return pick(m)
}

// 계절·이벤트 대사 (randomLine 에서 가끔)
func seasonLine() string {
	now := time.Now()
	n := voc()
	if e, ok := eventOn(now); ok {
		return pick(e.lines(n))
	}
	if e, d, ok := nextEvent(now); ok && d <= 7 {
		return fmt.Sprintf("%s까지 %d일 남았어! 두근두근", e.name, d)
	}
	name, _ := seasonOf(now)
	switch name {
	case "겨울":
		return pick([]string{n + " 감기 조심해! 따뜻하게 입어~", "추워… 털모자 쓰고 싶다", "군고구마 먹고 싶다~"})
	case "봄":
		return pick([]string{"봄이다! 꽃 냄새 난다~", n + " 오늘 날씨 좋지?", "나른나른 봄이야…"})
	case "여름":
		return pick([]string{"더워… 바나나 아이스크림!", n + " 물 많이 마셔!", "여름엔 수박이지~"})
	}
	return pick([]string{"가을엔 책 읽기 좋아~", "선선하다~ 산책 가고 싶다", n + " 가을 타는 거 아니지?"})
}

// ── 날씨 쇼 일정 · 이벤트 인사 ──

var (
	nextShowAt   = time.Now().Add(25 * time.Second)
	lastSeasonCk time.Time
)

func playShow(kind string, talk bool) {
	if !startShow(kind) {
		return
	}
	dayLog().Weather = kind
	if !contains(S.SeenFx, kind) {
		S.SeenFx = append(S.SeenFx, kind)
	}
	saveState()
	if talk && !focusing() && !tucked && !bathing {
		switch mode {
		case "idle", "walk", "sit":
			setMode(pick([]string{"laugh", "wave"}), rd(2.5, 3.2))
		}
		say(showLine(kind), 4*time.Second)
		nextTalk = time.Now().Add(rd(60, 120))
	}
}

func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

func showGap() time.Duration {
	if _, ok := eventOn(time.Now()); ok {
		return rd(20*60, 40*60)
	}
	return rd(45*60, 90*60)
}

func seasonTick(now time.Time) {
	if now.Sub(lastSeasonCk) < time.Second || ownerName == "" || quitting {
		return
	}
	lastSeasonCk = now
	// 그날 처음 → 이벤트 인사 (켜 둔 채 날짜가 바뀌어도)
	if e, ok := eventOn(now); ok && S.EventDay != today() && bubText == "" && greeted && now.Sub(startedAt) > 15*time.Second {
		switch mode {
		case "idle", "walk", "sit":
			celebrateEvent(e)
			return
		}
	}
	if S.NoWeather || hiddenFS || focusing() || now.Before(nextShowAt) || showing() {
		return
	}
	nextShowAt = now.Add(showGap())
	kind := todayFx(now)
	playShow(kind, true)
}

// 오늘 날씨 쇼: 이벤트 날이면 이벤트 것, 아니면 계절 것
func todayFx(now time.Time) string {
	if e, ok := eventOn(now); ok {
		return e.fx
	}
	_, fx := seasonOf(now)
	return fx
}

var startedAt = time.Now()

func celebrateEvent(e event) {
	S.EventDay = today()
	S.EventCnt++
	S.Bananas += e.bananas
	dayLog().Event = e.name
	s := pick(e.lines(voc())) + fmt.Sprintf("\n바나나 %d개 받았어!", e.bananas)
	if e.gift != "" && giveItem(e.gift) {
		it, _ := findItem(e.gift)
		s += "\n선물로 " + josa(it.name, "을", "를") + " 받았어! 바로 써 볼게!"
	}
	addExp(5) // 저장 포함
	setMode(e.anim, rd(2.5, 3.2))
	say(e.icon+" "+s, 7*time.Second)
	toast("우땅이 "+e.name+"!", fmt.Sprintf("%s %s · 바나나 %d개", e.icon, e.name, e.bananas))
	nextTalk = time.Now().Add(rd(60, 120))
	if !S.NoWeather {
		playShow(e.fx, false)
		nextShowAt = time.Now().Add(showGap())
	}
}

// ── 메뉴 ──

const (
	idShowNow   = 1200
	idWeatherOn = 1201
)

func appendSeasonMenu(m uintptr) {
	now := time.Now()
	sm, _, _ := pCreatePopupMenu.Call()
	name, _ := seasonOf(now)
	head := "지금은 " + name
	if e, ok := eventOn(now); ok {
		head += " · 오늘은 " + e.name + " " + e.icon
	} else if e, d, ok := nextEvent(now); ok {
		head += fmt.Sprintf(" · %s까지 %d일", e.name, d)
	}
	appendMenu(sm, MF_GRAYED, 0, head)
	appendMenu(sm, MF_SEPARATOR, 0, "")
	appendMenu(sm, MF_STRING, idShowNow, "지금 "+fxNames[todayFx(now)]+" 내리게 하기")
	f := uintptr(MF_STRING)
	if !S.NoWeather {
		f |= MF_CHECKED
	}
	appendMenu(sm, f, idWeatherOn, "가끔 저절로 내리기")
	appendMenu(m, MF_POPUP, sm, "계절")
}

func onSeasonCmd(cmd uintptr) bool {
	switch cmd {
	case idShowNow:
		if hiddenFS {
			return true
		}
		playShow(todayFx(time.Now()), true)
		nextShowAt = time.Now().Add(showGap())
	case idWeatherOn:
		S.NoWeather = !S.NoWeather
		if S.NoWeather {
			showUntil = time.Now()
		}
		saveState()
	default:
		return false
	}
	return true
}
