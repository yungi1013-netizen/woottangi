package com.woottangi.pet;

import android.app.Activity;
import android.content.Intent;
import android.graphics.BitmapFactory;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import android.text.InputFilter;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.widget.Button;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import java.io.InputStream;
import java.util.Random;

/** 우땅이 앱 첫 화면: 꺼내기 · 권한 · 이름 · 크기 · 사용법 */
public class MainActivity extends Activity {
    PrefStore store;
    float dp;
    TextView status, permNote;
    Button mainBtn, permBtn;
    EditText nameIn;
    TextView[] sizeBtn = new TextView[3], speedBtn = new TextView[3];
    final Handler h = new Handler(Looper.getMainLooper());

    static final int BROWN = 0xFF4A3520, MUTED = 0xFF8A6A40, CREAM = 0xFFFFF8E6, YELLOW = 0xFFFFD54A, LINE = 0xFFE0B860;

    @Override protected void onCreate(Bundle b) {
        super.onCreate(b);
        store = new PrefStore(this);
        dp = getResources().getDisplayMetrics().density;
        getWindow().setStatusBarColor(CREAM);
        if (Build.VERSION.SDK_INT >= 23) getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR);

        LinearLayout col = new LinearLayout(this);
        col.setOrientation(LinearLayout.VERTICAL);
        col.setPadding(px(20), px(20), px(20), px(28));
        col.setGravity(Gravity.CENTER_HORIZONTAL);

        ImageView img = new ImageView(this);
        try (InputStream in = getAssets().open("frames/wave_2.webp")) {
            img.setImageBitmap(BitmapFactory.decodeStream(in));
        } catch (Exception ignored) { }
        col.addView(img, new LinearLayout.LayoutParams(px(130), px(162)));

        TextView title = text("우땅이", 26, BROWN);
        title.setTypeface(Typeface.DEFAULT_BOLD);
        title.setGravity(Gravity.CENTER);
        col.addView(title);
        status = text("", 14, MUTED);
        status.setGravity(Gravity.CENTER);
        status.setPadding(0, px(4), 0, px(14));
        col.addView(status);

        // 권한
        permNote = text("우땅이가 다른 앱 위에 나오려면 '다른 앱 위에 표시' 권한이 필요해요.\n설정 화면에서 우땅이를 찾아 켜 주세요.", 14, BROWN);
        permNote.setBackground(round(0xFFFFEFC2, 12));
        permNote.setPadding(px(14), px(12), px(14), px(12));
        col.addView(permNote, full());
        permBtn = button("권한 켜러 가기", true);
        permBtn.setOnClickListener(v -> startActivity(new Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                Uri.parse("package:" + getPackageName()))));
        col.addView(permBtn, full(10));

        mainBtn = button("", true);
        mainBtn.setOnClickListener(v -> toggle());
        col.addView(mainBtn, full(10));
        Button bookBtn = button("🏆 업적 · 🎁 선물 모음", false);
        bookBtn.setOnClickListener(v -> showBook());
        col.addView(bookBtn, full(10));
        Button newsBtn = button("📢 업데이트 소식", false);
        newsBtn.setOnClickListener(v -> showNews());
        col.addView(newsBtn, full(10));
        Button chatBtn = button("💬 우땅이랑 대화하기 (AI)", false);
        chatBtn.setOnClickListener(v -> startActivity(new Intent(this, GameActivity.class).putExtra(GameActivity.EXTRA_GAME, "chat")));
        col.addView(chatBtn, full(10));

        // 이름
        col.addView(section("우땅이가 부를 내 이름"));
        LinearLayout nameRow = new LinearLayout(this);
        nameIn = new EditText(this);
        nameIn.setSingleLine();
        nameIn.setHint("예: 민지 (비우면 '주인님')");
        nameIn.setFilters(new InputFilter[]{new InputFilter.LengthFilter(10)});
        nameIn.setText(store.getString("name", ""));
        nameIn.setImeOptions(EditorInfo.IME_ACTION_DONE);
        nameIn.setOnEditorActionListener((v, a, e) -> { saveName(); return true; });
        nameRow.addView(nameIn, new LinearLayout.LayoutParams(0, -2, 1));
        Button save = button("저장", false);
        save.setOnClickListener(v -> saveName());
        nameRow.addView(save, new LinearLayout.LayoutParams(-2, -2));
        col.addView(nameRow, full());

        // 크기 · 속도
        col.addView(section("우땅이 크기"));
        col.addView(choice(sizeBtn, new String[]{"작게", "보통", "크게"}, "size"), full());
        col.addView(section("걷는 속도"));
        col.addView(choice(speedBtn, new String[]{"보통", "느리게", "빠르게"}, "speed"), full());

        // 사용법
        col.addView(section("이렇게 놀아요"));
        TextView help = text("• 톡: 우땅이가 대답해요\n"
                + "• 두 번 톡톡: 쓰다듬기 (기분 업!)\n"
                + "• 꾹 누르기: 메뉴 (간식 · 바나나 요리, 놀아주기, 꾸미기, 출석 도장, 운세)\n"
                + "• 놀아주기: 바나나 받기, 짝 맞추기, 바나나 탑 쌓기, 두더지 잡기, 숨바꼭질, 바나나 뽑기, 친구 땅콩이(Lv.5부터)\n"
                + "• 바나나 요리: 바나나 3개로 요리 8가지를 모아요 (요리 도감)\n"
                + "• 음악을 틀면 우땅이가 춤춰요 · 업적을 깨면 꾸미기 아이템 선물\n"
                + "• 장난감: 공·풍선을 꺼내 주면 우땅이가 차고 머리로 통통! 장난감도 톡 누르거나 던질 수 있어요\n"
                + "• 선물: 우땅이가 가끔 발밑에 두고 가는 선물을 톡 눌러 주워요 (계절마다 달라요)\n"
                + "• 돌봄: 꼬질꼬질해지면 씻기기(손가락으로 문질문질), 졸려 하면 이불 덮어 재우기\n"
                + "• 집중 타이머: 25분·50분 동안 옆에서 조용히 기다렸다가 바나나를 줘요\n"
                + "• 이모티콘: 박수, 화이팅, 사랑해, 흥!, 신나는 춤, 갸우뚱, 브이, 엄지척, 노래, 안 돼요, 꽃 선물, 잘 먹었어요, 뭐해?, 꽃받침 (꾹 → 이모티콘에서 골라 보기)\n"
                + "• 끌어서 휙 던지기: 날아가서 통통 튀어요\n"
                + "• 끌어서 살살 놓기: 벽·천장 가까이면 매달리고, 공중이면 그 자리에 앉아요\n"
                + "• 앉아 있을 때 꾹 → '여기를 내 자리로': 가끔 폴짝 뛰어 올라와요\n"
                + "• 알림창에서 숨기기 / 집에 보내기\n\n"
                + "기록(바나나, 레벨)은 이 폰에만 저장돼요. PC 우땅이와는 따로예요.", 14, BROWN);
        help.setLineSpacing(0, 1.2f);
        help.setBackground(round(0xFFFFFFFF, 12));
        help.setPadding(px(14), px(12), px(14), px(12));
        col.addView(help, full());

        ScrollView sv = new ScrollView(this);
        sv.setBackgroundColor(CREAM);
        sv.setFillViewport(true);
        sv.addView(col);
        setContentView(sv);

        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission("android.permission.POST_NOTIFICATIONS") != 0)
            requestPermissions(new String[]{"android.permission.POST_NOTIFICATIONS"}, 1);

        // 업데이트하고 처음 열면 "새로워진 점"을 한 번 보여 줌 (기록은 그대로)
        if (!newsId().equals(store.getString("news_seen", ""))) h.postDelayed(this::showNews, 400);
    }

    /** 소식 번호: 같은 버전 안에서 소식이 늘면 뒤를 바꿔 한 번 더 보여 줌 (PC ach.go newsID 와 같게) */
    String newsId() { return appVersion() + "-ai"; }

    String appVersion() {
        try { return getPackageManager().getPackageInfo(getPackageName(), 0).versionName; }
        catch (Exception e) { return "?"; }
    }

    /** 이번 버전에서 새로워진 점 (공지사항과 같은 내용) */
    static final String NEWS =
            "💬 우땅이랑 AI 대화하기 (새로!)\n"
            + "   앱 화면이나 꾹 → 메뉴의 '우땅이랑 대화하기'에서 말을 걸면 우땅이가 대답하고 동작도 해요\n"
            + "   내 Anthropic API 키를 넣어야 켜져요 (사용 요금은 내 계정에서 나가요). 키가 없어도 지금처럼 잘 놀아요\n\n"
            +             "😊 새 이모티콘 8개\n"
            + "   브이~ ✌️ · 엄지척 👍 · 노래하기 🎤 · 안 돼요! 🙅 · 꽃 선물 🌼 · 잘 먹었어요 🍌 · 뭐해? 💬 · 꽃받침 🌸\n"
            + "   꾹 → 이모티콘에서 골라 보세요. 바나나를 다 먹으면 가끔 \"잘 먹었어요!\"도 해요\n\n"
            + "🔨 우땅이 두더지 잡기 (꾹 → 놀아주기)\n"
            + "   30초 동안 쏙 나오는 우땅이를 톡! 황금 우땅이는 +3, 꽃 단 땅콩이는 누르면 -2\n\n"
            + "🖼️ 이모티콘 GIF 저장 (꾹 → 이모티콘 → GIF로 저장)\n"
            + "   사진 앨범 '우땅이 이모티콘'에 저장돼서 카톡으로 보낼 수 있어요\n\n"
            + "🏅 업적 보상 꾸미기: 업적을 깨면 특별한 꾸미기 아이템을 받아요\n\n"
            + "🍳 바나나 요리 + 요리 도감: 바나나로 여러 가지 요리를 만들어 모아요\n\n"
            + "🎶 음악이 나오면 우땅이가 춤을 춰요\n\n"
            + "💾 바나나 · 레벨 · 꾸미기 · 업적 기록은 업데이트해도 그대로예요";

    void showNews() {
        if (isFinishing()) return;
        store.put("news_seen", newsId());
        store.commit();
        TextView t = text(NEWS, 14, BROWN);
        t.setLineSpacing(0, 1.2f);
        t.setPadding(px(20), px(12), px(20), px(8));
        ScrollView sv = new ScrollView(this);
        sv.addView(t);
        new android.app.AlertDialog.Builder(this).setTitle("📢 우땅이 " + appVersion() + " 업데이트 소식")
                .setView(sv).setPositiveButton("좋아요!", null).show();
    }

    @Override protected void onResume() {
        super.onResume();
        refresh();
        h.postDelayed(tick, 1000);
    }

    @Override protected void onPause() {
        super.onPause();
        h.removeCallbacks(tick);
    }

    final Runnable tick = new Runnable() {
        @Override public void run() { refresh(); h.postDelayed(this, 1500); }
    };

    boolean canOverlay() { return Settings.canDrawOverlays(this); }

    void refresh() {
        boolean ok = canOverlay();
        permNote.setVisibility(ok ? View.GONE : View.VISIBLE);
        permBtn.setVisibility(ok ? View.GONE : View.VISIBLE);
        mainBtn.setEnabled(ok);
        mainBtn.setAlpha(ok ? 1 : 0.45f);
        mainBtn.setText(PetService.running ? "🏠 우땅이 집에 보내기" : "🍌 우땅이 꺼내기");
        Pet p = new Pet(new Random());
        p.load(store, System.currentTimeMillis());
        int[] lv = Pet.levelOf(p.exp);
        status.setText("Lv." + lv[0] + " " + Pet.titleOf(lv[0]) + "  ·  🍌 " + p.bananas
                + "\n함께한 지 " + p.daysTogether(System.currentTimeMillis()) + "일째  ·  꾸미기 " + p.ownedCount() + "/" + Pet.ITEM_IDS.length
                + "\n🏆 업적 " + p.achCount() + "/" + Pet.ACH.length + "  ·  🎁 선물 " + p.giftKindsGot() + "/" + Pet.GIFT_IDS.length
                + (PetService.running ? "  ·  지금 화면에 나와 있어요" : ""));
        int s = store.getInt("size", 1), sp = store.getInt("speed", 0);
        for (int i = 0; i < 3; i++) { mark(sizeBtn[i], i == s); mark(speedBtn[i], i == sp); }
    }

    /** 업적 · 선물 모음 · 기록 */
    void showBook() {
        Pet p = new Pet(new Random());
        p.load(store, System.currentTimeMillis());
        LinearLayout col = new LinearLayout(this);
        col.setOrientation(LinearLayout.VERTICAL);
        col.setPadding(px(18), px(8), px(18), px(8));

        col.addView(section("기록"));
        col.addView(text("간식 " + p.snackCnt + "번 · 쓰다듬기 " + p.petTotal + "번 · 게임 " + p.playCnt + "판\n"
                + "목욕 " + p.bathCnt + "번 · 이불 " + p.tuckCnt + "번 · 집중 " + p.focusCnt + "번 (" + p.focusMin + "분)\n"
                + "장난감 놀이 " + p.toyCnt + "번 · 주운 선물 " + p.giftCnt + "개\n"
                + "바나나 받기 최고 " + p.best + "개 · 바나나 탑 최고 " + p.towerBest + "층\n"
                + "두더지 잡기 최고 " + p.whackBest + "점 · 요리 " + p.cookCnt + "번 · 음악 춤 " + p.danceCnt + "번", 14, BROWN), full());

        col.addView(section("🍳 요리 도감  " + p.cookedKinds() + "/" + Pet.COOK_IDS.length));
        LinearLayout row = null;
        for (int i = 0; i < Pet.COOK_IDS.length; i++) {
            if (i % 4 == 0) { row = new LinearLayout(this); col.addView(row, full(4)); }
            int n = p.cookedOf(Pet.COOK_IDS[i]);
            TextView t = text(n > 0 ? Pet.COOK_ICONS[i] + "\n" + Pet.COOK_NAMES[i] + "\n" + n + "번" : "❔\n???\n아직", 11, n > 0 ? BROWN : MUTED);
            t.setGravity(Gravity.CENTER);
            t.setPadding(px(2), px(6), px(2), px(6));
            GradientDrawable g = round(n > 0 ? 0xFFFFFFFF : 0xFFF1E9D8, 10);
            g.setStroke(px(1), LINE);
            t.setBackground(g);
            LinearLayout.LayoutParams l = new LinearLayout.LayoutParams(0, -1, 1);
            if (i % 4 > 0) l.leftMargin = px(4);
            row.addView(t, l);
        }
        col.addView(text("꾹 → 간식 주기 → 바나나 요리하기 (바나나 " + Pet.COOK_COST + "개)", 12, MUTED), full());

        col.addView(section("🎁 선물 모음  " + p.giftKindsGot() + "/" + Pet.GIFT_IDS.length));
        LinearLayout grid = null;
        for (int i = 0; i < Pet.GIFT_IDS.length; i++) {
            if (i % 5 == 0) { grid = new LinearLayout(this); col.addView(grid, full(4)); }
            int n = p.giftCount(Pet.GIFT_IDS[i]);
            LinearLayout cell = new LinearLayout(this);
            cell.setOrientation(LinearLayout.VERTICAL);
            cell.setGravity(Gravity.CENTER_HORIZONTAL);
            cell.setPadding(0, px(6), 0, px(6));
            GradientDrawable g = round(n > 0 ? 0xFFFFFFFF : 0xFFF1E9D8, 10);
            g.setStroke(px(1), LINE);
            cell.setBackground(g);
            ImageView im = new ImageView(this);
            try (InputStream in = getAssets().open("fx/" + Pet.GIFT_IDS[i] + ".webp")) {
                im.setImageBitmap(BitmapFactory.decodeStream(in));
            } catch (Exception ignored) { }
            if (n == 0) { im.setColorFilter(0xFF000000); im.setAlpha(0.25f); }
            cell.addView(im, new LinearLayout.LayoutParams(px(34), px(34)));
            TextView t = text(n > 0 ? Pet.GIFT_NAMES[i] + "\n" + n + "개" : "???\n아직", 11, n > 0 ? BROWN : MUTED);
            t.setGravity(Gravity.CENTER);
            cell.addView(t);
            LinearLayout.LayoutParams l = new LinearLayout.LayoutParams(0, -2, 1);
            if (i % 5 > 0) l.leftMargin = px(4);
            grid.addView(cell, l);
        }

        col.addView(section("🏆 업적  " + p.achCount() + "/" + Pet.ACH.length));
        for (String[] a : Pet.ACH) {
            boolean got = p.hasAch(a[0]);
            TextView t = text((got ? a[1] + "  " : "🔒  ") + a[2] + "  ·  🍌" + a[4] + "\n      " + a[3], 13, got ? BROWN : MUTED);
            t.setPadding(px(4), px(5), 0, px(5));
            col.addView(t, full());
        }

        ScrollView sv = new ScrollView(this);
        sv.addView(col);
        new android.app.AlertDialog.Builder(this).setTitle("우땅이 수첩").setView(sv).setPositiveButton("닫기", null).show();
    }

    void toggle() {
        if (!canOverlay()) return;
        Intent i = new Intent(this, PetService.class);
        if (PetService.running) {
            i.setAction(PetService.ACT_STOP);
            startService(i);
        } else {
            i.setAction(PetService.ACT_SHOW);
            startForegroundService(i);
            moveTaskToBack(true); // 홈 화면에서 우땅이를 바로 볼 수 있게
        }
        h.postDelayed(this::refresh, 400);
    }

    void saveName() {
        String n = nameIn.getText().toString().trim();
        store.put("name", n);
        store.commit();
        nameIn.clearFocus();
        ((InputMethodManager) getSystemService(INPUT_METHOD_SERVICE)).hideSoftInputFromWindow(nameIn.getWindowToken(), 0);
        android.widget.Toast.makeText(this, n.isEmpty() ? "이제 '주인님'이라고 부를게요" : "이제 '" + n + "'(이)라고 부를게요", android.widget.Toast.LENGTH_SHORT).show();
        pushSettings();
    }

    void pushSettings() {
        if (PetService.running) startService(new Intent(this, PetService.class).setAction(PetService.ACT_SETTINGS));
    }

    // ───────── 작은 화면 조각들 ─────────
    int px(float v) { return Math.round(v * dp); }

    TextView text(String t, int sp, int color) {
        TextView v = new TextView(this);
        v.setText(t);
        v.setTextSize(TypedValue.COMPLEX_UNIT_SP, sp);
        v.setTextColor(color);
        return v;
    }

    TextView section(String t) {
        TextView v = text(t, 15, BROWN);
        v.setTypeface(Typeface.DEFAULT_BOLD);
        v.setPadding(px(2), px(18), 0, px(6));
        LinearLayout.LayoutParams l = full();
        v.setLayoutParams(l);
        return v;
    }

    GradientDrawable round(int color, int r) {
        GradientDrawable g = new GradientDrawable();
        g.setColor(color);
        g.setCornerRadius(px(r));
        return g;
    }

    Button button(String t, boolean primary) {
        Button b = new Button(this);
        b.setText(t);
        b.setAllCaps(false);
        b.setTextSize(TypedValue.COMPLEX_UNIT_SP, primary ? 17 : 15);
        b.setTypeface(Typeface.DEFAULT_BOLD);
        b.setTextColor(BROWN);
        GradientDrawable g = round(primary ? YELLOW : 0xFFFFFFFF, 14);
        g.setStroke(px(2), LINE);
        b.setBackground(g);
        b.setPadding(px(16), px(primary ? 14 : 8), px(16), px(primary ? 14 : 8));
        b.setStateListAnimator(null);
        return b;
    }

    LinearLayout.LayoutParams full() { return new LinearLayout.LayoutParams(-1, -2); }

    LinearLayout.LayoutParams full(int top) {
        LinearLayout.LayoutParams l = full();
        l.topMargin = px(top);
        return l;
    }

    LinearLayout choice(TextView[] out, String[] labels, String key) {
        LinearLayout row = new LinearLayout(this);
        for (int i = 0; i < labels.length; i++) {
            TextView v = text(labels[i], 15, BROWN);
            v.setGravity(Gravity.CENTER);
            v.setPadding(0, px(10), 0, px(10));
            final int idx = i;
            v.setOnClickListener(x -> {
                store.put(key, idx);
                store.commit();
                refresh();
                pushSettings();
            });
            LinearLayout.LayoutParams l = new LinearLayout.LayoutParams(0, -2, 1);
            if (i > 0) l.leftMargin = px(8);
            row.addView(v, l);
            out[i] = v;
        }
        return row;
    }

    void mark(TextView v, boolean on) {
        GradientDrawable g = round(on ? YELLOW : 0xFFFFFFFF, 12);
        g.setStroke(px(on ? 2 : 1), on ? LINE : 0xFFEADBB8);
        v.setBackground(g);
        v.setTypeface(on ? Typeface.DEFAULT_BOLD : Typeface.DEFAULT);
    }
}
