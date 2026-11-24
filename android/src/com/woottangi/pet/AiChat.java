package com.woottangi.pet;

import android.content.Context;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.SocketTimeoutException;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Calendar;

/**
 * 우땅이랑 AI 대화 (Claude)
 *  - 사용자가 자기 Anthropic API 키를 넣어야 켜짐. 키는 폰의 백업 제외 폴더(no_backup/ai_key.txt)에만 저장
 *  - Gradle 없이 빌드하는 앱이라 Java SDK 대신 HttpURLConnection 으로 Messages API 를 직접 부름
 *  - 대화 기록은 메모리에만 (최근 12마디). 앱이 꺼지면 사라짐
 */
public final class AiChat {
    static final String URL_API = "https://api.anthropic.com/v1/messages";
    static final String MODEL = "claude-opus-5-5";
    static final int MAX_INPUT = 200, KEEP = 12;

    static final ArrayList<String[]> hist = new ArrayList<>(); // {"user"|"assistant", 글}
    static volatile boolean busy = false;

    private AiChat() { }

    static File keyFile(Context c) { return new File(c.getNoBackupFilesDir(), "ai_key.txt"); }

    static String loadKey(Context c) {
        try (InputStream in = new java.io.FileInputStream(keyFile(c))) {
            return new String(readAll(in), StandardCharsets.UTF_8).trim();
        } catch (Exception e) { return ""; }
    }

    static boolean validKey(String k) {
        if (!k.startsWith("sk-ant-") || k.length() >= 400) return false;
        for (char ch : k.toCharArray()) if (" \t\r\n\"'<>".indexOf(ch) >= 0) return false;
        return true;
    }

    /** 키 저장 ("" 이면 지움). 오류 문구 또는 null */
    static String saveKey(Context c, String k) {
        k = k == null ? "" : k.trim();
        if (k.isEmpty()) { keyFile(c).delete(); return null; }
        if (!validKey(k)) return "키 모양이 이상해요. sk-ant- 로 시작하는 키를 붙여 넣어 주세요.";
        try (OutputStream o = new FileOutputStream(keyFile(c))) {
            o.write(k.getBytes(StandardCharsets.UTF_8));
            return null;
        } catch (Exception e) { return "키를 저장하지 못했어요."; }
    }

    /** 한 마디 주고받기 (백그라운드 스레드에서). 결과 JSON 문자열: {reply,emo} 또는 {error[,nokey]} */
    static String ask(Context c, Pet pet, String text) {
        try {
            JSONObject out = new JSONObject();
            text = text == null ? "" : text.trim();
            if (text.isEmpty()) return out.put("error", "할 말을 적어 줘!").toString();
            if (text.codePointCount(0, text.length()) > MAX_INPUT) text = text.substring(0, text.offsetByCodePoints(0, MAX_INPUT));
            String key = loadKey(c);
            if (key.isEmpty()) return out.put("error", "먼저 API 키를 넣어 줘!").put("nokey", true).toString();
            if (busy) return out.put("error", "아직 생각 중이야… 잠깐만!").toString();
            busy = true;
            try {
                Calendar cal = Calendar.getInstance();
                String sys = pet.aiSystem(System.currentTimeMillis(), cal.get(Calendar.HOUR_OF_DAY), cal.get(Calendar.MINUTE));
                ArrayList<String[]> h;
                synchronized (hist) { h = new ArrayList<>(hist); }
                h.add(new String[]{"user", text});
                String raw;
                try { raw = call(key, sys, h); }
                catch (ApiError e) { return out.put("error", errLine(e.code)).toString(); }
                catch (SocketTimeoutException e) { return out.put("error", "생각이 너무 오래 걸렸어… 다시 말해 줄래?").toString(); }
                catch (Exception e) { return out.put("error", "인터넷이 안 되는 것 같아… 연결을 확인해 줘!").toString(); }
                String[] se = Pet.splitEmo(raw);
                synchronized (hist) {
                    hist.clear();
                    hist.addAll(h);
                    hist.add(new String[]{"assistant", se[0]});
                    while (hist.size() > KEEP) hist.remove(0);
                }
                return out.put("reply", se[0]).put("emo", se[1]).toString();
            } finally { busy = false; }
        } catch (Exception e) { return "{\"error\":\"앗, 대답을 못 받았어…\"}"; }
    }

    static final class ApiError extends Exception {
        final int code;
        ApiError(int code) { super("http " + code); this.code = code; }
    }

    static String errLine(int code) {
        switch (code) {
            case 401: case 403: return "API 키가 맞지 않는 것 같아… 키를 다시 확인해 줘!";
            case 402: return "API 크레딧이 모자란 것 같아… 결제 정보를 확인해 줘!";
            case 429: return "말을 너무 많이 했나 봐! 조금 쉬었다가 다시 말 걸어 줘~";
            case 500: case 502: case 503: case 529: return "AI 쪽이 지금 바쁜가 봐… 조금 이따 다시!";
        }
        return "앗, 대답을 못 받았어… (오류 " + code + ")";
    }

    /** Messages API 호출 → 답 글 (동작 꼬리표 포함) */
    static String call(String key, String system, ArrayList<String[]> h) throws Exception {
        JSONArray msgs = new JSONArray();
        for (String[] t : h) msgs.put(new JSONObject().put("role", t[0]).put("content", t[1]));
        JSONObject body = new JSONObject()
                .put("model", MODEL)
                .put("max_tokens", 4000)
                .put("fallbacks", "default") // 거절되면 다른 모델이 대신 답함
                .put("output_config", new JSONObject().put("effort", "low"))
                .put("system", system)
                .put("messages", msgs);
        HttpURLConnection con = (HttpURLConnection) new URL(URL_API).openConnection();
        try {
            con.setRequestMethod("POST");
            con.setConnectTimeout(15000);
            con.setReadTimeout(60000);
            con.setDoOutput(true);
            con.setRequestProperty("content-type", "application/json");
            con.setRequestProperty("x-api-key", key);
            con.setRequestProperty("anthropic-version", "2026-10-21");
            con.setRequestProperty("anthropic-beta", "server-side-fallback-2026-07-01");
            try (OutputStream o = con.getOutputStream()) { o.write(body.toString().getBytes(StandardCharsets.UTF_8)); }
            int code = con.getResponseCode();
            if (code / 100 != 2) throw new ApiError(code);
            JSONObject r;
            try (InputStream in = con.getInputStream()) { r = new JSONObject(new String(readAll(in), StandardCharsets.UTF_8)); }
            if ("refusal".equals(r.optString("stop_reason"))) return "음… 그건 잘 모르겠어! 다른 얘기 하자~ [동작:huh]";
            StringBuilder sb = new StringBuilder();
            JSONArray content = r.optJSONArray("content");
            if (content != null) for (int i = 0; i < content.length(); i++) {
                JSONObject b = content.optJSONObject(i);
                if (b != null && "text".equals(b.optString("type"))) sb.append(b.optString("text"));
            }
            if (sb.toString().trim().isEmpty()) throw new ApiError(0);
            return sb.toString();
        } finally { con.disconnect(); }
    }

    static byte[] readAll(InputStream in) throws java.io.IOException {
        ByteArrayOutputStream b = new ByteArrayOutputStream();
        byte[] buf = new byte[8192];
        for (int n; (n = in.read(buf)) > 0; ) b.write(buf, 0, n);
        return b.toByteArray();
    }
}
