package com.woottangi.pet;

import android.content.res.AssetManager;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.util.LruCache;

import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;

/** assets/{폴더}/{동작}_{번호}.webp 그림 불러오기 (최근 것만 메모리에 둠) */
public class Frames {
    private final AssetManager am;
    private final String dir;
    private final Map<String, Integer> counts = new HashMap<>();
    private final LruCache<String, Bitmap> cache;

    public Frames(AssetManager am, String dir) {
        this.am = am;
        this.dir = dir;
        cache = new LruCache<String, Bitmap>(dir.equals("frames") ? 24 * 1024 * 1024 : 8 * 1024 * 1024) {
            @Override protected int sizeOf(String k, Bitmap b) { return b.getByteCount(); }
        };
        try {
            for (String f : am.list(dir)) {
                int u = f.lastIndexOf('_');
                if (u <= 0 || !f.endsWith(".webp")) continue;
                String seq = f.substring(0, u);
                Integer c = counts.get(seq);
                counts.put(seq, c == null ? 1 : c + 1);
            }
        } catch (Exception ignored) { }
    }

    public int count(String seq) {
        Integer c = counts.get(seq);
        return c == null ? 0 : c;
    }

    public Bitmap get(String seq, int i) {
        if (count(seq) == 0) seq = "idle";
        String k = seq + "_" + i;
        Bitmap b = cache.get(k);
        if (b != null) return b;
        try (InputStream in = am.open(dir + "/" + k + ".webp")) {
            b = BitmapFactory.decodeStream(in);
        } catch (Exception e) {
            return null;
        }
        if (b != null) cache.put(k, b);
        return b;
    }
}
