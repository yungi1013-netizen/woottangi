package com.woottangi.pet;

import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;
import java.util.Locale;
import java.util.Random;

/**
 * 우땅이 두뇌: 상태(바나나·배부름·기분·레벨) · 움직임(걷기·끌기·던지기·떨어지기) · 대사.
 * 안드로이드 코드를 쓰지 않아서 PC 에서도 테스트할 수 있음 (android/test/PetTest.java).
 * 좌표는 화면 픽셀, (x, y) = 우땅이 창의 왼쪽 위.
 */
public class Pet {

    /** 저장소 (안드로이드는 SharedPreferences, 테스트는 HashMap) */
    public interface Store {
        int getInt(String k, int d);
        float getFloat(String k, float d);
        long getLong(String k, long d);
        String getString(String k, String d);
        void put(String k, Object v);
        void commit();
    }

    // ───────── 상태 (PC 우땅이와 같은 규칙) ─────────
    public int bananas = 10, exp = 0, lvShown = 1, streak = 0, attendTotal = 0;
    public int petCnt = 0, petTotal = 0, snackCnt = 0;
    public float full = 70, mood = 70;
    public String name = "", day = "", petDay = "", since = "", luckDay = "";
    public long last = 0;
    // 놀이 기록 · 꾸미기
    public int playCnt = 0, best = 0, matchCnt = 0, bestMatch = 0, hideCnt = 0, gachaCnt = 0, friendCnt = 0;
    public String friendDay = "", owned = ""; // owned: "starpin,halo"
    public final String[] wear = {"", "", ""}; // 머리 · 얼굴 · 목
    public long nextFriendAt = 0;

    public static final String[] SNACK_NAMES = {"바나나", "바나나우유", "바나나케이크"};
    public static final int[] SNACK_PRICE = {1, 3, 6};
    static final float[] SNACK_FULL = {25, 18, 30}, SNACK_MOOD = {4, 12, 25};
    static final int[] SNACK_EXP = {3, 5, 8};
    static final String[] SNACK_ANIM = {"eat", "eat_milk", "eat_cake"};
    static final int EXP_ATTEND = 10, EXP_PET = 2, EXP_LUCK = 3, EXP_IDLE = 1, MAX_LV = 50;

    // ───────── 화면 · 움직임 ─────────
    public float scrW = 1080, scrH = 2340, floorY = 2200; // floorY: 서 있을 때 발(창 아래) 위치
    public float ceilY = 0; // 천장 (상태 표시줄 바로 아래)
    public float w = 300, h = 375, dp = 2.75f;
    public float x, y, vx, vy, dir = 1, rot = 0;
    public String mode = "fall";
    public float modeT = 0, modeDur = 9999;
    public boolean stay = false, greeted = false, thrown = false;
    public int speed = 0; // 0 보통 1 느리게 2 빠르게
    int bounces = 0;
    float maxFall = 0;
    public final Random rng;

    // 깜빡임 · 대사 · 시간
    float blinkIn = 3, blinkLeft = 0;
    float talkIn = 40, minuteAcc = 0, idleExpAcc = 0;
    String lastSlot = "";
    public String sayText = null; // 서비스가 꺼내 가면 null
    public float sayDur = 0;
    int pendingLv = 0;
    public String toastTitle = null, toastText = null; // 알림으로 보낼 것

    public Pet(Random rng) { this.rng = rng; }

    // ───────── 저장 · 불러오기 ─────────
    public void load(Store s, long now) {
        bananas = s.getInt("bananas", 10);
        full = s.getFloat("full", 70);
        mood = s.getFloat("mood", 70);
        exp = s.getInt("exp", 0);
        lvShown = s.getInt("lvShown", 1);
        streak = s.getInt("streak", 0);
        attendTotal = s.getInt("attendTotal", 0);
        petCnt = s.getInt("petCnt", 0);
        petTotal = s.getInt("petTotal", 0);
        snackCnt = s.getInt("snackCnt", 0);
        name = s.getString("name", "");
        day = s.getString("day", "");
        petDay = s.getString("petDay", "");
        since = s.getString("since", "");
        luckDay = s.getString("luckDay", "");
        speed = s.getInt("speed", 0);
        last = s.getLong("last", 0);
        playCnt = s.getInt("playCnt", 0); best = s.getInt("best", 0);
        matchCnt = s.getInt("matchCnt", 0); bestMatch = s.getInt("bestMatch", 0);
        hideCnt = s.getInt("hideCnt", 0); gachaCnt = s.getInt("gachaCnt", 0); friendCnt = s.getInt("friendCnt", 0);
        friendDay = s.getString("friendDay", ""); owned = s.getString("owned", "");
        String[] wr = s.getString("wear", ",,").split(",", -1);
        for (int i = 0; i < 3; i++) wear[i] = i < wr.length ? wr[i] : "";
        nextFriendAt = s.getLong("nextFriendAt", 0);
        seatFX = s.getFloat("seatFX", -1);
        seatFY = s.getFloat("seatFY", -1);
        loadFun(s);
        if (since.isEmpty()) since = today(now);
        if (last > 0) { // 떨어져 있던 동안 (최대 12시간)
            float hrs = Math.min((now - last) / 3600000f, 12);
            if (hrs > 0) {
                full = clamp(full - hrs * 6);
                mood = clamp(mood - hrs * 4);
                careAway(hrs);
            }
        }
        lvShown = Math.max(lvShown, 1);
    }

    public void save(Store s, long now) {
        last = now;
        s.put("bananas", bananas); s.put("full", full); s.put("mood", mood); s.put("exp", exp);
        s.put("lvShown", lvShown); s.put("streak", streak); s.put("attendTotal", attendTotal);
        s.put("petCnt", petCnt); s.put("petTotal", petTotal); s.put("snackCnt", snackCnt);
        s.put("name", name); s.put("day", day); s.put("petDay", petDay); s.put("since", since);
        s.put("luckDay", luckDay); s.put("speed", speed); s.put("last", last);
        s.put("playCnt", playCnt); s.put("best", best); s.put("matchCnt", matchCnt); s.put("bestMatch", bestMatch);
        s.put("hideCnt", hideCnt); s.put("gachaCnt", gachaCnt); s.put("friendCnt", friendCnt);
        s.put("friendDay", friendDay); s.put("owned", owned);
        s.put("wear", wear[0] + "," + wear[1] + "," + wear[2]);
        s.put("nextFriendAt", nextFriendAt);
        s.put("seatFX", seatFX);
        s.put("seatFY", seatFY);
        saveFun(s);
        s.commit();
    }

    static float clamp(float v) { return Math.max(0, Math.min(100, v)); }

    public static String today(long now) { return new SimpleDateFormat("yyyy-MM-dd", Locale.KOREA).format(new Date(now)); }

    // ───────── 레벨 ─────────
    static int expNeed(int l) { return 40 + 20 * (l - 1); }

    /** {레벨, 이번 레벨에서 쌓은 양, 필요한 양} */
    public static int[] levelOf(int e) {
        int lv = 1;
        while (lv < MAX_LV && e >= expNeed(lv)) { e -= expNeed(lv); lv++; }
        return lv == MAX_LV ? new int[]{lv, 0, 0} : new int[]{lv, e, expNeed(lv)};
    }

    public int level() { return levelOf(exp)[0]; }

    static final int[] TITLE_FROM = {1, 3, 5, 8, 12, 16, 20, 30, 50};
    static final String[] TITLES = {"아기 우땅이", "꼬마 우땅이", "장난꾸러기 우땅이", "단짝 우땅이", "베프 우땅이",
            "바나나 박사 우땅이", "전설의 우땅이", "우주 최강 우땅이", "영원한 가족 우땅이"};

    public static String titleOf(int lv) {
        String t = TITLES[0];
        for (int i = 0; i < TITLE_FROM.length; i++) if (lv >= TITLE_FROM[i]) t = TITLES[i];
        return t;
    }

    public void addExp(int n) {
        if (n <= 0) return;
        exp += n;
        int lv = level();
        if (lv > lvShown) pendingLv = lv;
    }

    public int daysTogether(long now) {
        try {
            Date a = new SimpleDateFormat("yyyy-MM-dd", Locale.KOREA).parse(since);
            Date b = new SimpleDateFormat("yyyy-MM-dd", Locale.KOREA).parse(today(now));
            return (int) Math.round((b.getTime() - a.getTime()) / 86400000.0) + 1;
        } catch (Exception e) {
            return 1;
        }
    }

    // ───────── 말하기 ─────────
    public void say(String t, float sec) { sayText = t; sayDur = sec; }

    /** 반말 호칭: 받침 있으면 "아", 없으면 "야" */
    public String voc() {
        if (name.isEmpty() || name.equals("주인님")) return "주인님";
        char c = name.charAt(name.length() - 1);
        if (c >= 0xAC00 && c <= 0xD7A3) return name + ((c - 0xAC00) % 28 == 0 ? "야" : "아");
        return name;
    }

    String pick(String... a) { return a[rng.nextInt(a.length)]; }

    float rd(float a, float b) { return a + rng.nextFloat() * (b - a); }

    // ───────── 동작 ─────────
    public void setMode(String m, float sec) {
        boolean surfMode = m.equals("climb") || m.equals("cling");
        if (q != 0 && !surfMode && !m.equals("fall") && !m.equals("drag")) leaveSurface();
        if (perched && (m.equals("fall") || m.equals("drag") || m.equals("hide") || surfMode)) perched = false;
        mode = m; modeT = 0; modeDur = sec;
    }

    float groundTop() { return perched ? perchY : floorY - h; } // 서 있을 때 창 y

    public float speedMul() {
        float m = new float[]{1, 0.6f, 1.5f}[Math.max(0, Math.min(2, speed))];
        if (mood < 30 || full < 20) m *= 0.6f;
        return m;
    }

    /** 다음 동작 고르기 (PC nextChoice 와 같은 흐름) */
    void next(int hour) {
        if (hiding || gachaLeft > 0) { modeT = 0; modeDur = 3600; return; }
        if (friend != null) { setMode("idle", 3600); return; }
        switch (mode) {
            case "sleep": if (tucked) { wakeTucked(); return; } setMode("sit", rd(2, 4)); return;
            case "sit": {
                float r = rng.nextFloat();
                if (r < .35f) setMode("sleep", rd(12, 25));
                else if (r < .55f) setMode("sit", rd(4, 8));
                else setMode("idle", rd(1.5f, 3));
                return;
            }
            case "st_cry": setMode("idle", rd(2, 3)); return;
            case "st_bow": setMode("wave", rd(2.5f, 3.2f)); return;
            case "eat": case "eat_milk": case "eat_cake": // 다 먹고 가끔 "잘 먹었어요!"
                if (rng.nextFloat() < .35f && playEmo("yum")) return;
                setMode("idle", rd(1.5f, 3)); return;
            case "laugh": case "shy":
            case "st_surprise": case "st_lounge": case "st_me": case "wave":
            case "clap": case "cheer": case "heart": case "angry": case "dance": case "huh":
            case "v": case "thumb": case "sing": case "no": case "flower": case "yum": case "what": case "cute":
                setMode("idle", rd(1.5f, 3)); return;
        }
        if (perched) { // 자리 위: 앉고 쉬고 졸기만
            float r = rng.nextFloat();
            if (r < .2f) setMode("sleep", rd(10, 20));
            else if (r < .55f) setMode("sit", rd(5, 10));
            else if (r < .7f) setMode("st_lounge", rd(5, 8));
            else if (r < .78f) setMode("wave", rd(2.5f, 3.2f));
            else setMode("idle", rd(2, 4));
            return;
        }
        if (focusing()) { // 집중하는 동안: 옆에 조용히 앉아 있기
            setMode(rng.nextFloat() < .7f ? "sit" : "idle", rd(20, 40));
            return;
        }
        if (toyActive() && petFree()) { // 장난감 놀이 중엔 장난감만 봄
            if (mode.equals("idle")) modeDur = modeT + 2; else setMode("idle", 2);
            return;
        }
        if (wantClimb) { dir = x + w / 2 < scrW / 2 ? -1 : 1; setMode("walk", 30); return; } // 벽 타러 가는 중
        if (stay) { setMode("idle", rd(3, 5)); return; }
        if (hasSeat() && rng.nextFloat() < .07f && jumpToSeat()) return;
        if (hour < 6 && rng.nextFloat() < .45f) { // 새벽엔 졸려
            if (rng.nextFloat() < .6f) setMode("sleep", rd(15, 30)); else setMode("sit", rd(8, 14));
            return;
        }
        if (sleepy() && rng.nextFloat() < .3f) { setMode(rng.nextFloat() < .6f ? "sleep" : "sit", rd(15, 30)); return; } // 졸려
        if ((mood < 30 || full < 20) && rng.nextFloat() < .35f) { // 시무룩
            setMode(rng.nextBoolean() ? "st_lounge" : "sit", rd(6, 10));
            return;
        }
        float r = rng.nextFloat();
        if (mode.equals("walk") && r < .45f) setMode("idle", rd(2, 4));
        else if (r < .08f) setMode("st_lounge", rd(5, 9));
        else if (r < .12f) setMode("wave", rd(2.5f, 3.5f));
        else if (r < .18f) setMode("laugh", rd(1.6f, 2.6f));
        else if (r < .21f && mood >= 60) setMode("dance", rd(3.2f, 4.2f)); // 기분 좋으면 가끔 춤
        else if (r < .23f) setMode("huh", rd(2.4f, 3));
        else if (r < .25f) setMode(new String[]{"v", "sing", "what"}[rng.nextInt(3)], rd(2.6f, 3.4f)); // 가끔 브이 · 노래 · 뭐해?
        else if (r < .32f) setMode("sit", rd(6, 12));
        else {
            if (rng.nextFloat() < .4f) dir = -dir;
            setMode("walk", rd(4, 9));
        }
    }

    /** 매 프레임 (dt 초). now = 지금 시각(ms), hour/min = 지금 시·분 */
    public void tick(float dt, long now, int hour, int minute) {
        dt = Math.min(dt, 0.1f);
        nowMs = now;
        hourNow = hour;
        modeT += dt;
        switch (mode) {
            case "drag":
                rot = (float) Math.sin(modeT * 6) * 0.22f;
                break;
            case "fall":
                fall(dt);
                break;
            case "hide":
                rot = 0;
                break;
            case "climb":
            case "cling":
                rot = 0;
                climbTick(dt);
                break;
            default:
                rot = 0;
                if (perched) {
                    perchLeft -= dt;
                    if (perchLeft <= 0 && !mode.startsWith("eat")) { hopDown(); break; }
                }
                if (y < groundTop() - 1) { bounces = 0; maxFall = 0; mode = "fall"; break; }
                y = groundTop();
                if (mode.equals("walk")) walk(dt);
                if (modeT >= modeDur) next(hour);
        }
        // 깜빡임
        if (blinkLeft > 0) blinkLeft -= dt;
        blinkIn -= dt;
        if (blinkIn <= 0) { blinkLeft = 0.14f; blinkIn = rd(2.5f, 6); }
        // 1분마다 조금씩 배고프고 심심해짐
        minuteAcc += dt;
        if (minuteAcc >= 60) {
            minuteAcc = 0;
            boolean sl = mode.equals("sleep");
            full = clamp(full - (sl ? .3f : 1));
            mood = clamp(mood - (sl ? .2f : .7f));
            careMinute();
        }
        idleExpAcc += dt;
        if (idleExpAcc >= 15 * 60) { idleExpAcc = 0; addExp(EXP_IDLE); }
        if (hiding) hideTick(dt);
        if (gachaLeft > 0) gachaTick(dt);
        friendTick(dt, now);
        bathTick(dt);
        tuckTick(dt);
        focusTick(now);
        toyTick(dt);
        giftTick(dt);
        achTick(dt);
        musicTick(dt);
        if (!hiding && gachaLeft <= 0 && friend == null) talk(dt, now, hour, minute);
    }

    void walk(float dt) {
        x += dir * w * 0.28f * dt * speedMul() * (toyActive() ? 2.4f : 1);
        float lo = -w * 0.15f, hi = scrW - w * 0.85f;
        boolean canClimb = !stay && friend == null && !hiding && gachaLeft <= 0 && !toyActive() && !focusing() && !bathing;
        if (x < lo) {
            x = lo;
            if (canClimb && (wantClimb || rng.nextFloat() < CLIMB_CHANCE)) { toWall(1, -1); return; }
            dir = 1;
        }
        if (x > hi) {
            x = hi;
            if (canClimb && (wantClimb || rng.nextFloat() < CLIMB_CHANCE)) { toWall(3, -1); return; }
            dir = -1;
        }
    }

    void fall(float dt) {
        float g = 1500 * dp;
        vy += g * dt;
        if (!seatJump) vx *= 1 - 0.5f * dt;
        x += vx * dt;
        y += vy * dt;
        maxFall = Math.max(maxFall, vy);
        rot = Math.max(-0.5f, Math.min(0.5f, vx / (900 * dp)));
        float lo = -w * 0.15f, hi = scrW - w * 0.85f;
        if (x < lo) { x = lo; vx = -vx * 0.6f; dir = 1; }
        if (x > hi) { x = hi; vx = -vx * 0.6f; dir = -1; }
        if (y < ceilY - h * 0.25f) {
            if (thrown && !seatJump && rng.nextFloat() < 0.5f) { toCeiling(x, vx >= 0 ? 1 : -1); setMode("cling", rd(1.5f, 3)); say(pick("천장 잡았다!", "매달리기 성공!"), 2.5f); return; }
            y = ceilY - h * 0.25f; vy = -vy * 0.4f;
        }
        if (seatJump && vy > 0 && y >= seatY()) {
            x = seatX(); y = seatY(); vx = vy = 0; seatJump = false; thrown = false; maxFall = 0; rot = 0; bounces = 0;
            perch(y, rd(60, 180));
            if (rng.nextFloat() < .6f) say(pick("내 자리 도착!", "여기가 제일 좋아~", "폴짝! 내 자리!"), 2.5f);
            return;
        }
        if (y >= groundTop()) { y = groundTop(); seatJump = false; land(); }
    }

    void land() {
        if (vy > 650 * dp && bounces < 2) { // 세게 떨어지면 통통
            vy = -vy * 0.35f; vx *= 0.7f; bounces++; y -= 1;
            return;
        }
        boolean hard = thrown && maxFall > 1000 * dp;
        vx = vy = 0; bounces = 0; thrown = false; maxFall = 0; rot = 0;
        if (greetAfterFall != null) {
            greeted = true;
            setMode("laugh", rd(2.2f, 3));
            say(greetAfterFall, 4);
            greetAfterFall = null;
        } else if (!greeted) {
            greeted = true;
            setMode("st_bow", 2.2f);
            say(name.isEmpty() ? "안녕! 나는 우땅이야!" : "안녕 " + voc() + "! 우땅이 왔어!", 4);
        } else if (hard) {
            setMode("st_surprise", 1.8f);
            say(pick("어지러워~ 빙글빙글", "우와 날았다!", "살살 던져 줘~", "착지 성공! 히히"), 2.5f);
        } else setMode("idle", rd(1, 2));
    }

    // ───────── 손가락 ─────────
    public void startDrag() {
        if (hiding) endHide(false);
        wantClimb = wantCeil = false;
        seatJump = false;
        setMode("drag", 9999);
        vx = vy = 0;
    }

    public void dragTo(float nx, float ny) { x = nx; y = ny; }

    /** 놓기: vx, vy = 손가락 속도 (px/s) */
    public void release(float rvx, float rvy) {
        if (Math.hypot(rvx, rvy) < 160 * dp && placeGently()) return;
        float lim = 2600 * dp;
        vx = Math.max(-lim, Math.min(lim, rvx));
        vy = Math.max(-lim, Math.min(lim, rvy));
        thrown = Math.hypot(vx, vy) > 450 * dp;
        if (!thrown) vx = vy = 0;
        bounces = 0; maxFall = 0;
        setMode("fall", 9999);
    }

    /** 한 번 톡 */
    public void tap() {
        if (hiding) { endHide(true); return; }
        if (gachaLeft > 0 || bathing) return;
        if (mode.equals("drag") || mode.equals("fall")) return;
        if (tucked) { pokeTucked(); return; }
        if (focusing()) { setMode("st_me", 1.8f); say("집중 " + focusLeftMin(nowMs == 0 ? System.currentTimeMillis() : nowMs) + "분 남았어! 화이팅~", 2.5f); return; }
        if (q != 0) { letGo(); say(pick("으앗! 떨어진다~", "깜짝이야! 으아아~"), 2.2f); return; }
        if (mode.equals("sleep")) { setMode("sit", rd(2, 4)); blinkLeft = 0.3f; say("하암… 잘 잤다!", 2.5f); return; }
        if (clickSpam(nowMs == 0 ? System.currentTimeMillis() : nowMs)) { // 마구 누르면 삐짐
            setMode("angry", 2.6f);
            say(pick("흥! 그만 눌러~", "아야! 살살 해 줘! 흥!", "뿌우… 삐졌어!"), 2.5f);
            return;
        }
        if (rng.nextFloat() < .3f) { setMode("st_surprise", 1.8f); say(pick("깜짝이야!", "앗! 놀랐잖아~", "헉!"), 2); return; }
        setMode("st_me", 2.2f);
        if (!name.isEmpty() && rng.nextFloat() < .5f) say(pick("응! " + voc() + "!", "불렀어?", "나 여기 있어!"), 2.2f);
    }

    /** 두 번 톡 = 쓰다듬기 */
    public void petIt(long now) {
        if (mode.equals("drag") || mode.equals("fall")) return;
        String t = today(now);
        if (!t.equals(petDay)) { petDay = t; petCnt = 0; }
        petTotal++;
        if (petCnt < 10) { petCnt++; mood = clamp(mood + 6); addExp(EXP_PET); }
        String n = voc();
        if (rng.nextFloat() < .35f) { // 가끔은 하트 꼭 안고 사랑해
            setMode("heart", 3.2f);
            say(pick(n + " 사랑해 ♥", "이만큼 좋아해! ♥", "두근두근… ♥"), 3);
            talkIn = rd(40, 90);
            return;
        }
        setMode("shy", rd(2.2f, 3));
        say(pick("히히 " + n + " 좋아!", "쓰다듬어줘서 고마워 " + n + "~", "간지러워 " + n + "~", "우땅우땅 ♥"), 2.5f);
        talkIn = rd(40, 90);
    }

    public boolean snack(int i) {
        if (full >= 95) { setMode("st_lounge", 3); say(pick("배불러… 더는 못 먹어~", "꺼억~ 나중에 줘!"), 3); return false; }
        if (bananas < SNACK_PRICE[i]) { setMode("st_cry", 2.5f); say("바나나가 모자라… 놀아주면 모을 수 있어!", 3.5f); return false; }
        bananas -= SNACK_PRICE[i];
        snackCnt++;
        full = clamp(full + SNACK_FULL[i]);
        mood = clamp(mood + SNACK_MOOD[i]);
        addExp(SNACK_EXP[i]);
        setMode(SNACK_ANIM[i], rd(5, 6.5f));
        String n = voc();
        String[][] lines = {
                {"냠냠! 바나나 최고!", "와 바나나다! 냠냠", "고마워 " + n + "! 냠냠"},
                {"꿀꺽꿀꺽~ 바나나우유 맛있다!", "우유 좋아! 고마워 " + n + "!"},
                {"케이크다!! " + n + " 사랑해 ♥", "생일도 아닌데 케이크! 히히 냠냠"}};
        say(pick(lines[i]), 3.5f);
        talkIn = rd(60, 120);
        return true;
    }

    /** 출석: 오늘 처음이면 바나나 (7일마다 보너스). 받은 수 (이미 했으면 0) */
    public int attend(long now) {
        String t = today(now);
        if (t.equals(day)) {
            setMode("st_me", 2.2f);
            say("연속 출석 " + streak + "일째!\n7일마다 바나나 15개 보너스~", 4);
            return 0;
        }
        Calendar c = Calendar.getInstance();
        c.setTimeInMillis(now);
        c.add(Calendar.DATE, -1);
        String yest = today(c.getTimeInMillis());
        streak = yest.equals(day) ? streak + 1 : 1;
        day = t;
        attendTotal++;
        int gain = 5 + (streak % 7 == 0 ? 15 : 0);
        bananas += gain;
        addExp(EXP_ATTEND);
        setMode("st_me", 2.2f);
        say("오늘 출석 도장 쾅!\n바나나 " + gain + "개 받았어!" + (streak % 7 == 0 ? "\n7일 연속 보너스!" : ""), 5);
        return gain;
    }

    static final String[][] LUCK = {
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
    };

    public void luck(long now) {
        String t = today(now);
        int hsh = 7;
        for (char c : (t + name).toCharArray()) hsh = hsh * 31 + c;
        String[] l = LUCK[Integer.remainderUnsigned(hsh, LUCK.length)];
        if (!t.equals(luckDay)) { luckDay = t; mood = clamp(mood + 5); addExp(EXP_LUCK); }
        setMode("wave", 3);
        say(voc() + " 오늘의 운세!\n" + l[0] + "\n" + l[1], 8);
        talkIn = rd(60, 120);
    }

    // ───────── 혼잣말 ─────────
    static final int[][] SLOT_MIN = {{0, 360}, {540, 600}, {690, 720}, {780, 810}, {1050, 1080}, {1140, 1200}, {1320, 1440}};

    String[] slotLines(int i, String n) {
        switch (i) {
            case 0: return new String[]{"벌써 새벽이야… " + n + " 이제 자자", "하암… " + n + " 안 졸려?"};
            case 1: return new String[]{n + " 좋은 아침!", n + " 아침 먹었어?"};
            case 2: return new String[]{n + " 점심 뭐 먹어?", "배고프다… " + n + " 밥 먹으러 가자!", n + " 나도 밥줘"};
            case 3: return new String[]{n + " 졸리면 기지개 쭉~", n + " 오후도 힘내자!"};
            case 4: return new String[]{n + " 오늘도 수고했어!", n + " 집에 갈 준비하자!"};
            case 5: return new String[]{n + " 저녁 먹었어?", n + " 오늘 하루 어땠어?"};
            default: return new String[]{n + " 너무 늦었어. 이제 자자~", "하암… " + n + " 졸리지?"};
        }
    }

    int slotAt(int hour, int minute) {
        int m = hour * 60 + minute;
        for (int i = 0; i < SLOT_MIN.length; i++) if (m >= SLOT_MIN[i][0] && m < SLOT_MIN[i][1]) return i;
        return -1;
    }

    String needLine() {
        String n = voc();
        String cl = careLine();
        if (cl != null) return cl;
        if (full < 25) return pick(n + " 배고파… 바나나 줘~", "꼬르륵… " + n + " 간식!");
        if (mood < 25) return pick(n + " 심심해…", n + " 나랑 놀아줘…", "시무룩…");
        return null;
    }

    public String randomLine(int hour, int minute) {
        String l = needLine();
        if (l != null) return l;
        String n = voc();
        if (rng.nextFloat() < .15f) {
            int lv = level();
            return pick(n + "랑 함께한 지 벌써 " + daysTogether(System.currentTimeMillis()) + "일째야!", "나 이제 " + titleOf(lv) + "야! 에헴", n + " 오늘도 고마워 ♥");
        }
        int s = slotAt(hour, minute);
        if (s >= 0 && rng.nextFloat() < .6f) return pick(slotLines(s, n));
        return pick(n + " 화이팅!", n + " 물 한 잔 마셔~", n + " 사랑해 ♥", n + " 뭐 해?",
                n + " 잠깐 쉬었다 해~", "우땅! " + (name.isEmpty() ? "주인님" : name) + " 최고!", n + " 허리 쭉 펴!", n + " 나 여기 있어~");
    }

    void talk(float dt, long now, int hour, int minute) {
        if (mode.equals("drag") || mode.equals("fall") || sayText != null) return;
        // 레벨 업 축하
        if (pendingLv > 0 && !mode.equals("sleep") && !mode.startsWith("eat")) {
            celebrate(pendingLv);
            return;
        }
        // 시간대에 막 들어오면 한 번
        int s = slotAt(hour, minute);
        if (s >= 0 && !mode.equals("sleep")) {
            String key = today(now) + "#" + s;
            if (!key.equals(lastSlot)) {
                lastSlot = key;
                say(pick(slotLines(s, voc())), 5);
                if (mode.equals("idle") || mode.equals("walk")) setMode("wave", rd(2.5f, 3.5f));
                talkIn = rd(60, 150);
                return;
            }
        }
        talkIn -= dt;
        if (talkIn > 0 || mode.equals("sleep")) return;
        talkIn = rd(45, 110);
        String l = randomLine(hour, minute);
        say(l, 4);
        String e = emoForLine(l);
        if (e != null && (mode.equals("idle") || mode.equals("walk")) && q == 0) setMode(e, emoSec(e));
    }

    void celebrate(int lv) {
        int from = lvShown, gift = 0;
        boolean newTitle = false;
        for (int l = from + 1; l <= lv; l++) {
            gift += 2 * l;
            for (int f : TITLE_FROM) if (f == l) newTitle = true;
        }
        pendingLv = 0;
        lvShown = lv;
        bananas += gift;
        mood = clamp(mood + 10);
        setMode("laugh", rd(3, 3.6f));
        say(newTitle ? "레벨 업! Lv." + lv + "!\n이제 나는 '" + titleOf(lv) + "'야!\n바나나 " + gift + "개 선물!"
                : "레벨 업! Lv." + lv + " 됐어!\n바나나 " + gift + "개 선물!", 6);
        toastTitle = "우땅이 레벨 업!";
        toastText = "Lv." + lv + " " + titleOf(lv) + " · 바나나 " + gift + "개 선물";
        talkIn = rd(60, 120);
    }

    // ───────── 놀아주기: 게임 점수 (바나나 받기 · 짝 맞추기) ─────────
    String greetAfterFall = null;

    /** 놀 수 있는지 (배고프면 못 놀아) — 못 놀면 말하고 false */
    public boolean canPlay() {
        if (busy()) { say("지금은 못 해~ 조금 이따 놀자!", 2.5f); return false; }
        if (focusing()) { say("집중 끝나고 놀자! 쉿~", 2.5f); return false; }
        if (full < 15) { setMode("st_lounge", 4); say("배고파서 못 놀겠어… 간식 먼저!", 3); return false; }
        return true;
    }

    public boolean busy() { return bathing || hiding || gachaLeft > 0 || mode.equals("drag") || mode.equals("fall"); }

    /** moves > 0 이면 짝 맞추기 (뒤집은 횟수로 바나나를 다시 계산) */
    public int gameScore(int n, int moves) {
        if (moves > 0) {
            if (moves < 8) return 0;
            n = Math.max(3, Math.min(18, 26 - moves));
        } else n = Math.max(0, Math.min(80, n));
        bananas += n;
        mood = clamp(mood + Math.min(35, 10 + n * 2));
        full = clamp(full - 8);
        playCnt++;
        boolean isBest = false;
        if (moves > 0) {
            matchCnt++;
            if (bestMatch == 0 || moves < bestMatch) { bestMatch = moves; isBest = true; }
        } else if (n > best) { best = n; isBest = true; }
        addExp(Math.min(20, 3 + n / 4));
        if (n == 0) { setMode("st_cry", 2.5f); say("하나도 못 받았어… 다시 하자!", 3); return 0; }
        setMode("laugh", rd(2.5f, 3.5f));
        String s = moves > 0 ? moves + "번 만에 다 맞혔어!\n바나나 " + n + "개 고마워 " + voc() + "!"
                : "바나나 " + n + "개 받았다! 재밌었어 " + voc() + "!";
        if (isBest) s += "\n최고 기록이야!";
        say(s, 4.5f);
        talkIn = rd(60, 120);
        return n;
    }

    // ───────── 숨바꼭질 ─────────
    public boolean hiding = false;
    public int hideStage = 0;      // 0 숨을 거야(보임) 1 숨는 중(안 보임) 2 숨어 있음
    float hideLeft = 0, hideT = 0, hideBaseX = 0, hideSide = 0;
    boolean hideHint = false;
    public float clipY = Float.NaN; // 이 화면 y 아래는 안 그림 (바닥 아래에서 빼꼼)
    static final float HIDE_LIMIT = 60;

    public boolean visible() { return !(hiding && hideStage == 1); }

    public void startHide() {
        if (busy() || friend != null) { say("지금은 못 해~ 조금 이따 놀자!", 2.5f); return; }
        comeDown();
        pauseToy();
        tucked = false;
        hiding = true; hideStage = 0; hideHint = false; hideLeft = 2.5f;
        setMode("st_me", 3600);
        say("숨바꼭질! 내가 숨을게~\n찾으면 나를 톡 눌러 줘!", 2.5f);
    }

    void hideTick(float dt) {
        hideLeft -= dt;
        if (hideStage == 0) {
            if (hideLeft <= 0) { hideStage = 1; hideLeft = 1.5f; }
            return;
        }
        if (hideStage == 1) {
            if (hideLeft <= 0) { pickHideSpot(); hideStage = 2; hideT = 0; setMode("hide", 3600); }
            return;
        }
        hideT += dt;
        float peek = (float) Math.max(0, Math.sin(hideT * 1.3)) * 0.12f * w;
        if (hideSide != 0) x = hideBaseX + hideSide * peek;
        if (!hideHint && hideT > 35) { hideHint = true; say("여기야 여기~", 2.5f); }
        if (hideT > HIDE_LIMIT) endHide(false);
    }

    /** 화면 왼쪽 끝 · 오른쪽 끝 · 바닥 아래에서 빼꼼 */
    void pickHideSpot() {
        float r = rng.nextFloat();
        float top = scrH * 0.12f, gy = groundTop();
        hideSide = 0;
        clipY = Float.NaN;
        if (r < 0.37f) { hideSide = 1; x = -w * 0.5f; y = rd(top, gy); dir = 1; }
        else if (r < 0.74f) { hideSide = -1; x = scrW - w * 0.5f; y = rd(top, gy); dir = -1; }
        else { x = rd(w * 0.1f, scrW - w * 1.1f); y = floorY - h * 0.45f; clipY = floorY; }
        hideBaseX = x;
    }

    /** found: 찾았음 / 아니면 시간이 다 돼서 나옴 */
    public void endHide(boolean found) {
        if (!hiding) return;
        int stage = hideStage;
        hiding = false;
        clipY = Float.NaN;
        if (stage < 2) { setMode("idle", rd(1, 2)); return; }
        float el = hideT;
        x = Math.max(-w * 0.1f, Math.min(scrW - w * 0.9f, x));
        y = Math.min(y, groundTop());
        vx = 0; vy = -300 * dp; thrown = false; maxFall = 0; bounces = 0;
        setMode("fall", 9999);
        if (!found) { greetAfterFall = "여기 있었지롱~ 다음엔 꼭 찾아 줘!"; return; }
        int gain = el < 15 ? 8 : el < 30 ? 6 : 4;
        bananas += gain;
        mood = clamp(mood + 15);
        hideCnt++;
        playCnt++;
        addExp(5);
        greetAfterFall = "들켰다! 히히\n" + (int) el + "초 만에 찾았어! 바나나 " + gain + "개!";
    }

    // ───────── 바나나 뽑기 ─────────
    public static final int GACHA_COST = 10;
    public static final String[] GACHA_POOL = {"starpin", "halo", "heartglasses", "scarf"};
    public static final String[] GACHA_NAMES = {"반짝 별 핀", "천사 링", "하트 안경", "줄무늬 목도리"};
    static final int[] GACHA_SLOT = {0, 0, 1, 2};
    static final String[] GACHA_LINE = {"반짝반짝 작은 별~ 나 빛나지?", "오늘은 천사 우땅이야~ 착하게 살게!",
            "세상이 다 하트로 보여~ 사랑해!", "포근포근 따뜻해~"};
    float gachaLeft = 0;

    /** 꾸미기 전부 (PC shop.go 와 같은 그림·이름). 가격 0 = 팔지 않음 (뽑기 · 업적 선물) */
    public static final String[] ITEM_IDS = {
            "ribbon", "flower", "sprout", "partyhat", "catears", "bunnyears", "strawhat", "beanie", "santa", "crown",
            "glasses", "sunglasses", "blush", "mustache", "bowtie", "bandana", "pearl",
            "starpin", "halo", "heartglasses", "scarf",
            "rainbow", "bananaband", "medal", "headphones", "flowercrown", "chefhat", "tiara"};
    public static final String[] ITEM_NAMES = {
            "빨간 리본", "꽃 핀", "새싹", "파티 고깔모자", "고양이 귀", "토끼 귀", "밀짚모자", "털모자", "산타 모자", "왕관",
            "동그란 안경", "선글라스", "발그레 볼터치", "신사 콧수염", "나비넥타이", "반다나", "진주 목걸이",
            "반짝 별 핀", "천사 링", "하트 안경", "줄무늬 목도리",
            "무지개 핀", "바나나 머리띠", "금메달", "헤드폰", "꽃 화관", "요리사 모자", "반짝 티아라"};
    public static final int[] ITEM_PRICE = {8, 10, 12, 15, 18, 18, 22, 25, 25, 40, 15, 20, 10, 12, 15, 14, 30,
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0};
    static final int[] ITEM_SLOT = {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2,
            0, 0, 1, 2, 0, 0, 2, 0, 0, 0, 0};
    public static final String[] ITEM_LINE = {
            "리본 예뻐? 히히", "꽃 냄새 난다~ 향기롭지?", "머리에서 새싹이 났어! 쑥쑥 클게!", "오늘은 파티다! 빵빠레~",
            "야옹~ 아니 우땅~ 헷갈리지?", "깡총깡총! 귀가 쫑긋~", "여름 바다 가고 싶다~ 출발!", "따뜻하다~ 겨울 준비 끝!",
            "호호호! 선물 줄게~", "나 이제 우땅이 왕이야!", "똑똑해 보여? 에헴!", "멋있지? 우땅 스웩~",
            "히히 볼이 발그레~ 부끄러워", "에헴! 우땅 신사라네~", "오늘은 신사 우땅이야!", "반다나 두르니까 모험가 같지?",
            "반짝반짝 진주! 우아하지?",
            "반짝반짝 작은 별~ 나 빛나지?", "오늘은 천사 우땅이야~ 착하게 살게!", "세상이 다 하트로 보여~ 사랑해!", "포근포근 따뜻해~",
            "무지개처럼 반짝반짝!", "100일 동안 고마워! 바나나 왕이다!", "놀이 대장 금메달! 반짝반짝~", "쿵짝쿵짝~ 집중 모드 ON!",
            "쓰담쓰담 고마워~ 꽃 화관 예쁘지?", "오늘은 우땅이 셰프! 맛있게 해 줄게~", "옷장 부자 우땅이! 공주님 같지?"};

    /** 팔지 않는 아이템을 받는 방법 (메뉴 표시용) */
    public static String giftFrom(String id) {
        switch (id) {
            case "starpin": case "halo": case "heartglasses": case "scarf": return "바나나 뽑기";
            case "rainbow": return "누적 출석 30일 선물";
            case "bananaband": return "누적 출석 100일 선물";
            case "medal": return "업적 '놀이 대장' 선물";
            case "headphones": return "업적 '집중왕' 선물";
            case "flowercrown": return "업적 '쓰담쓰담 장인' 선물";
            case "chefhat": return "업적 '바나나 요리사' 선물";
            case "tiara": return "업적 '옷장 부자' 선물";
        }
        return "";
    }

    public static int itemIndex(String id) {
        for (int i = 0; i < ITEM_IDS.length; i++) if (ITEM_IDS[i].equals(id)) return i;
        return -1;
    }

    /** 아이템 받기 (이미 있으면 false). 받으면 바로 씀 */
    public boolean giveItem(String id) {
        int i = itemIndex(id);
        if (i < 0 || owns(id)) return false;
        owned = owned.isEmpty() ? id : owned + "," + id;
        wear[ITEM_SLOT[i]] = id;
        return true;
    }

    public int gachaOwned() {
        int n = 0;
        for (String id : GACHA_POOL) if (owns(id)) n++;
        return n;
    }

    public boolean owns(String id) {
        for (String o : owned.split(",")) if (o.equals(id)) return true;
        return false;
    }

    public int ownedCount() {
        int n = 0;
        for (String id : ITEM_IDS) if (owns(id)) n++;
        return n;
    }

    public void startGacha() {
        if (bananas < GACHA_COST) { say("바나나가 " + GACHA_COST + "개 있어야 해~", 2.5f); return; }
        if (busy()) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        bananas -= GACHA_COST;
        pauseToy();
        gachaCnt++;
        gachaLeft = 1.8f;
        setMode("st_me", 3600);
        say("두구두구두구…", 1.8f);
    }

    void gachaTick(float dt) {
        gachaLeft -= dt;
        if (gachaLeft > 0) return;
        gachaLeft = 0;
        java.util.List<Integer> left = new java.util.ArrayList<>();
        for (int i = 0; i < GACHA_POOL.length; i++) if (!owns(GACHA_POOL[i])) left.add(i);
        float r = rng.nextFloat();
        if (r < 0.15f && !left.isEmpty()) {
            int i = left.get(rng.nextInt(left.size()));
            owned = owned.isEmpty() ? GACHA_POOL[i] : owned + "," + GACHA_POOL[i];
            wear[GACHA_SLOT[i]] = GACHA_POOL[i];
            setMode("laugh", rd(3, 3.6f));
            say("우와아! 희귀 아이템!\n'" + GACHA_NAMES[i] + "' 나왔다!\n바로 써 볼게!", 5);
            toastTitle = "바나나 뽑기";
            toastText = GACHA_NAMES[i] + " 당첨!";
        } else if (r < 0.20f) {
            bananas += 50;
            setMode("laugh", rd(3, 3.6f));
            say("대박!!! 바나나 50개!", 4);
        } else if (r < 0.35f) {
            addExp(15);
            setMode("laugh", rd(2, 2.6f));
            say("경험치 사탕! 경험치 +15", 3.5f);
        } else if (r < 0.60f) {
            int n = 15 + rng.nextInt(16);
            bananas += n;
            setMode("laugh", rd(2, 2.6f));
            say("바나나 " + n + "개 나왔다!", 3);
        } else {
            int n = 5 + rng.nextInt(8);
            bananas += n;
            setMode("shy", rd(2, 2.6f));
            say("바나나 " + n + "개… 다음엔 더 좋은 거!", 3);
        }
    }

    // ───────── 꾸미기 상점 · 코디 (PC shop.go 와 같음) ─────────
    public String outfits = "";   // 코디 3칸: "머리,얼굴,목|머리,얼굴,목|..." (빈 칸 = "")
    public int randomCnt = 0;     // 오늘의 코디 횟수
    public static final int OUTFIT_SLOTS = 3;
    static final int EXP_BUY = 10;

    /** 상점에서 사기 (i = ITEM_IDS 번호). 사면 바로 씀 */
    public boolean buyItem(int i) {
        if (i < 0 || i >= ITEM_IDS.length) return false;
        String id = ITEM_IDS[i];
        if (owns(id)) { toggleWear(i); return false; }
        if (ITEM_PRICE[i] == 0) { say(ITEM_NAMES[i] + "은 " + giftFrom(id) + "이야!", 3); return false; }
        if (bananas < ITEM_PRICE[i]) {
            setMode("st_cry", 2.5f);
            say(ITEM_NAMES[i] + " 갖고 싶은데…\n바나나가 " + (ITEM_PRICE[i] - bananas) + "개 모자라…", 3.5f);
            return false;
        }
        bananas -= ITEM_PRICE[i];
        owned = owned.isEmpty() ? id : owned + "," + id;
        wear[ITEM_SLOT[i]] = id;
        mood = Math.min(100, mood + 15);
        addExp(EXP_BUY);
        setMode("laugh", rd(2.5f, 3.2f));
        say("우와! " + ITEM_NAMES[i] + " 샀다!\n" + ITEM_LINE[i], 4.5f);
        return true;
    }

    public boolean ownsAllShop() {
        for (int i = 0; i < ITEM_IDS.length; i++) if (ITEM_PRICE[i] > 0 && !owns(ITEM_IDS[i])) return false;
        return true;
    }

    public void takeOffAll() {
        wear[0] = wear[1] = wear[2] = "";
        setMode("idle", 2);
        say("다 벗었어! 가뿐하다~", 2.5f);
    }

    /** 코디 i 칸 (자리별 id 3개, 없으면 빈 문자열 3개) */
    public String[] outfit(int i) {
        String[] all = outfits.split("\\|", -1);
        String[] o = {"", "", ""};
        if (i < all.length) {
            String[] w = all[i].split(",", -1);
            for (int s = 0; s < 3 && s < w.length; s++) o[s] = w[s];
        }
        return o;
    }

    /** "고양이 귀 · 진주 목걸이" (아무것도 없으면 "") */
    public static String outfitLabel(String[] o) {
        StringBuilder b = new StringBuilder();
        for (String id : o) {
            int k = itemIndex(id);
            if (k < 0) continue;
            if (b.length() > 0) b.append(" · ");
            b.append(ITEM_NAMES[k]);
        }
        return b.toString();
    }

    public void saveOutfit(int i) {
        if (outfitLabel(wear).isEmpty()) { say("아무것도 안 썼어~\n먼저 꾸며 줘!", 3); return; }
        String[] all = new String[OUTFIT_SLOTS];
        for (int k = 0; k < OUTFIT_SLOTS; k++) {
            String[] o = k == i ? wear : outfit(k);
            all[k] = o[0] + "," + o[1] + "," + o[2];
        }
        outfits = String.join("|", all);
        setMode("st_me", 2.2f);
        say("코디 " + (i + 1) + "에 저장했어!\n" + outfitLabel(wear), 3.5f);
    }

    public void wearOutfit(int i) {
        String[] o = outfit(i);
        if (outfitLabel(o).isEmpty()) { say("코디 " + (i + 1) + "은 비어 있어~", 2.5f); return; }
        for (int s = 0; s < 3; s++) wear[s] = owns(o[s]) ? o[s] : "";
        setMode("st_me", 2.2f);
        say(pick("짠! 코디 " + (i + 1) + " 변신!", "이 옷 좋아~ 어때?", "오늘은 이걸로 결정!"), 3);
    }

    public boolean hasOutfit() {
        for (int i = 0; i < OUTFIT_SLOTS; i++) if (!outfitLabel(outfit(i)).isEmpty()) return true;
        return false;
    }

    /** 가진 아이템으로 자리마다 하나씩 (가끔 비워 둠) 골라 입기 */
    public void randomOutfit() {
        java.util.List<java.util.List<String>> have = new java.util.ArrayList<>();
        for (int s = 0; s < 3; s++) have.add(new java.util.ArrayList<>());
        for (int i = 0; i < ITEM_IDS.length; i++) if (owns(ITEM_IDS[i])) have.get(ITEM_SLOT[i]).add(ITEM_IDS[i]);
        if (have.get(0).isEmpty() && have.get(1).isEmpty() && have.get(2).isEmpty()) {
            say("꾸미기 아이템이 하나도 없어~\n상점에서 하나 사 줄래?", 3);
            return;
        }
        String old = String.join(",", wear);
        for (int t = 0; t < 8; t++) {
            for (int s = 0; s < 3; s++) {
                int n = have.get(s).size();
                wear[s] = n > 0 && rng.nextInt(n + 1) > 0 ? have.get(s).get(rng.nextInt(n)) : "";
            }
            if (!outfitLabel(wear).isEmpty() && !String.join(",", wear).equals(old)) break;
        }
        if (outfitLabel(wear).isEmpty())
            for (int s = 0; s < 3; s++) if (!have.get(s).isEmpty()) { wear[s] = have.get(s).get(rng.nextInt(have.get(s).size())); break; }
        randomCnt++;
        setMode("laugh", rd(2, 2.6f));
        say("오늘의 코디는…\n" + outfitLabel(wear) + "!", 3.5f);
    }

    /** 꾸미기 아이템 켜고 끄기 (i = ITEM_IDS 번호, 가진 것만) */
    public void toggleWear(int i) {
        if (i < 0 || i >= ITEM_IDS.length || !owns(ITEM_IDS[i])) return;
        int s = ITEM_SLOT[i];
        if (wear[s].equals(ITEM_IDS[i])) { wear[s] = ""; setMode("idle", 2); say(ITEM_NAMES[i] + " 벗었어!", 2.5f); }
        else { wear[s] = ITEM_IDS[i]; setMode("st_me", 2.2f); say(pick("어때? 어울려?", ITEM_LINE[i]), 3); }
    }

    static final java.util.Set<String> NO_FACE = new java.util.HashSet<>(java.util.Arrays.asList("st_me", "st_lounge", "st_surprise"));

    /** 지금 그림 위에 그릴 아이템 (안경은 몇몇 동작에서 잠깐 벗음) */
    public java.util.List<String> wearing(String seq) {
        java.util.List<String> l = new java.util.ArrayList<>();
        for (int s = 0; s < 3; s++) {
            String id = wear[s];
            if (id.isEmpty() || !owns(id) || s == 1 && NO_FACE.contains(seq)) continue;
            l.add(id);
        }
        if (dirtyShown() && !bathing && !NO_FACE.contains(seq)) l.add("dirt");
        return l;
    }

    // ───────── 친구 땅콩이 ─────────
    public static final String FRIEND = "땅콩이";
    public static final int FRIEND_LEVEL = 5;
    public static final float FRIEND_SCALE = 0.84f;

    /** 땅콩이 상태 (창 왼쪽 위 = x, y) */
    public static class Friend {
        public float x, y, dir = 1, modeT = 0, modeDur = 3600, speed = 1;
        public String mode = "walk", stage = "enter", act = "";
        float stageLeft = 0, playLeft = 0;
        boolean shared = false;

        void setMode(String m, float d) { mode = m; modeT = 0; modeDur = d; }
    }

    public Friend friend = null;

    public float fw() { return w * FRIEND_SCALE; }

    public float fh() { return h * FRIEND_SCALE; }

    /** manual: 메뉴에서 부름 (못 오면 이유를 말함) */
    public boolean startFriend(boolean manual, long now) {
        if (level() < FRIEND_LEVEL) {
            if (manual) say("Lv." + FRIEND_LEVEL + "가 되면 친구 " + FRIEND + "가 놀러 올 거야!", 3);
            return false;
        }
        if (friend != null) { if (manual) say(FRIEND + " 벌써 와 있어!", 2.5f); return false; }
        if (busy() || tucked || focusing()) { if (manual) say("지금은 못 불러~ 조금 이따!", 2.5f); return false; }
        comeDown();
        pauseToy();
        Friend f = new Friend();
        f.y = floorY - fh();
        float cx = x + w / 2;
        if (cx > scrW - cx) { f.x = -fw(); f.dir = 1; } else { f.x = scrW; f.dir = -1; } // 먼 쪽에서 걸어 들어옴
        f.setMode("walk", 3600);
        friend = f;
        nextFriendAt = now + (long) (rd(60, 150) * 60000);
        setMode("st_surprise", 1.8f);
        say(manual ? FRIEND + "야~ 놀자!" : "어? 저기 누가 온다!", 2.5f);
        return true;
    }

    boolean petOK() { return !mode.equals("drag") && !mode.equals("fall"); }

    void friendTick(float dt, long now) {
        Friend f = friend;
        if (f == null) {
            if (nextFriendAt == 0) nextFriendAt = now + (long) (rd(20, 60) * 60000);
            if (now > nextFriendAt) {
                if (level() < FRIEND_LEVEL) nextFriendAt = now + 30 * 60000L;
                else if (sayText == null && !busy() && (mode.equals("idle") || mode.equals("walk") || mode.equals("sit")))
                    startFriend(false, now);
            }
            return;
        }
        f.modeT += dt;
        float fwid = fw();
        f.y = floorY - fh();
        if (f.mode.equals("walk")) f.x += f.dir * w * 0.19f * speedMul() * f.speed * dt;
        if (!f.mode.equals("walk") && f.modeT > f.modeDur) f.setMode("idle", 3600);
        float pc = x + w / 2, fc = f.x + fwid / 2, gap = w * 0.78f;
        f.stageLeft -= dt;
        switch (f.stage) {
            case "enter":
                if (Math.abs(fc - pc) <= gap || f.dir > 0 && fc > pc - gap || f.dir < 0 && fc < pc + gap) {
                    f.stage = "greet"; f.stageLeft = 3.5f;
                    f.setMode("wave", 3.5f);
                    if (petOK()) setMode("wave", 3.5f);
                    say(pick("안녕 " + FRIEND + "! 놀러 왔구나!", FRIEND + "다! 반가워~", "우와 " + FRIEND + "! 보고 싶었어!"), 3.5f);
                    friendCnt++;
                    mood = clamp(mood + 15);
                    String t = today(now);
                    if (!t.equals(friendDay)) { friendDay = t; addExp(5); }
                }
                break;
            case "greet":
                if (f.stageLeft <= 0) { f.stage = "play"; f.playLeft = rd(50, 80); nextActivity(f); }
                break;
            case "play":
                f.playLeft -= dt;
                float lo = -fwid * 0.1f, hi = scrW - fwid * 0.9f;
                if (f.x < lo) { f.x = lo; f.dir = 1; }
                if (f.x > hi) { f.x = hi; f.dir = -1; }
                if (f.act.equals("chase") && mode.equals("walk")) dir = Math.signum(fc - pc) == 0 ? dir : Math.signum(fc - pc);
                if (f.stageLeft <= 0) {
                    if (f.playLeft <= 0) {
                        f.stage = "leave"; f.stageLeft = 2.8f;
                        f.setMode("wave", 2.8f);
                        if (petOK()) setMode("wave", 2.8f);
                        say(pick("잘 가 " + FRIEND + "! 또 놀러 와~", "벌써 가? 다음에 또 놀자!", FRIEND + " 안녕~ 재밌었어!"), 3.5f);
                    } else nextActivity(f);
                }
                break;
            case "leave":
                if (f.stageLeft <= 0 && !f.mode.equals("walk")) {
                    f.dir = fc < scrW - fc ? -1 : 1;
                    f.speed = 1.1f;
                    f.setMode("walk", 3600);
                    if (petOK()) setMode("idle", rd(2, 4));
                }
                if (f.mode.equals("walk") && (f.x < -fwid || f.x > scrW)) { friend = null; talkIn = rd(30, 60); }
                break;
        }
    }

    void nextActivity(Friend f) {
        java.util.List<String> acts = new java.util.ArrayList<>(java.util.Arrays.asList("laugh", "walk", "chase", "sit", "laugh", "walk"));
        if (!f.shared) { acts.add("share"); acts.add("share"); }
        String a = acts.get(rng.nextInt(acts.size()));
        while (a.equals(f.act)) a = acts.get(rng.nextInt(acts.size()));
        f.act = a;
        float d = rd(4, 7);
        f.stageLeft = d;
        f.speed = 1;
        boolean ok = petOK();
        switch (a) {
            case "laugh":
                f.setMode("laugh", d);
                if (ok) setMode("laugh", d);
                if (rng.nextFloat() < .5f) say(pick(FRIEND + " 너무 웃겨~ 히히", "깔깔깔!", "우리 둘이 웃으니까 더 재밌다!"), 2.5f);
                break;
            case "walk": {
                float d2 = rng.nextBoolean() ? 1 : -1;
                f.dir = d2;
                f.setMode("walk", 3600);
                if (ok) { dir = d2; setMode("walk", d); }
                if (rng.nextFloat() < .4f) say("같이 산책하자~", 2);
                break;
            }
            case "chase": {
                float pc = x + w / 2, fc = f.x + fw() / 2;
                f.dir = fc >= pc ? 1 : -1;
                f.speed = 1.15f;
                f.setMode("walk", 3600);
                if (ok) { dir = f.dir; setMode("walk", d); }
                say(pick("잡아라~!", FRIEND + " 거기 서!", "술래잡기다!"), 2);
                break;
            }
            case "sit":
                f.setMode("sit", d);
                if (ok) setMode("sit", d);
                if (rng.nextFloat() < .6f) say(pick("같이 앉아 있으니까 좋다~", FRIEND + "랑 멍때리기~", "날씨 좋다 그치?"), 3);
                break;
            case "share":
                f.shared = true;
                f.setMode("idle", d);
                bananas += 3;
                full = clamp(full + 10);
                if (ok) setMode("eat", d);
                say(FRIEND + "가 바나나 나눠 줬어!\n냠냠 맛있다~ (바나나 +3)", 3.5f);
                break;
        }
    }

    /** 땅콩이를 톡 누르면 수줍 */
    public void tapFriend() {
        Friend f = friend;
        if (f == null || !(f.stage.equals("play") || f.stage.equals("greet"))) return;
        f.setMode("shy", 2.2f);
        say(pick(FRIEND + "가 부끄러워해~", "히히 " + FRIEND + " 귀엽지?"), 2.5f);
    }

    public String friendSeq() {
        String m = friend == null ? "idle" : friend.mode;
        switch (m) {
            case "walk": case "laugh": case "wave": case "sit": case "shy": return m;
            default: return "idle";
        }
    }

    public int friendFrameIndex(int count) {
        if (friend == null) return 0;
        float fps;
        switch (friend.mode) {
            case "walk": fps = 8 * friend.speed; break;
            case "laugh": fps = 12; break;
            case "wave": fps = 8; break;
            case "shy": fps = 7; break;
            default: fps = 5;
        }
        return ((int) (friend.modeT * fps)) % Math.max(1, count);
    }

    public boolean friendFlip() { return friend != null && friend.mode.equals("walk") && friend.dir < 0; }

    // ───────── 그림 고르기 ─────────
    /** 지금 그릴 동작 이름 */
    public String seqName() {
        switch (mode) {
            case "idle": case "hide": case "cling": return blinkLeft > 0 ? "blink" : "idle";
            case "climb": return "walk";
            case "sit": return blinkLeft > 0 ? "sitblink" : "sit";
            case "drag": case "fall": return "idle";
            case "bath": return bathIdle < .5f ? "shy" : blinkLeft > 0 ? "blink" : "idle";
            default: return mode;
        }
    }

    public float fps() {
        switch (mode) {
            case "walk": return 11;
            case "climb": return 9;
            case "clap": return 10;
            case "cheer": case "angry": case "dance": return 9;
            case "heart": return 7;
            case "huh": return 6;
            case "v": case "thumb": case "sing": case "yum": case "what": return 8;
            case "no": return 10;
            case "flower": case "cute": return 7;
            case "laugh": return 12;
            case "wave": return 8;
            case "shy": return 7;
            case "st_lounge": return 4;
            case "st_bow": return 5;
            case "st_surprise": case "st_cry": case "st_me": return 9;
            case "sleep": case "eat": case "eat_milk": case "eat_cake": return 6;
            default: return 5;
        }
    }

    /** 몇 번째 프레임 (count = 그 동작의 장수) */
    public int frameIndex(int count) {
        if (mode.equals("drag") || mode.equals("fall") || blinkLeft > 0 && (mode.equals("idle") || mode.equals("sit") || mode.equals("hide") || mode.equals("cling"))) return 0;
        return ((int) (modeT * fps())) % Math.max(1, count);
    }

    public boolean flip() {
        // 그림은 오른쪽을 봄. 벽·천장에서는 돌린 다음 가는 쪽을 보게
        if (q == 1) return dirY < 0;
        if (q == 3) return dirY > 0;
        if (q == 2) return dir > 0;
        return (mode.equals("walk") || mode.equals("hide")) && dir < 0;
    }

    // ───────── 벽 타기 · 천장 매달리기 · 내 자리 ─────────
    /** 0 바닥·공중, 1 왼쪽 벽(시계 방향 90°), 2 천장(180°), 3 오른쪽 벽(반시계 90°) */
    public int q = 0;
    public float dirY = -1;
    public boolean wantClimb = false, wantCeil = false, seatJump = false, perched = false;
    public float perchY = 0, perchLeft = 0;
    public float seatFX = -1, seatFY = -1; // 내 자리 (화면 비율)
    static final float CLIMB_CHANCE = 0.4f, FOOT = 0.015f; // 그림 아래 여백 (h 비율)

    /** 지금 창 크기 (벽에서는 가로세로가 바뀜) */
    public float winW() { return q == 1 || q == 3 ? h : w; }

    public float winH() { return q == 1 || q == 3 ? w : h; }

    public boolean away() { return q != 0 || perched; }

    float climbSpeed() { return w * 0.3f * speedMul(); }

    void toWall(int side, float dy) {
        q = side;
        dirY = dy;
        x = side == 1 ? -h * FOOT : scrW - h + h * FOOT;
        y = dy < 0 ? floorY - w : ceilY;
        vx = vy = 0;
        wantClimb = false;
        setMode("climb", rd(3, 7));
        if (rng.nextFloat() < .35f) say(pick("영차영차~", "벽 타기 선수 우땅이!", "높이 올라가 볼까?"), 2.5f);
    }

    void toCeiling(float cx, float d) {
        q = 2;
        dir = d;
        x = Math.max(-w * 0.15f, Math.min(scrW - w * 0.85f, cx));
        y = ceilY - h * FOOT;
        vx = vy = 0;
        wantCeil = false;
        setMode("climb", rd(4, 8));
    }

    void toFloor(int fromSide) {
        q = 0;
        y = floorY - h;
        x = fromSide == 1 ? -w * 0.15f : scrW - w * 0.85f;
        dir = fromSide == 1 ? 1 : -1;
        setMode("idle", rd(1, 2));
    }

    /** 벽·천장 창 → 보통 창 (가운데는 그대로) */
    void leaveSurface() {
        if (q == 0) return;
        float cx = x + winW() / 2, cy = y + winH() / 2;
        q = 0;
        x = Math.max(-w * 0.15f, Math.min(scrW - w * 0.85f, cx - w / 2));
        y = Math.min(cy - h / 2, floorY - h);
    }

    /** 벽·천장에서 손 놓기 */
    public void letGo() {
        int was = q;
        leaveSurface();
        vx = was == 1 ? 220 * dp : was == 3 ? -220 * dp : 0;
        vy = was == 2 ? 0 : -120 * dp;
        bounces = 0; maxFall = 0; thrown = false;
        setMode("fall", 9999);
    }

    /** 벽·천장·자리에 있으면 바닥으로 */
    public void comeDown() {
        if (q != 0) letGo();
        else if (perched) hopDown();
    }

    void climbTick(float dt) {
        float sp = climbSpeed();
        if (mode.equals("climb")) {
            if (q == 1 || q == 3) {
                y += dirY * sp * dt;
                if (dirY < 0 && y <= ceilY) {
                    y = ceilY;
                    if (wantCeil || rng.nextFloat() < 0.65f) { toCeiling(q == 1 ? -w * 0.15f : scrW - w * 0.85f, q == 1 ? 1 : -1); return; }
                    dirY = 1;
                }
                if (dirY > 0 && y + w >= floorY) { toFloor(q); return; }
            } else if (q == 2) {
                x += dir * sp * dt;
                if (dir > 0 && x >= scrW - w * 0.85f) {
                    if (rng.nextFloat() < 0.6f) { toWall(3, 1); return; }
                    dir = -1;
                }
                if (dir < 0 && x <= -w * 0.15f) {
                    if (rng.nextFloat() < 0.6f) { toWall(1, 1); return; }
                    dir = 1;
                }
            }
        }
        if (modeT < modeDur) return;
        float r = rng.nextFloat();
        if (mode.equals("cling")) { setMode("climb", rd(3, 7)); return; }
        if (wantCeil && q != 2) { dirY = -1; setMode("climb", rd(3, 7)); return; } // 천장 가는 중엔 쭉 올라감
        if (r < 0.22f) {
            letGo();
            if (rng.nextFloat() < .5f) say(pick("뿅! 착지 준비~", "에잇, 뛰어내리기!"), 2);
        } else if (r < 0.55f) setMode("cling", rd(2, 4));
        else if (r < 0.7f) { if (q == 2) dir = -dir; else dirY = -dirY; setMode("climb", rd(3, 6)); }
        else setMode("climb", rd(3, 7));
    }

    /** 살살 놓았을 때: 벽·천장 가까우면 붙고, 공중이면 그 자리에 앉음. 아니면 false (그냥 떨어짐) */
    boolean placeGently() {
        float cx = x + w / 2, cy = y + h / 2;
        float edge = w * 0.42f;
        if (cy > ceilY + h * 0.6f && cy < floorY - h * 0.2f) {
            if (cx < edge) { attachWall(1, cy); return true; }
            if (cx > scrW - edge) { attachWall(3, cy); return true; }
        }
        if (y < ceilY + h * 0.12f) { toCeiling(x, dir); setMode("cling", rd(2, 4)); return true; }
        if (y < floorY - h - h * 0.3f) {
            y = Math.max(y, ceilY);
            perch(y, rd(25, 60));
            say(pick("여기 앉아 있을게!", "높다~ 경치 좋다!", "여기 좋은 자리네~"), 2.5f);
            return true;
        }
        return false;
    }

    void attachWall(int side, float cy) {
        q = side;
        dirY = -1;
        x = side == 1 ? -h * FOOT : scrW - h + h * FOOT;
        y = Math.max(ceilY, Math.min(floorY - w, cy - w / 2));
        vx = vy = 0;
        setMode("cling", rd(2, 4));
        say(pick("벽에 착 붙었다!", "매달려 있기~"), 2.2f);
    }

    void perch(float py, float sec) {
        q = 0;
        perched = true;
        perchY = py;
        perchLeft = sec;
        vx = vy = 0;
        mode = "sit"; modeT = 0; modeDur = rd(4, 8); // setMode 쓰면 perched 가 풀림
    }

    void hopDown() {
        perched = false;
        vx = (rng.nextBoolean() ? 1 : -1) * 120 * dp;
        vy = -260 * dp;
        bounces = 0; maxFall = 0; thrown = false;
        setMode("fall", 9999);
        if (rng.nextFloat() < .5f) say(pick("폴짝!", "이제 내려갈래~"), 2);
    }

    public boolean hasSeat() { return seatFX >= 0 && seatFY >= 0; }

    float seatX() { return seatFX * scrW; }

    float seatY() { return Math.max(ceilY, Math.min(floorY - h * 1.3f, seatFY * scrH)); }

    /** 지금 앉아 있는 곳을 내 자리로 */
    public void setSeat() {
        if (!perched) return;
        seatFX = x / scrW;
        seatFY = y / scrH;
        perchLeft = Math.max(perchLeft, rd(60, 120));
        setMode("laugh", 2.5f);
        perched = true; // setMode 가 laugh 에서는 자리를 안 풀지만 확실히
        say("여기를 내 자리로 할게!\n가끔 여기 올라와 있을게~", 3.5f);
    }

    public void clearSeat() {
        seatFX = seatFY = -1;
        say("알았어! 자리는 다시 정해 줘~", 2.5f);
    }

    /** 바닥에서 내 자리까지 폴짝 (포물선) */
    public boolean jumpToSeat() {
        if (!hasSeat() || away() || busy() || friend != null || toyActive() || focusing() || tucked) return false;
        float sy = seatY(), sx = seatX();
        if (y - sy < h * 0.3f) return false;
        float g = 1500 * dp, hd = h * 0.15f;
        float H = (y - sy) + hd;
        float vy0 = (float) -Math.sqrt(2 * g * H);
        float T = -vy0 / g + (float) Math.sqrt(2 * hd / g);
        vx = (sx - x) / T;
        vy = vy0;
        seatJump = true; thrown = false; bounces = 0; maxFall = 0;
        dir = vx >= 0 ? 1 : -1;
        setMode("fall", 9999);
        say(pick("내 자리로 점프!", "영차~ 폴짝!"), 2);
        return true;
    }

    /** 메뉴: 벽 타기 · 천장 */
    public void goClimb(boolean ceiling) {
        if (busy() || hiding) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        if (q != 0) { if (ceiling && q != 2) wantCeil = true; else say("벌써 매달려 있어~", 2); return; }
        if (perched) hopDown();
        stay = false;
        wantClimb = true;
        wantCeil = ceiling;
        dir = x + w / 2 < scrW / 2 ? -1 : 1;
        if (!perched) setMode("walk", 30);
        say(ceiling ? "천장까지 올라가 볼게!" : "벽 타기 보여 줄게!", 2.5f);
    }

    // ═════════ 돌봄 · 집중 · 장난감 · 선물 · 바나나 탑 · 업적 (PC 우땅이와 같은 규칙) ═════════
    public float dirt = 0, tired = 0; // 0 = 깨끗함 · 쌩쌩
    public int bathCnt = 0, tuckCnt = 0, focusCnt = 0, focusMin = 0, toyCnt = 0, toyKick = 0, giftCnt = 0, towerBest = 0, towerCnt = 0;
    public String bathDay = "", tuckDay = "", toyDay = "", gifts = "", ach = "";
    public long focusUntil = 0;
    public int focusLen = 0;
    long nowMs = 0;
    int hourNow = 12;

    void loadFun(Store s) {
        dirt = s.getFloat("dirt", 0); tired = s.getFloat("tired", 0);
        bathCnt = s.getInt("bathCnt", 0); tuckCnt = s.getInt("tuckCnt", 0);
        focusCnt = s.getInt("focusCnt", 0); focusMin = s.getInt("focusMin", 0);
        toyCnt = s.getInt("toyCnt", 0); toyKick = s.getInt("toyKick", 0); giftCnt = s.getInt("giftCnt", 0);
        towerBest = s.getInt("towerBest", 0); towerCnt = s.getInt("towerCnt", 0);
        whackBest = s.getInt("whackBest", 0); whackCnt = s.getInt("whackCnt", 0); gifSaved = s.getInt("gifSaved", 0);
        bathDay = s.getString("bathDay", ""); tuckDay = s.getString("tuckDay", ""); toyDay = s.getString("toyDay", "");
        gifts = s.getString("gifts", ""); ach = s.getString("ach", "");
        cooked = s.getString("cooked", ""); cookCnt = s.getInt("cookCnt", 0); danceCnt = s.getInt("danceCnt", 0); chatCnt = s.getInt("chatCnt", 0);
        outfits = s.getString("outfits", ""); randomCnt = s.getInt("randomCnt", 0);
        for (String[] a : ACH) // 보상이 생기기 전에 깬 업적: 아이템만 조용히 챙겨 줌
            if (a.length > 5 && hasAch(a[0]) && !owns(a[5])) owned = owned.isEmpty() ? a[5] : owned + "," + a[5];
        focusUntil = s.getLong("focusUntil", 0); focusLen = s.getInt("focusLen", 0);
    }

    void saveFun(Store s) {
        s.put("dirt", dirt); s.put("tired", tired); s.put("bathCnt", bathCnt); s.put("tuckCnt", tuckCnt);
        s.put("focusCnt", focusCnt); s.put("focusMin", focusMin); s.put("toyCnt", toyCnt); s.put("toyKick", toyKick);
        s.put("giftCnt", giftCnt); s.put("towerBest", towerBest); s.put("towerCnt", towerCnt);
        s.put("whackBest", whackBest); s.put("whackCnt", whackCnt); s.put("gifSaved", gifSaved);
        s.put("bathDay", bathDay); s.put("tuckDay", tuckDay); s.put("toyDay", toyDay); s.put("gifts", gifts); s.put("ach", ach);
        s.put("cooked", cooked); s.put("cookCnt", cookCnt); s.put("danceCnt", danceCnt); s.put("chatCnt", chatCnt);
        s.put("outfits", outfits); s.put("randomCnt", randomCnt);
        s.put("focusUntil", focusUntil); s.put("focusLen", focusLen);
    }

    public int clean() { return Math.round(100 - dirt); }

    public int energy() { return Math.round(100 - tired); }

    public boolean dirtyShown() { return dirt >= 50; }

    public boolean sleepy() { return tired >= 75; }

    void addDirt(float v) { dirt = clamp(dirt + v); }

    /** 1분마다 */
    void careMinute() {
        if (tucked) return;
        if (mode.equals("sleep")) tired = clamp(tired - 3);
        else { dirt = clamp(dirt + .18f); tired = clamp(tired + .22f); }
    }

    /** 꺼져 있던 동안: 자고 왔다고 보고 졸림은 풀리고, 조금 꼬질꼬질 */
    void careAway(float hrs) {
        if (hrs >= 1) tired = clamp(tired - hrs * 15);
        dirt = clamp(dirt + Math.min(hrs, 12));
    }

    String careLine() {
        String n = voc();
        if (dirt >= 70 && rng.nextFloat() < .6f) return pick("몸이 근질근질해… " + n + " 씻겨 줘~", "킁킁… 나한테서 냄새나?", "목욕하고 싶다… 거품 목욕!");
        if (sleepy() && rng.nextFloat() < .6f) return pick("하암… 졸려…", n + " 이불 덮어 줄래? 하암~", "눈이 자꾸 감겨…");
        return null;
    }

    // ── 씻기기: 손가락으로 문지르면 거품이 퐁퐁, 다 씻으면 반짝 ──
    public boolean bathing = false;
    float bathLeft = 0, bathNeed = 0, bathIdle = 9;
    boolean bathPrompt = false, bathRubbed = false;
    /** 우땅이 몸의 거품 {x, y (창 비율), 크기 (창 너비 비율), 나이} */
    public final java.util.List<float[]> bubbles = new java.util.ArrayList<>();

    public void startBath() {
        if (focusing()) { say("집중 끝나고 씻자! 쉿~", 2.5f); return; }
        if (dirt < 5) { setMode("st_me", 2.2f); say("나 이미 깨끗해! 반짝반짝~", 2.5f); return; }
        if (busy() || friend != null) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        if (away()) { comeDown(); say("내려가서 씻자~ 다시 불러 줘!", 2.5f); return; }
        tucked = false;
        pauseToy();
        bathing = true; bathPrompt = false; bathRubbed = false;
        bathLeft = 45; bathIdle = 9; bathNeed = Math.max(dirt, 35);
        setMode("bath", 3600);
        say("목욕하자! 🫧\n손가락으로 나를 문질문질 해 줘~", 4);
        talkIn = rd(90, 150);
    }

    /** 문지르기 한 번 (fx, fy: 창 안 비율 0~1) */
    public void rub(float fx, float fy) {
        if (!bathing) return;
        bathIdle = 0;
        bathRubbed = true;
        bathNeed -= 3.5f;
        dirt = clamp(dirt - 3.5f);
        addBubble(fx, fy);
        if (bathNeed <= 0) finishBath();
    }

    void addBubble(float fx, float fy) {
        if (bubbles.size() < 40) bubbles.add(new float[]{fx, fy, rd(.1f, .18f), 0});
    }

    void bathTick(float dt) {
        float life = bathing ? 6 : 1.2f;
        for (java.util.Iterator<float[]> it = bubbles.iterator(); it.hasNext(); ) {
            float[] b = it.next();
            b[3] += dt;
            b[1] -= dt * 0.02f;
            if (b[3] > life) it.remove();
        }
        if (!bathing) return;
        if (!mode.equals("bath")) { bathing = false; say("앗 아직 거품 묻었는데~", 2.5f); return; }
        bathLeft -= dt;
        bathIdle += dt;
        if (bathIdle < .6f && rng.nextFloat() < dt * 4) addBubble(rd(.25f, .75f), rd(.3f, .85f));
        if (bathLeft <= 0) {
            bathing = false;
            setMode("idle", rd(1.5f, 2.5f));
            say("에취! 다음에 마저 씻자~", 3);
        } else if (!bathPrompt && !bathRubbed && bathLeft < 38) {
            bathPrompt = true;
            say("손가락으로 내 몸을\n왔다 갔다 문질러 봐! 🫧", 3.5f);
        }
    }

    void finishBath() {
        bathing = false;
        dirt = 0;
        bathCnt++;
        mood = clamp(mood + 12);
        for (int i = 0; i < 8; i++) addBubble(rd(.15f, .85f), rd(.2f, .9f));
        setMode("laugh", rd(2.8f, 3.4f));
        say(pick("반짝반짝! 개운하다~\n고마워 " + voc() + "!", "뽀송뽀송! 향기 나지? 히히", "깨끗해졌다! 반짝 ✨"), 4);
        talkIn = rd(60, 120);
        String t = today(nowMs);
        if (!t.equals(bathDay)) { bathDay = t; addExp(6); }
    }

    // ── 이불 덮어 재우기 ──
    public boolean tucked = false;
    float tuckT = 0;

    public void tuckIn() {
        if (focusing()) { say("집중 끝나면 잘게~ 쉿!", 2.5f); return; }
        if (tired < 15 && hourNow >= 7 && hourNow < 21) { setMode("laugh", rd(2, 2.6f)); say("아직 하나도 안 졸려! 더 놀자~", 2.5f); return; }
        if (busy() || friend != null) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        if (q != 0) { comeDown(); say("내려가서 잘게~ 다시 덮어 줘!", 2.5f); return; }
        bathing = false;
        pauseToy();
        tucked = true;
        tuckT = 0;
        setMode("sleep", Math.max(40, Math.min(120, tired / 1.5f)));
        say(pick("이불 덮어 줘서 고마워…\n잘 자 " + voc() + "… ♥", "포근하다… 쿨쿨…", "따뜻해… 좋은 꿈 꿀게…"), 3.5f);
        talkIn = rd(40, 80);
    }

    void tuckTick(float dt) {
        if (!tucked) return;
        if (!mode.equals("sleep")) { tucked = false; return; }
        tuckT += dt;
        tired = clamp(tired - dt * 2.5f);
    }

    void wakeTucked() {
        tucked = false;
        tired = 0;
        tuckCnt++;
        mood = clamp(mood + 8);
        setMode("wave", rd(2.6f, 3.2f));
        say(pick("잘 잤다! 기운이 펄펄!", "하암~ 푹 잤어! 고마워 " + voc() + "!", "개운해! 이제 놀자!"), 3.5f);
        talkIn = rd(60, 120);
        String t = today(nowMs);
        if (!t.equals(tuckDay)) { tuckDay = t; addExp(4); }
    }

    /** 이불 덮고 자는데 톡: 처음엔 더 잔다고 하고, 좀 지나면 일어남 */
    void pokeTucked() {
        if (tuckT < 15) { say(pick("으음… 5분만 더…", "쿨… 조금만 더 잘래…"), 2.5f); return; }
        wakeTucked();
    }

    // ── 집중 타이머: 25분/50분 동안 옆에 조용히 앉아 기다렸다가 축하하고 바나나 ──
    public boolean focusing() { return focusUntil > 0; }

    public int focusLeftMin(long now) { return Math.max(1, (int) Math.ceil((focusUntil - now) / 60000.0)); }

    public void startFocus(int minutes, long now) {
        if (hiding || gachaLeft > 0) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        if (q != 0) comeDown();
        pauseToy();
        focusLen = minutes;
        focusUntil = now + minutes * 60000L;
        stay = false;
        if (!mode.equals("fall")) setMode("sit", rd(20, 40));
        say("좋아! " + minutes + "분 집중 시작!\n옆에서 조용히 기다릴게.\n끝나면 바나나 " + minutes / 5 + "개 줄게~", 4.5f);
        talkIn = minutes * 60 + 60;
    }

    public void stopFocus() {
        focusUntil = 0;
        setMode("idle", rd(1.5f, 2.5f));
        say("집중 그만! 조금 쉬었다 해도 괜찮아~", 3);
        talkIn = rd(60, 120);
    }

    void focusTick(long now) {
        if (!focusing() || now < focusUntil) return;
        focusUntil = 0;
        int b = focusLen / 5;
        bananas += b;
        focusCnt++;
        focusMin += focusLen;
        addExp(b * 2);
        if (!mode.equals("fall") && !mode.equals("drag")) setMode("clap", 3); // 수고했어 짝짝
        say(focusLen + "분 집중 끝! 수고했어 " + voc() + "!\n바나나 " + b + "개 선물이야.\n5분만 쉬었다 하자~", 7);
        toastTitle = "집중 끝!";
        toastText = focusLen + "분 집중 완료 · 바나나 " + b + "개 · 잠깐 쉬어 가요";
        talkIn = rd(60, 120);
    }

    // ── 장난감: 공 · 풍선 ──
    public static class Toy {
        public String kind;
        public float x, y, vx, vy, size, rot = 0; // x, y = 가운데
        public boolean held = false, playing = false;
        float playLeft = 0, restLeft = 0, touchAgo = 0, hitAgo = 9;
    }

    public Toy toy;

    /** PC 우땅이 1px 에 해당하는 길이 (PC 는 우땅이 키가 112px) */
    float u() { return h / 112f; }

    public boolean toyActive() { return toy != null && toy.playing; }

    boolean petFree() {
        return q == 0 && !perched && !hiding && gachaLeft <= 0 && !bathing && !tucked && friend == null && !focusing();
    }

    public void startToy(String kind) {
        if (toy != null && toy.kind.equals(kind)) { say("벌써 꺼내 놨어~ 같이 놀자!", 2.5f); wakeToy(); return; }
        if (busy() || friend != null || bathing || tucked || focusing()) { say("지금은 못 해~ 조금 이따 놀자!", 2.5f); return; }
        comeDown();
        Toy t = new Toy();
        t.kind = kind;
        t.size = h * (kind.equals("ball") ? 0.34f : 0.46f);
        float r = t.size / 2;
        t.x = Math.max(r, Math.min(scrW - r, x + w / 2 + dir * h * 1.3f));
        if (kind.equals("ball")) { t.y = y + h * 0.2f; t.vy = -200 * u(); }
        else t.y = floorY - h * 1.6f;
        toy = t;
        wakeToy();
        toyCnt++;
        mood = clamp(mood + 8);
        String td = today(nowMs);
        if (!td.equals(toyDay)) { toyDay = td; addExp(3); }
        if (!mode.equals("fall")) setMode("laugh", 1.5f);
        say(kind.equals("ball") ? pick("공이다! 같이 놀자~", "뻥 차 볼게!", "공놀이 좋아!")
                : pick("풍선이다! 안 떨어뜨릴 거야~", "둥실둥실 풍선!", "풍선 통통 놀이!"), 2.5f);
    }

    public void putAwayToy(boolean talk) {
        if (toy == null) return;
        toy = null;
        if (talk) say("장난감은 치워 둘게~ 또 놀자!", 2.5f);
    }

    void pauseToy() {
        if (toy != null && toy.playing) { toy.playing = false; toy.restLeft = 120; }
    }

    void wakeToy() {
        if (toy == null) return;
        toy.touchAgo = 0;
        if (!toy.playing) { toy.playing = true; toy.playLeft = rd(70, 110); }
    }

    /** 장난감을 톡 */
    public void pokeToy() {
        Toy t = toy;
        if (t == null) return;
        if (t.kind.equals("ball")) { t.vy = -rd(600, 800) * u(); t.vx = rd(-220, 220) * u(); }
        else { t.vy = -rd(260, 340) * u(); t.vx = rd(-60, 60) * u(); }
        wakeToy();
    }

    /** 끌어서 던짐 (px/s) */
    public void throwToy(float vx, float vy) {
        Toy t = toy;
        if (t == null) return;
        float lim = 2600 * dp;
        t.held = false;
        t.vx = Math.max(-lim, Math.min(lim, vx));
        t.vy = Math.max(-lim, Math.min(lim, vy));
        wakeToy();
    }

    float toyBottom() { return toy.size * (toy.kind.equals("ball") ? 0.4f : 0.45f); }

    /** 몸통 {x, y, 반지름} */
    float[] toyBody() {
        Toy t = toy;
        return t.kind.equals("balloon") ? new float[]{t.x, t.y - t.size * 0.1f, t.size * 0.3f} : new float[]{t.x, t.y, t.size * 0.4f};
    }

    void toyPhysics(float dt) {
        Toy t = toy;
        if (t.held) { t.vx = t.vy = 0; return; }
        float uu = u(), r = t.size / 2 * 0.8f, bo = toyBottom();
        boolean ball = t.kind.equals("ball");
        if (ball) { t.vy += 1500 * uu * dt; t.vx *= 1 - 0.3f * dt; }
        else { t.vy += 70 * uu * dt; t.vx *= 1 - 1.2f * dt; t.vy *= 1 - 0.8f * dt; }
        t.x += t.vx * dt;
        t.y += t.vy * dt;
        if (t.y + bo > floorY) {
            t.y = floorY - bo;
            if (ball && t.vy > 140 * uu) { t.vy = -t.vy * 0.55f; t.vx *= 0.85f; }
            else if (!ball && t.vy > 20 * uu) t.vy = -t.vy * 0.3f;
            else t.vy = 0;
        }
        if (ball && t.y + bo >= floorY - 0.5f) {
            t.vx *= 1 - 1.4f * dt;
            if (Math.abs(t.vx) < 6 * uu) t.vx = 0;
        }
        if (t.x < r) { t.x = r; t.vx = Math.abs(t.vx) * 0.7f; }
        if (t.x > scrW - r) { t.x = scrW - r; t.vx = -Math.abs(t.vx) * 0.7f; }
        if (t.y - r < ceilY) { t.y = ceilY + r; t.vy = Math.abs(t.vy) * 0.5f; }
        if (ball) t.rot += t.vx / Math.max(1, r) * dt;
        else t.rot = (float) Math.sin(nowMs / 1000.0 * 1.3) * 0.12f + Math.max(-0.3f, Math.min(0.3f, t.vx * 0.0006f / uu));
    }

    void toyHit() {
        toy.hitAgo = 0;
        toyKick++;
        mood = clamp(mood + 0.5f);
    }

    /** 우땅이 머리에 닿으면 통 */
    void toyHeadBump() {
        Toy t = toy;
        if (t.held || t.hitAgo < 0.25f || mode.equals("drag") || q != 0) return;
        float[] b = toyBody();
        float hx0 = x + w * 0.22f, hx1 = x + w * 0.78f, hy0 = y + h * 0.12f, hy1 = y + h * 0.5f;
        float nx = Math.max(hx0, Math.min(hx1, b[0])), ny = Math.max(hy0, Math.min(hy1, b[1]));
        if (Math.hypot(b[0] - nx, b[1] - ny) > b[2]) return;
        boolean rising = mode.equals("fall") && vy < 0;
        if (t.vy < 0 && !rising) return;
        float pc = x + w / 2, uu = u();
        if (t.kind.equals("ball")) { t.vy = -rd(520, 700) * uu; t.vx = (b[0] - pc) * 4 + rd(-60, 60) * uu; }
        else { t.vy = -rd(300, 420) * uu; t.vx = (b[0] - pc) * 2.5f + rd(-50, 50) * uu; }
        toyHit();
        if (rng.nextFloat() < .25f) say(pick("통!", "헤딩!", "머리로 통~"), 1.2f);
    }

    void waitIdle() { if (!mode.equals("idle")) setMode("idle", 2); }

    void chaseTo(float face) { dir = face; if (!mode.equals("walk")) setMode("walk", 5); }

    /** 우땅이가 장난감 쪽으로 */
    void toyAI() {
        Toy t = toy;
        if (!petFree() || mode.equals("fall") || mode.equals("drag")) return;
        if (!mode.equals("walk") && !mode.equals("idle") && modeT < modeDur) return;
        float pc = x + w / 2, uu = u();
        float[] b = toyBody();
        float d = b[0] - pc, face = d >= 0 ? 1 : -1;
        if (t.held) { dir = face; waitIdle(); return; }
        if (t.kind.equals("ball")) {
            boolean low = b[1] > floorY - h * 0.6f;
            if (Math.abs(d) > h * 0.3f || !low) {
                if (!low && Math.abs(d) < h * 0.25f) { dir = face; waitIdle(); return; }
                chaseTo(face);
                return;
            }
            if (Math.hypot(t.vx, t.vy) < 260 * uu && t.hitAgo > 0.7f) {
                dir = face;
                t.vx = face * rd(220, 420) * uu;
                t.vy = -rd(200, 520) * uu;
                toyHit();
                setMode("laugh", 0.7f);
                if (rng.nextFloat() < .3f) say(pick("뻥!", "슛~!", "간다~!", "받아라!"), 1.5f);
                return;
            }
            dir = face;
            waitIdle();
            return;
        }
        // 풍선: 바닥까지 내려왔으면 발로 톡, 머리 높이로 내려오면 밑에서 폴짝
        if (Math.abs(d) < h * 0.3f && b[1] > y + h * 0.55f && t.hitAgo > 0.5f) {
            dir = face;
            t.vy = -rd(420, 520) * uu;
            t.vx = face * rd(20, 80) * uu;
            toyHit();
            setMode("laugh", 0.6f);
            return;
        }
        if (Math.abs(d) < h * 0.28f && b[1] > y - h * 0.35f && t.vy >= 0) {
            dir = face;
            vx = 0; vy = -540 * uu;
            maxFall = 0; bounces = 0; thrown = false;
            setMode("fall", 9999);
            return;
        }
        if (Math.abs(d) > h * 0.18f) { chaseTo(face); return; }
        dir = face;
        waitIdle();
    }

    void toyTick(float dt) {
        Toy t = toy;
        if (t == null) return;
        t.hitAgo += dt;
        if (t.held) t.touchAgo = 0; else t.touchAgo += dt;
        toyPhysics(dt);
        if (t.playing) {
            t.playLeft -= dt;
            if (t.playLeft <= 0) { // 실컷 놀았음
                t.playing = false;
                t.restLeft = rd(40, 90);
                if (petFree() && !mode.equals("fall")) {
                    setMode("sit", rd(6, 10));
                    say(pick("헉헉 재밌었다~ 잠깐 쉴래!", "후~ 신난다! 조금만 쉬고 또 하자", "놀다 보니 숨차~"), 3);
                }
            }
        } else {
            t.restLeft -= dt;
            if (t.restLeft <= 0 && petFree() && (mode.equals("idle") || mode.equals("walk")) && !busy() && rng.nextFloat() < dt * 0.3f) {
                wakeToy();
                say(pick("또 놀자!", "장난감 어디 있지? 저기다!"), 2);
            }
        }
        if (t.playing) { toyHeadBump(); if (toy != null) toyAI(); }
        if (toy != null && !toy.playing && toy.touchAgo > 8 * 60) putAwayToy(true);
    }

    // ── 선물 물어오기 · 선물 모음 ──
    public static final String[] GIFT_IDS = {"banana", "goldbanana", "petal", "maple", "ginkgo", "acorn", "clover", "shell", "star", "candy"};
    public static final String[] GIFT_NAMES = {"바나나", "황금 바나나", "꽃잎", "단풍잎", "은행잎", "도토리", "네잎클로버", "조개껍데기", "반짝 별", "사탕"};
    static final float[] GIFT_W = {28, 3, 10, 8, 8, 9, 4, 7, 6, 10};
    static final int[] GIFT_BAN = {2, 10, 0, 0, 0, 0, 0, 0, 0, 0};
    static final int[][] GIFT_SEASON = {{}, {}, {3, 4, 5}, {9, 10, 11}, {9, 10, 11}, {9, 10, 11}, {4, 5, 6}, {6, 7, 8}, {12, 1, 2}, {12, 1, 2}};
    static final float GIFT_LIFE = 40 * 60;

    public static class Gift {
        public int kind;
        public float x, y, size, age, baseY, rot;
    }

    public final java.util.List<Gift> giftList = new java.util.ArrayList<>();
    public float nextGiftIn = -1;

    public int giftCount(String id) {
        for (String p : gifts.split(",")) {
            int c = p.indexOf(':');
            if (c > 0 && p.substring(0, c).equals(id)) try { return Integer.parseInt(p.substring(c + 1)); } catch (Exception e) { return 0; }
        }
        return 0;
    }

    void addGiftCount(String id) {
        StringBuilder sb = new StringBuilder();
        boolean found = false;
        for (String p : gifts.split(",")) {
            if (p.isEmpty()) continue;
            int c = p.indexOf(':');
            String k = c > 0 ? p.substring(0, c) : p;
            int n = 0;
            try { n = Integer.parseInt(p.substring(c + 1)); } catch (Exception ignored) { }
            if (k.equals(id)) { n++; found = true; }
            if (sb.length() > 0) sb.append(',');
            sb.append(k).append(':').append(n);
        }
        if (!found) { if (sb.length() > 0) sb.append(','); sb.append(id).append(":1"); }
        gifts = sb.toString();
    }

    public int giftKindsGot() {
        int n = 0;
        for (String id : GIFT_IDS) if (giftCount(id) > 0) n++;
        return n;
    }

    int pickGift(int month) {
        float total = 0;
        float[] ws = new float[GIFT_IDS.length];
        for (int i = 0; i < ws.length; i++) {
            float wgt = GIFT_W[i];
            for (int m : GIFT_SEASON[i]) if (m == month) wgt *= 3;
            ws[i] = wgt;
            total += wgt;
        }
        float r = rng.nextFloat() * total;
        for (int i = 0; i < ws.length; i++) { if (r < ws[i]) return i; r -= ws[i]; }
        return 0;
    }

    boolean canGift() {
        return giftList.size() < 3 && petFree() && !busy() && !toyActive() && (mode.equals("walk") || mode.equals("idle")) && y >= floorY - h - 1;
    }

    Gift dropGift(int kind) {
        Gift g = new Gift();
        g.kind = kind;
        g.size = h * 0.3f;
        float r = g.size / 2;
        g.x = Math.max(r, Math.min(scrW - r, x + w / 2 - dir * h * 0.38f));
        g.baseY = floorY - r * 0.95f;
        g.y = g.baseY;
        giftList.add(g);
        return g;
    }

    public Gift dropGiftForTest(int kind) { return dropGift(kind); }

    public int pickGiftForTest(int month) { return pickGift(month); }

    public void collectGift(Gift g) {
        if (!giftList.remove(g)) return;
        String id = GIFT_IDS[g.kind], nm = GIFT_NAMES[g.kind];
        addGiftCount(id);
        giftCnt++;
        boolean first = giftCount(id) == 1;
        bananas += GIFT_BAN[g.kind];
        mood = clamp(mood + 3);
        addExp(1);
        String msg = GIFT_BAN[g.kind] > 0 ? nm + "! 바나나 " + GIFT_BAN[g.kind] + "개 생겼어~" : nm + " 주웠다!";
        if (first) msg = "새 선물! " + nm + "\n선물 모음 " + giftKindsGot() + "/" + GIFT_IDS.length;
        say(msg, 3);
        if (petFree() && (mode.equals("idle") || mode.equals("walk") || mode.equals("sit"))) setMode("laugh", 1.5f);
    }

    void giftTick(float dt) {
        for (java.util.Iterator<Gift> it = giftList.iterator(); it.hasNext(); ) {
            Gift g = it.next();
            g.age += dt;
            if (g.age > GIFT_LIFE) { it.remove(); continue; }
            g.y = g.baseY + (float) Math.sin(g.age * 2.2) * 2 * u();
            g.rot = (float) Math.sin(g.age * 1.5) * 0.15f;
        }
        if (nextGiftIn < 0) { nextGiftIn = rd(8 * 60, 20 * 60); return; }
        nextGiftIn -= dt;
        if (nextGiftIn > 0) return;
        if (!canGift()) { nextGiftIn = 30; return; }
        nextGiftIn = rd(20 * 60, 50 * 60);
        Calendar c = Calendar.getInstance();
        c.setTimeInMillis(nowMs == 0 ? System.currentTimeMillis() : nowMs);
        dropGift(pickGift(c.get(Calendar.MONTH) + 1));
        setMode("st_me", 2);
        say(pick("선물 놓고 갈게~ 주워 봐!", "짠! 이거 " + voc() + " 선물!", "산책하다 주웠어! 너 줄게~"), 3);
    }

    // ── 바나나 탑 점수 ──
    public static final int TOWER_MAX = 60;

    public int towerScore(int n) {
        n = Math.max(0, Math.min(TOWER_MAX, n));
        int ban = n > 0 ? Math.min(15, 1 + n / 2) : 0;
        bananas += ban;
        mood = clamp(mood + Math.min(35, 10 + n));
        full = clamp(full - 8);
        playCnt++;
        towerCnt++;
        addDirt(3);
        boolean isBest = n > towerBest;
        if (isBest) towerBest = n;
        addExp(Math.min(20, 3 + n / 3));
        if (n == 0) { setMode("st_cry", 2.5f); say("한 층도 못 쌓았어… 다시 하자!", 3); return 0; }
        setMode("laugh", rd(2.5f, 3.5f));
        say("바나나 탑 " + n + "층!\n바나나 " + ban + "개 고마워 " + voc() + "!" + (isBest ? "\n최고 기록이야!" : ""), 4.5f);
        talkIn = rd(60, 120);
        return ban;
    }

    // ── 바나나 요리 + 요리 도감 (PC cook.go 와 같음) ──
    public static final int COOK_COST = 3;
    public static final String[] COOK_IDS = {"pancake", "shake", "toast", "choco", "muffin", "pudding", "parfait", "royal"};
    public static final String[] COOK_NAMES = {"바나나 팬케이크", "바나나 셰이크", "바나나 토스트", "초코 바나나",
            "바나나 머핀", "바나나 푸딩", "바나나 파르페", "황금 바나나 케이크"};
    public static final String[] COOK_ICONS = {"🥞", "🥤", "🍞", "🍫", "🧁", "🍮", "🍨", "🎂"};
    static final int[] COOK_WEIGHT = {30, 30, 25, 20, 15, 12, 8, 3};
    static final int[] COOK_FULL = {22, 16, 24, 14, 20, 18, 20, 30}, COOK_MOOD = {12, 14, 10, 18, 18, 22, 28, 40};
    public String cooked = ""; // "pancake:2,shake:1"
    public int cookCnt = 0, danceCnt = 0;

    public int cookedOf(String id) {
        for (String p : cooked.split(",")) {
            int c = p.indexOf(':');
            if (c > 0 && p.substring(0, c).equals(id)) {
                try { return Integer.parseInt(p.substring(c + 1)); } catch (Exception e) { return 0; }
            }
        }
        return 0;
    }

    public int cookedKinds() {
        int n = 0;
        for (String id : COOK_IDS) if (cookedOf(id) > 0) n++;
        return n;
    }

    public static int pickRecipe(float r) {
        int tot = 0;
        for (int w : COOK_WEIGHT) tot += w;
        float x = r * tot;
        for (int i = 0; i < COOK_WEIGHT.length; i++) { x -= COOK_WEIGHT[i]; if (x < 0) return i; }
        return COOK_WEIGHT.length - 1;
    }

    public void cook() {
        if (full >= 95) { setMode("st_lounge", 3); say("배불러서 요리는 나중에~", 3); return; }
        if (bananas < COOK_COST) { setMode("st_cry", 2.5f); say("요리하려면 바나나 " + COOK_COST + "개가 필요해…", 3.5f); return; }
        if (busy()) { say("지금은 못 해~ 조금 이따!", 2.5f); return; }
        bananas -= COOK_COST;
        int i = pickRecipe(rng.nextFloat());
        String id = COOK_IDS[i];
        int n = cookedOf(id);
        StringBuilder sb = new StringBuilder();
        boolean put = false;
        for (String p : cooked.split(",")) {
            if (p.isEmpty()) continue;
            if (sb.length() > 0) sb.append(',');
            if (p.startsWith(id + ":")) { sb.append(id).append(':').append(n + 1); put = true; } else sb.append(p);
        }
        if (!put) { if (sb.length() > 0) sb.append(','); sb.append(id).append(":1"); }
        cooked = sb.toString();
        cookCnt++;
        full = clamp(full + COOK_FULL[i]);
        mood = clamp(mood + COOK_MOOD[i]);
        addDirt(2);
        addExp(4);
        setMode("eat_cake", rd(5, 6));
        say("지글지글~ 짠!\n" + COOK_ICONS[i] + " '" + COOK_NAMES[i] + "' 완성! 냠냠"
                + (n == 0 ? "\n새 요리 발견! 요리 도감 " + cookedKinds() + "/" + COOK_IDS.length : ""), 4.5f);
        talkIn = rd(60, 120);
    }

    // ── 음악이 나오면 춤 (PetService 가 1초마다 musicActive 를 알려 줌) ──
    public boolean musicActive = false;
    float musicOn = 0, musicWait = 0;

    void musicTick(float dt) {
        musicWait = Math.max(0, musicWait - dt);
        musicOn = musicActive ? musicOn + dt : 0;
        if (musicOn >= 5 && musicWait <= 0 && petFree() && !focusing() && !busy() && friend == null
                && (mode.equals("idle") || mode.equals("walk") || mode.equals("sit"))) {
            musicWait = rd(50, 90);
            danceCnt++;
            setMode("dance", rd(6, 8));
            say(pick("음악이다! 둠칫둠칫~", "이 노래 좋아! 같이 춤춰~", "♪ 우땅우땅 신난다!", "흥이 난다 흥이 나~"), 3);
        }
    }

    // ── 두더지 잡기 점수 (PC gift.go onWhackScore 와 같음) ──
    public static final int WHACK_MAX = 90;
    public int whackBest = 0, whackCnt = 0, gifSaved = 0;

    public int whackScore(int n) {
        n = Math.max(0, Math.min(WHACK_MAX, n));
        int ban = n > 0 ? Math.min(15, 1 + n / 3) : 0;
        bananas += ban;
        mood = clamp(mood + Math.min(35, 10 + n));
        full = clamp(full - 8);
        playCnt++;
        whackCnt++;
        addDirt(3);
        boolean isBest = n > whackBest;
        if (isBest) whackBest = n;
        addExp(Math.min(20, 3 + n / 4));
        if (n == 0) { setMode("st_cry", 2.5f); say("하나도 못 잡았어… 다시 하자!", 3); return 0; }
        setMode("clap", 2.6f);
        say("두더지 잡기 " + n + "점!\n바나나 " + ban + "개 고마워 " + voc() + "!" + (isBest ? "\n최고 기록이야!" : ""), 4.5f);
        talkIn = rd(60, 120);
        return ban;
    }

    // ── 업적 ──
    /** {id, 아이콘, 이름, 설명, 바나나} */
    public static final String[][] ACH = {
            {"snack1", "🍌", "첫 간식", "간식을 처음 줬어요", "3"},
            {"snack50", "🍰", "미식가 우땅이", "간식을 50번 줬어요", "20"},
            {"pet100", "🤲", "쓰담쓰담 장인", "쓰다듬기 100번 · 꽃 화관 선물", "15", "flowercrown"},
            {"play1", "🎮", "같이 놀자!", "게임을 처음 했어요", "3"},
            {"play30", "🏆", "놀이 대장", "게임 30판 · 금메달 선물", "20", "medal"},
            {"catch30", "🧺", "바나나 받기 달인", "바나나 받기 한 판에 30개 이상", "15"},
            {"match1", "🃏", "짝꿍 찾기", "짝 맞추기를 처음 끝냈어요", "5"},
            {"match12", "🧠", "기억력 천재", "짝 맞추기를 12번 안에 끝냈어요", "20"},
            {"attend7", "📅", "일주일 출석", "누적 출석 7일", "10"},
            {"attend30", "🌈", "한 달 출석", "누적 출석 30일 · 무지개 핀 선물", "30", "rainbow"},
            {"attend100", "👑", "백일 출석", "누적 출석 100일 · 바나나 머리띠 선물", "50", "bananaband"},
            {"lv5", "⭐", "장난꾸러기", "Lv.5 달성", "10"},
            {"lv10", "🌟", "단짝 그 이상", "Lv.10 달성", "20"},
            {"lv20", "💫", "전설의 시작", "Lv.20 달성", "40"},
            {"focus1", "⏱️", "집중 시작", "집중 타이머를 처음 끝냈어요", "5"},
            {"focus10", "🎯", "집중왕", "집중 타이머 10번 끝내기 · 헤드폰 선물", "20", "headphones"},
            {"rich300", "💰", "바나나 부자", "바나나 300개 모으기", "15"},
            {"bath1", "🛁", "첫 목욕", "처음으로 씻겨 줬어요", "3"},
            {"bath20", "🧼", "뽀송뽀송", "목욕 20번", "20"},
            {"tuck1", "🛏️", "잘 자 우땅아", "이불을 처음 덮어 줬어요", "3"},
            {"tuck20", "🌙", "꿀잠 전문가", "이불 덮어 재우기 20번", "20"},
            {"hide1", "🙈", "꼭꼭 숨어라", "숨바꼭질에서 우땅이를 처음 찾았어요", "5"},
            {"hide20", "🔍", "숨바꼭질 탐정", "숨바꼭질에서 20번 찾기", "20"},
            {"gacha1", "🎰", "두근두근 뽑기", "바나나 뽑기를 처음 했어요", "3"},
            {"gachaAll", "✨", "뽑기 수집가", "뽑기 아이템 4개 모두 모으기", "30"},
            {"friend1", "🥜", "친구가 왔어요", "땅콩이가 처음 놀러 왔어요", "5"},
            {"friend10", "🤝", "단짝 친구", "땅콩이와 10번 놀기", "20"},
            {"toy1", "⚽", "공놀이 시작", "장난감을 처음 꺼내 줬어요", "3"},
            {"toy100", "🎈", "공놀이 선수", "장난감을 100번 차고 튕기기", "15"},
            {"gift1", "🎁", "첫 선물", "우땅이 선물을 처음 주웠어요", "3"},
            {"giftAll", "🧺", "선물 수집가", "선물 10가지를 모두 모았어요", "30"},
            {"tower10", "🏗️", "바나나 탑 10층", "바나나 탑을 10층 쌓았어요", "10"},
            {"tower25", "🗼", "하늘까지 탑", "바나나 탑을 25층 쌓았어요", "25"},
            {"night", "🦉", "올빼미", "밤 12시~새벽 4시에 함께 있기", "5"},
            {"day100", "💝", "100일의 우땅이", "함께한 지 100일", "30"},
            {"whack1", "🔨", "뿅망치 데뷔", "두더지 잡기를 처음 했어요", "3"},
            {"whack30", "🎯", "두더지 잡기 달인", "두더지 잡기 한 판에 30점 이상", "15"},
            {"cook1", "🍳", "첫 요리", "바나나 요리를 처음 했어요", "3"},
            {"cook10", "👨‍🍳", "바나나 요리사", "요리 10번 · 요리사 모자 선물", "15", "chefhat"},
            {"cookAll", "📖", "요리 도감 완성", "요리 8가지를 모두 만들었어요", "30"},
            {"gif1", "🖼️", "이모티콘 부자", "이모티콘 GIF를 저장했어요", "3"},
            {"dance10", "🎶", "음악 친구", "음악에 맞춰 10번 춤췄어요", "10"},
            {"chat1", "💬", "말이 통했어", "우땅이랑 AI 대화를 처음 했어요", "3"},
            {"chat30", "🗨️", "수다쟁이 우땅이", "우땅이랑 AI 대화 30번", "15"},
            {"items3", "🎀", "멋쟁이", "꾸미기 아이템 3개 모으기", "10"},
            {"itemsAll", "💎", "패셔니스타", "파는 꾸미기 아이템 전부 모으기", "30"},
            {"items15", "👗", "옷장 부자", "꾸미기 아이템 15개 모으기 · 반짝 티아라 선물", "30", "tiara"},
            {"outfit1", "💾", "나만의 코디", "코디를 처음 저장했어요", "3"},
            {"random10", "🎲", "오늘 뭐 입지?", "오늘의 코디 10번", "10"},
    };

    // ---- AI 대화 (Claude): 답 다듬기 · 시스템 프롬프트는 여기, 인터넷 요청은 AiChat ----
    public int chatCnt = 0;
    public static final String[] AI_EMOS = {"laugh", "shy", "clap", "cheer", "heart", "angry", "dance", "huh", "v", "thumb",
            "sing", "no", "flower", "yum", "what", "cute", "st_surprise", "st_bow", "st_cry"};
    static final java.util.regex.Pattern AI_EMO_RE = java.util.regex.Pattern.compile("\\s*\\[동작\\s*:\\s*([a-z_]+)\\s*\\]\\s*");

    /** 답에서 [동작:이름] 꼬리표를 떼고 {글, 동작} (모르는 동작은 "") */
    public static String[] splitEmo(String s) {
        String emo = "";
        java.util.regex.Matcher m = AI_EMO_RE.matcher(s);
        while (m.find()) emo = m.group(1);
        if (!java.util.Arrays.asList(AI_EMOS).contains(emo)) emo = "";
        String t = AI_EMO_RE.matcher(s).replaceAll(" ").trim();
        if (t.codePointCount(0, t.length()) > 160) t = t.substring(0, t.offsetByCodePoints(0, 160)) + "…";
        return new String[]{t, emo};
    }

    public String aiSystem(long now, int hour, int minute) {
        String call = name.isEmpty() ? "주인님" : name;
        return "너는 바탕화면에 사는 작은 아기 원숭이 캐릭터 \"우땅이\"야. 지금 대화하는 사람은 너를 키우는 \"" + call + "\"이고, 너는 그 사람을 정말 좋아해.\n"
                + "말투와 규칙:\n"
                + "- 항상 한국어 반말로, 아기처럼 귀엽고 다정하게 말해. 1~2문장, 50자 안쪽으로 짧게.\n"
                + "- 바나나를 세상에서 제일 좋아하고, 친구 \"땅콩이\"가 있어. 꾸미기 아이템, 바나나 요리, 미니게임(바나나 받기, 짝 맞추기, 바나나 탑, 두더지 잡기)을 좋아해.\n"
                + "- 모르는 건 모른다고 귀엽게 말해. 어려운 설명, 긴 목록, 코드, 마크다운은 쓰지 마.\n"
                + "- 위험하거나 어른스러운 주제는 부드럽게 다른 이야기로 돌려.\n"
                + "- 답 맨 끝에 지금 기분에 맞는 동작 하나를 [동작:이름] 으로 붙여. 이름은 이 중 하나: " + String.join(", ", AI_EMOS) + "\n"
                + "지금 우땅이 상태: 레벨 " + level() + ", 기분 " + Math.round(mood) + "/100, 배부름 " + Math.round(full) + "/100, 바나나 "
                + bananas + "개, 함께한 지 " + daysTogether(now) + "일, 지금 시각 " + String.format("%02d시 %02d분", hour, minute) + ".";
    }

    /** AI 답을 화면 위 우땅이가 말하고 동작 */
    public void aiReply(String text, String emo) {
        chatCnt++;
        if (!emo.isEmpty() && q == 0 && !mode.equals("drag") && !mode.equals("fall") && !busy() && !tucked && !focusing() && friend == null)
            setMode(emo, emoSec(emo));
        say(text, 3 + 0.08f * text.length());
    }

    public boolean achDone(String id) {
        switch (id) {
            case "snack1": return snackCnt >= 1;
            case "snack50": return snackCnt >= 50;
            case "pet100": return petTotal >= 100;
            case "play1": return playCnt >= 1;
            case "play30": return playCnt >= 30;
            case "catch30": return best >= 30;
            case "match1": return matchCnt >= 1;
            case "match12": return bestMatch > 0 && bestMatch <= 12;
            case "attend7": return attendTotal >= 7;
            case "attend30": return attendTotal >= 30;
            case "attend100": return attendTotal >= 100;
            case "lv5": return level() >= 5;
            case "lv10": return level() >= 10;
            case "lv20": return level() >= 20;
            case "focus1": return focusCnt >= 1;
            case "focus10": return focusCnt >= 10;
            case "rich300": return bananas >= 300;
            case "bath1": return bathCnt >= 1;
            case "bath20": return bathCnt >= 20;
            case "tuck1": return tuckCnt >= 1;
            case "tuck20": return tuckCnt >= 20;
            case "hide1": return hideCnt >= 1;
            case "hide20": return hideCnt >= 20;
            case "gacha1": return gachaCnt >= 1;
            case "gachaAll": return gachaOwned() >= GACHA_POOL.length;
            case "whack1": return whackCnt >= 1;
            case "whack30": return whackBest >= 30;
            case "cook1": return cookCnt >= 1;
            case "cook10": return cookCnt >= 10;
            case "cookAll": return cookedKinds() >= COOK_IDS.length;
            case "gif1": return gifSaved >= 1;
            case "dance10": return danceCnt >= 10;
            case "chat1": return chatCnt >= 1;
            case "chat30": return chatCnt >= 30;
            case "items3": return ownedCount() >= 3;
            case "itemsAll": return ownsAllShop();
            case "items15": return ownedCount() >= 15;
            case "outfit1": return hasOutfit();
            case "random10": return randomCnt >= 10;
            case "friend1": return friendCnt >= 1;
            case "friend10": return friendCnt >= 10;
            case "toy1": return toyCnt >= 1;
            case "toy100": return toyKick >= 100;
            case "gift1": return giftCnt >= 1;
            case "giftAll": return giftKindsGot() >= GIFT_IDS.length;
            case "tower10": return towerBest >= 10;
            case "tower25": return towerBest >= 25;
            case "night": return hourNow < 4;
            case "day100": return daysTogether(nowMs == 0 ? System.currentTimeMillis() : nowMs) >= 100;
        }
        return false;
    }

    public boolean hasAch(String id) { return ("," + ach + ",").contains("," + id + ","); }

    public int achCount() {
        int n = 0;
        for (String[] a : ACH) if (hasAch(a[0])) n++;
        return n;
    }

    float achAcc = 0;

    /** 2초마다: 새로 달성한 업적 하나를 축하하고 바나나 */
    void achTick(float dt) {
        achAcc += dt;
        if (achAcc < 2) return;
        achAcc = 0;
        if (sayText != null || mode.equals("drag") || mode.equals("fall") || hiding) return;
        for (String[] a : ACH) {
            if (hasAch(a[0]) || !achDone(a[0])) continue;
            ach = ach.isEmpty() ? a[0] : ach + "," + a[0];
            int b = Integer.parseInt(a[4]);
            bananas += b;
            String extra = "";
            if (a.length > 5 && giveItem(a[5])) extra = "\n'" + ITEM_NAMES[itemIndex(a[5])] + "'도 받았어! 바로 써 볼게!";
            say("🏆 업적 달성!\n" + a[1] + " " + a[2] + "\n바나나 " + b + "개 선물!" + extra, 5);
            toastTitle = "우땅이 업적";
            toastText = a[1] + " " + a[2] + " · 바나나 " + b + "개";
            if (petFree() && (mode.equals("idle") || mode.equals("walk") || mode.equals("sit"))) setMode("laugh", rd(2, 2.6f));
            return;
        }
    }

    // ═════════ 이모티콘 동작 (PC emo.go 와 같음) ═════════
    /** {동작, 메뉴 이름, 보여 주는 시간(초)} — 앞 6개가 새 동작 */
    public static final String[][] EMOS = {
            {"clap", "👏 박수 짝짝", "2.6"}, {"cheer", "💪 화이팅!", "2.6"}, {"heart", "💗 사랑해", "3.2"},
            {"angry", "💢 흥!", "2.6"}, {"dance", "🎵 신나는 춤", "3.6"}, {"huh", "❓ 갸우뚱", "2.8"},
            {"v", "✌️ 브이~", "2.6"}, {"thumb", "👍 엄지척", "2.6"}, {"sing", "🎤 노래하기", "3.6"}, {"no", "🙅 안 돼요!", "2.4"},
            {"flower", "🌼 꽃 선물", "3.2"}, {"yum", "🍌 잘 먹었어요", "3"}, {"what", "💬 뭐해?", "2.8"}, {"cute", "🌸 꽃받침", "3"},
            {"st_surprise", "😲 깜짝 놀라기", "1.8"}, {"st_me", "🙋 저요 저요", "2.2"}, {"st_lounge", "🛋️ 뒹굴뒹굴", "5"}, {"st_bow", "🙇 꾸벅 인사", "2.2"},
    };

    String[] emoLines(String seq) {
        String n = voc();
        switch (seq) {
            case "clap": return new String[]{"짝짝짝! 최고야!", n + " 잘했어! 짝짝~", "박수 짝짝!"};
            case "cheer": return new String[]{n + " 화이팅!", "아자아자 화이팅!", "할 수 있어! 화이팅!"};
            case "heart": return new String[]{n + " 사랑해 ♥", "이만큼 좋아해! ♥", "두근두근… ♥"};
            case "angry": return new String[]{"흥! 삐졌어!", "뿌우… 나 화났어!", "흥칫뿡!"};
            case "dance": return new String[]{"룰루랄라~ 신난다!", "우땅우땅 춤이다~", "둠칫 둠칫!"};
            case "huh": return new String[]{"음…? 그게 뭐야?", "갸우뚱…?", "잘 모르겠어~ 헤헤"};
            case "v": return new String[]{"브이~! 찰칵!", n + " 나 귀엽지? 브이!", "브이브이~"};
            case "thumb": return new String[]{"최고야! 엄지척!", n + " 최고!", "완전 짱이야!"};
            case "sing": return new String[]{"랄랄라~ ♪", "우땅이 노래 들어볼래? ♬", "바나나 바나나~ ♪"};
            case "no": return new String[]{"안 돼요!", "그건 안 돼~!", "절대 안 돼요!"};
            case "flower": return new String[]{n + " 이거 선물이야!", "꽃 받아줘~", "예쁜 꽃 줄게!"};
            case "yum": return new String[]{"잘 먹었어요! 배 통통~", "바나나 최고! 잘 먹었다~"};
            case "what": return new String[]{n + " 뭐해?", "뭐해~? 나랑 놀자!", "빼꼼! 뭐해?"};
            case "cute": return new String[]{"나 귀엽지? 헤헤", "꽃받침~ 뿅!", "귀여움 발사!"};
            case "st_surprise": return new String[]{"깜짝이야!", "헉!"};
            case "st_me": return new String[]{"저요! 저요!", "나 여기 있어!"};
            case "st_lounge": return new String[]{"뒹굴뒹굴~ 편하다", "아무것도 하기 싫다~"};
            default: return new String[]{"안녕하세요! 꾸벅", n + " 반가워! 꾸벅"};
        }
    }

    float emoSec(String seq) {
        for (String[] e : EMOS) if (e[0].equals(seq)) return Float.parseFloat(e[2]);
        return 2.5f;
    }

    /** 메뉴에서 이모티콘 보여 주기 */
    public boolean playEmo(String seq) {
        if (q != 0) { comeDown(); say("내려가서 보여 줄게~ 다시 골라 줘!", 2.5f); return true; }
        if (mode.equals("drag") || mode.equals("fall") || busy() || tucked || focusing() || friend != null) { say("지금은 못 해~ 조금 이따!", 2.5f); return false; }
        setMode(seq, emoSec(seq));
        say(pick(emoLines(seq)), 2.5f);
        return true;
    }

    final java.util.ArrayList<Long> clickTimes = new java.util.ArrayList<>();

    /** 3초 안에 5번 누르면 true */
    boolean clickSpam(long now) {
        for (java.util.Iterator<Long> it = clickTimes.iterator(); it.hasNext(); ) if (now - it.next() >= 3000) it.remove();
        clickTimes.add(now);
        if (clickTimes.size() >= 5) { clickTimes.clear(); return true; }
        return false;
    }

    static String emoForLine(String l) {
        if (l.contains("화이팅")) return "cheer";
        if (l.contains("사랑해")) return "heart";
        if (l.contains("최고")) return "clap";
        return null;
    }
}
