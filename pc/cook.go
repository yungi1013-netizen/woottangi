package main

// 바나나 요리하기 + 요리 도감
//   - 바나나 3개로 요리를 하면 8가지 중 하나가 나옴 (귀한 요리는 잘 안 나옴)
//   - 처음 만든 요리는 도감에 올라가고, 도감은 우땅이 수첩(/book)에서 봄
//   - 폰 앱(Pet.java COOK)과 같은 요리 · 같은 확률

import (
	"fmt"
	"html"
	"strings"
	"time"
)

type recipe struct {
	id, name, icon string
	weight         int
	full, mood     float64
}

var recipes = []recipe{
	{"pancake", "바나나 팬케이크", "🥞", 30, 22, 12},
	{"shake", "바나나 셰이크", "🥤", 30, 16, 14},
	{"toast", "바나나 토스트", "🍞", 25, 24, 10},
	{"choco", "초코 바나나", "🍫", 20, 14, 18},
	{"muffin", "바나나 머핀", "🧁", 15, 20, 18},
	{"pudding", "바나나 푸딩", "🍮", 12, 18, 22},
	{"parfait", "바나나 파르페", "🍨", 8, 20, 28},
	{"royal", "황금 바나나 케이크", "🎂", 3, 30, 40},
}

const cookCost = 3

// 0~1 사이 r 로 요리 고르기 (테스트에서 r 을 정해 줄 수 있게)
func pickRecipe(r float64) int {
	tot := 0
	for _, c := range recipes {
		tot += c.weight
	}
	x := r * float64(tot)
	for i, c := range recipes {
		x -= float64(c.weight)
		if x < 0 {
			return i
		}
	}
	return len(recipes) - 1
}

func cookedKinds(m map[string]int) int {
	n := 0
	for _, c := range recipes {
		if m[c.id] > 0 {
			n++
		}
	}
	return n
}

func doCook() {
	if S.Full >= 95 {
		setMode("st_lounge", 3*time.Second)
		say("배불러서 요리는 나중에~", 3*time.Second)
		return
	}
	if S.Bananas < cookCost {
		setMode("st_cry", 2500*time.Millisecond)
		say(fmt.Sprintf("요리하려면 바나나 %d개가 필요해…", cookCost), 3500*time.Millisecond)
		return
	}
	S.Bananas -= cookCost
	i := pickRecipe(rng.Float64())
	c := recipes[i]
	if S.Cooked == nil {
		S.Cooked = map[string]int{}
	}
	isNew := S.Cooked[c.id] == 0
	S.Cooked[c.id]++
	S.CookCnt++
	S.Full = clamp(S.Full + c.full)
	S.Mood = clamp(S.Mood + c.mood)
	addDirt(2)
	addExp(4) // 저장 포함
	setMode("eat_cake", rd(5, 6))
	s := "지글지글~ 짠!\n'" + c.name + "' 완성! 냠냠"
	if isNew {
		s += fmt.Sprintf("\n새 요리 발견! 요리 도감 %d/%d", cookedKinds(S.Cooked), len(recipes))
	}
	say(s, 4500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}

func cookBookHTML(st petState) string {
	e := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, `<h2 id="cook">요리 도감 <small>%d / %d</small></h2><div class="gifts">`, cookedKinds(st.Cooked), len(recipes))
	for _, c := range recipes {
		if n := st.Cooked[c.id]; n > 0 {
			fmt.Fprintf(&b, `<div class="gf"><div class="dish">%s</div><b>%s</b><span>%d번</span></div>`, c.icon, e(c.name), n)
		} else {
			b.WriteString(`<div class="gf no"><div class="dish">❔</div><b>???</b><span>아직</span></div>`)
		}
	}
	fmt.Fprintf(&b, `</div><p class="empty">우클릭 메뉴 → 바나나 요리하기 (바나나 %d개). 귀한 요리일수록 잘 안 나와요.</p>`, cookCost)
	return b.String()
}

const cookBookCSS = `.dish{font-size:34px;line-height:44px;height:44px}.gf.no .dish{opacity:.35}`
