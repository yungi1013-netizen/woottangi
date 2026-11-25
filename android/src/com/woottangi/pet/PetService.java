package com.woottangi.pet;

import android.annotation.SuppressLint;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.pm.ServiceInfo;
import android.content.res.Configuration;
import android.graphics.Color;
import android.graphics.PixelFormat;
import android.graphics.drawable.GradientDrawable;
import android.os.Build;
import android.os.IBinder;
import android.provider.Settings;
import android.util.TypedValue;
import android.view.Choreographer;
import android.view.GestureDetector;
import android.view.Gravity;
import android.view.HapticFeedbackConstants;
import android.view.MotionEvent;
import android.view.VelocityTracker;
import android.view.View;
import android.view.ViewConfiguration;
import android.view.WindowInsets;
import android.view.WindowManager;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;

import android.graphics.Bitmap;
import android.graphics.BitmapFactory;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.InputStream;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collections;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Random;

/** 화면 위 우땅이: 다른 앱 위에 떠서 돌아다니는 창 + 말풍선 + 메뉴 + 알림 */
public class PetService extends Service implements Choreographer.FrameCallback {
    public static final String ACT_HIDE = "hide", ACT_SHOW = "show", ACT_STOP = "stop", ACT_SETTINGS = "settings";
    public static volatile boolean running = false;
    static PetService inst;
    static final int NOTI_ID = 1;
    static final float[] SIZES = {0.8f, 1f, 1.25f};

    WindowManager wm;
    PrefStore store;
    Pet pet;
    Frames frames, ffr;
    PetView view, fview;
    WindowManager.LayoutParams flp;
    final Map<String, Bitmap> accBmp = new HashMap<>();
    final Map<String, float[][]> anchors = new HashMap<>();
    final Map<String, Bitmap> fxBmp = new HashMap<>();
    /** 화면 위 물건 창 (장난감 · 선물) */
    static class ThingWin { ThingView v; WindowManager.LayoutParams p; }
    final Map<Object, ThingWin> things = new HashMap<>();
    int focusShown = -1;
    float rubDist = 0;
    boolean rubbing = false;
    boolean gaming = false;
    final android.os.Handler h = new android.os.Handler(android.os.Looper.getMainLooper());
    BubbleView bubble;
    View menu;
    WindowManager.LayoutParams lp, blp, mlp;
    boolean shown = false, hidden = false, screenOn = true, looping = false, bubbleOn = false;
    long lastFrame = 0, bubbleUntil = 0, lastSave = 0, lastClock = 0;
    int hour = 12, minute = 0;
    float dp;

    // 손가락
    GestureDetector gd;
    VelocityTracker vt;
    boolean dragging = false;
    float downX, downY, grabX, grabY;
    int slop;

    final BroadcastReceiver screenRx = new BroadcastReceiver() {
        @Override public void onReceive(Context c, Intent i) {
            screenOn = Intent.ACTION_SCREEN_ON.equals(i.getAction());
            if (screenOn) startLoop(); else { save(); stopLoop(); }
        }
    };

    @Override public IBinder onBind(Intent i) { return null; }

    @Override public void onCreate() {
        super.onCreate();
        dp = getResources().getDisplayMetrics().density;
        wm = (WindowManager) getSystemService(WINDOW_SERVICE);
        store = new PrefStore(this);
        frames = new Frames(getAssets(), "frames");
        ffr = new Frames(getAssets(), "friend");
        loadAccessories();
        pet = new Pet(new Random());
        pet.load(store, System.currentTimeMillis());
        slop = ViewConfiguration.get(this).getScaledTouchSlop();
        IntentFilter f = new IntentFilter();
        f.addAction(Intent.ACTION_SCREEN_ON);
        f.addAction(Intent.ACTION_SCREEN_OFF);
        registerReceiver(screenRx, f);
        running = true;
        inst = this;
    }

    void loadAccessories() {
        try (InputStream in = getAssets().open("anchors.json")) {
            java.io.ByteArrayOutputStream bo = new java.io.ByteArrayOutputStream();
            byte[] buf = new byte[8192];
            int r;
            while ((r = in.read(buf)) > 0) bo.write(buf, 0, r);
            JSONObject o = new JSONObject(bo.toString("UTF-8"));
            for (Iterator<String> it = o.keys(); it.hasNext(); ) {
                String k = it.next();
                JSONArray a = o.getJSONArray(k);
                float[][] l = new float[a.length()][];
                for (int i = 0; i < a.length(); i++) {
                    JSONArray v = a.getJSONArray(i);
                    l[i] = new float[]{(float) v.getDouble(0), (float) v.getDouble(1), (float) v.getDouble(2), (float) v.getDouble(3)};
                }
                anchors.put(k, l);
            }
        } catch (Exception ignored) { }
        List<String> accIds = new ArrayList<>(java.util.Arrays.asList(Pet.ITEM_IDS));
        accIds.add("dirt");
        for (String id : accIds) {
            try (InputStream in = getAssets().open("acc/" + id + ".webp")) {
                Bitmap b = BitmapFactory.decodeStream(in);
                if (b != null) accBmp.put(id, b);
            } catch (Exception ignored) { }
        }
    }

    Bitmap fx(String name) {
        if (fxBmp.containsKey(name)) return fxBmp.get(name);
        Bitmap b = null;
        try (InputStream in = getAssets().open("fx/" + name + ".webp")) { b = BitmapFactory.decodeStream(in); } catch (Exception ignored) { }
        fxBmp.put(name, b);
        return b;
    }

    float[] anchorOf(String seq, int i) {
        float[][] l = anchors.get(seq);
        return l == null || i >= l.length ? null : l[i];
    }

    // ───────── 게임 화면 (GameActivity) 과 주고받기 ─────────
    static void gameOpen(boolean open) {
        PetService s = inst;
        if (s == null || s.view == null) return;
        s.gaming = open;
        s.closeMenu();
        int v = open || s.hidden ? View.GONE : View.VISIBLE;
        s.view.setVisibility(v);
        if (s.fview != null) s.fview.setVisibility(v);
        s.thingsVisible(v);
        if (open) { s.hideBubble(); s.stopLoop(); s.save(); } else s.startLoop();
    }

    static boolean canPlay(Context c) {
        PetService s = inst;
        if (s != null) return s.pet.full >= 15;
        Pet p = new Pet(new Random());
        p.load(new PrefStore(c), System.currentTimeMillis());
        return p.full >= 15;
    }

    static void gameScore(Context c, int n, int moves) {
        PetService s = inst;
        if (s != null) { s.pet.gameScore(n, moves); s.save(); return; }
        PrefStore st = new PrefStore(c);
        Pet p = new Pet(new Random());
        p.load(st, System.currentTimeMillis());
        p.gameScore(n, moves);
        p.save(st, System.currentTimeMillis());
    }

    static void towerScore(Context c, int n) {
        PetService s = inst;
        if (s != null) { s.pet.towerScore(n); s.save(); return; }
        PrefStore st = new PrefStore(c);
        Pet p = new Pet(new Random());
        p.load(st, System.currentTimeMillis());
        p.towerScore(n);
        p.save(st, System.currentTimeMillis());
    }

    static void whackScore(Context c, int n) {
        PetService s = inst;
        if (s != null) { s.pet.whackScore(n); s.save(); return; }
        PrefStore st = new PrefStore(c);
        Pet p = new Pet(new Random());
        p.load(st, System.currentTimeMillis());
        p.whackScore(n);
        p.save(st, System.currentTimeMillis());
    }

    /** AI 대화용: 지금 놀고 있는 우땅이 (없으면 저장된 기록으로) */
    static Pet petFor(Context c) {
        PetService s = inst;
        if (s != null) return s.pet;
        Pet p = new Pet(new Random());
        p.load(new PrefStore(c), System.currentTimeMillis());
        return p;
    }

    /** AI 답: 화면 위 우땅이가 말하고 동작 */
    static void aiReply(Context c, String text, String emo) {
        PetService s = inst;
        if (s != null) { s.pet.aiReply(text, emo); s.save(); return; }
        PrefStore st = new PrefStore(c);
        Pet p = new Pet(new Random());
        p.load(st, System.currentTimeMillis());
        p.chatCnt++;
        p.save(st, System.currentTimeMillis());
    }

    /** 꾹 → 이모티콘 → GIF로 저장 (사진 앨범 "우땅이 이모티콘") */
    void saveEmoGifs() {
        pet.say("이모티콘 저장하는 중…", 2);
        android.os.Handler mh = new android.os.Handler(android.os.Looper.getMainLooper());
        new Thread(() -> {
            int n = EmoGif.saveAll(this);
            mh.post(() -> {
                if (n == 0) { pet.say("앗, 저장을 못 했어…", 3); return; }
                pet.gifSaved++;
                pet.playEmo("cheer");
                pet.say("이모티콘 " + n + "개를 저장했어!\n카톡에 보내 봐~", 4);
                save();
                Toast.makeText(this, Build.VERSION.SDK_INT >= 29
                        ? "사진 앨범 '" + EmoGif.ALBUM + "'에 GIF " + n + "개를 저장했어요"
                        : "GIF " + n + "개를 저장했어요 (파일 앱 > Android/data/" + getPackageName() + "/files/Pictures)", Toast.LENGTH_LONG).show();
            });
        }, "emo-gif").start();
    }

    void openChat() {
        pet.say("나랑 얘기하자 " + pet.voc() + "!", 2.5f);
        startActivity(new Intent(this, GameActivity.class).putExtra(GameActivity.EXTRA_GAME, "chat")
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_NO_ANIMATION));
    }

    void openGame(String g, String label) {
        if (!pet.canPlay()) return;
        pet.setMode("laugh", 2.5f);
        pet.say("놀자 " + pet.voc() + "! " + label + "!", 2.5f);
        Intent i = new Intent(this, GameActivity.class).putExtra(GameActivity.EXTRA_GAME, g)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_NO_ANIMATION);
        h.postDelayed(() -> startActivity(i), 700);
    }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        goForeground();
        if (!Settings.canDrawOverlays(this)) { stopSelf(); return START_NOT_STICKY; }
        String a = intent == null ? null : intent.getAction();
        if (ACT_STOP.equals(a)) { stopSelf(); return START_NOT_STICKY; }
        if (ACT_HIDE.equals(a)) { setHidden(true); return START_STICKY; }
        if (ACT_SHOW.equals(a)) { setHidden(false); }
        if (ACT_SETTINGS.equals(a)) applySettings();
        if (!shown) showPet();
        return START_STICKY;
    }

    // ───────── 알림 (서비스가 계속 돌려면 필요) ─────────
    void goForeground() {
        NotificationManager nm = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        if (nm.getNotificationChannel("pet") == null) {
            NotificationChannel ch = new NotificationChannel("pet", "화면 위 우땅이", NotificationManager.IMPORTANCE_LOW);
            ch.setDescription("우땅이가 화면에 나와 있는 동안 보이는 알림");
            ch.setShowBadge(false);
            nm.createNotificationChannel(ch);
        }
        Notification n = buildNoti();
        if (Build.VERSION.SDK_INT >= 34) startForeground(NOTI_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE);
        else startForeground(NOTI_ID, n);
    }

    PendingIntent act(String a, int code) {
        Intent i = new Intent(this, PetService.class).setAction(a);
        return PendingIntent.getService(this, code, i, PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
    }

    @SuppressWarnings("deprecation")
    Notification buildNoti() {
        PendingIntent open = PendingIntent.getActivity(this, 0, new Intent(this, MainActivity.class), PendingIntent.FLAG_IMMUTABLE);
        Notification.Builder b = new Notification.Builder(this, "pet")
                .setSmallIcon(R.drawable.ic_noti)
                .setContentTitle(hidden ? "우땅이가 잠깐 숨어 있어요" : "우땅이가 화면에서 놀고 있어요")
                .setContentText(pet != null && pet.focusing()
                        ? "⏱️ 집중 중 · " + pet.focusLeftMin(System.currentTimeMillis()) + "분 남았어요"
                        : "우땅이를 꾹 누르면 메뉴가 나와요")
                .setContentIntent(open)
                .setOngoing(true)
                .setShowWhen(false)
                .addAction(new Notification.Action.Builder(null, hidden ? "보이기" : "숨기기", act(hidden ? ACT_SHOW : ACT_HIDE, 1)).build())
                .addAction(new Notification.Action.Builder(null, "집에 보내기", act(ACT_STOP, 2)).build());
        return b.build();
    }

    void updateNoti() {
        ((NotificationManager) getSystemService(NOTIFICATION_SERVICE)).notify(NOTI_ID, buildNoti());
    }

    // ───────── 화면 크기 ─────────
    @SuppressWarnings("deprecation")
    void measureScreen() {
        int sw, sh, bottom;
        if (Build.VERSION.SDK_INT >= 30) {
            android.view.WindowMetrics m = wm.getCurrentWindowMetrics();
            sw = m.getBounds().width();
            sh = m.getBounds().height();
            bottom = m.getWindowInsets().getInsetsIgnoringVisibility(WindowInsets.Type.navigationBars()).bottom;
            pet.ceilY = m.getWindowInsets().getInsetsIgnoringVisibility(WindowInsets.Type.statusBars()).top;
        } else {
            android.graphics.Point p = new android.graphics.Point();
            wm.getDefaultDisplay().getRealSize(p);
            sw = p.x; sh = p.y;
            int id = getResources().getIdentifier("navigation_bar_height", "dimen", "android");
            bottom = id > 0 && sh > sw ? getResources().getDimensionPixelSize(id) : 0;
            int sid = getResources().getIdentifier("status_bar_height", "dimen", "android");
            pet.ceilY = sid > 0 ? getResources().getDimensionPixelSize(sid) : 24 * dp;
        }
        pet.scrW = sw;
        pet.scrH = sh;
        pet.floorY = sh - bottom;
        float s = SIZES[Math.max(0, Math.min(2, store.getInt("size", 1)))];
        pet.w = Math.round(110 * dp * s);
        pet.h = Math.round(pet.w * 1.25f);
        pet.dp = dp;
    }

    void applySettings() {
        pet.name = store.getString("name", "");
        pet.speed = store.getInt("speed", 0);
        if (view == null) return;
        measureScreen();
        pet.comeDown();
        lp.width = (int) pet.w;
        lp.height = (int) pet.h;
        pet.x = Math.max(-pet.w * 0.15f, Math.min(pet.scrW - pet.w * 0.85f, pet.x));
        if (!pet.mode.equals("drag")) pet.setMode("fall", 9999);
        safeUpdate(view, lp);
    }

    @Override public void onConfigurationChanged(Configuration c) {
        super.onConfigurationChanged(c);
        if (view == null) return;
        closeMenu();
        pet.comeDown();
        measureScreen();
        pet.x = Math.max(-pet.w * 0.15f, Math.min(pet.scrW - pet.w * 0.85f, pet.x));
        pet.y = Math.min(pet.y, pet.floorY - pet.h);
        if (!pet.mode.equals("drag")) { pet.vx = pet.vy = 0; pet.setMode("fall", 9999); }
    }

    // ───────── 창 ─────────
    WindowManager.LayoutParams params(int w, int h, int extra) {
        WindowManager.LayoutParams p = new WindowManager.LayoutParams(w, h,
                WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY,
                WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE | WindowManager.LayoutParams.FLAG_LAYOUT_NO_LIMITS
                        | WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN | extra,
                PixelFormat.TRANSLUCENT);
        p.gravity = Gravity.TOP | Gravity.LEFT;
        if (Build.VERSION.SDK_INT >= 28)
            p.layoutInDisplayCutoutMode = WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
        return p;
    }

    void safeUpdate(View v, WindowManager.LayoutParams p) {
        try { wm.updateViewLayout(v, p); } catch (Exception ignored) { }
    }

    void showPet() {
        measureScreen();
        pet.name = store.getString("name", "");
        pet.x = pet.scrW * 0.5f - pet.w * 0.5f;
        pet.y = -pet.h * 0.2f;
        pet.vx = pet.vy = 0;
        pet.greeted = false;
        pet.setMode("fall", 9999);

        view = new PetView(this);
        lp = params((int) pet.w, (int) pet.h, 0);
        lp.x = (int) pet.x;
        lp.y = (int) pet.y;
        setupTouch();
        wm.addView(view, lp);

        bubble = new BubbleView(this);
        bubble.setVisibility(View.GONE);
        blp = params(WindowManager.LayoutParams.WRAP_CONTENT, WindowManager.LayoutParams.WRAP_CONTENT,
                WindowManager.LayoutParams.FLAG_NOT_TOUCHABLE);
        wm.addView(bubble, blp);
        shown = true;
        lastFrame = 0;
        startLoop();
    }

    void setHidden(boolean h) {
        if (hidden == h) return;
        hidden = h;
        closeMenu();
        if (view != null) view.setVisibility(h ? View.GONE : View.VISIBLE);
        if (fview != null) fview.setVisibility(h ? View.GONE : View.VISIBLE);
        thingsVisible(h ? View.GONE : View.VISIBLE);
        if (h) hideBubble();
        else if (view != null) { pet.y = -pet.h * 0.2f; pet.vy = 0; pet.setMode("fall", 9999); }
        if (h) stopLoop(); else startLoop();
        updateNoti();
    }

    // ───────── 매 프레임 ─────────
    void startLoop() {
        if (looping || !shown || hidden || gaming || !screenOn) return;
        looping = true;
        lastFrame = 0;
        Choreographer.getInstance().postFrameCallback(this);
    }

    void stopLoop() {
        looping = false;
        Choreographer.getInstance().removeFrameCallback(this);
    }

    @Override public void doFrame(long nanos) {
        if (!looping) return;
        long now = System.currentTimeMillis();
        float dt = lastFrame == 0 ? 1 / 60f : (nanos - lastFrame) / 1e9f;
        lastFrame = nanos;
        if (now - lastClock > 1000) {
            lastClock = now;
            try { // 음악이 나오는지 (소리 내용은 안 들음)
                pet.musicActive = ((android.media.AudioManager) getSystemService(AUDIO_SERVICE)).isMusicActive();
            } catch (Exception e) { pet.musicActive = false; }
            Calendar c = Calendar.getInstance();
            hour = c.get(Calendar.HOUR_OF_DAY);
            minute = c.get(Calendar.MINUTE);
        }
        if (menu == null) pet.tick(dt, now, hour, minute); // 메뉴 열려 있으면 잠깐 멈춤
        draw();
        if (pet.sayText != null) {
            showBubble(pet.sayText, pet.sayDur);
            pet.sayText = null;
        }
        if (pet.toastTitle != null) {
            Toast.makeText(this, pet.toastText, Toast.LENGTH_LONG).show();
            pet.toastTitle = pet.toastText = null;
        }
        if (bubbleOn) {
            if (now > bubbleUntil || pet.mode.equals("drag")) hideBubble();
            else placeBubble();
        }
        if (now - lastSave > 30000) { lastSave = now; save(); }
        int fm = pet.focusing() ? pet.focusLeftMin(now) : 0;
        if (fm != focusShown) { focusShown = fm; updateNoti(); }
        Choreographer.getInstance().postFrameCallback(this);
    }

    void draw() {
        String seq = pet.seqName();
        int n = frames.count(seq);
        if (n == 0) { seq = "idle"; n = frames.count(seq); }
        int fi = pet.frameIndex(n);
        List<Bitmap> acc = Collections.emptyList();
        List<String> ids = pet.wearing(seq);
        if (!ids.isEmpty()) {
            acc = new ArrayList<>();
            for (String id : ids) { Bitmap b = accBmp.get(id); if (b != null) acc.add(b); }
        }
        float clip = Float.isNaN(pet.clipY) ? Float.NaN : pet.clipY - pet.y;
        view.set(frames.get(seq, fi), pet.flip(), pet.rot, acc, anchorOf(seq, fi), clip, pet.q);
        List<float[]> bs = Collections.emptyList();
        if (!pet.bubbles.isEmpty()) {
            bs = new ArrayList<>();
            float life = pet.bathing ? 6 : 1.2f;
            for (float[] b : pet.bubbles) {
                float a = Math.min(1, b[3] / 0.2f) * Math.min(1, (life - b[3]) / Math.min(1, life));
                bs.add(new float[]{b[0], b[1], b[2], a});
            }
        }
        view.setExtras(pet.tucked && seq.equals("sleep") ? fx("blanket") : null, fx("bubble"), bs);
        drawThings();
        int vis = pet.visible() ? View.VISIBLE : View.INVISIBLE;
        if (view.getVisibility() != vis && view.getVisibility() != View.GONE) view.setVisibility(vis);
        drawFriend();
        int nx = Math.round(pet.x), ny = Math.round(pet.y), ww = Math.round(pet.winW()), wh = Math.round(pet.winH());
        if (nx != lp.x || ny != lp.y || ww != lp.width || wh != lp.height) {
            lp.x = nx;
            lp.y = ny;
            lp.width = ww;
            lp.height = wh;
            safeUpdate(view, lp);
        }
    }

    void save() { if (pet != null) pet.save(store, System.currentTimeMillis()); }

    // ───────── 친구 땅콩이 창 ─────────
    void drawFriend() {
        Pet.Friend f = pet.friend;
        if (f == null) {
            if (fview != null) {
                try { wm.removeView(fview); } catch (Exception ignored) { }
                fview = null;
            }
            return;
        }
        if (fview == null) {
            fview = new PetView(this);
            flp = params((int) pet.fw(), (int) pet.fh(), 0);
            flp.x = (int) f.x;
            flp.y = (int) f.y;
            fview.setOnClickListener(v -> pet.tapFriend());
            try { wm.addView(fview, flp); } catch (Exception e) { fview = null; return; }
            // 땅콩이가 우땅이·말풍선을 가리지 않게 다시 맨 위로
            try { wm.removeView(view); wm.addView(view, lp); } catch (Exception ignored) { }
            if (bubble != null) try { wm.removeView(bubble); wm.addView(bubble, blp); } catch (Exception ignored) { }
        }
        String seq = pet.friendSeq();
        int n = ffr.count(seq);
        fview.set(ffr.get(seq, pet.friendFrameIndex(n)), pet.friendFlip(), 0, Collections.<Bitmap>emptyList(), null, Float.NaN, 0);
        int nx = Math.round(f.x), ny = Math.round(f.y), fw = (int) pet.fw(), fh = (int) pet.fh();
        if (nx != flp.x || ny != flp.y || fw != flp.width || fh != flp.height) {
            flp.x = nx; flp.y = ny; flp.width = fw; flp.height = fh;
            safeUpdate(fview, flp);
        }
    }

    // ───────── 말풍선 ─────────
    void showBubble(String t, float sec) {
        if (bubble == null || hidden) return;
        bubble.setText(t);
        bubble.setVisibility(View.VISIBLE);
        bubbleOn = true;
        bubbleUntil = System.currentTimeMillis() + (long) (sec * 1000);
        placeBubble();
    }

    void placeBubble() {
        bubble.measure(View.MeasureSpec.UNSPECIFIED, View.MeasureSpec.UNSPECIFIED);
        int bw = bubble.getMeasuredWidth(), bh = bubble.getMeasuredHeight();
        float cx = pet.x + pet.winW() / 2;
        int x = (int) Math.max(4 * dp, Math.min(pet.scrW - bw - 4 * dp, cx - bw / 2f));
        float top = pet.q == 0 ? pet.y + pet.h * 0.08f : pet.y; // 서 있을 땐 그림 위 여백만큼 내려서
        int y = (int) (top - bh);
        if (y < pet.ceilY + 4 * dp) y = (int) Math.min(pet.y + pet.winH() + 2 * dp, pet.floorY - bh); // 위에 자리 없으면 아래에
        float tx = cx - x;
        if (blp.x != x || blp.y != y || Math.abs(bubble.tailX - tx) > 1) {
            bubble.tailX = tx;
            bubble.invalidate();
            blp.x = x;
            blp.y = y;
            safeUpdate(bubble, blp);
        }
    }

    void hideBubble() {
        bubbleOn = false;
        if (bubble != null) bubble.setVisibility(View.GONE);
    }

    // ───────── 손가락 ─────────
    void setupTouch() {
        gd = new GestureDetector(this, new GestureDetector.SimpleOnGestureListener() {
            @Override public boolean onDown(MotionEvent e) { return true; }

            @Override public boolean onSingleTapConfirmed(MotionEvent e) {
                if (dragging) return false;
                if (menu != null) { closeMenu(); return true; }
                pet.tap();
                return true;
            }

            @Override public boolean onDoubleTap(MotionEvent e) {
                if (dragging) return false;
                view.performHapticFeedback(HapticFeedbackConstants.VIRTUAL_KEY);
                pet.petIt(System.currentTimeMillis());
                return true;
            }

            @Override public void onLongPress(MotionEvent e) {
                if (dragging) return;
                view.performHapticFeedback(HapticFeedbackConstants.LONG_PRESS);
                openMenu();
            }
        });
        view.setOnTouchListener((v, e) -> {
            if (vt == null) vt = VelocityTracker.obtain();
            MotionEvent raw = rawEvent(e);
            vt.addMovement(raw);
            raw.recycle();
            switch (e.getActionMasked()) {
                case MotionEvent.ACTION_DOWN:
                    dragging = false;
                    rubbing = false;
                    rubDist = 0;
                    downX = e.getRawX(); downY = e.getRawY();
                    grabX = downX - pet.x; grabY = downY - pet.y;
                    break;
                case MotionEvent.ACTION_MOVE:
                    if (pet.bathing && !dragging) { // 목욕 중: 끌지 않고 문지르기
                        float d = (float) Math.hypot(e.getRawX() - downX, e.getRawY() - downY);
                        if (!rubbing && d > slop) {
                            rubbing = true;
                            MotionEvent c = MotionEvent.obtain(e);
                            c.setAction(MotionEvent.ACTION_CANCEL);
                            gd.onTouchEvent(c);
                            c.recycle();
                        }
                        if (rubbing) {
                            rubDist += d;
                            downX = e.getRawX(); downY = e.getRawY();
                            if (rubDist > pet.w * 0.12f) {
                                rubDist = 0;
                                pet.rub(e.getX() / Math.max(1, v.getWidth()), e.getY() / Math.max(1, v.getHeight()));
                            }
                        }
                        break;
                    }
                    if (!dragging && Math.hypot(e.getRawX() - downX, e.getRawY() - downY) > slop * 1.5f) {
                        dragging = true;
                        closeMenu();
                        hideBubble();
                        pet.startDrag();
                        grabX = Math.max(pet.w * 0.1f, Math.min(pet.w * 0.9f, downX - pet.x));
                        grabY = Math.max(pet.h * 0.1f, Math.min(pet.h * 0.9f, downY - pet.y));
                        MotionEvent c = MotionEvent.obtain(e);
                        c.setAction(MotionEvent.ACTION_CANCEL);
                        gd.onTouchEvent(c); // 꾹 누르기 취소
                        c.recycle();
                    }
                    if (dragging) pet.dragTo(e.getRawX() - grabX, e.getRawY() - grabY);
                    break;
                case MotionEvent.ACTION_UP:
                case MotionEvent.ACTION_CANCEL:
                    if (dragging) {
                        vt.computeCurrentVelocity(1000);
                        pet.release(vt.getXVelocity(), vt.getYVelocity());
                        dragging = false;
                    }
                    vt.recycle();
                    vt = null;
                    break;
            }
            if (!dragging && !rubbing) gd.onTouchEvent(e);
            return true;
        });
    }

    // ───────── 장난감 · 선물 창 ─────────
    void thingsVisible(int vis) {
        for (ThingWin t : things.values()) t.v.setVisibility(vis);
    }

    ThingWin thingWin(Object key, int size) {
        ThingWin t = new ThingWin();
        t.v = new ThingView(this);
        t.p = params(size, size, 0);
        t.v.setVisibility(hidden || gaming ? View.GONE : View.VISIBLE);
        try { wm.addView(t.v, t.p); } catch (Exception e) { return null; }
        // 우땅이·말풍선이 물건 위에 오게
        try { wm.removeView(view); wm.addView(view, lp); } catch (Exception ignored) { }
        if (bubble != null) try { wm.removeView(bubble); wm.addView(bubble, blp); } catch (Exception ignored) { }
        things.put(key, t);
        return t;
    }

    void placeThing(ThingWin t, float cx, float cy, int size) {
        int nx = Math.round(cx - size / 2f), ny = Math.round(cy - size / 2f);
        if (nx != t.p.x || ny != t.p.y || size != t.p.width) {
            t.p.x = nx; t.p.y = ny; t.p.width = t.p.height = size;
            safeUpdate(t.v, t.p);
        }
    }

    void drawThings() {
        java.util.Set<Object> alive = new java.util.HashSet<>();
        Pet.Toy toy = pet.toy;
        if (toy != null) {
            alive.add(toy);
            int sz = Math.round(toy.size);
            ThingWin t = things.get(toy);
            if (t == null && (t = thingWin(toy, sz)) != null) setupToyTouch(t, toy);
            if (t != null) {
                t.v.set(fx(toy.kind), toy.rot, 1);
                placeThing(t, toy.x, toy.y, sz);
            }
        }
        for (Pet.Gift g : pet.giftList) {
            alive.add(g);
            int sz = Math.round(g.size);
            ThingWin t = things.get(g);
            if (t == null && (t = thingWin(g, sz)) != null) t.v.setOnClickListener(x -> pet.collectGift(g));
            if (t != null) {
                float a = Math.min(1, Math.min(g.age / 0.4f, (40 * 60 - g.age) / 3));
                t.v.set(fx(Pet.GIFT_IDS[g.kind]), g.rot, a);
                placeThing(t, g.x, g.y, sz);
            }
        }
        for (Iterator<Map.Entry<Object, ThingWin>> it = things.entrySet().iterator(); it.hasNext(); ) {
            Map.Entry<Object, ThingWin> en = it.next();
            if (alive.contains(en.getKey())) continue;
            try { wm.removeView(en.getValue().v); } catch (Exception ignored) { }
            it.remove();
        }
    }

    /** 장난감: 톡 = 통 튀기기, 끌기 = 잡고 던지기 */
    @SuppressLint("ClickableViewAccessibility")
    void setupToyTouch(ThingWin t, Pet.Toy toy) {
        final float[] st = new float[4]; // downX, downY, grabX, grabY
        final boolean[] drag = {false};
        final VelocityTracker[] tv = {null};
        t.v.setOnTouchListener((v, e) -> {
            if (tv[0] == null) tv[0] = VelocityTracker.obtain();
            MotionEvent raw = rawEvent(e);
            tv[0].addMovement(raw);
            raw.recycle();
            switch (e.getActionMasked()) {
                case MotionEvent.ACTION_DOWN:
                    drag[0] = false;
                    st[0] = e.getRawX(); st[1] = e.getRawY();
                    st[2] = st[0] - toy.x; st[3] = st[1] - toy.y;
                    break;
                case MotionEvent.ACTION_MOVE:
                    if (!drag[0] && Math.hypot(e.getRawX() - st[0], e.getRawY() - st[1]) > slop) { drag[0] = true; toy.held = true; }
                    if (drag[0]) { toy.x = e.getRawX() - st[2]; toy.y = e.getRawY() - st[3]; }
                    break;
                case MotionEvent.ACTION_UP:
                case MotionEvent.ACTION_CANCEL:
                    if (drag[0]) {
                        tv[0].computeCurrentVelocity(1000);
                        pet.throwToy(tv[0].getXVelocity(), tv[0].getYVelocity());
                    } else if (e.getActionMasked() == MotionEvent.ACTION_UP) pet.pokeToy();
                    toy.held = false;
                    drag[0] = false;
                    tv[0].recycle();
                    tv[0] = null;
                    break;
            }
            return true;
        });
    }

    /** 창이 손가락 따라 움직이니까 화면 좌표로 바꾼 이벤트로 속도 계산 */
    MotionEvent rawEvent(MotionEvent e) {
        MotionEvent c = MotionEvent.obtain(e);
        c.setLocation(e.getRawX(), e.getRawY());
        return c;
    }

    // ───────── 메뉴 (꾹 누르기) ─────────
    TextView item(String t, Runnable r) {
        TextView v = new TextView(this);
        v.setText(t);
        v.setTextSize(TypedValue.COMPLEX_UNIT_SP, 15);
        v.setTextColor(0xFF4A3520);
        int p = (int) (10 * dp);
        v.setPadding((int) (14 * dp), p, (int) (14 * dp), p);
        GradientDrawable g = new GradientDrawable();
        g.setCornerRadius(10 * dp);
        g.setColor(0x00000000);
        v.setBackground(new android.graphics.drawable.RippleDrawable(
                android.content.res.ColorStateList.valueOf(0x33E0B860), g, null));
        v.setOnClickListener(x -> { closeMenu(); r.run(); });
        return v;
    }

    TextView note(String s) {
        TextView t = new TextView(this);
        t.setText(s);
        t.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
        t.setTextColor(0xFF8A6A40);
        t.setPadding((int) (14 * dp), (int) (6 * dp), (int) (14 * dp), (int) (6 * dp));
        return t;
    }

    void openMenu() { openMenu(0); }

    /** page 0 메인 · 1 간식 · 2 놀아주기 · 3 꾸미기 · 4 돌봄 · 집중 · 5 이모티콘 · 6 꾸미기 상점 · 7 코디 저장 */
    void openMenu(int page) {
        closeMenu();
        if (pet.hiding) { pet.tap(); return; } // 숨바꼭질 중에 꾹 = 찾았다
        hideBubble();
        int[] lv = Pet.levelOf(pet.exp);
        LinearLayout m = new LinearLayout(this);
        m.setOrientation(LinearLayout.VERTICAL);
        int p = (int) (8 * dp);
        m.setPadding(p, p, p, p);
        GradientDrawable bg = new GradientDrawable();
        bg.setColor(0xFFFFFDF5);
        bg.setCornerRadius(16 * dp);
        bg.setStroke((int) (2 * dp), 0xFFE0B860);
        m.setBackground(bg);
        m.setElevation(8 * dp);

        TextView head = new TextView(this);
        head.setText("Lv." + lv[0] + " " + Pet.titleOf(lv[0]) + "\n🍌 " + pet.bananas + "   배부름 " + (int) pet.full + "   기분 " + (int) pet.mood);
        head.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
        head.setTextColor(0xFF8A6A40);
        head.setPadding((int) (14 * dp), (int) (6 * dp), (int) (14 * dp), (int) (6 * dp));
        m.addView(head);

        long now = System.currentTimeMillis();
        if (page == 0) {
            m.addView(item("🍌 간식 주기  ›", () -> openMenu(1)));
            m.addView(item("🎮 놀아주기  ›", () -> openMenu(2)));
            m.addView(item("🤚 쓰다듬기", () -> pet.petIt(System.currentTimeMillis())));
            m.addView(item("🫧 돌봄 · 집중  ›", () -> openMenu(4)));
            m.addView(item("😊 이모티콘  ›", () -> openMenu(5)));
            m.addView(item("🎀 꾸미기  ›", () -> openMenu(3)));
            if (pet.perched) m.addView(item("📍 여기를 내 자리로 정하기", () -> pet.setSeat()));
            if (pet.hasSeat()) {
                if (!pet.perched) m.addView(item("🪑 내 자리로 가기", () -> { if (!pet.jumpToSeat()) pet.say("지금은 못 가~ 조금 이따!", 2.5f); }));
                m.addView(item("❌ 내 자리 지우기", () -> pet.clearSeat()));
            }
            if (pet.away()) m.addView(item("⬇️ 내려오기", () -> pet.comeDown()));
            m.addView(item(Pet.today(now).equals(pet.day) ? "📅 출석 도장 (오늘 완료)" : "📅 출석 도장 찍기", () -> pet.attend(System.currentTimeMillis())));
            m.addView(item("💬 우땅이랑 대화하기 (AI)", this::openChat));
            m.addView(item("🔮 오늘의 운세", () -> pet.luck(System.currentTimeMillis())));
            m.addView(item(pet.stay ? "🚶 다시 돌아다니기" : "🧍 여기 가만히 있기", () -> {
                pet.stay = !pet.stay;
                pet.say(pet.stay ? "알았어! 여기 얌전히 있을게" : "야호! 산책 가자~", 2.5f);
            }));
            m.addView(item("🙈 잠깐 숨기", () -> setHidden(true)));
            m.addView(item("🏠 집에 보내기", () -> startService(new Intent(this, PetService.class).setAction(ACT_STOP))));
            m.addView(item("⚙️ 우땅이 앱 열기", () -> startActivity(new Intent(this, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))));
        } else if (page == 1) {
            m.addView(item("🍌 바나나 (1개)", () -> pet.snack(0)));
            m.addView(item("🥛 바나나우유 (3개)", () -> pet.snack(1)));
            m.addView(item("🎂 바나나케이크 (6개)", () -> pet.snack(2)));
            m.addView(item("🍳 바나나 요리하기 (" + Pet.COOK_COST + "개)", () -> pet.cook()));
            m.addView(item("‹ 뒤로", () -> openMenu(0)));
        } else if (page == 2) {
            m.addView(item("🧺 바나나 받기", () -> openGame("game", "바나나 받기")));
            m.addView(item("🃏 짝 맞추기", () -> openGame("match", "짝 맞추기")));
            m.addView(item("🏗️ 바나나 탑 쌓기", () -> openGame("tower", "바나나 탑 쌓기")));
            m.addView(item("🔨 우땅이 두더지 잡기", () -> openGame("whack", "두더지 잡기")));
            m.addView(item("⚽ 공 꺼내기", () -> pet.startToy("ball")));
            m.addView(item("🎈 풍선 꺼내기", () -> pet.startToy("balloon")));
            if (pet.toy != null) m.addView(item("🧺 장난감 치우기", () -> pet.putAwayToy(true)));
            m.addView(item("🙈 숨바꼭질", () -> pet.startHide()));
            m.addView(item("🎰 바나나 뽑기 (10개)", () -> pet.startGacha()));
            m.addView(item("🧗 벽 타기", () -> pet.goClimb(false)));
            m.addView(item("🙃 천장 매달리기", () -> pet.goClimb(true)));
            m.addView(item(pet.level() >= Pet.FRIEND_LEVEL ? "🥜 친구 땅콩이 부르기" : "🥜 친구 땅콩이 (Lv." + Pet.FRIEND_LEVEL + "부터)",
                    () -> pet.startFriend(true, System.currentTimeMillis())));
            m.addView(item("‹ 뒤로", () -> openMenu(0)));
        } else if (page == 5) {
            for (String[] e : Pet.EMOS) m.addView(item(e[1], () -> pet.playEmo(e[0])));
            m.addView(item("🖼️ 이모티콘 GIF로 저장하기", this::saveEmoGifs));
            m.addView(item("‹ 뒤로", () -> openMenu(0)));
        } else if (page == 4) {
            TextView t = new TextView(this);
            t.setText("청결 " + pet.clean() + "%   기운 " + pet.energy() + "%   🏆 업적 " + pet.achCount() + "/" + Pet.ACH.length);
            t.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
            t.setTextColor(0xFF8A6A40);
            t.setPadding((int) (14 * dp), 0, (int) (14 * dp), (int) (6 * dp));
            m.addView(t);
            m.addView(item("🛁 씻기기", () -> pet.startBath()));
            m.addView(item("🛏️ 이불 덮어 재우기", () -> pet.tuckIn()));
            if (pet.focusing()) m.addView(item("⏹️ 집중 그만 (" + pet.focusLeftMin(now) + "분 남음)", () -> pet.stopFocus()));
            else {
                m.addView(item("⏱️ 25분 집중하기", () -> pet.startFocus(25, System.currentTimeMillis())));
                m.addView(item("⏱️ 50분 집중하기", () -> pet.startFocus(50, System.currentTimeMillis())));
            }
            m.addView(item("‹ 뒤로", () -> openMenu(0)));
        } else if (page == 6) {
            m.addView(note("🛍️ 꾸미기 상점 · 사면 바로 써요"));
            for (int i = 0; i < Pet.ITEM_IDS.length; i++) {
                final int k = i;
                String id = Pet.ITEM_IDS[i];
                if (pet.owns(id)) {
                    if (Pet.ITEM_PRICE[i] > 0) m.addView(item("✔ " + Pet.ITEM_NAMES[i] + "  (가지고 있음)", () -> { pet.toggleWear(k); openMenu(6); }));
                } else if (Pet.ITEM_PRICE[i] > 0) {
                    TextView v = item("🛒 " + Pet.ITEM_NAMES[i] + "  (🍌 " + Pet.ITEM_PRICE[i] + ")", () -> { pet.buyItem(k); openMenu(6); });
                    if (pet.bananas < Pet.ITEM_PRICE[i]) v.setAlpha(0.45f);
                    m.addView(v);
                } else {
                    TextView v = item("🎁 " + Pet.ITEM_NAMES[i] + "  (" + Pet.giftFrom(id) + ")", () -> pet.buyItem(k));
                    v.setAlpha(0.45f);
                    m.addView(v);
                }
            }
            m.addView(item("‹ 꾸미기로", () -> openMenu(3)));
        } else if (page == 7) {
            m.addView(note("💾 지금 모습을 어느 칸에 저장할까요?"));
            for (int i = 0; i < Pet.OUTFIT_SLOTS; i++) {
                final int k = i;
                String l = Pet.outfitLabel(pet.outfit(i));
                m.addView(item("코디 " + (i + 1) + (l.isEmpty() ? " (비어 있음)" : " (지금: " + l + " → 바꾸기)"), () -> { pet.saveOutfit(k); openMenu(3); }));
            }
            m.addView(item("‹ 꾸미기로", () -> openMenu(3)));
        } else {
            m.addView(note("🎀 꾸미기 " + pet.ownedCount() + "/" + Pet.ITEM_IDS.length + "  ·  누르면 쓰기/벗기"));
            int have = 0;
            for (int i = 0; i < Pet.ITEM_IDS.length; i++) {
                if (!pet.owns(Pet.ITEM_IDS[i])) continue;
                have++;
                final int k = i;
                boolean on = java.util.Arrays.asList(pet.wear).contains(Pet.ITEM_IDS[i]);
                m.addView(item((on ? "✅ " : "⬜ ") + Pet.ITEM_NAMES[i], () -> { pet.toggleWear(k); openMenu(3); }));
            }
            if (have == 0) m.addView(note("아직 꾸미기 아이템이 없어요.\n상점에서 바나나로 사 보세요!"));
            m.addView(item("🛍️ 꾸미기 상점  ›", () -> openMenu(6)));
            m.addView(item("🎲 오늘의 코디 (아무거나 골라 입기)", () -> pet.randomOutfit()));
            for (int i = 0; i < Pet.OUTFIT_SLOTS; i++) {
                final int k = i;
                String l = Pet.outfitLabel(pet.outfit(i));
                if (!l.isEmpty()) m.addView(item("👗 코디 " + (i + 1) + " 입기: " + l, () -> pet.wearOutfit(k)));
            }
            m.addView(item("💾 지금 모습 코디로 저장  ›", () -> openMenu(7)));
            if (have > 0) m.addView(item("🙅 모두 벗기", () -> pet.takeOffAll()));
            m.addView(item("‹ 뒤로", () -> openMenu(0)));
        }

        mlp = params(WindowManager.LayoutParams.WRAP_CONTENT, WindowManager.LayoutParams.WRAP_CONTENT,
                WindowManager.LayoutParams.FLAG_WATCH_OUTSIDE_TOUCH);
        m.measure(View.MeasureSpec.UNSPECIFIED, View.MeasureSpec.UNSPECIFIED);
        int mw = m.getMeasuredWidth(), mh = m.getMeasuredHeight();
        // 화면보다 길면 (상점 · 이모티콘) 스크롤되게
        View root = m;
        int maxH = (int) (pet.floorY - pet.ceilY - 8 * dp);
        if (maxH > 0 && mh > maxH) {
            android.widget.ScrollView sv = new android.widget.ScrollView(this);
            sv.setBackground(m.getBackground());
            m.setBackground(null);
            sv.setElevation(8 * dp);
            sv.addView(m);
            root = sv;
            mh = maxH;
            mlp.height = maxH;
        }
        float right = pet.x + pet.winW(), left = pet.x;
        int x = right + mw < pet.scrW - 4 * dp ? (int) (right - pet.winW() * 0.1f) : (int) (left - mw + pet.winW() * 0.1f);
        x = (int) Math.max(4 * dp, Math.min(pet.scrW - mw - 4 * dp, x));
        int y = (int) Math.max(pet.ceilY + 4 * dp, Math.min(pet.floorY - mh - 4 * dp, pet.y + pet.winH() * 0.5f - mh / 2f));
        mlp.x = x;
        mlp.y = y;
        root.setOnTouchListener((v, e) -> {
            if (e.getActionMasked() == MotionEvent.ACTION_OUTSIDE) { closeMenu(); return true; }
            return false;
        });
        menu = root;
        try { wm.addView(root, mlp); } catch (Exception ex) { menu = null; }
        if (!pet.mode.equals("sleep") && pet.q == 0 && !pet.bathing && !pet.focusing()) pet.setMode("st_me", 9999);
    }

    void closeMenu() {
        if (menu == null) return;
        try { wm.removeView(menu); } catch (Exception ignored) { }
        menu = null;
        if (pet.mode.equals("st_me") && pet.modeDur > 999) pet.setMode("idle", 1.5f);
    }

    // ───────── 끝 ─────────
    @Override public void onDestroy() {
        running = false;
        stopLoop();
        save();
        try { unregisterReceiver(screenRx); } catch (Exception ignored) { }
        closeMenu();
        inst = null;
        h.removeCallbacksAndMessages(null);
        for (ThingWin t : things.values()) try { wm.removeView(t.v); } catch (Exception ignored) { }
        things.clear();
        for (View v : new View[]{view, bubble, fview}) {
            if (v != null) try { wm.removeView(v); } catch (Exception ignored) { }
        }
        view = null;
        bubble = null;
        fview = null;
        shown = false;
        super.onDestroy();
    }
}
