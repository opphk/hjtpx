const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export const register = async userData => {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(userData)
    });

    const data = await response.json();

    if (response.ok) {
      return { success: true, message: '注册成功' };
    }
    return { success: false, message: data.message || '注册失败' };
  } catch (error) {
    console.error('Register error:', error);
    return { success: false, message: '网络错误，请稍后重试' };
  }
};

export const authService = {
  async login(credentials) {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(credentials)
      });

      const data = await response.json();

      if (response.ok) {
        if (data.token) {
          localStorage.setItem('authToken', data.token);
        }
        if (data.user) {
          localStorage.setItem('user', JSON.stringify(data.user));
        }
        return { success: true, user: data.user, token: data.token };
      }
      return { success: false, message: data.message || '登录失败' };
    } catch (error) {
      console.error('Login error:', error);
      return { success: false, message: '网络错误，请稍后重试' };
    }
  },

  async register(userData) {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(userData)
      });

      const data = await response.json();

      if (response.ok) {
        return { success: true, message: '注册成功' };
      }
      return { success: false, message: data.message || '注册失败' };
    } catch (error) {
      console.error('Register error:', error);
      return { success: false, message: '网络错误，请稍后重试' };
    }
  },

  logout() {
    localStorage.removeItem('authToken');
    localStorage.removeItem('user');
    return { success: true };
  },

  getToken() {
    return localStorage.getItem('authToken');
  },

  getUser() {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  isAuthenticated() {
    return !!this.getToken();
  }
};
