package main

// 키우기 콘텐츠: 바나나 화폐 · 배부름/기분 · 간식 · 출석 도장 · 오늘의 운세 · 바나나 받기 게임

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type petState struct {
	Bananas int     `json:"bananas"`
	Full    float64 `json:"full"`
	Mood    float64 `json:"mood"`
	Last    int64   `json:"last"`
	Day     string  `json:"day"`
	Streak  int     `json:"streak"`
	LuckDay string  `json:"luckDay"`
	Best    int     `json:"best"`
	PetDay  string  `json:"petDay"`
	PetCnt  int     `json:"petCnt"`
	// 알림 설정 (기본값 켜짐)
	NoToast     bool `json:"noToast"`
	NoNeedAlert bool `json:"noNeedAlert"`
	// 꾸미기 (shop.go)
	Owned []string `json:"owned,omitempty"` // 산 아이템 id
	Wear  []string `json:"wear,omitempty"`  // 자리별 착용 id (머리·얼굴·목)
	// 레벨 · 친밀도 (level.go)
	Exp      int    `json:"exp"`
	LvShown  int    `json:"lvShown"`  // 축하까지 마친 레벨
	Since    string `json:"since"`    // 처음 함께한 날
	AnnivDay string `json:"annivDay"` // 기념일 인사한 날
	// 기록 · 업적 (ach.go)
	AttendTotal int               `json:"attendTotal"` // 누적 출석 일수
	SnackCnt    int               `json:"snackCnt"`
	PetTotal    int               `json:"petTotal"`
	PlayCnt     int               `json:"playCnt"`
	MatchCnt    int               `json:"matchCnt"`
	BestMatch   int               `json:"bestMatch"`     // 짝 맞추기 최소 횟수 (0 = 아직 없음)
	Ach         map[string]string `json:"ach,omitempty"` // 업적 id → 달성 날짜
	// 집중 타이머 (focus.go)
	FocusCnt int `json:"focusCnt"`
	FocusMin int `json:"focusMin"`
	// 설정 (settings.go)
	Scale      float64 `json:"scale,omitempty"`
	Opacity    int     `json:"opacity,omitempty"` // 0=100% 1=85% 2=70% 3=55%
	Speed      int     `json:"speed,omitempty"`   // 0 보통 1 느리게 2 빠르게
	NoAutoHide bool    `json:"noAutoHide,omitempty"`
	// 돌봄 (care.go): 0 = 깨끗함 · 쌩쌩
	Dirt    float64 `json:"dirt,omitempty"`
	Tired   float64 `json:"tired,omitempty"`
	BathCnt int     `json:"bathCnt,omitempty"`
	TuckCnt int     `json:"tuckCnt,omitempty"`
	BathDay string  `json:"bathDay,omitempty"`
	TuckDay string  `json:"tuckDay,omitempty"`
	// 계절 · 이벤트 (season.go)
	EventDay  string   `json:"eventDay,omitempty"`
	EventCnt  int      `json:"eventCnt,omitempty"`
	NoWeather bool     `json:"noWeather,omitempty"`
	SeenFx    []string `json:"seenFx,omitempty"`
	// 그림일기 (diary.go)
	Diary []dayRec `json:"diary,omitempty"`
}

var S = petState{Bananas: 10, Full: 70, Mood: 70}
var lastDecay = time.Now()

var statePathOverride string // 테스트용 저장 폴더 (비어 있으면 %AppData%\Woottangi)

func statePath() string {
	if statePathOverride != "" {
		return filepath.Join(statePathOverride, "state.json")
	}
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	d = filepath.Join(d, "Woottangi")
	os.MkdirAll(d, 0755)
	return filepath.Join(d, "state.json")
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func today() string { return time.Now().Format("2006-01-02") }

func loadState() {
	if b, err := os.ReadFile(statePath()); err == nil {
		json.Unmarshal(b, &S)
	}
	if S.Last > 0 { // 자리 비운 동안 천천히 줄어듦 (최대 12시간치)
		h := time.Since(time.Unix(S.Last, 0)).Hours()
		if h > 12 {
			h = 12
		}
		if h > 0 {
			S.Full = clamp(S.Full - h*6)
			S.Mood = clamp(S.Mood - h*4)
			careAway(h)
		}
	}
	saveState()
}

func saveState() {
	S.Last = time.Now().Unix()
	snapshotState()
	if b, err := json.Marshal(S); err == nil {
		os.WriteFile(statePath(), b, 0644)
	}
}

// 1분마다 조금씩 (PC는 하루 종일 켜두니까 천천히)
func stateTick() {
	if time.Since(lastDecay) < time.Minute {
		return
	}
	lastDecay = time.Now()
	if mode == "sleep" {
		S.Full, S.Mood = clamp(S.Full-0.1), clamp(S.Mood-0.05)
	} else {
		S.Full, S.Mood = clamp(S.Full-0.25), clamp(S.Mood-0.2)
	}
	if S.Dirt >= 70 { // 꼬질꼬질하면 기분이 조금 더 빨리 내려감
		S.Mood = clamp(S.Mood - 0.1)
	}
	careMinute()
	dayLog().Min++
	saveState()
}

// ── 출석 도장 ──
func checkAttendance() (gain int, isNew bool) {
	t := today()
	if S.Day == t {
		return 0, false
	}
	ys := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if S.Day == ys {
		S.Streak++
	} else {
		S.Streak = 1
	}
	S.Day = t
	S.AttendTotal++
	gain = 5
	if S.Streak%7 == 0 {
		gain += 15
	}
	S.Bananas += gain
	addExp(expAttend) // 저장 포함
	return gain, true
}

func attendLine(gain int, isNew bool) string {
	n := (S.Streak-1)%7 + 1
	stamps := ""
	for i := 1; i <= 7; i++ {
		if i <= n {
			stamps += "●"
		} else {
			stamps += "○"
		}
	}
	if isNew {
		s := fmt.Sprintf("오늘 출석 도장 쾅!\n%s\n바나나 %d개 받았어!", stamps, gain)
		if S.Streak%7 == 0 {
			s += "\n7일 연속 보너스!"
		}
		return s + "\n" + attendGiftLine()
	}
	return fmt.Sprintf("연속 출석 %d일째!\n%s\n7일마다 바나나 15개 보너스~\n%s", S.Streak, stamps, attendGiftLine())
}

func doAttend() {
	g, isNew := checkAttendance()
	setMode("st_me", 2200*time.Millisecond)
	say(attendLine(g, isNew), 5*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

// ── 간식 ──
type snack struct {
	name       string
	price      int
	full, mood float64
}

var snacks = []snack{
	{"바나나", 1, 25, 4},
	{"바나나우유", 3, 18, 12},
	{"바나나케이크", 6, 30, 25},
}

func doSnack(i int) {
	f := snacks[i]
	if S.Full >= 95 {
		setMode("st_lounge", 3*time.Second)
		say(pick([]string{"배불러… 더는 못 먹어~", "꺼억~ 나중에 줘!"}), 3*time.Second)
		return
	}
	if S.Bananas < f.price {
		setMode("st_cry", 2500*time.Millisecond)
		say("바나나가 모자라… 놀아주면 모을 수 있어!", 3500*time.Millisecond)
		return
	}
	S.Bananas -= f.price
	S.SnackCnt++
	dayLog().Snack++
	addDirt(2) // 부스러기
	S.Full = clamp(S.Full + f.full)
	S.Mood = clamp(S.Mood + f.mood)
	addExp(expSnack[i]) // 저장 포함
	setMode([]string{"eat", "eat_milk", "eat_cake"}[i], rd(5, 6.5))
	lines := map[int][]string{
		0: {"냠냠! 바나나 최고!", "와 바나나다! 냠냠", "고마워 " + voc() + "! 냠냠"},
		1: {"꿀꺽꿀꺽~ 바나나우유 맛있다!", "우유 좋아! 고마워 " + voc() + "!"},
		2: {"케이크다!! " + voc() + " 사랑해 ♥", "생일도 아닌데 케이크! 히히 냠냠"},
	}
	say(pick(lines[i]), 3500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}

// ── 오늘의 운세 (하루 하나, 날짜+이름으로 고정) ──
var luck = [][2]string{
	{"대박! 오늘은 바나나가 주렁주렁 열리는 날이야!", "행운의 간식: 바나나케이크"},
	{"생각지도 못한 칭찬을 듣게 될 거야.", "행운의 색: 노랑"},
	{"오늘은 천천히 가도 괜찮아. 쉬어가는 날!", "행운의 행동: 기지개 쭉~"},
	{"점심 메뉴 고민 끝! 처음 떠오른 걸 먹어!", "행운의 시간: 11시 30분"},
	{"오랜만에 연락 온 친구가 좋은 소식을 줄 거야.", "행운의 숫자: 7"},
	{"작은 실수가 있어도 금방 해결돼. 걱정 마!", "행운의 물건: 물 한 잔"},
	{"오늘 한 일은 다 잘될 거야. 우땅이가 응원해!", "행운의 말: 할 수 있다!"},
	{"퇴근길에 기분 좋은 일이 기다리고 있어.", "행운의 간식: 바나나우유"},
	{"새로운 걸 시작하기 딱 좋은 날이야.", "행운의 방향: 오른쪽"},
	{"오늘은 웃을 일이 세 번 생길 거야. 세어 봐!", "행운의 표정: 히히"},
}

func doLuck() {
	h := uint32(7)
	for _, c := range today() + ownerName {
		h = h*31 + uint32(c)
	}
	l := luck[h%uint32(len(luck))]
	if S.LuckDay != today() {
		S.LuckDay = today()
		S.Mood = clamp(S.Mood + 5)
		addExp(expLuck) // 저장 포함
	}
	setMode("wave", 3*time.Second)
	say(voc()+" 오늘의 운세!\n"+l[0]+"\n"+l[1], 8*time.Second)
	nextTalk = time.Now().Add(rd(60, 120))
}

// ── 쓰다듬기 / 혼내기 (게이지 반영) ──
func petBonus() {
	if S.PetDay != today() {
		S.PetDay, S.PetCnt = today(), 0
	}
	S.PetTotal++
	dayLog().Pet++
	if S.PetCnt < 10 {
		S.PetCnt++
		S.Mood = clamp(S.Mood + 6)
		addExp(expPet)
	}
	saveState()
}

func scoldPenalty() {
	S.Mood = clamp(S.Mood - 12)
	saveState()
}

// 게이지가 낮으면 하는 말 (없으면 "")
func needLine() string {
	n := voc()
	if S.Full < 30 && rng.Float64() < 0.7 {
		return pick([]string{n + " 배고파…", "꼬르륵… " + n + " 간식 줘~", "바나나 먹고 싶다…"})
	}
	if S.Mood < 30 && rng.Float64() < 0.7 {
		return pick([]string{n + " 심심해…", n + " 나랑 놀아줘…", "시무룩…"})
	}
	return careLine()
}

// ── 바나나 받기 게임: 내장 웹페이지를 작은 앱 창으로 띄움 ──
//
//go:embed game.html
var gameHTML []byte

//go:embed match.html
var matchHTML []byte

const WM_APP_SCORE = 0x8000 + 1

var (
	gameURL   string
	gameOnce  sync.Once
	gameMu    sync.Mutex
	gameToken string
)

func startGameServer() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	gameURL = fmt.Sprintf("http://%s/", ln.Addr().String())
	mux := http.NewServeMux()
	page := func(b []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
		}
	}
	mux.HandleFunc("/", page(gameHTML))
	mux.HandleFunc("/match", page(matchHTML))
	addBookRoutes(mux)
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		gameMu.Lock()
		defer gameMu.Unlock()
		if S.Full < 15 {
			w.Write([]byte(""))
			return
		}
		gameToken = strconv.FormatInt(rand.Int63(), 36)
		w.Write([]byte(gameToken))
	})
	mux.HandleFunc("/score", func(w http.ResponseWriter, r *http.Request) {
		gameMu.Lock()
		ok := gameToken != "" && r.URL.Query().Get("t") == gameToken
		gameToken = ""
		gameMu.Unlock()
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if !ok || n < 0 {
			http.Error(w, "no", 400)
			return
		}
		moves := 0 // 0 = 바나나 받기, 1 이상 = 짝 맞추기에서 뒤집은 횟수
		if r.URL.Query().Get("g") == "match" {
			moves, _ = strconv.Atoi(r.URL.Query().Get("m"))
			if moves < 8 {
				http.Error(w, "no", 400)
				return
			}
			n = max(3, min(18, 26-moves)) // 페이지가 보낸 값 대신 횟수로 다시 계산
		} else if n > 80 {
			n = 80
		}
		pPostMessageW.Call(hwnd, WM_APP_SCORE, uintptr(n), uintptr(moves))
		w.Write([]byte("ok"))
	})
	go http.Serve(ln, mux)
	return nil
}

var gameNames = []string{"바나나 받기", "짝 맞추기"}
var gamePaths = []string{"", "match"}

func openGame(which int) {
	if S.Full < 15 {
		setMode("st_lounge", 4*time.Second)
		say("배고파서 못 놀겠어… 간식 먼저!", 3*time.Second)
		return
	}
	if !openPage(gamePaths[which], 440, 700) {
		say("게임 창을 못 열었어… 흑", 3*time.Second)
		return
	}
	setMode("laugh", rd(2, 3))
	say("놀자 "+voc()+"! "+gameNames[which]+"!", 3*time.Second)
	nextTalk = time.Now().Add(rd(90, 150))
}

// 내장 웹페이지를 작은 앱 창으로 열기 (Edge/Chrome 없으면 기본 브라우저)
func openPage(path string, w, h int) bool {
	var err error
	gameOnce.Do(func() { err = startGameServer() })
	if err != nil || gameURL == "" {
		return false
	}
	url := gameURL + path
	size := fmt.Sprintf("--window-size=%d,%d", w, h)
	pf := os.Getenv("ProgramFiles(x86)")
	pf2 := os.Getenv("ProgramFiles")
	la := os.Getenv("LOCALAPPDATA")
	cands := []string{
		filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf2, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf2, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(la, `Google\Chrome\Application\chrome.exe`),
	}
	for _, c := range cands {
		if _, e := os.Stat(c); e == nil {
			if exec.Command(c, "--app="+url, size).Start() == nil {
				return true
			}
		}
	}
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start() == nil
}

// 게임 점수 도착 (UI 스레드). moves > 0 이면 짝 맞추기
func onGameScore(n, moves int) {
	S.Bananas += n
	S.Mood = clamp(S.Mood + float64(min(35, 10+n*2)))
	S.Full = clamp(S.Full - 8)
	S.PlayCnt++
	dayLog().Play++
	addDirt(3)
	best := false
	if moves > 0 {
		S.MatchCnt++
		if S.BestMatch == 0 || moves < S.BestMatch {
			S.BestMatch, best = moves, true
		}
	} else if n > S.Best {
		S.Best, best = n, true
	}
	addExp(min(20, 3+n/4)) // 저장 포함
	if n == 0 {
		setMode("st_cry", 2500*time.Millisecond)
		say("하나도 못 받았어… 다시 하자!", 3*time.Second)
		return
	}
	setMode("laugh", rd(2.5, 3.5))
	s := fmt.Sprintf("바나나 %d개 받았다! 재밌었어 %s!", n, voc())
	if moves > 0 {
		s = fmt.Sprintf("%d번 만에 다 맞혔어!\n바나나 %d개 고마워 %s!", moves, n, voc())
	}
	if best {
		s += "\n최고 기록이야!"
	}
	say(s, 4500*time.Millisecond)
	nextTalk = time.Now().Add(rd(60, 120))
}
