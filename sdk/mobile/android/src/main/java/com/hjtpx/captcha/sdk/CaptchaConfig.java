package com.hjtpx.captcha.sdk;

import android.content.Context;
import android.util.DisplayMetrics;
import android.view.Display;
import android.view.WindowManager;

public class CaptchaConfig {
    private final Context context;
    private int captchaWidth;
    private int captchaHeight;
    private boolean enableHapticFeedback;
    private boolean enableSoundEffect;
    private int sliderTrackHeight;
    private int sliderThumbSize;

    public CaptchaConfig(Context context) {
        this.context = context;
        initDefaults();
    }

    private void initDefaults() {
        WindowManager windowManager = (WindowManager) context.getSystemService(Context.WINDOW_SERVICE);
        Display display = windowManager.getDefaultDisplay();
        DisplayMetrics metrics = new DisplayMetrics();
        display.getMetrics(metrics);

        int screenWidth = metrics.widthPixels;
        int screenHeight = metrics.heightPixels;

        this.captchaWidth = Math.min(screenWidth - 32, 320);
        this.captchaHeight = (int) (captchaWidth * 0.6);
        this.enableHapticFeedback = true;
        this.enableSoundEffect = true;
        this.sliderTrackHeight = 40;
        this.sliderThumbSize = 50;
    }

    public int getCaptchaWidth() {
        return captchaWidth;
    }

    public void setCaptchaWidth(int width) {
        this.captchaWidth = width;
    }

    public int getCaptchaHeight() {
        return captchaHeight;
    }

    public void setCaptchaHeight(int height) {
        this.captchaHeight = height;
    }

    public boolean isEnableHapticFeedback() {
        return enableHapticFeedback;
    }

    public void setEnableHapticFeedback(boolean enable) {
        this.enableHapticFeedback = enable;
    }

    public boolean isEnableSoundEffect() {
        return enableSoundEffect;
    }

    public void setEnableSoundEffect(boolean enable) {
        this.enableSoundEffect = enable;
    }

    public int getSliderTrackHeight() {
        return sliderTrackHeight;
    }

    public void setSliderTrackHeight(int height) {
        this.sliderTrackHeight = height;
    }

    public int getSliderThumbSize() {
        return sliderThumbSize;
    }

    public void setSliderThumbSize(int size) {
        this.sliderThumbSize = size;
    }

    public int getDeviceDensity() {
        return context.getResources().getDisplayMetrics().densityDpi;
    }

    public float getDeviceScale() {
        return context.getResources().getDisplayMetrics().density;
    }
}
