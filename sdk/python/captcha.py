"""
行为验证系统 Python SDK

提供完整的验证码类型支持、错误处理、连接池管理和自动重试机制。
"""

import requests
import json
import time
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass, asdict
from enum import Enum
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
import logging

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class CaptchaType(Enum):
    """验证码类型枚举"""
    SLIDER = "slider"
    CLICK = "click"
    IMAGE = "image"
    ROTATION = "rotation"
    GESTURE = "gesture"
    JIGSAW = "jigsaw"


class ClickMode(Enum):
    """点击验证码模式"""
    NUMBER = "number"
    LETTER = "letter"
    CHINESE = "chinese"
    MIXED = "mixed"
    ICON = "icon"


# 异常类定义
class CaptchaError(Exception):
    """验证码 SDK 基础异常"""
    pass


class CaptchaAPIError(CaptchaError):
    """API 错误异常"""
    def __init__(self, message: str, code: Optional[int] = None, data: Optional[Any] = None):
        self.message = message
        self.code = code
        self.data = data
        super().__init__(f"API Error: {message} (code: {code})")


class CaptchaNetworkError(CaptchaError):
    """网络错误异常"""
    pass


class CaptchaTimeoutError(CaptchaNetworkError):
    """超时错误异常"""
    pass


class CaptchaValidationError(CaptchaError):
    """验证失败异常"""
    pass


class CaptchaSessionExpiredError(CaptchaError):
    """会话过期异常"""
    pass


# 数据类定义
@dataclass
class TrajectoryPoint:
    """轨迹点"""
    x: int
    y: int
    t: int
    
    def to_dict(self) -> Dict[str, int]:
        """转换为字典"""
        return asdict(self)
    
    @classmethod
    def from_dict(cls, data: Dict[str, int]) -> 'TrajectoryPoint':
        """从字典创建"""
        return cls(**data)


@dataclass
class SliderCaptchaResponse:
    """滑块验证码响应"""
    session_id: str
    image_url: str
    puzzle_url: str
    hint_url: Optional[str] = None
    shape: Optional[int] = None
    secret_y: Optional[int] = None
    image_width: Optional[int] = None
    image_height: Optional[int] = None
    tolerance: Optional[int] = None


@dataclass
class ClickCaptchaResponse:
    """点击验证码响应"""
    session_id: str
    image_url: str
    hint: str
    hint_order: List[int]
    max_points: int
    mode: str
    allow_shuffle: bool
    points: Optional[List[List[int]]] = None


@dataclass
class ImageCaptchaResponse:
    """图形验证码响应"""
    challenge_id: str
    image: str


@dataclass
class RotationCaptchaResponse:
    """旋转验证码响应"""
    challenge_id: str
    image: str


@dataclass
class GestureCaptchaResponse:
    """手势验证码响应"""
    session_id: str
    pattern: Optional[str] = None
    grid_size: Optional[int] = None
    hint: Optional[str] = None


@dataclass
class JigsawPiece:
    """拼图碎片"""
    index: int
    original_x: int
    original_y: int
    current_x: int
    current_y: int
    width: int
    height: int
    rotation: int = 0
    
    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return asdict(self)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'JigsawPiece':
        """从字典创建"""
        return cls(**data)


@dataclass
class JigsawCaptchaResponse:
    """拼图验证码响应"""
    session_id: str
    image_url: str
    pieces: List[JigsawPiece]
    piece_images: List[str]
    grid_size: int
    piece_width: int
    piece_height: int
    image_width: int
    image_height: int


@dataclass
class VerifyResult:
    """验证结果"""
    success: bool
    message: str
    remaining_attempts: Optional[int] = None
    trajectory_result: Optional[Dict] = None
    risk_score: Optional[float] = None
    captcha_pass: Optional[bool] = None
    fail_reason: Optional[str] = None


@dataclass
class LoginResponse:
    """登录响应"""
    access_token: str
    refresh_token: str
    expires_in: int
    user: Dict


class CaptchaClient:
    """验证码客户端
    
    提供完整的验证码功能，包括连接池管理和自动重试机制。
    """
    
    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        timeout: int = 30,
        max_retries: int = 3,
        retry_backoff_factor: float = 0.5,
        pool_connections: int = 10,
        pool_maxsize: int = 10,
    ):
        """
        初始化客户端
        
        Args:
            base_url: API 基础 URL
            api_key: API 密钥
            timeout: 请求超时时间（秒）
            max_retries: 最大重试次数
            retry_backoff_factor: 重试退避因子
            pool_connections: 连接池大小
            pool_maxsize: 最大连接数
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self._token = None
        self._refresh_token = None
        
        # 创建会话并配置连接池和重试
        self.session = requests.Session()
        
        # 配置重试策略
        retry_strategy = Retry(
            total=max_retries,
            read=max_retries,
            connect=max_retries,
            backoff_factor=retry_backoff_factor,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["HEAD", "GET", "OPTIONS", "POST", "PUT", "DELETE"],
        )
        
        # 配置适配器
        adapter = HTTPAdapter(
            max_retries=retry_strategy,
            pool_connections=pool_connections,
            pool_maxsize=pool_maxsize,
        )
        
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
    
    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'Content-Type': 'application/json',
            'User-Agent': 'Captcha-Python-SDK/1.0',
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
        params: Optional[Dict] = None,
    ) -> Any:
        """
        发送请求
        
        Args:
            method: HTTP 方法
            path: API 路径
            data: 请求数据
            params: URL 参数
            
        Returns:
            响应数据
            
        Raises:
            CaptchaNetworkError: 网络错误
            CaptchaTimeoutError: 超时错误
            CaptchaAPIError: API 错误
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
        
        try:
            logger.debug(f"Request: {method} {url}")
            response = self.session.request(method, url, **kwargs)
            
            # 检查 HTTP 状态码
            try:
                response.raise_for_status()
            except requests.HTTPError as e:
                logger.error(f"HTTP error: {e}")
                raise CaptchaAPIError(
                    f"HTTP error: {response.status_code}",
                    code=response.status_code,
                ) from e
            
            # 解析响应
            result = response.json()
            logger.debug(f"Response: {result}")
            
            # 检查业务状态码
            code = result.get('code')
            if code != 0 and code is not None:
                message = result.get('message', 'Unknown error')
                data = result.get('data')
                
                # 根据错误码抛不同异常
                if code == 404 or '不存在' in message or '过期' in message:
                    raise CaptchaSessionExpiredError(message)
                if code == 400:
                    raise CaptchaValidationError(message)
                
                raise CaptchaAPIError(message, code=code, data=data)
            
            return result.get('data')
            
        except requests.Timeout as e:
            logger.error(f"Request timeout: {e}")
            raise CaptchaTimeoutError(f"Request timed out after {self.timeout}s") from e
        except requests.ConnectionError as e:
            logger.error(f"Connection error: {e}")
            raise CaptchaNetworkError(f"Connection error: {e}") from e
        except requests.RequestException as e:
            logger.error(f"Request failed: {e}")
            raise CaptchaNetworkError(f"Request failed: {e}") from e
        except json.JSONDecodeError as e:
            logger.error(f"Failed to decode JSON response: {e}")
            raise CaptchaAPIError("Invalid JSON response") from e
    
    # ==================== 滑块验证码 ====================
    def get_slider_captcha(
        self,
        width: int = 320,
        height: int = 160,
        tolerance: int = 8,
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
        data = self._request(
            'GET',
            '/api/v1/captcha/slider',
            params={
                'width': width,
                'height': height,
                'tolerance': tolerance,
            },
        )
        return SliderCaptchaResponse(**data)
    
    def verify_slider_captcha(
        self,
        session_id: str,
        x: int,
        y: Optional[int] = None,
        trajectory: Optional[List[TrajectoryPoint]] = None,
        behavior_data: Optional[List[Dict]] = None,
    ) -> VerifyResult:
        """
        验证滑块验证码
        
        Args:
            session_id: 会话 ID
            x: X 坐标
            y: Y 坐标
            trajectory: 轨迹数据
            behavior_data: 行为数据
            
        Returns:
            验证结果
        """
        req_data = {
            'session_id': session_id,
            'type': 'slider',
            'x': x,
        }
        if y is not None:
            req_data['y'] = y
        if trajectory:
            req_data['trajectory'] = [asdict(p) for p in trajectory]
        if behavior_data:
            req_data['behavior_data'] = behavior_data
        
        data = self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            trajectory_result=data.get('trajectory_result'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )
    
    # ==================== 点击验证码 ====================
    def get_click_captcha(
        self,
        mode: Union[ClickMode, str] = ClickMode.NUMBER,
        max_points: int = 3,
        allow_shuffle: bool = True,
    ) -> ClickCaptchaResponse:
        """
        获取点击验证码
        
        Args:
            mode: 验证码模式
            max_points: 最大点数
            allow_shuffle: 是否允许打乱顺序
            
        Returns:
            点击验证码响应
        """
        if isinstance(mode, ClickMode):
            mode = mode.value
            
        data = self._request(
            'GET',
            '/api/v1/captcha/click',
            params={
                'mode': mode,
                'points': str(max_points),
                'shuffle': str(allow_shuffle).lower(),
            },
        )
        return ClickCaptchaResponse(
            session_id=data.get('session_id', ''),
            image_url=data.get('image_url', ''),
            hint=data.get('hint', ''),
            hint_order=data.get('hint_order', []),
            max_points=data.get('max_points', 0),
            mode=data.get('mode', mode),
            allow_shuffle=data.get('allow_shuffle', allow_shuffle),
            points=data.get('points'),
        )
    
    def verify_click_captcha(
        self,
        session_id: str,
        points: List[List[int]],
        click_sequence: Optional[List[int]] = None,
        behavior_data: Optional[List[Dict]] = None,
    ) -> VerifyResult:
        """
        验证点击验证码
        
        Args:
            session_id: 会话 ID
            points: 点击坐标列表
            click_sequence: 点击顺序
            behavior_data: 行为数据
            
        Returns:
            验证结果
        """
        req_data = {
            'session_id': session_id,
            'type': 'click',
            'points': points,
        }
        if click_sequence:
            req_data['click_sequence'] = click_sequence
        if behavior_data:
            req_data['behavior_data'] = behavior_data
        
        data = self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )
    
    # ==================== 图形验证码 ====================
    def get_image_captcha(
        self,
        type_: str = 'mixed',
        count: int = 4,
        noise_mode: int = 0,
        line_mode: int = 0,
    ) -> ImageCaptchaResponse:
        """
        获取图形验证码
        
        Args:
            type_: 验证码类型
            count: 字符数量
            noise_mode: 噪音模式
            line_mode: 线条模式
            
        Returns:
            图形验证码响应
        """
        data = self._request(
            'GET',
            '/api/v1/captcha/image',
            params={
                'type': type_,
                'count': count,
                'noise_mode': noise_mode,
                'line_mode': line_mode,
            },
        )
        return ImageCaptchaResponse(**data)
    
    def verify_image_captcha(
        self,
        challenge_id: str,
        answer: str,
    ) -> VerifyResult:
        """
        验证图形验证码
        
        Args:
            challenge_id: 挑战 ID
            answer: 答案
            
        Returns:
            验证结果
        """
        data = self._request(
            'POST',
            '/api/v1/captcha/image/verify',
            data={
                'challenge_id': challenge_id,
                'answer': answer,
            },
        )
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )
    
    # ==================== 旋转验证码 ====================
    def get_rotation_captcha(self) -> RotationCaptchaResponse:
        """
        获取旋转验证码
        
        Returns:
            旋转验证码响应
        """
        data = self._request('GET', '/api/v1/captcha/rotation')
        return RotationCaptchaResponse(**data)
    
    def verify_rotation_captcha(
        self,
        challenge_id: str,
        angle: int,
    ) -> VerifyResult:
        """
        验证旋转验证码
        
        Args:
            challenge_id: 挑战 ID
            angle: 旋转角度
            
        Returns:
            验证结果
        """
        data = self._request(
            'POST',
            '/api/v1/captcha/rotation/verify',
            data={
                'challenge_id': challenge_id,
                'angle': angle,
            },
        )
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )
    
    # ==================== 手势验证码 ====================
    def get_gesture_captcha(self) -> GestureCaptchaResponse:
        """
        获取手势验证码
        
        Returns:
            手势验证码响应
        """
        data = self._request('GET', '/api/v1/captcha/gesture')
        return GestureCaptchaResponse(
            session_id=data.get('session_id', ''),
            pattern=data.get('pattern'),
            grid_size=data.get('grid_size'),
            hint=data.get('hint'),
        )
    
    def verify_gesture_captcha(
        self,
        session_id: str,
        pattern: List[int],
    ) -> VerifyResult:
        """
        验证手势验证码
        
        Args:
            session_id: 会话 ID
            pattern: 手势模式
            
        Returns:
            验证结果
        """
        data = self._request(
            'POST',
            '/api/v1/captcha/gesture/verify',
            data={
                'session_id': session_id,
                'pattern': pattern,
            },
        )
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )
    
    # ==================== 拼图验证码 ====================
    def get_jigsaw_captcha(
        self,
        width: int = 300,
        height: int = 300,
        grid_size: int = 3,
    ) -> JigsawCaptchaResponse:
        """
        获取拼图验证码
        
        Args:
            width: 图片宽度
            height: 图片高度
            grid_size: 网格大小 (2, 3, 4)
            
        Returns:
            拼图验证码响应
        """
        data = self._request(
            'GET',
            '/api/v1/captcha/jigsaw',
            params={
                'width': width,
                'height': height,
                'grid_size': grid_size,
            },
        )
        # 解析碎片数据
        pieces = [JigsawPiece(**p) for p in data.get('pieces', [])]
        return JigsawCaptchaResponse(
            session_id=data.get('session_id', ''),
            image_url=data.get('image_url', ''),
            pieces=pieces,
            piece_images=data.get('piece_images', []),
            grid_size=data.get('grid_size', 3),
            piece_width=data.get('piece_width', 0),
            piece_height=data.get('piece_height', 0),
            image_width=data.get('image_width', width),
            image_height=data.get('image_height', height),
        )
    
    def verify_jigsaw_captcha(
        self,
        session_id: str,
        pieces: List[JigsawPiece],
    ) -> VerifyResult:
        """
        验证拼图验证码
        
        Args:
            session_id: 会话 ID
            pieces: 碎片列表
            
        Returns:
            验证结果
        """
        data = self._request(
            'POST',
            '/api/v1/captcha/jigsaw/verify',
            data={
                'session_id': session_id,
                'pieces': [asdict(p) for p in pieces],
            },
        )
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
        )
    
    # ==================== 通用验证方法 ====================
    def verify_captcha(
        self,
        captcha_type: Union[CaptchaType, str],
        session_id: str,
        **kwargs,
    ) -> VerifyResult:
        """
        通用验证方法
        
        Args:
            captcha_type: 验证码类型
            session_id: 会话 ID
            **kwargs: 其他验证参数
            
        Returns:
            验证结果
        """
        if isinstance(captcha_type, CaptchaType):
            captcha_type = captcha_type.value
            
        req_data = {
            'session_id': session_id,
            'type': captcha_type,
            **kwargs,
        }
        
        data = self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return VerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            trajectory_result=data.get('trajectory_result'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )
    
    # ==================== 用户认证 ====================
    def auth(self) -> 'UserAuth':
        """获取用户认证 API"""
        return UserAuth(self)
    
    # ==================== 环境检测 ====================
    def env(self) -> 'Environment':
        """获取环境检测 API"""
        return Environment(self)
    
    def close(self):
        """关闭客户端，释放资源"""
        self.session.close()
    
    def __enter__(self) -> 'CaptchaClient':
        """上下文管理器入口"""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """上下文管理器退出"""
        self.close()


class UserAuth:
    """用户认证 API"""
    
    def __init__(self, client: CaptchaClient):
        self.client = client
    
    def register(
        self,
        username: str,
        email: str,
        password: str,
        behavior_data: Optional[str] = None,
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
        captcha_token: Optional[str] = None,
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
            raise CaptchaError("No refresh token available")
        
        result = self.client._request(
            'POST',
            '/api/v1/auth/refresh',
            data={'refresh_token': token},
        )
        
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
        return self.client._request(
            'GET',
            '/api/v1/auth/verify-email',
            params={'token': token},
        )
    
    def resend_verification(self, email: str) -> Dict:
        """
        重新发送验证邮件
        
        Args:
            email: 邮箱
            
        Returns:
            发送结果
        """
        return self.client._request(
            'POST',
            '/api/v1/auth/resend-verification',
            data={'email': email},
        )
    
    def request_password_reset(self, email: str) -> Dict:
        """
        请求重置密码
        
        Args:
            email: 邮箱
            
        Returns:
            请求结果
        """
        return self.client._request(
            'POST',
            '/api/v1/auth/request-password-reset',
            data={'email': email},
        )
    
    def reset_password(self, token: str, new_password: str) -> Dict:
        """
        重置密码
        
        Args:
            token: 重置令牌
            new_password: 新密码
            
        Returns:
            重置结果
        """
        return self.client._request(
            'POST',
            '/api/v1/auth/reset-password',
            data={
                'token': token,
                'new_password': new_password,
            },
        )


class Environment:
    """环境检测 API"""
    
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
        response = self.client.session.get(
            url,
            params=params,
            timeout=self.client.timeout,
        )
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
