package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAISplitEmo(t *testing.T) {
	s, e := splitEmo("안녕! 바나나 먹자~ [동작:v]")
	if s != "안녕! 바나나 먹자~" || e != "v" {
		t.Fatalf("%q %q", s, e)
	}
	if s, e = splitEmo("헤헤 [동작:rm_rf]"); e != "" || s != "헤헤" {
		t.Fatal("모르는 동작은 버림")
	}
	if s, _ = splitEmo(strings.Repeat("가", 300)); len([]rune(s)) > 161 {
		t.Fatal("너무 긴 답 자르기")
	}
	if !validAIKey("sk-ant-api03-abc_DEF-123") || validAIKey("hello") || validAIKey("sk-ant-a b") {
		t.Fatal("키 모양 검사")
	}
}

// 실제로 나가는 요청 모양 (가짜 API 서버로 확인 · 돈 안 듦)
func TestAICallRequest(t *testing.T) {
	var body map[string]any
	var hdr http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		if strings.Contains(r.Header.Get("Authorization")+r.Header.Get("X-Api-Key"), "sk-ant-bad") {
			w.WriteHeader(401)
			w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5",
			"content":[{"type":"thinking","thinking":"","signature":"x"},{"type":"text","text":"안녕 민지! [동작:heart]"}],
			"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}}`))
	}))
	defer srv.Close()
	aiBaseURL = srv.URL
	defer func() { aiBaseURL = "" }()

	hist := []aiTurn{{true, "안녕"}, {false, "안녕! 반가워"}, {true, "뭐해?"}}
	got, err := callClaude(context.Background(), "sk-ant-test", "시스템", hist)
	if err != nil || got != "안녕 민지! [동작:heart]" {
		t.Fatalf("답: %q %v", got, err)
	}
	if body["model"] != aiModel || body["fallbacks"] != "default" {
		t.Fatalf("모델/대체 모델: %v %v", body["model"], body["fallbacks"])
	}
	if oc, _ := body["output_config"].(map[string]any); oc["effort"] != "low" {
		t.Fatalf("effort: %v", body["output_config"])
	}
	if _, bad := body["temperature"]; bad {
		t.Fatal("temperature 를 보내면 안 됨")
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 3 || msgs[1].(map[string]any)["role"] != "assistant" {
		t.Fatalf("대화 기록: %v", msgs)
	}
	t.Logf("auth=%q xkey=%q", hdr.Get("Authorization"), hdr.Get("X-Api-Key"))
	if hdr.Get("Authorization") != "Bearer sk-ant-test" && hdr.Get("X-Api-Key") != "sk-ant-test" || !strings.Contains(hdr.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Fatalf("헤더: key=%q beta=%q", hdr.Get("X-Api-Key"), hdr.Get("Anthropic-Beta"))
	}
	// 키가 틀리면 우땅이 말로 알려 줌
	_, err = callClaude(context.Background(), "sk-ant-bad", "시스템", hist)
	if err == nil || !strings.Contains(aiErrLine(err), "API 키") {
		t.Fatalf("401: %v / %s", err, aiErrLine(err))
	}
	if !strings.Contains(aiErrLine(errors.New("dial tcp")), "인터넷") {
		t.Fatal("연결 오류 문구")
	}
}

// 한 마디 주고받기: 키 없으면 안내, 있으면 답 + 바탕화면 우땅이 동작
func TestAIAsk(t *testing.T) {
	setupMove(t)
	aiKeyOver = t.TempDir()
	defer func() { aiKeyOver = ""; aiCallFn = callClaude; aiHist = nil }()
	aiHist = nil
	if r := aiAsk(context.Background(), "안녕"); r["nokey"] != true {
		t.Fatalf("키 없을 때: %v", r)
	}
	if err := writeFileForTest(aiKeyPath(), "sk-ant-test"); err != nil {
		t.Fatal(err)
	}
	var sys string
	aiCallFn = func(ctx context.Context, key, system string, hist []aiTurn) (string, error) {
		sys = system
		if key != "sk-ant-test" || len(hist) == 0 || !hist[len(hist)-1].user {
			t.Fatalf("부르는 값: %q %v", key, hist)
		}
		return "나도 바나나 좋아! [동작:v]", nil
	}
	r := aiAsk(context.Background(), "  바나나 좋아?  ")
	if r["reply"] != "나도 바나나 좋아!" || r["emo"] != "v" || len(aiHist) != 2 {
		t.Fatalf("답: %v hist=%d", r, len(aiHist))
	}
	if !strings.Contains(sys, "우땅이") || !strings.Contains(sys, "[동작:이름]") {
		t.Fatal("시스템 프롬프트")
	}
	posX, posY = 800, ground
	setMode("idle", time.Second)
	n := S.ChatCnt
	aiTick()
	if mode != "v" || S.ChatCnt != n+1 {
		t.Fatalf("바탕화면 동작: %s", mode)
	}
	for i := 0; i < 10; i++ {
		aiAsk(context.Background(), "또")
	}
	if len(aiHist) > aiKeep {
		t.Fatalf("기억 %d", len(aiHist))
	}
	for len(aiInbox) > 0 {
		<-aiInbox
	}
}

func writeFileForTest(p, s string) error { return os.WriteFile(p, []byte(s), 0o600) }
