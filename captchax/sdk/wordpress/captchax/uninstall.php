<?php
/**
 * CaptchaX Uninstall
 * 
 * This file is executed when the user deletes the plugin.
 * It removes all plugin options and data from the database.
 */

if (!defined('WP_UNINSTALL_PLUGIN')) {
    exit;
}

delete_option('captchax_api_key');
delete_option('captchax_api_secret');
delete_option('captchax_server_url');
delete_option('captchax_theme');
delete_option('captchax_captcha_type');
delete_option('captchax_enabled_forms');
