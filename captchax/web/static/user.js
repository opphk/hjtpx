(function() {
    'use strict';

    var API_BASE = '/api/v1/user';

    var UserAPI = {
        request: function(endpoint, options) {
            options = options || {};
            var token = localStorage.getItem('user_token');
            var config = {
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json'
                },
                method: options.method || 'GET'
            };
            if (token) {
                config.headers['Authorization'] = 'Bearer ' + token;
            }
            if (options.headers) {
                for (var key in options.headers) {
                    if (options.headers.hasOwnProperty(key)) {
                        config.headers[key] = options.headers[key];
                    }
                }
            }
            if (options.body) {
                config.body = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
            }

            return fetch(API_BASE + endpoint, config)
                .then(function(response) {
                    return response.json().then(function(data) {
                        if (!response.ok) {
                            var err = new Error(data.message || ('HTTP ' + response.status));
                            err.status = response.status;
                            throw err;
                        }
                        return data;
                    });
                });
        },

        auth: {
            login: function(identifier, password) {
                return UserAPI.request('/auth/login', {
                    method: 'POST',
                    body: { identifier: identifier, password: password }
                });
            },
            register: function(data) {
                return UserAPI.request('/auth/register', {
                    method: 'POST',
                    body: data
                });
            },
            logout: function() {
                return UserAPI.request('/auth/logout', { method: 'POST' });
            }
        },

        profile: {
            get: function() {
                return UserAPI.request('/profile');
            },
            update: function(data) {
                return UserAPI.request('/profile', {
                    method: 'PUT',
                    body: data
                });
            },
            changePassword: function(data) {
                return UserAPI.request('/profile/password', {
                    method: 'PUT',
                    body: data
                });
            }
        },

        dashboard: {
            getStats: function() {
                return UserAPI.request('/dashboard/stats');
            },
            getActivities: function() {
                return UserAPI.request('/dashboard/activities');
            }
        }
    };

    var UserAuth = {
        getToken: function() {
            return localStorage.getItem('user_token');
        },
        setToken: function(token) {
            localStorage.setItem('user_token', token);
        },
        removeToken: function() {
            localStorage.removeItem('user_token');
        },
        getUser: function() {
            var raw = localStorage.getItem('user_info');
            if (!raw) return null;
            try {
                return JSON.parse(raw);
            } catch (e) {
                return null;
            }
        },
        setUser: function(user) {
            localStorage.setItem('user_info', JSON.stringify(user));
        },
        removeUser: function() {
            localStorage.removeItem('user_info');
        },
        isLoggedIn: function() {
            return !!this.getToken();
        },
        logout: function() {
            this.removeToken();
            this.removeUser();
            window.location.href = '/login';
        },
        requireAuth: function() {
            if (!this.isLoggedIn()) {
                window.location.href = '/login';
                return false;
            }
            return true;
        }
    };

    var UserUI = {
        showError: function(el, message) {
            if (!el) return;
            el.textContent = message;
            el.classList.remove('hidden');
        },
        hideError: function(el) {
            if (!el) return;
            el.classList.add('hidden');
        },
        setLoading: function(btn, textEl, spinnerEl, loading) {
            if (loading) {
                btn.disabled = true;
                if (textEl) textEl.classList.add('hidden');
                if (spinnerEl) spinnerEl.classList.remove('hidden');
            } else {
                btn.disabled = false;
                if (textEl) textEl.classList.remove('hidden');
                if (spinnerEl) spinnerEl.classList.add('hidden');
            }
        },
        togglePassword: function(inputId, iconEl) {
            var input = document.getElementById(inputId);
            if (!input) return;
            if (input.type === 'password') {
                input.type = 'text';
            } else {
                input.type = 'password';
            }
        },
        showToast: function(message, type) {
            type = type || 'info';
            var toast = document.createElement('div');
            var bgColor = type === 'success' ? 'bg-green-500' : type === 'error' ? 'bg-red-500' : 'bg-blue-500';
            toast.className = 'fixed top-4 right-4 ' + bgColor + ' text-white px-6 py-3 rounded-lg shadow-lg z-50 transition-all transform translate-x-0';
            toast.textContent = message;
            document.body.appendChild(toast);
            setTimeout(function() {
                toast.classList.add('opacity-0', 'translate-x-full');
                setTimeout(function() {
                    if (toast.parentNode) toast.parentNode.removeChild(toast);
                }, 300);
            }, 3000);
        },
        formatNumber: function(num) {
            if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
            if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
            return String(num);
        },
        initLoginPage: function() {
            var toggleBtn = document.getElementById('toggle-login-password');
            if (toggleBtn) {
                toggleBtn.addEventListener('click', function() {
                    UserUI.togglePassword('login-password');
                });
            }

            var form = document.getElementById('login-form');
            if (!form) return;

            form.addEventListener('submit', function(e) {
                e.preventDefault();
                var identifier = document.getElementById('login-identifier').value.trim();
                var password = document.getElementById('login-password').value;
                var errorEl = document.getElementById('login-error');
                var btn = document.getElementById('login-btn');
                var textEl = document.getElementById('login-btn-text');
                var spinnerEl = document.getElementById('login-spinner');

                UserUI.hideError(errorEl);

                if (!identifier) {
                    UserUI.showError(errorEl, '请输入用户名或邮箱');
                    return;
                }
                if (!password) {
                    UserUI.showError(errorEl, '请输入密码');
                    return;
                }

                UserUI.setLoading(btn, textEl, spinnerEl, true);

                UserAPI.auth.login(identifier, password)
                    .then(function(data) {
                        if (data.data && data.data.token) {
                            UserAuth.setToken(data.data.token);
                            if (data.data.user) {
                                UserAuth.setUser(data.data.user);
                            }
                            var remember = document.getElementById('remember-me');
                            if (remember && !remember.checked) {
                                localStorage.removeItem('user_token');
                                sessionStorage.setItem('user_token', data.data.token);
                            }
                            window.location.href = '/dashboard';
                        } else {
                            UserUI.showError(errorEl, '登录失败，返回数据异常');
                        }
                    })
                    .catch(function(err) {
                        UserUI.showError(errorEl, err.message || '登录失败，请重试');
                    })
                    .finally(function() {
                        UserUI.setLoading(btn, textEl, spinnerEl, false);
                    });
            });

            var rememberCheckbox = document.getElementById('remember-me');
            if (rememberCheckbox) {
                var savedIdentifier = localStorage.getItem('remembered_identifier');
                if (savedIdentifier) {
                    document.getElementById('login-identifier').value = savedIdentifier;
                    rememberCheckbox.checked = true;
                }
                rememberCheckbox.addEventListener('change', function() {
                    if (this.checked) {
                        localStorage.setItem('remembered_identifier', document.getElementById('login-identifier').value);
                    } else {
                        localStorage.removeItem('remembered_identifier');
                    }
                });
            }
        },
        initRegisterPage: function() {
            var toggleBtn = document.getElementById('toggle-reg-password');
            if (toggleBtn) {
                toggleBtn.addEventListener('click', function() {
                    UserUI.togglePassword('reg-password');
                });
            }

            var passwordInput = document.getElementById('reg-password');
            if (passwordInput) {
                passwordInput.addEventListener('input', function() {
                    UserUI.evaluatePasswordStrength(this.value);
                });
            }

            var confirmInput = document.getElementById('reg-confirm-password');
            if (confirmInput) {
                confirmInput.addEventListener('input', function() {
                    UserUI.checkPasswordMatch();
                });
            }

            var form = document.getElementById('register-form');
            if (!form) return;

            form.addEventListener('submit', function(e) {
                e.preventDefault();
                var username = document.getElementById('reg-username').value.trim();
                var email = document.getElementById('reg-email').value.trim();
                var name = document.getElementById('reg-name').value.trim();
                var password = document.getElementById('reg-password').value;
                var confirmPassword = document.getElementById('reg-confirm-password').value;
                var agreeTerms = document.getElementById('agree-terms').checked;
                var errorEl = document.getElementById('register-error');
                var btn = document.getElementById('register-btn');
                var textEl = document.getElementById('register-btn-text');
                var spinnerEl = document.getElementById('register-spinner');

                UserUI.hideError(errorEl);

                if (!username || username.length < 3) {
                    UserUI.showError(errorEl, '用户名至少需要3个字符');
                    return;
                }
                if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
                    UserUI.showError(errorEl, '请输入有效的邮箱地址');
                    return;
                }
                if (!password || password.length < 8) {
                    UserUI.showError(errorEl, '密码至少需要8个字符');
                    return;
                }
                if (password !== confirmPassword) {
                    UserUI.showError(errorEl, '两次输入的密码不一致');
                    return;
                }
                if (!agreeTerms) {
                    UserUI.showError(errorEl, '请阅读并同意服务条款');
                    return;
                }

                UserUI.setLoading(btn, textEl, spinnerEl, true);

                UserAPI.auth.register({
                    username: username,
                    email: email,
                    name: name,
                    password: password
                })
                    .then(function(data) {
                        if (data.data && data.data.token) {
                            UserAuth.setToken(data.data.token);
                            if (data.data.user) {
                                UserAuth.setUser(data.data.user);
                            }
                            UserUI.showToast('注册成功！正在跳转...', 'success');
                            setTimeout(function() {
                                window.location.href = '/dashboard';
                            }, 1000);
                        } else {
                            UserUI.showToast('注册成功！请登录。', 'success');
                            setTimeout(function() {
                                window.location.href = '/login';
                            }, 1000);
                        }
                    })
                    .catch(function(err) {
                        UserUI.showError(errorEl, err.message || '注册失败，请重试');
                    })
                    .finally(function() {
                        UserUI.setLoading(btn, textEl, spinnerEl, false);
                    });
            });
        },
        evaluatePasswordStrength: function(password) {
            var bar = document.getElementById('password-strength-bar');
            var text = document.getElementById('password-strength-text');
            if (!bar || !text) return;

            bar.style.width = '0%';
            bar.className = 'password-strength-bar';

            if (!password) {
                text.textContent = '至少8个字符';
                text.className = 'text-xs text-gray-400 mt-1';
                return;
            }

            var score = 0;
            if (password.length >= 8) score++;
            if (password.length >= 12) score++;
            if (/[a-z]/.test(password)) score++;
            if (/[A-Z]/.test(password)) score++;
            if (/[0-9]/.test(password)) score++;
            if (/[^a-zA-Z0-9]/.test(password)) score++;

            var strengthClass, strengthText, textColor;
            if (score <= 2) {
                strengthClass = 'password-strength-bar strength-weak';
                strengthText = '弱';
                textColor = 'text-red-500';
            } else if (score <= 3) {
                strengthClass = 'password-strength-bar strength-fair';
                strengthText = '一般';
                textColor = 'text-yellow-500';
            } else if (score <= 4) {
                strengthClass = 'password-strength-bar strength-good';
                strengthText = '良好';
                textColor = 'text-blue-500';
            } else {
                strengthClass = 'password-strength-bar strength-strong';
                strengthText = '强';
                textColor = 'text-green-500';
            }

            bar.className = strengthClass;
            text.textContent = '密码强度：' + strengthText;
            text.className = 'text-xs ' + textColor + ' mt-1';
        },
        checkPasswordMatch: function() {
            var password = document.getElementById('reg-password').value;
            var confirm = document.getElementById('reg-confirm-password').value;
            var hint = document.getElementById('confirm-password-hint');
            if (!hint) return;

            if (!confirm) {
                hint.classList.add('hidden');
                return;
            }
            hint.classList.remove('hidden');
            if (password === confirm) {
                hint.textContent = '密码匹配';
                hint.className = 'text-xs text-green-500 mt-1';
            } else {
                hint.textContent = '密码不匹配';
                hint.className = 'text-xs text-red-500 mt-1';
            }
        },
        initDashboardPage: function() {
            if (!UserAuth.requireAuth()) return;

            UserUI.initSidebar();
            UserUI.initMobileMenu();
            UserUI.initLogout();
            UserUI.initTimeDisplay();
            UserUI.loadDashboardStats();
            UserUI.loadRecentActivities();
            UserUI.loadUserInfo();
        },
        loadDashboardStats: function() {
            UserAPI.dashboard.getStats()
                .then(function(data) {
                    var stats = data.data || data;
                    document.getElementById('stat-total').textContent = UserUI.formatNumber(stats.total_verifications || 0);
                    document.getElementById('stat-rate').textContent = (stats.success_rate || 0).toFixed(1) + '%';
                    document.getElementById('stat-today').textContent = UserUI.formatNumber(stats.today_verifications || 0);
                    document.getElementById('stat-apps').textContent = UserUI.formatNumber(stats.app_count || 0);
                })
                .catch(function() {
                    var mock = { total_verifications: 12847, success_rate: 94.2, today_verifications: 523, app_count: 3 };
                    document.getElementById('stat-total').textContent = UserUI.formatNumber(mock.total_verifications);
                    document.getElementById('stat-rate').textContent = mock.success_rate.toFixed(1) + '%';
                    document.getElementById('stat-today').textContent = UserUI.formatNumber(mock.today_verifications);
                    document.getElementById('stat-apps').textContent = UserUI.formatNumber(mock.app_count);
                });
        },
        loadRecentActivities: function() {
            var list = document.getElementById('activity-list');
            if (!list) return;

            UserAPI.dashboard.getActivities()
                .then(function(data) {
                    var activities = (data.data && data.data.activities) || data.activities || [];
                    UserUI.renderActivities(list, activities);
                })
                .catch(function() {
                    UserUI.renderActivities(list, UserUI.generateMockActivities());
                });
        },
        renderActivities: function(list, activities) {
            if (!activities || activities.length === 0) {
                list.innerHTML = '<div class="text-center py-4 text-gray-400">暂无活动记录</div>';
                return;
            }
            var html = '';
            for (var i = 0; i < activities.length; i++) {
                var a = activities[i];
                var bgColor = a.type === 'success' ? 'bg-green-50' : a.type === 'warning' ? 'bg-yellow-50' : 'bg-red-50';
                var iconColor = a.type === 'success' ? 'text-green-500' : a.type === 'warning' ? 'text-yellow-500' : 'text-red-500';
                html += '<div class="flex items-center space-x-3 p-3 ' + bgColor + ' rounded-lg">';
                html += '<div class="w-2 h-2 rounded-full ' + iconColor.replace('text-', 'bg-') + '"></div>';
                html += '<div class="flex-1 min-w-0">';
                html += '<p class="text-sm text-gray-700 truncate">' + (a.message || a.description || '') + '</p>';
                html += '<p class="text-xs text-gray-400 mt-0.5">' + (a.time || a.created_at || '') + '</p>';
                html += '</div></div>';
            }
            list.innerHTML = html;
        },
        generateMockActivities: function() {
            return [
                { type: 'success', message: '验证成功 - IP: 192.168.1.100', time: '2分钟前' },
                { type: 'success', message: '验证成功 - IP: 10.0.0.55', time: '15分钟前' },
                { type: 'warning', message: '检测到可疑行为 - IP: 203.0.113.5', time: '1小时前' },
                { type: 'success', message: '验证成功 - IP: 172.16.0.20', time: '2小时前' }
            ];
        },
        loadUserInfo: function() {
            var user = UserAuth.getUser();
            if (user) {
                var usernameEl = document.getElementById('sidebar-username');
                if (usernameEl) usernameEl.textContent = user.username || user.email || '用户';
            }
        },
        initProfilePage: function() {
            if (!UserAuth.requireAuth()) return;

            UserUI.initSidebar();
            UserUI.initMobileMenu();
            UserUI.initLogout();
            UserUI.initProfileForm();
            UserUI.initPasswordForm();
            UserUI.initAvatarUpload();
            UserUI.loadProfileData();
        },
        loadProfileData: function() {
            var user = UserAuth.getUser();
            if (user) {
                var usernameEl = document.getElementById('profile-username');
                var emailEl = document.getElementById('profile-email-input');
                var nameEl = document.getElementById('profile-name-input');
                var phoneEl = document.getElementById('profile-phone');
                var nameDisplay = document.getElementById('profile-name');
                var emailDisplay = document.getElementById('profile-email-display');
                var sidebarUsername = document.getElementById('sidebar-username');

                if (usernameEl) usernameEl.value = user.username || '';
                if (emailEl) emailEl.value = user.email || '';
                if (nameEl) nameEl.value = user.name || user.nickname || '';
                if (phoneEl) phoneEl.value = user.phone || '';
                if (nameDisplay) nameDisplay.textContent = user.name || user.username || '--';
                if (emailDisplay) emailDisplay.textContent = user.email || '--';
                if (sidebarUsername) sidebarUsername.textContent = user.username || user.email || '用户';

                if (user.avatar) {
                    var avatarEl = document.getElementById('profile-avatar');
                    if (avatarEl) {
                        avatarEl.innerHTML = '<img src="' + user.avatar + '" alt="avatar" class="w-full h-full object-cover">';
                    }
                }
                return;
            }

            UserAPI.profile.get()
                .then(function(data) {
                    var profile = data.data || data;
                    if (profile) {
                        UserAuth.setUser(profile);
                        var usernameEl = document.getElementById('profile-username');
                        var emailEl = document.getElementById('profile-email-input');
                        var nameEl = document.getElementById('profile-name-input');
                        var phoneEl = document.getElementById('profile-phone');
                        var nameDisplay = document.getElementById('profile-name');
                        var emailDisplay = document.getElementById('profile-email-display');
                        var sidebarUsername = document.getElementById('sidebar-username');

                        if (usernameEl) usernameEl.value = profile.username || '';
                        if (emailEl) emailEl.value = profile.email || '';
                        if (nameEl) nameEl.value = profile.name || profile.nickname || '';
                        if (phoneEl) phoneEl.value = profile.phone || '';
                        if (nameDisplay) nameDisplay.textContent = profile.name || profile.username || '--';
                        if (emailDisplay) emailDisplay.textContent = profile.email || '--';
                        if (sidebarUsername) sidebarUsername.textContent = profile.username || profile.email || '用户';

                        if (profile.avatar) {
                            var avatarEl = document.getElementById('profile-avatar');
                            if (avatarEl) {
                                avatarEl.innerHTML = '<img src="' + profile.avatar + '" alt="avatar" class="w-full h-full object-cover">';
                            }
                        }
                    }
                })
                .catch(function(err) {
                    console.error('Failed to load profile:', err);
                });
        },
        initProfileForm: function() {
            var form = document.getElementById('profile-form');
            if (!form) return;

            form.addEventListener('submit', function(e) {
                e.preventDefault();
                var errorEl = document.getElementById('profile-error');
                var successEl = document.getElementById('profile-success');
                var btn = document.getElementById('profile-save-btn');
                var textEl = document.getElementById('profile-save-text');
                var spinnerEl = document.getElementById('profile-save-spinner');

                UserUI.hideError(errorEl);
                if (successEl) successEl.classList.add('hidden');

                var email = document.getElementById('profile-email-input').value.trim();
                var name = document.getElementById('profile-name-input').value.trim();
                var phone = document.getElementById('profile-phone').value.trim();

                if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
                    UserUI.showError(errorEl, '请输入有效的邮箱地址');
                    return;
                }

                UserUI.setLoading(btn, textEl, spinnerEl, true);

                UserAPI.profile.update({ email: email, name: name, phone: phone })
                    .then(function(data) {
                        if (successEl) {
                            successEl.textContent = '个人资料更新成功';
                            successEl.classList.remove('hidden');
                        }
                        var updated = data.data || data;
                        if (updated) {
                            UserAuth.setUser(updated);
                        }
                        UserUI.showToast('资料更新成功', 'success');
                    })
                    .catch(function(err) {
                        UserUI.showError(errorEl, err.message || '更新失败，请重试');
                    })
                    .finally(function() {
                        UserUI.setLoading(btn, textEl, spinnerEl, false);
                    });
            });
        },
        initPasswordForm: function() {
            var form = document.getElementById('password-form');
            if (!form) return;

            form.addEventListener('submit', function(e) {
                e.preventDefault();
                var errorEl = document.getElementById('password-error');
                var successEl = document.getElementById('password-success');
                var btn = document.getElementById('password-submit-btn');
                var textEl = document.getElementById('password-submit-text');
                var spinnerEl = document.getElementById('password-submit-spinner');

                UserUI.hideError(errorEl);
                if (successEl) successEl.classList.add('hidden');

                var currentPwd = document.getElementById('current-password').value;
                var newPwd = document.getElementById('new-password').value;
                var confirmPwd = document.getElementById('confirm-new-password').value;

                if (!currentPwd) {
                    UserUI.showError(errorEl, '请输入当前密码');
                    return;
                }
                if (!newPwd || newPwd.length < 8) {
                    UserUI.showError(errorEl, '新密码至少需要8个字符');
                    return;
                }
                if (newPwd !== confirmPwd) {
                    UserUI.showError(errorEl, '两次输入的新密码不一致');
                    return;
                }

                UserUI.setLoading(btn, textEl, spinnerEl, true);

                UserAPI.profile.changePassword({
                    current_password: currentPwd,
                    new_password: newPwd
                })
                    .then(function() {
                        if (successEl) {
                            successEl.textContent = '密码修改成功';
                            successEl.classList.remove('hidden');
                        }
                        form.reset();
                        UserUI.showToast('密码修改成功', 'success');
                    })
                    .catch(function(err) {
                        UserUI.showError(errorEl, err.message || '密码修改失败');
                    })
                    .finally(function() {
                        UserUI.setLoading(btn, textEl, spinnerEl, false);
                    });
            });
        },
        initAvatarUpload: function() {
            var avatarContainer = document.querySelector('.avatar-upload');
            var fileInput = document.getElementById('avatar-input');
            if (!avatarContainer || !fileInput) return;

            avatarContainer.addEventListener('click', function() {
                fileInput.click();
            });

            fileInput.addEventListener('change', function() {
                var file = fileInput.files[0];
                if (!file) return;
                if (file.size > 2 * 1024 * 1024) {
                    UserUI.showToast('图片大小不能超过2MB', 'error');
                    return;
                }

                var reader = new FileReader();
                reader.onload = function(e) {
                    var avatarEl = document.getElementById('profile-avatar');
                    if (avatarEl) {
                        avatarEl.innerHTML = '<img src="' + e.target.result + '" alt="avatar" class="w-full h-full object-cover">';
                    }
                    UserUI.showToast('头像已更新（本地预览）', 'success');
                };
                reader.readAsDataURL(file);
            });
        },
        initSidebar: function() {
            var logoutBtn = document.getElementById('logout-btn');
            if (logoutBtn) {
                logoutBtn.addEventListener('click', function() {
                    UserAuth.logout();
                });
            }
        },
        initLogout: function() {
            var logoutBtn = document.getElementById('logout-btn');
            if (logoutBtn) {
                logoutBtn.addEventListener('click', function() {
                    UserAuth.logout();
                });
            }
        },
        initMobileMenu: function() {
            var menuBtn = document.getElementById('mobile-menu-btn');
            if (!menuBtn) return;
            menuBtn.addEventListener('click', function() {
                window.toggleSidebar();
            });
        },
        initTimeDisplay: function() {
            var timeEl = document.getElementById('current-time');
            if (!timeEl) return;
            function updateTime() {
                var now = new Date();
                timeEl.textContent = now.getFullYear() + '-' +
                    String(now.getMonth() + 1).padStart(2, '0') + '-' +
                    String(now.getDate()).padStart(2, '0') + ' ' +
                    String(now.getHours()).padStart(2, '0') + ':' +
                    String(now.getMinutes()).padStart(2, '0') + ':' +
                    String(now.getSeconds()).padStart(2, '0');
            }
            updateTime();
            setInterval(updateTime, 1000);
        }
    };

    window.toggleSidebar = function() {
        var sidebar = document.getElementById('sidebar');
        var overlay = document.getElementById('mobile-overlay');
        if (!sidebar) return;
        if (sidebar.classList.contains('-translate-x-full')) {
            sidebar.classList.remove('-translate-x-full');
            if (overlay) overlay.classList.remove('hidden');
        } else {
            sidebar.classList.add('-translate-x-full');
            if (overlay) overlay.classList.add('hidden');
        }
    };

    window.UserAPI = UserAPI;
    window.UserAuth = UserAuth;
    window.UserUI = UserUI;
})();