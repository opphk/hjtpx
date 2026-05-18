$(function() {
    const Toast = Swal.mixin({
        toast: true,
        position: 'top-end',
        showConfirmButton: false,
        timer: 3000,
        timerProgressBar: true
    });

    const defaultConfig = {
        brand_name: '墨盾验证',
        primary_color: '#007bff',
        success_color: '#28a745',
        warning_color: '#ffc107',
        danger_color: '#dc3545',
        custom_css: '',
        is_enabled: false,
        logo_url: '',
        favicon_url: ''
    };

    let currentConfig = { ...defaultConfig };

    // 颜色输入双向绑定
    function setupColorBindings() {
        const colors = ['primary', 'success', 'warning', 'danger'];
        colors.forEach(color => {
            $(`#${color}Color`).on('input', function() {
                $(`#${color}ColorText`).val(this.value);
                updatePreview();
            });
            $(`#${color}ColorText`).on('input', function() {
                if (/^#[0-9A-Fa-f]{6}$/.test(this.value) || /^#[0-9A-Fa-f]{3}$/.test(this.value)) {
                    $(`#${color}Color`).val(this.value);
                    updatePreview();
                }
            });
        });
    }

    // 更新预览区域
    function updatePreview() {
        const primary = $('#primaryColor').val();
        const success = $('#successColor').val();
        const warning = $('#warningColor').val();
        const danger = $('#dangerColor').val();

        // 更新按钮
        $('#previewPrimary').css({
            'background-color': primary,
            'border-color': primary,
            'color': '#fff'
        });
        $('#previewSuccess').css({
            'background-color': success,
            'border-color': success,
            'color': '#fff'
        });
        $('#previewWarning').css({
            'background-color': warning,
            'border-color': warning,
            'color': '#1f2d3d'
        });
        $('#previewDanger').css({
            'background-color': danger,
            'border-color': danger,
            'color': '#fff'
        });

        // 更新徽章
        $('#previewBadgePrimary').css({
            'background-color': primary,
            'color': '#fff'
        });
        $('#previewBadgeSuccess').css({
            'background-color': success,
            'color': '#fff'
        });
        $('#previewBadgeWarning').css({
            'background-color': warning,
            'color': '#1f2d3d'
        });
        $('#previewBadgeDanger').css({
            'background-color': danger,
            'color': '#fff'
        });
    }

    // 加载配置
    function loadConfig() {
        $.ajax({
            url: '/admin/api/whitelabel',
            method: 'GET',
            dataType: 'json',
            success: function(response) {
                if (response.code === 0 && response.data) {
                    currentConfig = { ...defaultConfig, ...response.data };
                    applyConfigToForm(currentConfig);
                    updatePreview();
                }
            },
            error: function() {
                console.log('加载配置失败，使用默认配置');
                applyConfigToForm(defaultConfig);
                updatePreview();
            }
        });
    }

    // 应用配置到表单
    function applyConfigToForm(config) {
        $('#isEnabled').prop('checked', config.is_enabled);
        $('#brandName').val(config.brand_name);
        $('#primaryColor').val(config.primary_color);
        $('#primaryColorText').val(config.primary_color);
        $('#successColor').val(config.success_color);
        $('#successColorText').val(config.success_color);
        $('#warningColor').val(config.warning_color);
        $('#warningColorText').val(config.warning_color);
        $('#dangerColor').val(config.danger_color);
        $('#dangerColorText').val(config.danger_color);
        $('#customCSS').val(config.custom_css);

        // 更新Logo预览
        if (config.logo_url) {
            $('#logoImage').attr('src', config.logo_url).show();
        }
        if (config.favicon_url) {
            $('#faviconImage').attr('src', config.favicon_url).show();
        }
    }

    // 上传Logo
    async function uploadLogo(file, type) {
        const formData = new FormData();
        formData.append('file', file);

        return $.ajax({
            url: `/admin/api/whitelabel/logo/${type}`,
            method: 'POST',
            data: formData,
            processData: false,
            contentType: false
        });
    }

    // 删除Logo
    async function deleteLogo(type) {
        return $.ajax({
            url: `/admin/api/whitelabel/logo/${type}`,
            method: 'DELETE'
        });
    }

    // 保存配置
    async function saveConfig() {
        const config = {
            brand_name: $('#brandName').val(),
            primary_color: $('#primaryColor').val(),
            success_color: $('#successColor').val(),
            warning_color: $('#warningColor').val(),
            danger_color: $('#dangerColor').val(),
            custom_css: $('#customCSS').val(),
            is_enabled: $('#isEnabled').is(':checked')
        };

        return $.ajax({
            url: '/admin/api/whitelabel',
            method: 'PUT',
            contentType: 'application/json',
            data: JSON.stringify(config)
        });
    }

    // 表单提交处理
    $('#whitelabelForm').on('submit', async function(e) {
        e.preventDefault();

        const $btn = $('#saveBtn');
        const originalText = $btn.html();
        $btn.html('<i class="fas fa-spinner fa-spin mr-2"></i>保存中...').prop('disabled', true);

        try {
            // 处理Logo
            const logoFile = $('#logoFile')[0].files[0];
            if (logoFile) {
                await uploadLogo(logoFile, 'logo');
            } else if ($('#deleteLogo').is(':checked')) {
                await deleteLogo('logo');
            }

            // 处理Favicon
            const faviconFile = $('#faviconFile')[0].files[0];
            if (faviconFile) {
                await uploadLogo(faviconFile, 'favicon');
            } else if ($('#deleteFavicon').is(':checked')) {
                await deleteLogo('favicon');
            }

            // 保存配置
            const response = await saveConfig();
            if (response.code === 0) {
                Toast.fire({
                    icon: 'success',
                    title: '配置保存成功'
                });
                loadConfig();
            } else {
                Toast.fire({
                    icon: 'error',
                    title: '配置保存失败: ' + (response.message || '未知错误')
                });
            }
        } catch (xhr) {
            Toast.fire({
                icon: 'error',
                title: '请求失败: ' + (xhr.responseJSON?.message || xhr.statusText)
            });
        } finally {
            $btn.html(originalText).prop('disabled', false);
        }
    });

    // 重置按钮
    $('#resetBtn').on('click', function() {
        Swal.fire({
            title: '确认重置',
            text: '确定要将所有主题配置重置为默认值吗？',
            icon: 'warning',
            showCancelButton: true,
            confirmButtonColor: '#3085d6',
            cancelButtonColor: '#d33',
            confirmButtonText: '确定重置',
            cancelButtonText: '取消'
        }).then(function(result) {
            if (result.isConfirmed) {
                $.ajax({
                    url: '/admin/api/whitelabel/reset',
                    method: 'POST',
                    success: function(response) {
                        if (response.code === 0) {
                            Toast.fire({
                                icon: 'success',
                                title: '已重置为默认值'
                            });
                            loadConfig();
                        }
                    },
                    error: function() {
                        Toast.fire({
                            icon: 'error',
                            title: '重置失败'
                        });
                    }
                });
            }
        });
    });

    // 预览按钮
    $('#previewBtn').on('click', function() {
        // 创建临时样式应用到当前页面
        const primary = $('#primaryColor').val();
        const success = $('#successColor').val();
        const warning = $('#warningColor').val();
        const danger = $('#dangerColor').val();

        // 移除旧的临时样式
        $('#whitelabel-preview-style').remove();

        // 创建新的临时样式
        const style = $('<style id="whitelabel-preview-style">').text(`
            :root {
                --primary: ${primary};
                --success: ${success};
                --warning: ${warning};
                --danger: ${danger};
            }
            .btn-primary { background-color: ${primary} !important; border-color: ${primary} !important; }
            .btn-success { background-color: ${success} !important; border-color: ${success} !important; }
            .btn-warning { background-color: ${warning} !important; border-color: ${warning} !important; }
            .btn-danger { background-color: ${danger} !important; border-color: ${danger} !important; }
            .text-primary { color: ${primary} !important; }
            .text-success { color: ${success} !important; }
            .text-warning { color: ${warning} !important; }
            .text-danger { color: ${danger} !important; }
            .bg-primary { background-color: ${primary} !important; }
            .bg-success { background-color: ${success} !important; }
            .bg-warning { background-color: ${warning} !important; }
            .bg-danger { background-color: ${danger} !important; }
            .sidebar-dark-primary { background-color: ${primary} !important; }
        `);
        $('head').append(style);

        Toast.fire({
            icon: 'success',
            title: '预览模式已应用'
        });
    });

    // 初始化
    setupColorBindings();
    loadConfig();
});
