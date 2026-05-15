const API_BASE = '/api/v1';
const TOKEN_KEY = 'admin_token';
const USER_KEY = 'admin_user';

const auth = {
    getToken() {
        return localStorage.getItem(TOKEN_KEY);
    },

    setToken(token) {
        localStorage.setItem(TOKEN_KEY, token);
    },

    removeToken() {
        localStorage.removeItem(TOKEN_KEY);
    },

    getUser() {
        const userStr = localStorage.getItem(USER_KEY);
        return userStr ? JSON.parse(userStr) : null;
    },

    setUser(user) {
        localStorage.setItem(USER_KEY, JSON.stringify(user));
    },

    removeUser() {
        localStorage.removeItem(USER_KEY);
    },

    isAuthenticated() {
        const token = this.getToken();
        return !!token;
    },

    async login(username, password) {
        try {
            const response = await fetch(`${API_BASE}/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });

            const data = await response.json();

            if (response.ok && data.code === 0) {
                this.setToken(data.data.token);
                this.setUser(data.data.user);
                return { success: true };
            }

            return { success: false, message: data.message || '登录失败' };
        } catch (error) {
            return { success: false, message: '网络错误，请稍后重试' };
        }
    },

    async logout() {
        try {
            const token = this.getToken();
            if (token) {
                await fetch(`${API_BASE}/auth/logout`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${token}`
                    }
                });
            }
        } catch (error) {
        } finally {
            this.removeToken();
            this.removeUser();
            window.location.href = '/admin/login';
        }
    },

    async request(url, options = {}) {
        const token = this.getToken();
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        try {
            const response = await fetch(`${API_BASE}${url}`, {
                ...options,
                headers
            });

            if (response.status === 401) {
                this.removeToken();
                this.removeUser();
                window.location.href = '/admin/login';
                throw new Error('未授权');
            }

            const data = await response.json();
            return data;
        } catch (error) {
            if (error.message !== '未授权') {
                console.error('请求错误:', error);
            }
            throw error;
        }
    }
};

document.addEventListener('DOMContentLoaded', () => {
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        if (auth.isAuthenticated()) {
            window.location.href = '/admin/';
        }

        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const loginBtn = document.getElementById('loginBtn');
            const errorDiv = document.getElementById('loginError');

            loginBtn.disabled = true;
            loginBtn.textContent = '登录中...';
            errorDiv.textContent = '';

            const result = await auth.login(username, password);

            if (result.success) {
                window.location.href = '/admin/';
            } else {
                errorDiv.textContent = result.message;
                loginBtn.disabled = false;
                loginBtn.textContent = '登录';
            }
        });
    }

    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        if (!auth.isAuthenticated()) {
            window.location.href = '/admin/login';
        }

        logoutBtn.addEventListener('click', (e) => {
            e.preventDefault();
            if (confirm('确定要退出登录吗？')) {
                auth.logout();
            }
        });

        const user = auth.getUser();
        if (user) {
            const usernameDisplay = document.getElementById('usernameDisplay');
            const userAvatar = document.getElementById('userAvatar');
            if (usernameDisplay) {
                usernameDisplay.textContent = user.username || '管理员';
            }
            if (userAvatar) {
                userAvatar.textContent = (user.username || 'A').charAt(0).toUpperCase();
            }
        }
    }
});
