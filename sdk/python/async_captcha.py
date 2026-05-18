"""
Asyncio Async Support for Python SDK

提供异步/await兼容的完整SDK方法，支持高并发场景。
"""

import asyncio
import aiohttp
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass, asdict
from enum import Enum
import logging
import json

logger = logging.getLogger(__name__)


class AsyncCaptchaType(Enum):
    """验证码类型枚举"""
    SLIDER = "slider"
    CLICK = "click"
    IMAGE = "image"
    ROTATION = "rotation"
    GESTURE = "gesture"
    JIGSAW = "jigsaw"


class AsyncClickMode(Enum):
    """点击验证码模式枚举"""
    NUMBER = "number"
    LETTER = "letter"
    CHINESE = "chinese"
    MIXED = "mixed"
    ICON = "icon"


class AsyncCaptchaError(Exception):
    """基础异步验证码错误"""
    pass


class AsyncCaptchaAPIError(AsyncCaptchaError):
    """API错误异常"""
    def __init__(self, message: str, code: Optional[int] = None, data: Optional[Any] = None):
        self.message = message
        self.code = code
        self.data = data
        super().__init__(f"API Error: {message} (code: {code})")


class AsyncCaptchaNetworkError(AsyncCaptchaError):
    """网络错误异常"""
    pass


class AsyncCaptchaTimeoutError(AsyncCaptchaNetworkError):
    """超时错误异常"""
    pass


class AsyncCaptchaValidationError(AsyncCaptchaError):
    """验证失败异常"""
    pass


@dataclass
class AsyncTrajectoryPoint:
    """异步客户端轨迹点"""
    x: int
    y: int
    t: int

    def to_dict(self) -> Dict[str, int]:
        """转换为字典格式"""
        return asdict(self)


@dataclass
class AsyncSliderCaptchaResponse:
    """异步滑块验证码响应"""
    session_id: str
    image_url: str
    puzzle_url: str
    hint_url: Optional[str] = None
    shape: Optional[int] = None
    secret_y: Optional[int] = None
    image_width: Optional[int] = None
    image_height: Optional[int] = None


@dataclass
class AsyncClickCaptchaResponse:
    """异步点击验证码响应"""
    session_id: str
    image_url: str
    hint: str
    hint_order: List[int]
    max_points: int
    mode: str
    allow_shuffle: bool
    points: Optional[List[List[int]]] = None


@dataclass
class AsyncImageCaptchaResponse:
    """异步图形验证码响应"""
    challenge_id: str
    image: str


@dataclass
class AsyncRotationCaptchaResponse:
    """异步旋转验证码响应"""
    challenge_id: str
    image: str


@dataclass
class AsyncGestureCaptchaResponse:
    """异步手势验证码响应"""
    session_id: str
    pattern: Optional[str] = None
    grid_size: Optional[int] = None
    hint: Optional[str] = None


@dataclass
class AsyncJigsawPiece:
    """异步拼图碎片"""
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


@dataclass
class AsyncJigsawCaptchaResponse:
    """异步拼图验证码响应"""
    session_id: str
    image_url: str
    pieces: List[AsyncJigsawPiece]
    grid_size: int
    piece_width: int
    piece_height: int
    image_width: int
    image_height: int


@dataclass
class AsyncVerifyResult:
    """异步验证结果"""
    success: bool
    message: str
    remaining_attempts: Optional[int] = None
    trajectory_result: Optional[Dict] = None
    risk_score: Optional[float] = None
    captcha_pass: Optional[bool] = None
    fail_reason: Optional[str] = None


class AsyncCaptchaClient:
    """异步验证码客户端

    提供完整的异步验证码功能，支持高并发场景。
    支持上下文管理器，方便资源管理。
    """

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        timeout: int = 30,
        max_retries: int = 3,
        retry_backoff_factor: float = 0.5,
        max_connections: int = 100,
    ):
        """
        初始化异步客户端

        Args:
            base_url: API基础URL
            api_key: API密钥
            timeout: 请求超时时间（秒）
            max_retries: 最大重试次数
            retry_backoff_factor: 重试退避因子
            max_connections: 最大并发连接数
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.max_retries = max_retries
        self.retry_backoff_factor = retry_backoff_factor
        self.max_connections = max_connections
        self._token = None
        self._refresh_token = None
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        """获取或创建aiohttp会话"""
        if self._session is None or self._session.closed:
            connector = aiohttp.TCPConnector(
                limit=self.max_connections,
                limit_per_host=self.max_connections,
            )
            self._session = aiohttp.ClientSession(
                connector=connector,
                timeout=self.timeout,
            )
        return self._session

    def _get_headers(self) -> Dict[str, str]:
        """获取请求头"""
        headers = {
            'Content-Type': 'application/json',
            'User-Agent': 'Captcha-Python-Async-SDK/1.0',
        }
        if self.api_key:
            headers['X-API-Key'] = self.api_key
        if self._token:
            headers['Authorization'] = f'Bearer {self._token}'
        return headers

    async def _request(
        self,
        method: str,
        path: str,
        data: Optional[Dict] = None,
        params: Optional[Dict] = None,
    ) -> Any:
        """
        发送异步请求

        Args:
            method: HTTP方法
            path: API路径
            data: 请求数据
            params: URL参数

        Returns:
            响应数据
        """
        url = f"{self.base_url}{path}"
        session = await self._get_session()
        headers = self._get_headers()

        for attempt in range(self.max_retries + 1):
            try:
                async with session.request(
                    method, url, json=data, params=params, headers=headers,
                ) as response:
                    response.raise_for_status()
                    result = await response.json()

                    code = result.get('code')
                    if code != 0 and code is not None:
                        message = result.get('message', 'Unknown error')
                        raise AsyncCaptchaAPIError(message, code=code)

                    return result.get('data')

            except asyncio.TimeoutError:
                if attempt >= self.max_retries:
                    raise AsyncCaptchaTimeoutError("Request timed out")
                delay = self.retry_backoff_factor * (2 ** attempt)
                logger.warning(f"Request timeout, retrying in {delay}s...")
                await asyncio.sleep(delay)

            except aiohttp.ClientError as e:
                if attempt >= self.max_retries:
                    raise AsyncCaptchaNetworkError(str(e))
                delay = self.retry_backoff_factor * (2 ** attempt)
                logger.warning(f"Request failed, retrying in {delay}s...")
                await asyncio.sleep(delay)

    async def get_slider_captcha(
        self,
        width: int = 320,
        height: int = 160,
        tolerance: int = 8,
    ) -> AsyncSliderCaptchaResponse:
        """
        异步获取滑块验证码

        Args:
            width: 图片宽度
            height: 图片高度
            tolerance: 容差值

        Returns:
            滑块验证码响应
        """
        data = await self._request(
            'GET', '/api/v1/captcha/slider',
            params={'width': width, 'height': height, 'tolerance': tolerance},
        )
        return AsyncSliderCaptchaResponse(**data)

    async def verify_slider_captcha(
        self,
        session_id: str,
        x: int,
        y: Optional[int] = None,
        trajectory: Optional[List[AsyncTrajectoryPoint]] = None,
        behavior_data: Optional[List[Dict]] = None,
    ) -> AsyncVerifyResult:
        """
        异步验证滑块验证码

        Args:
            session_id: 会话ID
            x: X坐标
            y: Y坐标
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
            req_data['trajectory'] = [p.to_dict() for p in trajectory]
        if behavior_data:
            req_data['behavior_data'] = behavior_data

        data = await self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            trajectory_result=data.get('trajectory_result'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )

    async def get_click_captcha(
        self,
        mode: Union[AsyncClickMode, str] = AsyncClickMode.NUMBER,
        max_points: int = 3,
        allow_shuffle: bool = True,
    ) -> AsyncClickCaptchaResponse:
        """
        异步获取点击验证码

        Args:
            mode: 验证码模式
            max_points: 最大点数
            allow_shuffle: 是否允许打乱顺序

        Returns:
            点击验证码响应
        """
        if isinstance(mode, AsyncClickMode):
            mode = mode.value

        data = await self._request(
            'GET', '/api/v1/captcha/click',
            params={'mode': mode, 'points': str(max_points), 'shuffle': str(allow_shuffle).lower()},
        )
        return AsyncClickCaptchaResponse(
            session_id=data.get('session_id', ''),
            image_url=data.get('image_url', ''),
            hint=data.get('hint', ''),
            hint_order=data.get('hint_order', []),
            max_points=data.get('max_points', 0),
            mode=data.get('mode', mode),
            allow_shuffle=data.get('allow_shuffle', allow_shuffle),
            points=data.get('points'),
        )

    async def verify_click_captcha(
        self,
        session_id: str,
        points: List[List[int]],
        click_sequence: Optional[List[int]] = None,
        behavior_data: Optional[List[Dict]] = None,
    ) -> AsyncVerifyResult:
        """
        异步验证点击验证码

        Args:
            session_id: 会话ID
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

        data = await self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )

    async def get_image_captcha(
        self,
        type_: str = 'mixed',
        count: int = 4,
    ) -> AsyncImageCaptchaResponse:
        """
        异步获取图形验证码

        Args:
            type_: 验证码类型
            count: 字符数量

        Returns:
            图形验证码响应
        """
        data = await self._request(
            'GET', '/api/v1/captcha/image',
            params={'type': type_, 'count': count},
        )
        return AsyncImageCaptchaResponse(**data)

    async def verify_image_captcha(
        self,
        challenge_id: str,
        answer: str,
    ) -> AsyncVerifyResult:
        """
        异步验证图形验证码

        Args:
            challenge_id: 挑战ID
            answer: 答案

        Returns:
            验证结果
        """
        data = await self._request(
            'POST', '/api/v1/captcha/image/verify',
            data={'challenge_id': challenge_id, 'answer': answer},
        )
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )

    async def get_rotation_captcha(self) -> AsyncRotationCaptchaResponse:
        """异步获取旋转验证码"""
        data = await self._request('GET', '/api/v1/captcha/rotation')
        return AsyncRotationCaptchaResponse(**data)

    async def verify_rotation_captcha(
        self,
        challenge_id: str,
        angle: int,
    ) -> AsyncVerifyResult:
        """
        异步验证旋转验证码

        Args:
            challenge_id: 挑战ID
            angle: 旋转角度

        Returns:
            验证结果
        """
        data = await self._request(
            'POST', '/api/v1/captcha/rotation/verify',
            data={'challenge_id': challenge_id, 'angle': angle},
        )
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )

    async def get_gesture_captcha(self) -> AsyncGestureCaptchaResponse:
        """异步获取手势验证码"""
        data = await self._request('GET', '/api/v1/captcha/gesture')
        return AsyncGestureCaptchaResponse(
            session_id=data.get('session_id', ''),
            pattern=data.get('pattern'),
            grid_size=data.get('grid_size'),
            hint=data.get('hint'),
        )

    async def verify_gesture_captcha(
        self,
        session_id: str,
        pattern: List[int],
    ) -> AsyncVerifyResult:
        """
        异步验证手势验证码

        Args:
            session_id: 会话ID
            pattern: 手势模式

        Returns:
            验证结果
        """
        data = await self._request(
            'POST', '/api/v1/captcha/gesture/verify',
            data={'session_id': session_id, 'pattern': pattern},
        )
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
        )

    async def get_jigsaw_captcha(
        self,
        width: int = 300,
        height: int = 300,
        grid_size: int = 3,
    ) -> AsyncJigsawCaptchaResponse:
        """
        异步获取拼图验证码

        Args:
            width: 图片宽度
            height: 图片高度
            grid_size: 网格大小

        Returns:
            拼图验证码响应
        """
        data = await self._request(
            'GET', '/api/v1/captcha/jigsaw',
            params={'width': width, 'height': height, 'grid_size': grid_size},
        )
        pieces = [AsyncJigsawPiece(**p) for p in data.get('pieces', [])]
        return AsyncJigsawCaptchaResponse(
            session_id=data.get('session_id', ''),
            image_url=data.get('image_url', ''),
            pieces=pieces,
            grid_size=data.get('grid_size', 3),
            piece_width=data.get('piece_width', 0),
            piece_height=data.get('piece_height', 0),
            image_width=data.get('image_width', width),
            image_height=data.get('image_height', height),
        )

    async def verify_jigsaw_captcha(
        self,
        session_id: str,
        pieces: List[AsyncJigsawPiece],
    ) -> AsyncVerifyResult:
        """
        异步验证拼图验证码

        Args:
            session_id: 会话ID
            pieces: 碎片列表

        Returns:
            验证结果
        """
        data = await self._request(
            'POST', '/api/v1/captcha/jigsaw/verify',
            data={'session_id': session_id, 'pieces': [p.to_dict() for p in pieces]},
        )
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
        )

    async def verify_captcha(
        self,
        captcha_type: Union[AsyncCaptchaType, str],
        session_id: str,
        **kwargs,
    ) -> AsyncVerifyResult:
        """
        通用异步验证方法

        Args:
            captcha_type: 验证码类型
            session_id: 会话ID
            **kwargs: 其他验证参数

        Returns:
            验证结果
        """
        if isinstance(captcha_type, AsyncCaptchaType):
            captcha_type = captcha_type.value

        req_data = {
            'session_id': session_id,
            'type': captcha_type,
            **kwargs,
        }

        data = await self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            trajectory_result=data.get('trajectory_result'),
            risk_score=data.get('risk_score'),
            captcha_pass=data.get('captcha_pass'),
            fail_reason=data.get('fail_reason'),
        )

    async def close(self):
        """关闭异步客户端，释放资源"""
        if self._session and not self._session.closed:
            await self._session.close()

    async def __aenter__(self) -> 'AsyncCaptchaClient':
        """上下文管理器入口"""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """上下文管理器退出"""
        await self.close()


async def async_basic_example():
    """基础异步使用示例"""
    print("\n" + "="*50)
    print("异步基础示例")
    print("="*50)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        slider = await client.get_slider_captcha(width=320, height=160)
        print(f"会话ID: {slider.session_id}")

        result = await client.verify_slider_captcha(
            session_id=slider.session_id,
            x=150,
            y=slider.secret_y,
        )
        print(f"验证成功: {result.success}")


async def async_concurrent_example():
    """并发请求示例"""
    print("\n" + "="*50)
    print("异步并发示例")
    print("="*50)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        tasks = [
            client.get_slider_captcha(width=320, height=160)
            for _ in range(10)
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        success_count = 0
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                print(f"请求 {i+1} 失败: {result}")
            else:
                print(f"请求 {i+1} 成功: {result.session_id[:20]}...")
                success_count += 1

        print(f"\n成功率: {success_count}/{len(results)}")


async def async_mixed_captcha_example():
    """混合验证码示例"""
    print("\n" + "="*50)
    print("异步混合验证码示例")
    print("="*50)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        tasks = [
            client.get_slider_captcha(),
            client.get_click_captcha(),
            client.get_image_captcha(),
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        for i, result in enumerate(results):
            if isinstance(result, Exception):
                print(f"验证码 {i+1} 获取失败: {result}")
            else:
                print(f"验证码 {i+1} 获取成功")


async def async_error_handling_example():
    """异步错误处理示例"""
    print("\n" + "="*50)
    print("异步错误处理示例")
    print("="*50)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        try:
            result = await client.verify_slider_captcha(
                session_id="invalid-session-id",
                x=100,
            )
            print(f"验证结果: {result.success}")
        except AsyncCaptchaAPIError as e:
            print(f"API错误: {e.message}, 代码: {e.code}")
        except AsyncCaptchaTimeoutError as e:
            print(f"超时错误: {e}")
        except AsyncCaptchaNetworkError as e:
            print(f"网络错误: {e}")
        except AsyncCaptchaError as e:
            print(f"验证码错误: {e}")


async def async_batch_verification_example():
    """批量验证示例"""
    print("\n" + "="*50)
    print("异步批量验证示例")
    print("="*50)

    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        sliders = await asyncio.gather(
            client.get_slider_captcha() for _ in range(5)
        )

        verify_tasks = [
            client.verify_slider_captcha(
                session_id=slider.session_id,
                x=150,
                y=slider.secret_y,
            )
            for slider in sliders
        ]
        verify_results = await asyncio.gather(*verify_tasks, return_exceptions=True)

        success_count = sum(
            1 for r in verify_results
            if not isinstance(r, Exception) and r.success
        )
        print(f"批量验证成功率: {success_count}/{len(verify_results)}")


async def main():
    """主函数，运行所有示例"""
    print("="*50)
    print("Python 异步 SDK 完整示例")
    print("="*50)

    examples = [
        ("基础异步示例", async_basic_example),
        ("并发请求示例", async_concurrent_example),
        ("混合验证码示例", async_mixed_captcha_example),
        ("错误处理示例", async_error_handling_example),
        ("批量验证示例", async_batch_verification_example),
    ]

    for name, func in examples:
        try:
            await func()
        except Exception as e:
            print(f"示例 '{name}' 执行失败: {e}")

    print("\n" + "="*50)
    print("所有示例运行完成")
    print("="*50)


if __name__ == "__main__":
    asyncio.run(main())
