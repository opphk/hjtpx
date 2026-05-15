<?php
/**
 * Plugin Name: CaptchaX - 行为验证码
 * Plugin URI: https://github.com/opphk/hjtpx
 * Description: CaptchaX 现代化行为验证码系统，支持滑块、点选、拼图等多种验证方式
 * Version: 1.0.0
 * Author: CaptchaX Team
 * Author URI: https://captchax.com
 * License: GPL v2 or later
 * Text Domain: captchax
 * Domain Path: /languages
 */

if (!defined('ABSPATH')) {
    exit;
}

define('CAPTCHAX_VERSION', '1.0.0');
define('CAPTCHAX_PLUGIN_DIR', plugin_dir_path(__FILE__));
define('CAPTCHAX_PLUGIN_URL', plugin_dir_url(__FILE__));
define('CAPTCHAX_PLUGIN_BASENAME', plugin_basename(__FILE__));

class CaptchaX {
    
    protected $loader;
    protected $plugin_name;
    protected $version;
    
    public function __construct() {
        $this->version = CAPTCHAX_VERSION;
        $this->plugin_name = 'captchax';
        $this->load_dependencies();
        $this->define_admin_hooks();
        $this->define_public_hooks();
    }
    
    private function load_dependencies() {
        require_once CAPTCHAX_PLUGIN_DIR . 'includes/class-captchax-loader.php';
        require_once CAPTCHAX_PLUGIN_DIR . 'admin/class-captchax-admin.php';
        require_once CAPTCHAX_PLUGIN_DIR . 'includes/class-captchax-public.php';
        
        $this->loader = new CaptchaX_Loader();
    }
    
    private function define_admin_hooks() {
        $admin = new CaptchaX_Admin($this->get_plugin_name(), $this->get_version());
        
        $this->loader->add_action('admin_menu', $admin, 'add_plugin_admin_menu');
        $this->loader->add_action('admin_enqueue_scripts', $admin, 'enqueue_styles');
        $this->loader->add_action('admin_enqueue_scripts', $admin, 'enqueue_scripts');
    }
    
    private function define_public_hooks() {
        $public = new CaptchaX_Public($this->get_plugin_name(), $this->get_version());
        
        $this->loader->add_action('comment_form_after_fields', $public, 'add_captcha_to_comment_form');
        $this->loader->add_filter('preprocess_comment', $public, 'verify_comment_captcha');
        
        $this->loader->add_action('login_form', $public, 'add_captcha_to_login_form');
        $this->loader->add_filter('authenticate', $public, 'verify_login_captcha', 30, 3);
        
        $this->loader->add_action('register_form', $public, 'add_captcha_to_register_form');
        $this->loader->add_filter('registration_errors', $public, 'verify_register_captcha', 10, 3);
        
        $this->loader->add_action('lostpassword_form', $public, 'add_captcha_to_lostpassword_form');
        $this->loader->add_filter('allow_password_reset', $public, 'verify_lostpassword_captcha');
        
        $this->loader->add_action('wp_enqueue_scripts', $public, 'enqueue_styles');
        $this->loader->add_action('wp_enqueue_scripts', $public, 'enqueue_scripts');
        
        $this->loader->add_action('wp_ajax_captchax_verify', $public, 'ajax_verify');
        $this->loader->add_action('wp_ajax_nopriv_captchax_verify', $public, 'ajax_verify');
    }
    
    public function run() {
        $this->loader->run();
    }
    
    public function get_plugin_name() {
        return $this->plugin_name;
    }
    
    public function get_version() {
        return $this->version;
    }
}

function run_captchax() {
    $plugin = new CaptchaX();
    $plugin->run();
}
run_captchax();

register_activation_hook(__FILE__, 'captchax_activate');
function captchax_activate() {
    add_option('captchax_api_key', '');
    add_option('captchax_api_secret', '');
    add_option('captchax_server_url', 'https://api.captchax.com');
    add_option('captchax_enabled_forms', ['comment', 'login', 'register', 'lostpassword']);
    add_option('captchax_theme', 'light');
    add_option('captchax_captcha_type', 'slider');
}

register_deactivation_hook(__FILE__, 'captchax_deactivate');
function captchax_deactivate() {
}
