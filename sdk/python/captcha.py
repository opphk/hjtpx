"""
行为验证系统 Python SDK
"""

import requests
import json
import time
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict


@dataclass
class TrajectoryPoint:
    """轨迹点"""
    x: int
    y: int
    t: int


@dataclass
class SliderCaptchaResponse:
    """滑块验证码响应"""
    session_id: str
    image_url: str
    puzzle_url: str
    hint_url: str
    shape: int
    secret_y: int
    image_width: int
    image_height: int


@dataclass
class VerifyCaptchaResponse:
    """验证码验证响应"""
    success: bool
    message: str
    remaining_attempts: int
    trajectory_result: Optional[Dict] = None


@dataclass
class LoginResponse:
    """登录响应"""
    access_token: str
    refresh_token: str
    expires_in: int
    user: Dict


class CaptchaClient:
    """验证码客户端"""
    
    def __init__(
        self, base_url: str, api_key: Optional[str] = None, timeout: int = 30):
        """
        初始化客户端
        
        Args:
            base_url: API基础URL
            api_key: API密钥
            timeout: 请求超时时间（秒）
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self.session = requests.Session()
        self._token = None
        self._refresh_token = None
    
    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'Content-Type': 'application/json',
        }
        if self.api_key:
            headers['X-API-Key'] = self.api_key
        if self._token:
            headers['Authorization'] = f'Bearer {self._token}'
        return headers
    
    def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict] = None,
        params: Optional[Dict] = None
    ) -> Any:
        """
        发送请求
        
        Args:
            method: HTTP方法
            path: API路径
            data: 请求数据
            params: URL参数
            
        Returns:
            响应数据
        """
        url = f"{self.base_url}{path}"
        
        kwargs = {
            'headers': self._get_headers(),
            'timeout': self.timeout,
        }
        
        if data:
            kwargs['json'] = data
        if params:
            kwargs['params'] = params
        
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        
        result = response.json()
        
        if result.get('code') != 0:
            raise Exception(f"API Error: {result.get('message', 'Unknown error')}")
        
        return result.get('data')
    
    def get_slider_captcha(
        self,
        width: int = 320,
        height: int = 160,
        tolerance: int = 8
    ) -> SliderCaptchaResponse:
        """
        获取滑块验证码
        
        Args:
            width: 图片宽度
            height: 图片高度
            tolerance: 容差值
            
        Returns:
            滑块验证码响应
        """
        data = self._request('GET', '/api/v1/captcha/slider', params={
            'width': width,
            'height': height,
            'tolerance': tolerance,
        })
        return SliderCaptchaResponse(**data)
    
    def verify_captcha(
        self,
        session_id: str,
        x: int,
        y: Optional[int] = None,
        trajectory: Optional[List[TrajectoryPoint]] = None
    ) -> VerifyCaptchaResponse:
        """
        验证滑块验证码
        
        Args:
            session_id: 会话ID
            x: X坐标
            y: Y坐标
            trajectory: 轨迹数据
            
        Returns:
            验证响应
        """
        req_data = {'session_id': session_id, 'x': x}
        if y is not None:
            req_data['y'] = y
        if trajectory:
            req_data['trajectory'] = [asdict(p) for p in trajectory]
        
        data = self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return VerifyCaptchaResponse(**data)
    
    def get_click_captcha(self) -> Dict:
        """获取点击验证码"""
        return self._request('GET', '/api/v1/captcha/click')
    
    def get_gesture_captcha(self) -> Dict:
        """获取手势验证码"""
        return self._request('GET', '/api/v1/captcha/gesture')
    
    def verify_gesture_captcha(self, session_id: str, pattern: List[int]) -> Dict:
        """
        验证手势验证码
        
        Args:
            session_id: 会话ID
            pattern: 手势模式
            
        Returns:
            验证响应
        """
        return self._request('POST', '/api/v1/captcha/gesture/verify', data={
            'session_id': session_id,
            'pattern': pattern,
        })
    
    def auth(self) -> 'UserAuth':
        """获取用户认证API"""
        return UserAuth(self)
    
    def env(self) -> 'Environment':
        """获取环境检测API"""
        return Environment(self)


class UserAuth:
    """用户认证API"""
    
    def __init__(self, client: CaptchaClient):
        self.client = client
    
    def register(
        self,
        username: str,
        email: str,
        password: str,
        behavior_data: Optional[str] = None
    ) -> Dict:
        """
        用户注册
        
        Args:
            username: 用户名
            email: 邮箱
            password: 密码
            behavior_data: 行为数据
            
        Returns:
            注册结果
        """
        data = {
            'username': username,
            'email': email,
            'password': password,
        }
        if behavior_data:
            data['behavior_data'] = behavior_data
        
        return self.client._request('POST', '/api/v1/auth/register', data=data)
    
    def login(
        self,
        username: str,
        password: str,
        captcha_token: Optional[str] = None
    ) -> LoginResponse:
        """
        用户登录
        
        Args:
            username: 用户名
            password: 密码
            captcha_token: 验证码令牌
            
        Returns:
            登录响应
        """
        data = {'username': username, 'password': password}
        if captcha_token:
            data['captcha_token'] = captcha_token
        
        result = self.client._request('POST', '/api/v1/auth/login', data=data)
        self.client._token = result.get('access_token')
        self.client._refresh_token = result.get('refresh_token')
        return LoginResponse(**result)
    
    def refresh_token(self, refresh_token: Optional[str] = None) -> Dict:
        """
        刷新访问令牌
        
        Args:
            refresh_token: 刷新令牌
            
        Returns:
            刷新结果
        """
        token = refresh_token or self.client._refresh_token
        if not token:
            raise Exception("No refresh token available")
        
        result = self.client._request('POST', '/api/v1/auth/refresh', data={
            'refresh_token': token,
        })
        
        self.client._token = result.get('access_token')
        if result.get('refresh_token'):
            self.client._refresh_token = result.get('refresh_token')
        
        return result
    
    def logout(self) -> None:
        """用户登出"""
        try:
            self.client._request('POST', '/api/v1/auth/logout')
        finally:
            self.client._token = None
            self.client._refresh_token = None
    
    def verify_email(self, token: str) -> Dict:
        """
        验证邮箱
        
        Args:
            token: 验证令牌
            
        Returns:
            验证结果
        """
        return self.client._request('GET', '/api/v1/auth/verify-email', params={'token': token})
    
    def resend_verification(self, email: str) -> Dict:
        """
        重新发送验证邮件
        
        Args:
            email: 邮箱
            
        Returns:
            发送结果
        """
        return self.client._request('POST', '/api/v1/auth/resend-verification', data={'email': email})
    
    def request_password_reset(self, email: str) -> Dict:
        """
        请求重置密码
        
        Args:
            email: 邮箱
            
        Returns:
            请求结果
        """
        return self.client._request('POST', '/api/v1/auth/request-password-reset', data={'email': email})
    
    def reset_password(self, token: str, new_password: str) -> Dict:
        """
        重置密码
        
        Args:
            token: 重置令牌
            new_password: 新密码
            
        Returns:
            重置结果
        """
        return self.client._request('POST', '/api/v1/auth/reset-password', data={
            'token': token,
            'new_password': new_password,
        })


class Environment:
    """环境检测API"""
    
    def __init__(self, client: CaptchaClient):
        self.client = client
    
    def get_detection_script(self, callback: Optional[str] = None) -> str:
        """
        获取检测脚本
        
        Args:
            callback: 回调函数名
            
        Returns:
            脚本内容
        """
        params = {}
        if callback:
            params['callback'] = callback
        
        url = f"{self.client.base_url}/api/v1/detect/script"
        response = self.client.session.get(url, params=params, timeout=self.client.timeout)
        response.raise_for_status()
        return response.text
    
    def submit_detection(self, data: Dict) -> Dict:
        """
        提交检测数据
        
        Args:
            data: 检测数据
            
        Returns:
            提交结果
        """
        return self.client._request('POST', '/api/v1/detect/submit', data=data)
    
    def check_environment(self, data: Dict) -> Dict:
        """
        环境检测
        
        Args:
            data: 检测数据
            
        Returns:
            检测结果
        """
        return self.client._request('POST', '/api/v1/detect/check', data=data)


# 使用示例
def example_slider_captcha():
    """滑块验证码示例"""
    client = CaptchaClient('http://localhost:8080')
    
    # 获取验证码
    captcha = client.get_slider_captcha(width=320, height=160, tolerance=8)
    print(f"Session ID: {captcha.session_id}")
    
    # 生成模拟轨迹
    trajectory = [
        TrajectoryPoint(x=0, y=captcha.secret_y, t=0),
        TrajectoryPoint(x=50, y=captcha.secret_y + 5, t=200),
        TrajectoryPoint(x=100, y=captcha.secret_y - 3, t=400),
        TrajectoryPoint(x=150, y=captcha.secret_y + 2, t=600),
        TrajectoryPoint(x=185, y=captcha.secret_y, t=800),
    ]
    
    # 验证
    result = client.verify_captcha(
        session_id=captcha.session_id,
        x=185,
        y=captcha.secret_y,
        trajectory=trajectory
    )
    
    print(f"Success: {result.success}, Message: {result.message}")


def example_user_login():
    """用户登录示例"""
    client = CaptchaClient('http://localhost:8080')
    auth = client.auth()
    
    try:
        login_result = auth.login(username='testuser', password='password123')
        print(f"Login successful, access token: {login_result.access_token[:50}...")
    except Exception as e:
        print(f"Login failed: {e}")


if __name__ == '__main__':
    print("行为验证系统 Python SDK 示例")
    print("1. 滑块验证码示例")
    print("2. 用户登录示例")
    
    choice = input("请选择示例 (1/2): ").strip()
    
    if choice == '1':
        example_slider_captcha()
    elif choice == '2':
        example_user_login()
    else:
        print("无效的选择")
