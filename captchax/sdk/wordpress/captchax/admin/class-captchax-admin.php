<?php

class CaptchaX_Admin {
    
    private $plugin_name;
    private $version;
    
    public function __construct($plugin_name, $version) {
        $this->plugin_name = $plugin_name;
        $this->version = $version;
    }
    
    public function add_plugin_admin_menu() {
        add_options_page(
            'CaptchaX 设置',
            'CaptchaX',
            'manage_options',
            'captchax',
            [$this, 'display_admin_page']
        );
    }
    
    public function display_admin_page() {
        if (isset($_POST['captchax_save'])) {
            check_admin_referer('captchax_settings');
            
            update_option('captchax_api_key', sanitize_text_field($_POST['api_key'] ?? ''));
            update_option('captchax_api_secret', sanitize_text_field($_POST['api_secret'] ?? ''));
            update_option('captchax_server_url', esc_url_raw($_POST['server_url'] ?? ''));
            update_option('captchax_theme', sanitize_text_field($_POST['theme'] ?? 'light'));
            update_option('captchax_captcha_type', sanitize_text_field($_POST['captcha_type'] ?? 'slider'));
            update_option('captchax_enabled_forms', array_map('sanitize_text_field', $_POST['enabled_forms'] ?? []));
            
            echo '<div class="updated"><p>设置已保存</p></div>';
        }
        
        include CAPTCHAX_PLUGIN_DIR . 'admin/partials/captchax-admin-display.php';
    }
    
    public function enqueue_styles() {
        wp_enqueue_style(
            $this->plugin_name . '-admin',
            CAPTCHAX_PLUGIN_URL . 'admin/css/captchax-admin.css',
            [],
            $this->version
        );
    }
    
    public function enqueue_scripts() {
        wp_enqueue_script(
            $this->plugin_name . '-admin',
            CAPTCHAX_PLUGIN_URL . 'admin/js/captchax-admin.js',
            ['jquery'],
            $this->version,
            true
        );
    }
}
