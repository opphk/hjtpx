<?php

require_once CAPTCHAX_PLUGIN_DIR . 'includes/class-captchax-api.php';

class CaptchaX_Public {
    
    private $plugin_name;
    private $version;
    private $api;
    
    public function __construct($plugin_name, $version) {
        $this->plugin_name = $plugin_name;
        $this->version = $version;
        $this->api = new CaptchaX_API();
    }
    
    public function enqueue_styles() {
        wp_enqueue_style(
            $this->plugin_name,
            CAPTCHAX_PLUGIN_URL . 'public/css/captchax-public.css',
            [],
            $this->version
        );
    }
    
    public function enqueue_scripts() {
        $script_url = $this->api->get_script_url();
        $app_id = $this->api->get_app_id();
        
        wp_enqueue_script(
            'captchax-sdk',
            $script_url,
            [],
            $this->version,
            true
        );
        
        wp_localize_script('captchax-sdk', 'captchaxConfig', [
            'appId' => $app_id,
            'theme' => get_option('captchax_theme', 'light'),
            'captchaType' => get_option('captchax_captcha_type', 'slider'),
            'ajaxUrl' => admin_url('admin-ajax.php'),
            'nonce' => wp_create_nonce('captchax_verify')
        ]);
        
        wp_enqueue_script(
            $this->plugin_name . '-public',
            CAPTCHAX_PLUGIN_URL . 'public/js/captchax-public.js',
            ['captchax-sdk'],
            $this->version,
            true
        );
    }
    
    public function add_captcha_to_comment_form() {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (!in_array('comment', $enabled_forms)) return;
        
        $this->render_captcha('comment');
    }
    
    public function add_captcha_to_login_form() {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (!in_array('login', $enabled_forms)) return;
        
        $this->render_captcha('login');
    }
    
    public function add_captcha_to_register_form() {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (!in_array('register', $enabled_forms)) return;
        
        $this->render_captcha('register');
    }
    
    public function add_captcha_to_lostpassword_form() {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (!in_array('lostpassword', $enabled_forms)) return;
        
        $this->render_captcha('lostpassword');
    }
    
    private function render_captcha($scene) {
        $captcha_type = get_option('captchax_captcha_type', 'slider');
        ?>
        <div class="captchax-container" id="captchax-<?php echo esc_attr($scene); ?>">
            <div id="captchax-<?php echo esc_attr($scene); ?>-element"></div>
            <input type="hidden" name="captchax_token" id="captchax-token-<?php echo esc_attr($scene); ?>" value="">
            <p class="captchax-error" style="display:none;color:#dc3232;">
                <?php _e('请先完成验证', 'captchax'); ?>
            </p>
        </div>
        <script>
            CaptchaX.init({
                element: '#captchax-<?php echo esc_attr($scene); ?>-element',
                scene: '<?php echo esc_attr($scene); ?>',
                theme: '<?php echo esc_attr(get_option('captchax_theme', 'light')); ?>',
                type: '<?php echo esc_attr($captcha_type); ?>',
                onSuccess: function(token) {
                    document.getElementById('captchax-token-<?php echo esc_attr($scene); ?>').value = token;
                    jQuery('.captchax-error').hide();
                },
                onError: function(error) {
                    console.error('CaptchaX Error:', error);
                }
            });
        </script>
        <?php
    }
    
    public function verify_comment_captcha($commentdata) {
        if (!is_user_logged_in()) {
            $enabled_forms = get_option('captchax_enabled_forms', []);
            if (in_array('comment', $enabled_forms)) {
                $token = sanitize_text_field($_POST['captchax_token'] ?? '');
                if (empty($token)) {
                    wp_die(__('请完成验证码', 'captchax'));
                }
                
                $result = $this->api->verify($token, 'comment');
                if (!$result['success']) {
                    wp_die(__('验证码验证失败', 'captchax'));
                }
            }
        }
        return $commentdata;
    }
    
    public function verify_login_captcha($user, $username, $password) {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (in_array('login', $enabled_forms)) {
            $token = sanitize_text_field($_POST['captchax_token'] ?? '');
            if (empty($token)) {
                return new WP_Error('captchax_error', __('请完成验证码', 'captchax'));
            }
            
            $result = $this->api->verify($token, 'login');
            if (!$result['success']) {
                return new WP_Error('captchax_error', __('验证码验证失败', 'captchax'));
            }
        }
        return $user;
    }
    
    public function verify_register_captcha($errors, $sanitized_user_login, $user_email) {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (in_array('register', $enabled_forms)) {
            $token = sanitize_text_field($_POST['captchax_token'] ?? '');
            if (empty($token)) {
                return new WP_Error('captchax_error', __('请完成验证码', 'captchax'));
            }
            
            $result = $this->api->verify($token, 'register');
            if (!$result['success']) {
                return new WP_Error('captchax_error', __('验证码验证失败', 'captchax'));
            }
        }
        return $errors;
    }
    
    public function verify_lostpassword_captcha($allow) {
        $enabled_forms = get_option('captchax_enabled_forms', []);
        if (in_array('lostpassword', $enabled_forms)) {
            $token = sanitize_text_field($_POST['captchax_token'] ?? '');
            if (empty($token)) {
                wp_die(__('请完成验证码', 'captchax'));
            }
            
            $result = $this->api->verify($token, 'lostpassword');
            if (!$result['success']) {
                wp_die(__('验证码验证失败', 'captchax'));
            }
        }
        return $allow;
    }
    
    public function ajax_verify() {
        check_ajax_referer('captchax_verify', 'nonce');
        
        $token = sanitize_text_field($_POST['token'] ?? '');
        $scene = sanitize_text_field($_POST['scene'] ?? 'default');
        
        $result = $this->api->verify($token, $scene);
        
        wp_send_json($result);
    }
}
