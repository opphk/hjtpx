(function($) {
    'use strict';
    
    $(document).ready(function() {
        $('.captchax-test-connection').on('click', function() {
            var $button = $(this);
            $button.prop('disabled', true).text('测试中...');
            
            $.ajax({
                url: ajaxurl,
                type: 'POST',
                data: {
                    action: 'captchax_test_connection',
                    nonce: $('#captchax_admin_nonce').val()
                },
                success: function(response) {
                    if (response.success) {
                        alert('连接成功！');
                    } else {
                        alert('连接失败：' + response.data);
                    }
                },
                error: function() {
                    alert('请求失败');
                },
                complete: function() {
                    $button.prop('disabled', false).text('测试连接');
                }
            });
        });
    });
    
})(jQuery);
