package main

// 업적 · 장기 출석 선물 · 우땅이 수첩(업적·도감 화면)
//
// - 업적은 조건만 적어 두면 levelTick 이 2초마다 확인해서 바나나(와 선물 아이템)를 주고 축하함
// - 수첩은 게임과 같은 내장 웹서버의 /book 페이지. 아이템 그림은 /item/{id}.png 로
//   실제 그리기 코드(drawLayer)로 우땅이에게 씌운 모습을 만들어 보여 줌

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

type achievement struct {
	id, icon, name, desc string
	bananas              int
	item                 string // 선물 아이템 id (없으면 "")
	done                 func() bool
}

func lvNow() int { l, _, _ := levelOf(S.Exp); return l }

var achievements = []achievement{
	{"snack1", "🍌", "첫 간식", "간식을 처음 줬어요", 3, "", func() bool { return S.SnackCnt >= 1 }},
	{"snack50", "🍰", "미식가 우땅이", "간식을 50번 줬어요", 20, "", func() bool { return S.SnackCnt >= 50 }},
	{"pet100", "🤲", "쓰담쓰담 장인", "쓰다듬기 100번 · 꽃 화관 선물", 15, "flowercrown", func() bool { return S.PetTotal >= 100 }},
	{"play1", "🎮", "같이 놀자!", "게임을 처음 했어요", 3, "", func() bool { return S.PlayCnt >= 1 }},
	{"play30", "🏆", "놀이 대장", "게임 30판 · 금메달 선물", 20, "medal", func() bool { return S.PlayCnt >= 30 }},
	{"catch30", "🧺", "바나나 받기 달인", "바나나 받기 한 판에 30개 이상", 15, "", func() bool { return S.Best >= 30 }},
	{"match1", "🃏", "짝꿍 찾기", "짝 맞추기를 처음 끝냈어요", 5, "", func() bool { return S.MatchCnt >= 1 }},
	{"match12", "🧠", "기억력 천재", "짝 맞추기를 12번 안에 끝냈어요", 20, "", func() bool { return S.BestMatch > 0 && S.BestMatch <= 12 }},
	{"attend7", "📅", "일주일 출석", "누적 출석 7일", 10, "", func() bool { return S.AttendTotal >= 7 }},
	{"attend30", "🌈", "한 달 출석", "누적 출석 30일 · 무지개 핀 선물", 30, "rainbow", func() bool { return S.AttendTotal >= 30 }},
	{"attend100", "👑", "백일 출석", "누적 출석 100일 · 바나나 머리띠 선물", 50, "bananaband", func() bool { return S.AttendTotal >= 100 }},
	{"lv5", "⭐", "장난꾸러기", "Lv.5 달성", 10, "", func() bool { return lvNow() >= 5 }},
	{"lv10", "🌟", "단짝 그 이상", "Lv.10 달성", 20, "", func() bool { return lvNow() >= 10 }},
	{"lv20", "💫", "전설의 시작", "Lv.20 달성", 40, "", func() bool { return lvNow() >= 20 }},
	{"items3", "🎀", "멋쟁이", "꾸미기 아이템 3개 모으기", 10, "", func() bool { return len(S.Owned) >= 3 }},
	{"itemsAll", "💎", "패셔니스타", "파는 꾸미기 아이템 전부 모으기", 30, "", ownsAllShop},
	{"items15", "👗", "옷장 부자", "꾸미기 아이템 15개 모으기 · 반짝 티아라 선물", 30, "tiara", func() bool { return len(S.Owned) >= 15 }},
	{"outfit1", "💾", "나만의 코디", "코디를 처음 저장했어요", 3, "", func() bool {
		for _, o := range S.Outfits {
			if outfitLabel(o) != "" {
				return true
			}
		}
		return false
	}},
	{"random10", "🎲", "오늘 뭐 입지?", "오늘의 코디 10번", 10, "", func() bool { return S.RandomCnt >= 10 }},
	{"focus1", "⏱️", "집중 시작", "집중 타이머를 처음 끝냈어요", 5, "", func() bool { return S.FocusCnt >= 1 }},
	{"focus10", "🎯", "집중왕", "집중 타이머 10번 끝내기 · 헤드폰 선물", 20, "headphones", func() bool { return S.FocusCnt >= 10 }},
	{"rich300", "💰", "바나나 부자", "바나나 300개 모으기", 15, "", func() bool { return S.Bananas >= 300 }},
	{"bath1", "🛁", "첫 목욕", "처음으로 씻겨 줬어요", 3, "", func() bool { return S.BathCnt >= 1 }},
	{"bath20", "🧼", "뽀송뽀송", "목욕 20번", 20, "", func() bool { return S.BathCnt >= 20 }},
	{"tuck1", "🛏️", "잘 자 우땅아", "이불을 처음 덮어 줬어요", 3, "", func() bool { return S.TuckCnt >= 1 }},
	{"tuck20", "🌙", "꿀잠 전문가", "이불 덮어 재우기 20번", 20, "", func() bool { return S.TuckCnt >= 20 }},
	{"event1", "🎉", "축제 좋아", "기념일 이벤트를 처음 함께했어요", 5, "", func() bool { return S.EventCnt >= 1 }},
	{"seasons", "🍂", "사계절 친구", "눈·벚꽃잎·비눗방울·낙엽을 모두 봤어요", 30, "", seenAllSeasons},
	{"diary7", "📔", "일기 일주일", "그림일기가 7일 쌓였어요", 10, "", func() bool { return diaryDays(S.Diary) >= 7 }},
	{"desk20", "🧺", "바탕화면 사냥꾼", "바탕화면 바나나 받기 한 판에 20개 이상", 15, "", func() bool { return S.DeskBest >= 20 }},
	{"hide1", "🙈", "꼭꼭 숨어라", "숨바꼭질에서 우땅이를 처음 찾았어요", 5, "", func() bool { return S.HideCnt >= 1 }},
	{"hide20", "🔍", "숨바꼭질 탐정", "숨바꼭질에서 20번 찾기", 20, "", func() bool { return S.HideCnt >= 20 }},
	{"gacha1", "🎰", "두근두근 뽑기", "바나나 뽑기를 처음 했어요", 3, "", func() bool { return S.GachaCnt >= 1 }},
	{"gachaAll", "✨", "뽑기 수집가", "뽑기 아이템 4개 모두 모으기", 30, "", func() bool {
		for _, id := range gachaPool {
			if !owns(id) {
				return false
			}
		}
		return true
	}},
	{"friend1", "🥜", "친구가 왔어요", "땅콩이가 처음 놀러 왔어요", 5, "", func() bool { return S.FriendCnt >= 1 }},
	{"friend10", "🤝", "단짝 친구", "땅콩이와 10번 놀기", 20, "", func() bool { return S.FriendCnt >= 10 }},
	{"memo1", "📝", "메모 부탁해", "포스트잇 메모를 처음 맡겼어요", 3, "", func() bool { return S.MemoCnt >= 1 }},
	{"stretch10", "🙆", "건강 지킴이", "스트레칭 알림 10번 받기", 10, "", func() bool { return S.StretchCnt >= 10 }},
	{"toy1", "⚽", "공놀이 시작", "장난감을 처음 꺼내 줬어요", 3, "", func() bool { return S.ToyCnt >= 1 }},
	{"toy100", "🎈", "공놀이 선수", "장난감을 100번 차고 튕기기", 15, "", func() bool { return S.ToyKick >= 100 }},
	{"gift1", "🎁", "첫 선물", "우땅이 선물을 처음 주웠어요", 3, "", func() bool { return S.GiftCnt >= 1 }},
	{"giftAll", "🧺", "선물 수집가", "선물 10가지를 모두 모았어요", 30, "", func() bool { return giftKindsGot(S.Gifts) >= len(giftKinds) }},
	{"tower10", "🏗️", "바나나 탑 10층", "바나나 탑을 10층 쌓았어요", 10, "", func() bool { return S.TowerBest >= 10 }},
	{"tower25", "🗼", "하늘까지 탑", "바나나 탑을 25층 쌓았어요", 25, "", func() bool { return S.TowerBest >= 25 }},
	{"night", "🦉", "올빼미", "밤 12시~새벽 4시에 함께 있기", 5, "", func() bool { return time.Now().Hour() < 4 }},
	{"day100", "💝", "100일의 우땅이", "함께한 지 100일", 30, "", func() bool { return daysTogether() >= 100 }},
	{"whack1", "🔨", "뿅망치 데뷔", "두더지 잡기를 처음 했어요", 3, "", func() bool { return S.WhackCnt >= 1 }},
	{"whack30", "🎯", "두더지 잡기 달인", "두더지 잡기 한 판에 30점 이상", 15, "", func() bool { return S.WhackBest >= 30 }},
	{"cook1", "🍳", "첫 요리", "바나나 요리를 처음 했어요", 3, "", func() bool { return S.CookCnt >= 1 }},
	{"cook10", "👨‍🍳", "바나나 요리사", "요리 10번 · 요리사 모자 선물", 15, "chefhat", func() bool { return S.CookCnt >= 10 }},
	{"cookAll", "📖", "요리 도감 완성", "요리 8가지를 모두 만들었어요", 30, "", func() bool { return cookedKinds(S.Cooked) >= len(recipes) }},
	{"gif1", "🖼️", "이모티콘 부자", "이모티콘 GIF를 저장했어요", 3, "", func() bool { return S.GifSaved }},
	{"chat1", "💬", "말이 통했어", "우땅이랑 AI 대화를 처음 했어요", 3, "", func() bool { return S.ChatCnt >= 1 }},
	{"dance10", "🎶", "음악 친구", "음악에 맞춰 10번 춤췄어요", 10, "", func() bool { return S.DanceCnt >= 10 }},
	{"chat30", "🗨️", "수다쟁이 우땅이", "우땅이랑 AI 대화 30번", 15, "", func() bool { return S.ChatCnt >= 30 }},
}

func seenAllSeasons() bool {
	for _, k := range []string{"snow", "petal", "bubble", "leaf"} {
		if !contains(S.SeenFx, k) && !(k == "snow" && contains(S.SeenFx, "xmas")) {
			return false
		}
	}
	return true
}

func ownsAllShop() bool {
	for _, it := range items {
		if it.price > 0 && !owns(it.id) {
			return false
		}
	}
	return true
}

// 레벨·업적 기능 전부터 키우던 경우: 누적 출석을 연속 출석 기록에서 시작
func initAch() {
	if S.Ach == nil {
		S.Ach = map[string]string{}
	}
	if S.AttendTotal == 0 && S.Day != "" {
		S.AttendTotal = max(1, S.Streak)
	}
	// 업데이트하고 처음 켜면 새로워진 점을 한 번 알려 줌 (처음 설치한 사람은 건너뜀)
	if S.SeenVer != newsID {
		if S.Day != "" {
			notices = append(notices, updateNews)
			toast("우땅이 "+appVersion+" 업데이트!", "꾸미기 8개 추가 · 코디 저장 · 오늘의 코디 · AI 대화 · 새 이모티콘")
		}
		S.SeenVer = newsID
	}
	// 보상 아이템이 생기기 전에 이미 깬 업적: 아이템만 조용히 챙겨 줌
	for _, a := range achievements {
		if a.item != "" && S.Ach[a.id] != "" && !owns(a.item) {
			S.Owned = append(S.Owned, a.item)
		}
	}
	saveState()
}

// 업데이트 소식 (말풍선). 폰 앱 MainActivity.NEWS 와 같은 내용
// 소식 번호: 같은 버전 안에서 소식이 늘면 뒤를 바꿔 한 번 더 보여 줌 (폰 MainActivity.newsId 와 같게)
var newsID = appVersion + "-deco"

const updateNews = "업데이트 됐어! 새로 생긴 것:\n" +
	"· 꾸미기 8개 (고양이 귀·토끼 귀·밀짚모자·진주 목걸이…)\n" +
	"· 코디 저장 3칸 · 오늘의 코디 (꾸미기 → 코디)\n" +
	"· 나랑 AI 대화하기 (메뉴 → 우땅이랑 대화하기)\n" +
	"· 이모티콘 8개 (브이·엄지척·노래…)\n" +
	"· 두더지 잡기 · 바나나 요리 도감\n" +
	"· 음악 나오면 춤추기 · 업적 꾸미기 선물\n" +
	"기록은 그대로야!"

// 출석 도장에 붙는 장기 출석 안내
func attendGiftLine() string {
	for _, g := range []struct {
		day  int
		name string
	}{{30, "무지개 핀"}, {100, "바나나 머리띠"}} {
		if S.AttendTotal < g.day {
			return fmt.Sprintf("누적 출석 %d일 · %d일 선물(%s)까지 %d일!", S.AttendTotal, g.day, g.name, g.day-S.AttendTotal)
		}
	}
	return fmt.Sprintf("누적 출석 %d일! 언제나 고마워~", S.AttendTotal)
}

// ── 달성 확인 · 축하 대기열 ──

var (
	notices     []string
	lastAchScan time.Time
	lastNotice  time.Time
)

func achTick() {
	if time.Since(lastAchScan) < 2*time.Second || ownerName == "" {
		return
	}
	lastAchScan = time.Now()
	checkAchievements()
}

// 새로 달성한 업적 id 목록
func checkAchievements() []string {
	if S.Ach == nil {
		S.Ach = map[string]string{}
	}
	var got []string
	for _, a := range achievements {
		if S.Ach[a.id] != "" || !a.done() {
			continue
		}
		S.Ach[a.id] = today()
		S.Bananas += a.bananas
		s := fmt.Sprintf("업적 달성!\n'%s'\n바나나 %d개 받았어!", a.name, a.bananas)
		if a.item != "" && giveItem(a.item) {
			it, _ := findItem(a.item)
			s += "\n선물로 " + josa(it.name, "을", "를") + " 받았어! 바로 써 볼게!"
		}
		notices = append(notices, s)
		toast("우땅이 업적 달성!", fmt.Sprintf("%s · 바나나 %d개", a.name, a.bananas))
		got = append(got, a.id)
	}
	if len(got) > 0 {
		saveState()
	}
	return got
}

// levelTick 에서 말풍선이 비었을 때 하나씩
func showNotice() bool {
	if len(notices) == 0 || time.Since(lastNotice) < 2*time.Second {
		return false
	}
	s := notices[0]
	notices = notices[1:]
	lastNotice = time.Now()
	setMode("laugh", rd(2.5, 3.2))
	say(s, 5500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
	return true
}

// ── 수첩 페이지 (웹서버 고루틴에서 읽으므로 저장할 때마다 복사본을 만들어 둠) ──

var (
	snapMu    sync.Mutex
	snap      petState
	snapOwner string
)

func snapshotState() {
	snapMu.Lock()
	snap = S
	snap.Owned = slices.Clone(S.Owned)
	snap.Wear = slices.Clone(S.Wear)
	snap.Ach = maps.Clone(S.Ach)
	snap.Cooked = maps.Clone(S.Cooked)
	snap.SeenFx = slices.Clone(S.SeenFx)
	snap.Diary = slices.Clone(S.Diary)
	snapOwner = ownerName
	snapMu.Unlock()
}

func openBook() {
	if !openPage("book", 560, 760) {
		say("수첩을 못 열었어… 흑", 3*time.Second)
		return
	}
	setMode("st_me", 2200*time.Millisecond)
	say(pick([]string{"내 수첩 보여줄게!", "짠! 우리 추억 수첩이야~"}), 3*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

var (
	itemPNGMu sync.Mutex
	itemPNG   = map[string][]byte{}
)

func addBookRoutes(mux *http.ServeMux) {
	addFxRoute(mux) // 선물 그림 (gift.go)
	mux.HandleFunc("/book", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		snapMu.Lock()
		st, owner := snap, snapOwner
		snapMu.Unlock()
		w.Write([]byte(bookHTML(st, owner)))
	})
	mux.HandleFunc("/item/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/item/"), ".png")
		if _, ok := accSprites[id]; !ok && id != "none" {
			http.NotFound(w, r)
			return
		}
		solo := r.URL.Query().Get("solo") != "" // 아직 없는 아이템: 아이템만 (도감 실루엣용)
		key := id
		if solo {
			key += "#solo"
		}
		itemPNGMu.Lock()
		b, ok := itemPNG[key]
		if !ok {
			b = renderItemPNG(id, solo)
			itemPNG[key] = b
		}
		itemPNGMu.Unlock()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(b)
	})
}

// idle 첫 프레임에 아이템 하나만 씌운 그림 (200x270). solo 면 몸 없이 아이템만
func renderItemPNG(id string, solo bool) []byte {
	fr := seqs["idle"][0]
	sp := accSprites[id]
	const x0, y0, ow, oh = 40, 60, 200, 270
	out := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			su, sv := float64(x0)+(float64(x)+0.5)*2-0.5, float64(y0)+(float64(y)+0.5)*2-0.5
			var c [4]float32
			if !solo {
				c = sample(fr, su, sv)
			}
			if sp != nil {
				drawLayer(fr, sp, su+0.5, sv+0.5, &c)
			}
			if c[3] < 1 {
				continue
			}
			a := c[3] / 255
			out.SetNRGBA(x, y, color.NRGBA{byte(c[2]/a + 0.5), byte(c[1]/a + 0.5), byte(c[0]/a + 0.5), byte(c[3] + 0.5)})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, out)
	return b.Bytes()
}

func bookHTML(st petState, owner string) string {
	e := html.EscapeString
	if owner == "" {
		owner = "주인님"
	}
	lv, cur, need := levelOf(st.Exp)
	pct := 100
	if need > 0 {
		pct = cur * 100 / need
	}
	got := 0
	for _, a := range achievements {
		if st.Ach[a.id] != "" {
			got++
		}
	}
	has := func(id string) bool { return slices.Contains(st.Owned, id) }

	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>우땅이 수첩</title><style>
*{box-sizing:border-box}body{margin:0;font-family:"Malgun Gothic","Apple SD Gothic Neo",system-ui,sans-serif;background:#fff4d6;color:#2b1d16}
main{max-width:560px;margin:0 auto;padding:16px}
h1{margin:4px 0 2px;font-size:24px}h2{margin:22px 0 10px;font-size:19px;display:flex;justify-content:space-between;align-items:baseline}h2 small{font-size:14px;color:#7d6556;font-weight:400}
.top{display:flex;gap:14px;align-items:center;background:#fff;border:3px solid #2b1d16;border-radius:16px;padding:12px;box-shadow:2px 4px 0 #2b1d16}
.top img{width:92px;flex:none}.top p{margin:2px 0;color:#7d6556;font-size:14px}
.bar{height:14px;border:2px solid #2b1d16;border-radius:8px;background:#fff;overflow:hidden;margin:6px 0 2px}.bar i{display:block;height:100%;background:#ffd84d}
.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin-top:12px}
.stats div{background:#fff;border:2px solid #2b1d16;border-radius:12px;padding:6px 8px;font-size:13px;color:#7d6556}.stats b{display:block;font-size:18px;color:#2b1d16}
.ach{display:grid;grid-template-columns:repeat(2,1fr);gap:8px}
.a{display:flex;gap:8px;align-items:center;background:#fff;border:2px solid #2b1d16;border-radius:12px;padding:8px}
.a .i{font-size:26px;width:34px;text-align:center}.a b{display:block;font-size:14px}.a span{font-size:12px;color:#7d6556}
.a.no{background:#f1e9d8;border-style:dashed;color:#9c8b7c}.a.no .i{filter:grayscale(1);opacity:.45}
.items{display:grid;grid-template-columns:repeat(3,1fr);gap:8px}
.it{background:#fff;border:2px solid #2b1d16;border-radius:12px;padding:6px;text-align:center;font-size:13px}
.it img{width:100%;aspect-ratio:200/270;object-fit:contain}.it b{display:block}.it span{color:#7d6556;font-size:12px}
.it.no{background:#f1e9d8;border-style:dashed}.it.no img{filter:brightness(0);opacity:.3}
.dy{background:#fff;border:2px solid #2b1d16;border-radius:12px;padding:10px 12px;margin-bottom:8px}
.dh{display:flex;justify-content:space-between;align-items:center;font-size:14px;margin-bottom:4px}.dh span{font-size:20px}
.dy p{margin:0;font-size:14px;line-height:22px;background:repeating-linear-gradient(transparent 0 21px,#f1e3c4 21px 22px)}
.empty{color:#7d6556;font-size:14px}
` + giftBookCSS + cookBookCSS + `
.on{display:inline-block;margin-top:2px;background:#ffd84d;border:2px solid #2b1d16;border-radius:8px;padding:0 6px;font-size:11px}
</style></head><body><main>`)
	fmt.Fprintf(&b, `<div class="top"><img src="/item/none.png" alt=""><div style="flex:1"><h1>Lv.%d %s</h1><p>%s 함께한 지 %d일째</p>`,
		lv, e(titleOf(lv)), e(josa(owner, "이랑", "랑")), daysTogetherOf(st.Since))
	if need > 0 {
		fmt.Fprintf(&b, `<div class="bar"><i style="width:%d%%"></i></div><p>경험치 %d / %d · 다음 레벨까지 %d</p>`, pct, cur, need, need-cur)
	} else {
		b.WriteString(`<div class="bar"><i style="width:100%"></i></div><p>최고 레벨!</p>`)
	}
	b.WriteString(`</div></div><div class="stats">`)
	bm := "-"
	if st.BestMatch > 0 {
		bm = fmt.Sprintf("%d번", st.BestMatch)
	}
	for _, s := range [][2]string{
		{"바나나", fmt.Sprint(st.Bananas)}, {"누적 출석", fmt.Sprintf("%d일", st.AttendTotal)}, {"연속 출석", fmt.Sprintf("%d일", st.Streak)},
		{"간식", fmt.Sprintf("%d번", st.SnackCnt)}, {"쓰다듬기", fmt.Sprintf("%d번", st.PetTotal)}, {"게임", fmt.Sprintf("%d판", st.PlayCnt)},
		{"바나나 받기 최고", fmt.Sprintf("%d개", st.Best)}, {"짝 맞추기 최고", bm}, {"꾸미기", fmt.Sprintf("%d/%d", len(st.Owned), len(items))},
		{"집중", fmt.Sprintf("%d번", st.FocusCnt)}, {"집중한 시간", fmt.Sprintf("%d분", st.FocusMin)}, {"업적", fmt.Sprintf("%d/%d", got, len(achievements))},
		{"목욕", fmt.Sprintf("%d번", st.BathCnt)}, {"이불 덮기", fmt.Sprintf("%d번", st.TuckCnt)}, {"그림일기", fmt.Sprintf("%d일", diaryDays(st.Diary))},
		{"숨바꼭질", fmt.Sprintf("%d번", st.HideCnt)}, {"바나나 뽑기", fmt.Sprintf("%d번", st.GachaCnt)}, {"땅콩이 방문", fmt.Sprintf("%d번", st.FriendCnt)},
		{"장난감 놀이", fmt.Sprintf("%d번", st.ToyKick)}, {"주운 선물", fmt.Sprintf("%d개", st.GiftCnt)}, {"바나나 탑 최고", fmt.Sprintf("%d층", st.TowerBest)},
		{"두더지 잡기 최고", fmt.Sprintf("%d점", st.WhackBest)}, {"요리", fmt.Sprintf("%d번", st.CookCnt)}, {"음악 춤", fmt.Sprintf("%d번", st.DanceCnt)},
	} {
		fmt.Fprintf(&b, `<div>%s<b>%s</b></div>`, s[0], e(s[1]))
	}
	fmt.Fprintf(&b, `</div><h2>업적 <small>%d / %d</small></h2><div class="ach">`, got, len(achievements))
	for _, a := range achievements {
		if d := st.Ach[a.id]; d != "" {
			fmt.Fprintf(&b, `<div class="a"><div class="i">%s</div><div><b>%s</b><span>%s · %s 달성</span></div></div>`, a.icon, e(a.name), e(a.desc), e(d))
		} else {
			fmt.Fprintf(&b, `<div class="a no"><div class="i">%s</div><div><b>%s</b><span>%s · 바나나 %d개</span></div></div>`, a.icon, e(a.name), e(a.desc), a.bananas)
		}
	}
	b.WriteString(`</div>`)
	b.WriteString(giftBookHTML(st))
	b.WriteString(cookBookHTML(st))
	b.WriteString(diaryHTML(st.Diary, owner, e))
	fmt.Fprintf(&b, `<h2>꾸미기 도감 <small>%d / %d</small></h2><div class="items">`, len(st.Owned), len(items))
	for _, it := range items {
		how := fmt.Sprintf("바나나 %d개", it.price)
		if it.price == 0 {
			how = giftFrom[it.id]
		}
		if has(it.id) {
			on := ""
			if slices.Contains(st.Wear, it.id) {
				on = `<span class="on">쓰는 중</span>`
			}
			fmt.Fprintf(&b, `<div class="it"><img src="/item/%s.png" alt=""><b>%s</b><span>%s</span>%s</div>`, it.id, e(it.name), e(slotNames[it.slot]), on)
		} else {
			fmt.Fprintf(&b, `<div class="it no"><img src="/item/%s.png?solo=1" alt=""><b>???</b><span>%s</span></div>`, it.id, e(how))
		}
	}
	b.WriteString(`</div><p style="text-align:center;color:#7d6556;font-size:13px;margin:22px 0 6px">우클릭 메뉴 → 꾸미기 에서 아이템을 쓰고 벗을 수 있어요</p></main>
<script>if(location.hash){var el=document.querySelector(location.hash);if(el)el.scrollIntoView()}</script></body></html>`)
	return b.String()
}

func daysTogetherOf(since string) int {
	t, err := time.ParseInLocation("2006-01-02", since, time.Local)
	if err != nil {
		return 1
	}
	y, m, d := time.Now().Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.Local).Sub(t).Hours()/24+0.5) + 1
}
