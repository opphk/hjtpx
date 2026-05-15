<div class="wrap">
    <h1>CaptchaX 设置</h1>
    
    <form method="post" action="">
        <?php wp_nonce_field('captchax_settings'); ?>
        
        <table class="form-table">
            <tr>
                <th>API Key</th>
                <td>
                    <input type="text" name="api_key" 
                           value="<?php echo esc_attr(get_option('captchax_api_key', '')); ?>" 
                           class="regular-text" />
                </td>
            </tr>
            <tr>
                <th>API Secret</th>
                <td>
                    <input type="password" name="api_secret" 
                           value="<?php echo esc_attr(get_option('captchax_api_secret', '')); ?>" 
                           class="regular-text" />
                </td>
            </tr>
            <tr>
                <th>服务器地址</th>
                <td>
                    <input type="url" name="server_url" 
                           value="<?php echo esc_attr(get_option('captchax_server_url', 'https://api.captchax.com')); ?>" 
                           class="regular-text" />
                </td>
            </tr>
            <tr>
                <th>主题</th>
                <td>
                    <select name="theme">
                        <option value="light" <?php selected('light', get_option('captchax_theme', 'light')); ?>>浅色</option>
                        <option value="dark" <?php selected('dark', get_option('captchax_theme', 'light')); ?>>深色</option>
                    </select>
                </td>
            </tr>
            <tr>
                <th>验证码类型</th>
                <td>
                    <select name="captcha_type">
                        <option value="slider" <?php selected('slider', get_option('captchax_captcha_type', 'slider')); ?>>滑块验证</option>
                        <option value="click" <?php selected('click', get_option('captchax_captcha_type', 'slider')); ?>>点选验证</option>
                        <option value="puzzle" <?php selected('puzzle', get_option('captchax_captcha_type', 'slider')); ?>>拼图验证</option>
                        <option value="rotate" <?php selected('rotate', get_option('captchax_captcha_type', 'slider')); ?>>旋转验证</option>
                        <option value="text" <?php selected('text', get_option('captchax_captcha_type', 'slider')); ?>>文字验证</option>
                    </select>
                </td>
            </tr>
            <tr>
                <th>启用表单</th>
                <td>
                    <label><input type="checkbox" name="enabled_forms[]" value="comment" 
                        <?php checked(in_array('comment', get_option('captchax_enabled_forms', ['comment']))); ?> />
                        评论表单</label><br>
                    <label><input type="checkbox" name="enabled_forms[]" value="login" 
                        <?php checked(in_array('login', get_option('captchax_enabled_forms', ['comment']))); ?> />
                        登录表单</label><br>
                    <label><input type="checkbox" name="enabled_forms[]" value="register" 
                        <?php checked(in_array('register', get_option('captchax_enabled_forms', ['comment']))); ?> />
                        注册表单</label><br>
                    <label><input type="checkbox" name="enabled_forms[]" value="lostpassword" 
                        <?php checked(in_array('lostpassword', get_option('captchax_enabled_forms', ['comment']))); ?> />
                        找回密码表单</label>
                </td>
            </tr>
        </table>
        
        <p class="submit">
            <input type="submit" name="captchax_save" class="button-primary" value="保存设置" />
        </p>
    </form>
</div>
