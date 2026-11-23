package main

// 우땅이 그림일기: 하루 동안 한 일(같이 있던 시간 · 쓰다듬기 · 간식 · 놀이 · 목욕 · 이불 · 집중 · 날기 · 레벨 · 날씨 · 이벤트)을
// 날마다 모아 두고, 수첩에서 우땅이 말투의 일기로 보여 줌 (최근 60일)

import (
	"fmt"
	"strings"
	"time"
)

type dayRec struct {
	Date    string `json:"d"`
	Min     int    `json:"min,omitempty"` // 켜 두고 같이 있던 시간(분)
	Pet     int    `json:"pet,omitempty"`
	Snack   int    `json:"snack,omitempty"`
	Play    int    `json:"play,omitempty"`
	Bath    int    `json:"bath,omitempty"`
	Tuck    int    `json:"tuck,omitempty"`
	Focus   int    `json:"focus,omitempty"` // 집중한 분
	Throw   int    `json:"throw,omitempty"`
	Level   int    `json:"lv,omitempty"` // 그날 오른 레벨
	Weather string `json:"w,omitempty"`  // 그날 본 날씨 쇼
	Event   string `json:"ev,omitempty"` // 그날 이벤트 이름
}

const diaryKeep = 60

// 오늘 기록 (없으면 새로)
func dayLog() *dayRec {
	t := today()
	if n := len(S.Diary); n > 0 && S.Diary[n-1].Date == t {
		return &S.Diary[n-1]
	}
	S.Diary = append(S.Diary, dayRec{Date: t})
	if len(S.Diary) > diaryKeep {
		S.Diary = append([]dayRec(nil), S.Diary[len(S.Diary)-diaryKeep:]...)
	}
	return &S.Diary[len(S.Diary)-1]
}

// 일기가 쓰인 날 수 (같이 1분 이상 있던 날)
func diaryDays(d []dayRec) int {
	n := 0
	for _, r := range d {
		if r.Min > 0 {
			n++
		}
	}
	return n
}

var weatherDiary = map[string]string{
	"snow": "오늘은 눈이 펑펑 내렸다. ❄️", "xmas": "하얀 눈이 내렸다. 크리스마스 느낌! ❄️",
	"petal": "벚꽃잎이 팔랑팔랑 날렸다. 🌸", "bubble": "비눗방울이 퐁퐁 떠올랐다. 🫧",
	"leaf": "낙엽이 바스락바스락 떨어졌다. 🍁", "confetti": "꽃가루가 펑펑! 🎉",
	"heart": "하트가 퐁퐁 날아다녔다. 💗", "spooky": "사탕이 하늘에서 떨어졌다! 🍬",
}

// 일기 문장들 (날짜로 고정된 무작위라 다시 열어도 같은 글)
func diaryText(r dayRec, owner string) []string {
	if owner == "" {
		owner = "주인님"
	}
	h := uint32(17)
	for _, c := range r.Date {
		h = h*31 + uint32(c)
	}
	pk := func(a ...string) string {
		h = h*1103515245 + 12345
		return a[(h>>8)%uint32(len(a))]
	}
	o := josa(owner, "이랑", "랑")
	var s []string
	switch {
	case r.Event != "":
		s = append(s, fmt.Sprintf("오늘은 %s! %s 같이 보내서 좋았다.", r.Event, o))
	case r.Weather != "":
		s = append(s, weatherDiary[r.Weather])
	}
	switch {
	case r.Min >= 60:
		s = append(s, fmt.Sprintf("%s %d시간 %d분이나 같이 있었다.", o, r.Min/60, r.Min%60))
	case r.Min > 0:
		s = append(s, fmt.Sprintf("%s %d분 같이 있었다. %s", o, r.Min, pk("조금 아쉽다.", "짧아도 좋았다!")))
	}
	switch {
	case r.Pet >= 10:
		s = append(s, fmt.Sprintf("쓰다듬어 준 게 무려 %d번! %s", r.Pet, pk("머리가 반들반들해졌다.", "행복해서 녹을 뻔했다.")))
	case r.Pet > 0:
		s = append(s, fmt.Sprintf("%d번 쓰다듬어 줬다. %s", r.Pet, pk("헤헤.", "기분 좋았다.")))
	}
	switch {
	case r.Snack >= 3:
		s = append(s, fmt.Sprintf("간식을 %d번이나 먹었다. %s", r.Snack, pk("배 터지겠다!", "냠냠, 최고의 하루.")))
	case r.Snack > 0:
		s = append(s, pk("간식을 먹었다. 바나나는 언제 먹어도 맛있다.", "냠냠! 간식 고마워."))
	}
	if r.Play > 0 {
		s = append(s, fmt.Sprintf("같이 게임을 %d판 했다. %s", r.Play, pk("재밌었다!", "다음엔 더 잘할 거야.")))
	}
	if r.Bath > 0 {
		s = append(s, pk("목욕을 했다. 몸이 뽀송뽀송!", "거품 목욕을 했다. 반짝반짝 ✨"))
	}
	if r.Tuck > 0 {
		s = append(s, pk("이불을 덮어 줘서 푹 잤다.", "포근한 이불 속에서 꿀잠 잤다."))
	}
	if r.Focus > 0 {
		s = append(s, fmt.Sprintf("%s %d분 동안 집중했다. 옆에서 조용히 응원했다.", josa(owner, "이", "가"), r.Focus))
	}
	if r.Throw > 0 {
		s = append(s, fmt.Sprintf("하늘을 %d번 날았다! %s", r.Throw, pk("무서웠지만 재밌었다.", "어질어질~")))
	}
	if r.Level > 0 {
		s = append(s, fmt.Sprintf("레벨이 올라서 Lv.%d이 됐다!", r.Level))
	}
	if len(s) == 0 || (r.Min == 0 && len(s) <= 1) {
		return append(s, josa(owner, "을", "를")+" 잠깐밖에 못 봤다… 내일은 더 오래 같이 있자.")
	}
	return append(s, pk("내일도 같이 놀자!", owner+" 최고. 끝!", "오늘도 행복했다 ♥", "내일은 바나나 케이크 먹고 싶다."))
}

// 그날 기분 그림
func diaryFace(r dayRec) string {
	n := r.Pet + r.Snack*2 + r.Play*2 + r.Bath*2 + r.Tuck*2
	switch {
	case n >= 12:
		return "😆"
	case n >= 4:
		return "😊"
	case r.Min >= 60:
		return "🙂"
	}
	return "🥺"
}

func diaryDate(d string) string {
	t, err := time.ParseInLocation("2006-01-02", d, time.Local)
	if err != nil {
		return d
	}
	return fmt.Sprintf("%d월 %d일 (%s)", t.Month(), t.Day(), []string{"일", "월", "화", "수", "목", "금", "토"}[t.Weekday()])
}

// 수첩 페이지의 일기 부분 (최신부터 30일)
func diaryHTML(d []dayRec, owner string, esc func(string) string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<h2 id="diary">우땅이 그림일기 <small>%d일</small></h2>`, diaryDays(d))
	shown := 0
	for i := len(d) - 1; i >= 0 && shown < 30; i-- {
		r := d[i]
		if r.Min == 0 && r.Pet+r.Snack+r.Play+r.Bath+r.Tuck == 0 {
			continue
		}
		shown++
		fmt.Fprintf(&b, `<div class="dy"><div class="dh"><b>%s</b><span>%s</span></div><p>%s</p></div>`,
			esc(diaryDate(r.Date)), diaryFace(r), esc(strings.Join(diaryText(r, owner), " ")))
	}
	if shown == 0 {
		b.WriteString(`<p class="empty">아직 일기가 없어요. 오늘 하루가 지나면 우땅이가 일기를 써요!</p>`)
	}
	return b.String()
}

func openDiary() {
	if !openPage("book#diary", 560, 760) {
		say("일기장을 못 열었어… 흑", 3*time.Second)
		return
	}
	setMode("shy", rd(2.5, 3))
	say(pick([]string{"내 일기 몰래 보는 거야? 히히", "부끄럽지만… 보여 줄게!"}), 3*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}
