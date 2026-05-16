package com.hjtpx.sdk;

public enum CaptchaType {
    NUMBER("number"),
    LETTER("letter"),
    MIXED("mixed");

    private final String value;

    CaptchaType(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }

    @Override
    public String toString() {
        return value;
    }
}
