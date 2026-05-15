<?php

class CaptchaX_API {
    
    private $api_key;
    private $api_secret;
    private $server_url;
    
    public function __construct() {
        $this->api_key = get_option('captchax_api_key', '');
        $this->api_secret = get_option('captchax_api_secret', '');
        $this->server_url = get_option('captchax_server_url', 'https://api.captchax.com');
    }
    
    public function verify($token, $scene = 'default') {
        $timestamp = time();
        $signature = $this->generate_signature($token, $timestamp);
        
        $response = wp_remote_post($this->server_url . '/api/v2/verify', [
            'headers' => [
                'Content-Type' => 'application/json',
                'X-API-Key' => $this->api_key,
                'X-Timestamp' => $timestamp,
                'X-Signature' => $signature
            ],
            'body' => json_encode([
                'token' => $token,
                'scene' => $scene,
                'ip' => $this->get_client_ip(),
                'userAgent' => $_SERVER['HTTP_USER_AGENT'] ?? ''
            ]),
            'timeout' => 30
        ]);
        
        if (is_wp_error($response)) {
            return [
                'success' => false,
                'error' => $response->get_error_message()
            ];
        }
        
        $body = json_decode(wp_remote_retrieve_body($response), true);
        return $body;
    }
    
    private function generate_signature($token, $timestamp) {
        $data = $token . ':' . $timestamp;
        return hash_hmac('sha256', $data, $this->api_secret);
    }
    
    private function get_client_ip() {
        $ip = '';
        if (!empty($_SERVER['HTTP_CLIENT_IP'])) {
            $ip = $_SERVER['HTTP_CLIENT_IP'];
        } elseif (!empty($_SERVER['HTTP_X_FORWARDED_FOR'])) {
            $ip = $_SERVER['HTTP_X_FORWARDED_FOR'];
        } else {
            $ip = $_SERVER['REMOTE_ADDR'] ?? '';
        }
        return $ip;
    }
    
    public function get_script_url() {
        return $this->server_url . '/captcha.js';
    }
    
    public function get_app_id() {
        return $this->api_key;
    }
}
