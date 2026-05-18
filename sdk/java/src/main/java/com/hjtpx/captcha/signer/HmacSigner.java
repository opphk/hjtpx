package com.hjtpx.captcha.signer;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.InvalidKeyException;
import java.security.NoSuchAlgorithmException;
import java.util.Base64;

public class HmacSigner {
    private static final String ALGORITHM = "HmacSHA256";
    private final String secretKey;

    public HmacSigner(String secretKey) {
        this.secretKey = secretKey;
    }

    public String sign(String data) {
        try {
            Mac mac = Mac.getInstance(ALGORITHM);
            SecretKeySpec secretKeySpec = new SecretKeySpec(
                secretKey.getBytes(StandardCharsets.UTF_8),
                ALGORITHM
            );
            mac.init(secretKeySpec);
            byte[] hash = mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
            return Base64.getEncoder().encodeToString(hash);
        } catch (NoSuchAlgorithmException | InvalidKeyException e) {
            throw new RuntimeException("Failed to sign data", e);
        }
    }

    public boolean verify(String data, String signature) {
        String expectedSignature = sign(data);
        return expectedSignature.equals(signature);
    }
}
