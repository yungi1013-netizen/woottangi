package main

// 선물 물어오기 · 선물 모음
//   - 우땅이가 20~50분마다 산책하다가 발밑에 작은 선물을 두고 감 (화면에 최대 3개, 40분 지나면 사라짐)
//   - 누르면 주워서 선물 모음에 쌓임. 바나나 선물은 바나나도 생김
//   - 계절마다 잘 나오는 선물이 다름 (봄 꽃잎 · 여름 조개 · 가을 낙엽·도토리 · 겨울 사탕·별)

import (
	"fmt"
	"html"
	"math"
	"net/http"
	"strings"
	"time"
)

type giftKind struct {
	id, name string
	w        float64 // 나오는 정도
	bananas  int
	season   []int // 많이 나오는 달
}

var giftKinds = []giftKind{
	{"banana", "바나나", 28, 2, nil},
	{"goldbanana", "황금 바나나", 3, 10, nil},
	{"petal", "꽃잎", 10, 0, []int{3, 4, 5}},
	{"maple", "단풍잎", 8, 0, []int{9, 10, 11}},
	{"ginkgo", "은행잎", 8, 0, []int{9, 10, 11}},
	{"acorn", "도토리", 9, 0, []int{9, 10, 11}},
	{"clover", "네잎클로버", 4, 0, []int{4, 5, 6}},
	{"shell", "조개껍데기", 7, 0, []int{6, 7, 8}},
	{"star", "반짝 별", 6, 0, []int{12, 1, 2}},
	{"candy", "사탕", 10, 0, []int{12, 1, 2}},
}

type giftT struct {
	th    *thing
	kind  *giftKind
	born  time.Time
	baseY float64
}

var (
	gifts      []*giftT
	nextGiftAt time.Time
)

const giftLife = 40 * time.Minute

func pickGift(month int) *giftKind {
	total := 0.0
	ws := make([]float64, len(giftKinds))
	for i, g := range giftKinds {
		w := g.w
		for _, m := range g.season {
			if m == month {
				w *= 3
			}
		}
		ws[i] = w
		total += w
	}
	r := rng.Float64() * total
	for i := range giftKinds {
		if r < ws[i] {
			return &giftKinds[i]
		}
		r -= ws[i]
	}
	return &giftKinds[0]
}

func canGift() bool {
	return len(gifts) < 3 && petFree() && !busy() && !hiddenFS && !toyActive() &&
		(mode == "walk" || mode == "idle") && posY >= ground-1
}

// 발밑에 선물 두기
func dropGift(k *giftKind) *giftT {
	c := cw()
	gs := int(c * 0.3)
	g := &giftT{kind: k, th: newThing(k.id, gs), born: time.Now()}
	r := g.th.r()
	g.th.x = math.Max(workL+r, math.Min(workR-r, posX+c/2-dir*c*0.38))
	g.baseY = ground + float64(canvasH) - 2 - r*0.95
	g.th.y = g.baseY
	g.th.onClick = func() { collectGift(g) }
	gifts = append(gifts, g)
	return g
}

func collectGift(g *giftT) {
	for i, x := range gifts {
		if x == g {
			gifts = append(gifts[:i], gifts[i+1:]...)
			break
		}
	}
	g.th.close()
	if S.Gifts == nil {
		S.Gifts = map[string]int{}
	}
	S.Gifts[g.kind.id]++
	S.GiftCnt++
	first := S.Gifts[g.kind.id] == 1
	S.Bananas += g.kind.bananas
	S.Mood = clamp(S.Mood + 3)
	addExp(1) // 저장 포함
	sfx("found")
	msg := g.kind.name + " 주웠다!"
	if g.kind.bananas > 0 {
		msg = fmt.Sprintf("%s! 바나나 %d개 생겼어~", g.kind.name, g.kind.bananas)
	}
	if first {
		msg = fmt.Sprintf("새 선물! %s\n선물 모음 %d/%d", g.kind.name, giftKindsGot(S.Gifts), len(giftKinds))
	}
	say(msg, 3*time.Second)
	if petFree() && (mode == "idle" || mode == "walk" || mode == "sit") {
		setMode("laugh", 1500*time.Millisecond)
	}
}

func giftKindsGot(m map[string]int) int {
	n := 0
	for _, g := range giftKinds {
		if m[g.id] > 0 {
			n++
		}
	}
	return n
}

// 매 프레임
func giftTick(now time.Time) {
	alive := gifts[:0]
	for _, g := range gifts {
		age := now.Sub(g.born)
		if age > giftLife {
			g.th.close()
			continue
		}
		a := age.Seconds()
		g.th.y = g.baseY + math.Sin(a*2.2)*2*dpi
		g.th.rot = math.Sin(a*1.5) * 0.15
		g.th.alpha = math.Min(1, math.Min(a/0.4, (giftLife-age).Seconds()/3))
		g.th.draw()
		alive = append(alive, g)
	}
	gifts = alive
	if nextGiftAt.IsZero() {
		nextGiftAt = now.Add(rd(8*60, 20*60))
		return
	}
	if now.Before(nextGiftAt) {
		return
	}
	if !canGift() {
		nextGiftAt = now.Add(30 * time.Second)
		return
	}
	nextGiftAt = now.Add(rd(20*60, 50*60))
	dropGift(pickGift(int(now.Month())))
	setMode("st_me", 2*time.Second)
	say(pick([]string{"선물 놓고 갈게~ 주워 봐!", "짠! 이거 " + voc() + " 선물!", "산책하다 주웠어! 너 줄게~"}), 3*time.Second)
	sfx("pop")
}

// ── 수첩: 선물 모음 ──

func giftBookHTML(st petState) string {
	e := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, `<h2>선물 모음 <small>%d / %d</small></h2><div class="gifts">`, giftKindsGot(st.Gifts), len(giftKinds))
	for _, g := range giftKinds {
		if n := st.Gifts[g.id]; n > 0 {
			fmt.Fprintf(&b, `<div class="gf"><img src="/fx/%s.png" alt=""><b>%s</b><span>%d개</span></div>`, g.id, e(g.name), n)
		} else {
			fmt.Fprintf(&b, `<div class="gf no"><img src="/fx/%s.png" alt=""><b>???</b><span>아직</span></div>`, g.id)
		}
	}
	b.WriteString(`</div><p class="empty">우땅이가 가끔 발밑에 두고 가는 선물을 눌러서 주워요.</p>`)
	return b.String()
}

const giftBookCSS = `.gifts{display:grid;grid-template-columns:repeat(5,1fr);gap:8px}
.gf{background:#fff;border:2px solid #2b1d16;border-radius:12px;padding:6px 4px;text-align:center;font-size:12px}
.gf img{width:44px;height:44px}.gf b{display:block;font-size:12px}.gf span{color:#7d6556}
.gf.no{background:#f1e9d8;border-style:dashed}.gf.no img{filter:brightness(0);opacity:.25}`

func addFxRoute(mux *http.ServeMux) {
	mux.HandleFunc("/fx/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/fx/")
		if strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".png") {
			http.NotFound(w, r)
			return
		}
		b, err := fxFS.ReadFile("fx/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(b)
	})
}

// ── 바나나 탑 점수 (UI 스레드) ──

const towerMax = 60

func onTowerScore(n int) {
	ban := 0
	if n > 0 {
		ban = min(15, 1+n/2)
	}
	S.Bananas += ban
	S.Mood = clamp(S.Mood + float64(min(35, 10+n)))
	S.Full = clamp(S.Full - 8)
	S.PlayCnt++
	S.TowerCnt++
	dayLog().Play++
	addDirt(3)
	best := n > S.TowerBest
	if best {
		S.TowerBest = n
	}
	addExp(min(20, 3+n/3)) // 저장 포함
	if n == 0 {
		setMode("st_cry", 2500*time.Millisecond)
		say("한 층도 못 쌓았어… 다시 하자!", 3*time.Second)
		return
	}
	setMode("laugh", rd(2.5, 3.5))
	s := fmt.Sprintf("바나나 탑 %d층!\n바나나 %d개 고마워 %s!", n, ban, voc())
	if best {
		s += "\n최고 기록이야!"
	}
	say(s, 4500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}

// ── 두더지 잡기 점수 (UI 스레드) ──

const whackMax = 90

func whackBananas(n int) int {
	if n <= 0 {
		return 0
	}
	return min(15, 1+n/3)
}

func onWhackScore(n int) {
	ban := whackBananas(n)
	S.Bananas += ban
	S.Mood = clamp(S.Mood + float64(min(35, 10+n)))
	S.Full = clamp(S.Full - 8)
	S.PlayCnt++
	S.WhackCnt++
	dayLog().Play++
	addDirt(3)
	best := n > S.WhackBest
	if best {
		S.WhackBest = n
	}
	addExp(min(20, 3+n/4)) // 저장 포함
	if n == 0 {
		setMode("st_cry", 2500*time.Millisecond)
		say("하나도 못 잡았어… 다시 하자!", 3*time.Second)
		return
	}
	setMode("clap", 2600*time.Millisecond)
	s := fmt.Sprintf("두더지 잡기 %d점!\n바나나 %d개 고마워 %s!", n, ban, voc())
	if best {
		s += "\n최고 기록이야!"
	}
	say(s, 4500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}
