package com.woottangi.pet;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.view.View;
import android.webkit.JavascriptInterface;
import android.webkit.WebChromeClient;
import android.webkit.WebSettings;
import android.webkit.WebView;

import java.util.Random;

/** 바나나 받기 · 짝 맞추기 (PC 와 같은 게임 페이지를 앱 안에서 엶) */
public class GameActivity extends Activity {
    public static final String EXTRA_GAME = "game"; // "game" · "match" · "tower"
    WebView web;
    final Handler h = new Handler(Looper.getMainLooper());
    String token = "";
    boolean sent = false, chat = false;

    @SuppressLint({"SetJavaScriptEnabled", "AddJavascriptInterface"})
    @Override protected void onCreate(Bundle b) {
        super.onCreate(b);
        getWindow().setStatusBarColor(0xFFC6ECFA);
        web = new WebView(this);
        WebSettings s = web.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setAllowFileAccess(false);
        s.setMediaPlaybackRequiresUserGesture(false);
        web.setOverScrollMode(View.OVER_SCROLL_NEVER);
        web.addJavascriptInterface(new Bridge(), "Woo");
        String g = getIntent().getStringExtra(EXTRA_GAME);
        if (!"match".equals(g) && !"tower".equals(g) && !"whack".equals(g) && !"chat".equals(g)) g = "game";
        chat = "chat".equals(g);
        if (chat) { getWindow().setStatusBarColor(0xFFFFE27A); web.setWebChromeClient(new WebChromeClient()); } // confirm() 창
        web.loadUrl("file:///android_asset/" + g + ".html");
        setContentView(web);
    }

    @Override protected void onResume() {
        super.onResume();
        if (!chat) PetService.gameOpen(true); // 게임하는 동안 화면 위 우땅이는 잠깐 숨김 (대화 중엔 같이 있음)
    }

    @Override protected void onPause() {
        super.onPause();
        if (!chat) PetService.gameOpen(false);
    }

    @Override protected void onDestroy() {
        if (web != null) { web.removeJavascriptInterface("Woo"); web.destroy(); }
        super.onDestroy();
    }

    /** 게임 페이지가 부르는 것: fetch("/start"), fetch("/score?..."), window.close() */
    class Bridge {
        @JavascriptInterface public String call(String url) {
            Uri u = Uri.parse("http://x" + (url.startsWith("/") ? url : "/" + url));
            String path = u.getPath() == null ? "" : u.getPath();
            if (path.equals("/start")) {
                if (!PetService.canPlay(GameActivity.this)) return "";
                token = Long.toString(new Random().nextLong() & Long.MAX_VALUE, 36);
                sent = false;
                return token;
            }
            if (path.equals("/score")) {
                String t = u.getQueryParameter("t");
                if (token.isEmpty() || t == null || !t.equals(token) || sent) return "";
                token = "";
                sent = true;
                int n = num(u.getQueryParameter("n")), m = "match".equals(u.getQueryParameter("g")) ? num(u.getQueryParameter("m")) : 0;
                if (n < 0 || "match".equals(u.getQueryParameter("g")) && m < 8) return "";
                if ("tower".equals(u.getQueryParameter("g"))) h.post(() -> PetService.towerScore(GameActivity.this, n));
                else if ("whack".equals(u.getQueryParameter("g"))) h.post(() -> PetService.whackScore(GameActivity.this, n));
                else h.post(() -> PetService.gameScore(GameActivity.this, n, m));
                return "ok";
            }
            return "";
        }

        /** 대화 페이지: /ai/status · /ai/key (바로 답) */
        @JavascriptInterface public String ai(String path, String body) {
            if (!chat) return "{}";
            try {
                if ("/ai/status".equals(path)) {
                    Pet p = PetService.petFor(GameActivity.this);
                    return new org.json.JSONObject().put("key", !AiChat.loadKey(GameActivity.this).isEmpty())
                            .put("owner", p.name.equals("주인님") ? "" : p.name).toString();
                }
                if ("/ai/key".equals(path)) {
                    String err = AiChat.saveKey(GameActivity.this, body);
                    return err == null ? "{\"ok\":true}" : new org.json.JSONObject().put("error", err).toString();
                }
            } catch (Exception ignored) { }
            return "{}";
        }

        /** 대화 페이지: /ai/chat (인터넷이라 따로 돌리고 끝나면 페이지에 알려 줌) */
        @JavascriptInterface public void ask(String body) {
            if (!chat) return;
            new Thread(() -> {
                String text = "";
                try { text = new org.json.JSONObject(body).optString("text"); } catch (Exception ignored) { }
                Pet p = PetService.petFor(GameActivity.this);
                String r = AiChat.ask(GameActivity.this, p, text);
                h.post(() -> {
                    try {
                        org.json.JSONObject o = new org.json.JSONObject(r);
                        if (o.has("reply")) PetService.aiReply(GameActivity.this, o.optString("reply"), o.optString("emo"));
                    } catch (Exception ignored) { }
                    if (web != null) web.evaluateJavascript("window.__wooAi(" + r + ")", null);
                });
            }).start();
        }

        @JavascriptInterface public void close() { h.post(GameActivity.this::finish); }
    }

    static int num(String s) {
        try { return Integer.parseInt(s); } catch (Exception e) { return -1; }
    }
}
