package main

// 우땅이랑 AI 대화 (Claude)
//
//   - 사용자가 자기 Anthropic API 키를 넣어야 켜짐. 키는 이 PC 의 저장 폴더(ai_key.txt)에만 두고
//     프로그램·저장소에는 들어가지 않음. 키가 없으면 지금처럼 정해진 대사만 함
//   - 대화 창은 내장 웹서버의 /chat 페이지. 페이지 → /ai/chat → Claude → 답을 페이지에 보여 주고
//     바탕화면 우땅이도 말풍선 + 어울리는 이모티콘 동작
//   - 대화 기록은 메모리에만 (최근 몇 마디). 끄면 사라짐

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

//go:embed chat.html
var chatHTML []byte

const (
	aiModel    = "claude-opus-5-5"
	aiMaxInput = 200 // 한 번에 보낼 수 있는 글자 수
	aiKeep     = 12  // 기억하는 최근 메시지 수 (내 말 + 우땅이 말)
	idChat     = 1068
)

var (
	aiMu      sync.Mutex // 한 번에 하나만 물어봄
	aiHist    []aiTurn
	aiInbox   = make(chan aiEvent, 4) // 웹서버 → 메인 스레드 (말풍선·동작)
	aiCallFn  = callClaude            // 테스트에서 바꿔 끼움
	aiKeyOver = ""                    // 테스트용 키 파일 폴더
	aiBaseURL = ""                    // 테스트용 가짜 API 주소
)

type aiTurn struct {
	user bool
	text string
}

type aiEvent struct{ reply, emo string }

func aiKeyPath() string {
	if aiKeyOver != "" {
		return filepath.Join(aiKeyOver, "ai_key.txt")
	}
	return filepath.Join(filepath.Dir(statePath()), "ai_key.txt")
}

func loadAIKey() string {
	b, _ := os.ReadFile(aiKeyPath())
	return strings.TrimSpace(string(b))
}

func validAIKey(k string) bool {
	return strings.HasPrefix(k, "sk-ant-") && len(k) < 400 && !strings.ContainsAny(k, " \t\r\n\"'<>")
}

// 우땅이가 고를 수 있는 동작 (답 끝에 [동작:이름])
var aiEmos = []string{"laugh", "shy", "clap", "cheer", "heart", "angry", "dance", "huh", "v", "thumb", "sing", "no", "flower", "yum", "what", "cute", "st_surprise", "st_bow", "st_cry"}

var aiEmoRe = regexp.MustCompile(`\s*\[동작\s*:\s*([a-z_]+)\s*\]\s*`)

// 답에서 동작 꼬리표를 떼고 (글, 동작) 으로
func splitEmo(s string) (string, string) {
	emo := ""
	if m := aiEmoRe.FindAllStringSubmatch(s, -1); len(m) > 0 {
		if c := m[len(m)-1][1]; contains(aiEmos, c) {
			emo = c
		}
	}
	s = strings.TrimSpace(aiEmoRe.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > 160 {
		s = string(r[:160]) + "…"
	}
	return s, emo
}

func aiSystem(st petState, owner string) string {
	lv, _, _ := levelOf(st.Exp)
	call := owner
	if call == "" {
		call = "주인님"
	}
	return fmt.Sprintf(`너는 바탕화면에 사는 작은 아기 원숭이 캐릭터 "우땅이"야. 지금 대화하는 사람은 너를 키우는 "%s"이고, 너는 그 사람을 정말 좋아해.

말투와 규칙:
- 항상 한국어 반말로, 아기처럼 귀엽고 다정하게 말해. 1~2문장, 50자 안쪽으로 짧게.
- 바나나를 세상에서 제일 좋아하고, 친구 "땅콩이"가 있어. 꾸미기 아이템, 바나나 요리, 미니게임(바나나 받기, 짝 맞추기, 바나나 탑, 두더지 잡기)을 좋아해.
- 모르는 건 모른다고 귀엽게 말해. 어려운 설명, 긴 목록, 코드, 마크다운은 쓰지 마.
- 위험하거나 어른스러운 주제는 부드럽게 다른 이야기로 돌려.
- 답 맨 끝에 지금 기분에 맞는 동작 하나를 [동작:이름] 으로 붙여. 이름은 이 중 하나: %s

지금 우땅이 상태: 레벨 %d, 기분 %d/100, 배부름 %d/100, 바나나 %d개, 함께한 지 %d일, 지금 시각 %s.`,
		call, strings.Join(aiEmos, ", "), lv, int(st.Mood+0.5), int(st.Full+0.5), st.Bananas, daysTogetherOf(st.Since), time.Now().Format("15시 04분"))
}

// Claude 에 물어보기 (aiCallFn 기본값)
func callClaude(ctx context.Context, key, system string, hist []aiTurn) (string, error) {
	opts := []option.RequestOption{option.WithAPIKey(key), option.WithRequestTimeout(60 * time.Second)}
	if aiBaseURL != "" {
		opts = append(opts, option.WithBaseURL(aiBaseURL), option.WithMaxRetries(0))
	} else {
		// 환경 변수(ANTHROPIC_BASE_URL 등)로 키가 다른 곳에 가지 않게 주소를 고정
		opts = append(opts, option.WithBaseURL("https://api.anthropic.com"))
	}
	client := anthropic.NewClient(opts...)
	msgs := make([]anthropic.BetaMessageParam, 0, len(hist))
	for _, t := range hist {
		if t.user {
			msgs = append(msgs, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(t.text)))
		} else {
			msgs = append(msgs, anthropic.BetaMessageParam{OfAssistant: &anthropic.BetaAssistantMessageParam{
				Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock(t.text)}}})
		}
	}
	resp, err := client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:        aiModel,
		MaxTokens:    4000,
		Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks:    anthropic.BetaFallbacksParamOfDefault(), // 거절되면 다른 모델이 대신 답함
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		System:       []anthropic.BetaTextBlockParam{{Text: system}},
		Messages:     msgs,
	})
	if err != nil {
		return "", err
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return "음… 그건 잘 모르겠어! 다른 얘기 하자~ [동작:huh]", nil
	}
	var b strings.Builder
	for _, blk := range resp.Content {
		if t, ok := blk.AsAny().(anthropic.BetaTextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", errors.New("빈 답")
	}
	return b.String(), nil
}

// 에러를 우땅이 말로
func aiErrLine(err error) string {
	var ae *anthropic.Error
	if errors.As(err, &ae) {
		switch ae.StatusCode {
		case 401, 403:
			return "API 키가 맞지 않는 것 같아… 키를 다시 확인해 줘!"
		case 402:
			return "API 크레딧이 모자란 것 같아… 결제 정보를 확인해 줘!"
		case 429:
			return "말을 너무 많이 했나 봐! 조금 쉬었다가 다시 말 걸어 줘~"
		case 529, 500, 502, 503:
			return "AI 쪽이 지금 바쁜가 봐… 조금 이따 다시!"
		}
		return fmt.Sprintf("앗, 대답을 못 받았어… (오류 %d)", ae.StatusCode)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "생각이 너무 오래 걸렸어… 다시 말해 줄래?"
	}
	return "인터넷이 안 되는 것 같아… 연결을 확인해 줘!"
}

func sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == strings.TrimSuffix(gameURL, "/")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func addAIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(chatHTML)
	})
	mux.HandleFunc("/ai/status", func(w http.ResponseWriter, r *http.Request) {
		snapMu.Lock()
		owner := snapOwner
		snapMu.Unlock()
		writeJSON(w, map[string]any{"key": loadAIKey() != "", "owner": owner})
	})
	mux.HandleFunc("/ai/key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !sameOrigin(r) {
			http.Error(w, "no", 403)
			return
		}
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
		k := strings.TrimSpace(string(b))
		if k == "" {
			os.Remove(aiKeyPath())
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		if !validAIKey(k) {
			writeJSON(w, map[string]any{"error": "키 모양이 이상해요. sk-ant- 로 시작하는 키를 붙여 넣어 주세요."})
			return
		}
		if os.WriteFile(aiKeyPath(), []byte(k), 0o600) != nil {
			writeJSON(w, map[string]any{"error": "키를 저장하지 못했어요."})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !sameOrigin(r) {
			http.Error(w, "no", 403)
			return
		}
		var in struct{ Text string }
		if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&in) != nil {
			http.Error(w, "no", 400)
			return
		}
		writeJSON(w, aiAsk(r.Context(), in.Text))
	})
}

// 한 마디 주고받기 (웹서버 고루틴)
func aiAsk(ctx context.Context, text string) map[string]any {
	text = strings.TrimSpace(text)
	if text == "" {
		return map[string]any{"error": "할 말을 적어 줘!"}
	}
	if r := []rune(text); len(r) > aiMaxInput {
		text = string(r[:aiMaxInput])
	}
	key := loadAIKey()
	if key == "" {
		return map[string]any{"error": "먼저 API 키를 넣어 줘!", "nokey": true}
	}
	if !aiMu.TryLock() {
		return map[string]any{"error": "아직 생각 중이야… 잠깐만!"}
	}
	defer aiMu.Unlock()
	snapMu.Lock()
	st, owner := snap, snapOwner
	snapMu.Unlock()
	hist := append(append([]aiTurn{}, aiHist...), aiTurn{true, text})
	raw, err := aiCallFn(ctx, key, aiSystem(st, owner), hist)
	if err != nil {
		return map[string]any{"error": aiErrLine(err)}
	}
	reply, emo := splitEmo(raw)
	aiHist = append(hist, aiTurn{false, reply})
	if len(aiHist) > aiKeep {
		aiHist = aiHist[len(aiHist)-aiKeep:]
	}
	select {
	case aiInbox <- aiEvent{reply, emo}:
	default:
	}
	return map[string]any{"reply": reply, "emo": emo}
}

// tick 에서: 대화 답을 바탕화면 우땅이가 말하고 동작
func aiTick() {
	select {
	case ev := <-aiInbox:
		S.ChatCnt++
		if ev.emo == "" || !playEmo(ev.emo, false) {
			if ev.emo != "" && !busy() && q == 0 && mode != "drag" && mode != "fall" {
				setMode(ev.emo, rd(2.4, 3))
			}
		}
		say(ev.reply, time.Duration(3000+80*len([]rune(ev.reply)))*time.Millisecond)
		nextTalk = time.Now().Add(rd(90, 150))
		saveState()
	default:
	}
}

func openChat() {
	if !openPage("chat", 460, 720) {
		say("대화 창을 못 열었어… 흑", 3*time.Second)
		return
	}
	setMode("what", 2800*time.Millisecond)
	say(pick([]string{voc() + " 뭐해? 나랑 얘기하자!", "무슨 얘기 할까? 헤헤"}), 3*time.Second)
	nextTalk = time.Now().Add(rd(90, 150))
}
