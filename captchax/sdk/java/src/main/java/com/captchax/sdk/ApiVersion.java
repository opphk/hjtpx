package com.captchax.sdk;

public enum ApiVersion {
    V1("v1"),
    V2("v2");

    private final String value;

    ApiVersion(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }
}
