package com.woottangi.pet;

import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.net.Uri;
import android.os.Build;
import android.os.Environment;
import android.provider.MediaStore;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;

/** 이모티콘 GIF 를 사진 앨범 "우땅이 이모티콘" 에 저장 (PC emo.go saveEmoGifs 와 같은 파일) */
public final class EmoGif {
    public static final String ALBUM = "우땅이 이모티콘";

    private EmoGif() { }

    /** 저장한 개수 (실패하면 0). 안드로이드 10 이상은 권한 없이 앨범에 바로, 그 아래는 앱 사진 폴더에 */
    public static int saveAll(Context c) {
        int n = 0;
        try (BufferedReader r = new BufferedReader(new InputStreamReader(c.getAssets().open("emo/names.txt"), StandardCharsets.UTF_8))) {
            String line;
            while ((line = r.readLine()) != null) {
                String[] p = line.split("\t");
                if (p.length != 2) continue;
                if (save(c, "emo/" + p[0] + ".gif", "우땅이_" + p[1] + ".gif")) n++;
            }
        } catch (Exception e) {
            return n;
        }
        return n;
    }

    static boolean save(Context c, String asset, String file) {
        try (InputStream in = c.getAssets().open(asset)) {
            if (Build.VERSION.SDK_INT >= 29) {
                ContentResolver cr = c.getContentResolver();
                Uri col = MediaStore.Images.Media.EXTERNAL_CONTENT_URI;
                String rel = Environment.DIRECTORY_PICTURES + "/" + ALBUM;
                // 예전에 저장한 같은 이름이 있으면 지우고 새로 (앨범에 _1, _2 가 쌓이지 않게)
                cr.delete(col, MediaStore.Images.Media.DISPLAY_NAME + "=? AND " + MediaStore.Images.Media.RELATIVE_PATH + "=?",
                        new String[]{file, rel + "/"});
                ContentValues v = new ContentValues();
                v.put(MediaStore.Images.Media.DISPLAY_NAME, file);
                v.put(MediaStore.Images.Media.MIME_TYPE, "image/gif");
                v.put(MediaStore.Images.Media.RELATIVE_PATH, rel);
                v.put(MediaStore.Images.Media.IS_PENDING, 1);
                Uri u = cr.insert(col, v);
                if (u == null) return false;
                try (OutputStream out = cr.openOutputStream(u)) {
                    if (out == null) return false;
                    copy(in, out);
                }
                v.clear();
                v.put(MediaStore.Images.Media.IS_PENDING, 0);
                cr.update(u, v, null, null);
                return true;
            }
            File dir = new File(c.getExternalFilesDir(Environment.DIRECTORY_PICTURES), ALBUM);
            if (!dir.isDirectory() && !dir.mkdirs()) return false;
            try (OutputStream out = new FileOutputStream(new File(dir, file))) {
                copy(in, out);
            }
            return true;
        } catch (Exception e) {
            return false;
        }
    }

    static void copy(InputStream in, OutputStream out) throws java.io.IOException {
        byte[] b = new byte[16384];
        int k;
        while ((k = in.read(b)) > 0) out.write(b, 0, k);
    }
}
