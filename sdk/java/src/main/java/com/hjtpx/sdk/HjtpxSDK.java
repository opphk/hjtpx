package com.hjtpx.sdk;

public class HjtpxSDK {
    private static final String VERSION = "1.0.0";

    public static String getVersion() {
        return VERSION;
    }

    public static void main(String[] args) {
        System.out.println("hjtpx Java SDK v" + VERSION);
        System.out.println("A comprehensive Java SDK for hjtpx captcha services");
    }
}
