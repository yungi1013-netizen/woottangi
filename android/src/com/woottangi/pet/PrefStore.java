package com.woottangi.pet;

import android.content.Context;
import android.content.SharedPreferences;

/** 우땅이 기록을 폰에 저장 (SharedPreferences) */
public class PrefStore implements Pet.Store {
    public final SharedPreferences sp;
    private SharedPreferences.Editor ed;

    public PrefStore(Context c) { sp = c.getSharedPreferences("woottangi", Context.MODE_PRIVATE); }

    public int getInt(String k, int d) { try { return sp.getInt(k, d); } catch (Exception e) { return d; } }
    public float getFloat(String k, float d) { try { return sp.getFloat(k, d); } catch (Exception e) { return d; } }
    public long getLong(String k, long d) { try { return sp.getLong(k, d); } catch (Exception e) { return d; } }
    public String getString(String k, String d) { try { return sp.getString(k, d); } catch (Exception e) { return d; } }

    public void put(String k, Object v) {
        if (ed == null) ed = sp.edit();
        if (v instanceof Integer) ed.putInt(k, (Integer) v);
        else if (v instanceof Float) ed.putFloat(k, (Float) v);
        else if (v instanceof Long) ed.putLong(k, (Long) v);
        else if (v instanceof Boolean) ed.putBoolean(k, (Boolean) v);
        else ed.putString(k, String.valueOf(v));
    }

    public void commit() {
        if (ed != null) { ed.apply(); ed = null; }
    }
}
