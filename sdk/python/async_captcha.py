"""
Asyncio Async Support for Python SDK

Provides async/await compatible versions of all SDK methods.
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
    """Captcha type enum"""
    SLIDER = "slider"
    CLICK = "click"
    IMAGE = "image"
    ROTATION = "rotation"
    GESTURE = "gesture"
    JIGSAW = "jigsaw"


class AsyncClickMode(Enum):
    """Click captcha mode enum"""
    NUMBER = "number"
    LETTER = "letter"
    CHINESE = "chinese"
    MIXED = "mixed"
    ICON = "icon"


class AsyncCaptchaError(Exception):
    """Base async captcha error"""
    pass


class AsyncCaptchaAPIError(AsyncCaptchaError):
    """API error exception"""
    def __init__(self, message: str, code: Optional[int] = None, data: Optional[Any] = None):
        self.message = message
        self.code = code
        self.data = data
        super().__init__(f"API Error: {message} (code: {code})")


class AsyncCaptchaNetworkError(AsyncCaptchaError):
    """Network error exception"""
    pass


class AsyncCaptchaTimeoutError(AsyncCaptchaNetworkError):
    """Timeout error exception"""
    pass


@dataclass
class AsyncTrajectoryPoint:
    """Trajectory point for async client"""
    x: int
    y: int
    t: int

    def to_dict(self) -> Dict[str, int]:
        return asdict(self)


@dataclass
class AsyncSliderCaptchaResponse:
    """Slider captcha response for async client"""
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
    """Click captcha response for async client"""
    session_id: str
    image_url: str
    hint: str
    hint_order: List[int]
    max_points: int
    mode: str
    allow_shuffle: bool
    points: Optional[List[List[int]]] = None


@dataclass
class AsyncVerifyResult:
    """Verify result for async client"""
    success: bool
    message: str
    remaining_attempts: Optional[int] = None
    trajectory_result: Optional[Dict] = None
    risk_score: Optional[float] = None


class AsyncCaptchaClient:
    """Async captcha client with asyncio support"""

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        timeout: int = 30,
        max_retries: int = 3,
        retry_backoff_factor: float = 0.5,
        max_connections: int = 100,
    ):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.max_retries = max_retries
        self.retry_backoff_factor = retry_backoff_factor
        self.max_connections = max_connections
        self._token = None
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        """Get or create aiohttp session"""
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
        """Get request headers"""
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
        """Send async request"""
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
                await asyncio.sleep(self.retry_backoff_factor * (2 ** attempt))

            except aiohttp.ClientError as e:
                if attempt >= self.max_retries:
                    raise AsyncCaptchaNetworkError(str(e))
                await asyncio.sleep(self.retry_backoff_factor * (2 ** attempt))

    async def get_slider_captcha(
        self,
        width: int = 320,
        height: int = 160,
        tolerance: int = 8,
    ) -> AsyncSliderCaptchaResponse:
        """Get slider captcha asynchronously"""
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
    ) -> AsyncVerifyResult:
        """Verify slider captcha asynchronously"""
        req_data = {
            'session_id': session_id,
            'type': 'slider',
            'x': x,
        }
        if y is not None:
            req_data['y'] = y
        if trajectory:
            req_data['trajectory'] = [p.to_dict() for p in trajectory]

        data = await self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            trajectory_result=data.get('trajectory_result'),
            risk_score=data.get('risk_score'),
        )

    async def get_click_captcha(
        self,
        mode: Union[AsyncClickMode, str] = AsyncClickMode.NUMBER,
        max_points: int = 3,
        allow_shuffle: bool = True,
    ) -> AsyncClickCaptchaResponse:
        """Get click captcha asynchronously"""
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
    ) -> AsyncVerifyResult:
        """Verify click captcha asynchronously"""
        req_data = {
            'session_id': session_id,
            'type': 'click',
            'points': points,
        }
        if click_sequence:
            req_data['click_sequence'] = click_sequence

        data = await self._request('POST', '/api/v1/captcha/verify', data=req_data)
        return AsyncVerifyResult(
            success=data.get('success', False),
            message=data.get('message', ''),
            remaining_attempts=data.get('remaining_attempts'),
            risk_score=data.get('risk_score'),
        )

    async def close(self):
        """Close the async client"""
        if self._session and not self._session.closed:
            await self._session.close()

    async def __aenter__(self) -> 'AsyncCaptchaClient':
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()


async def async_example():
    """Example of async captcha operations"""
    async with AsyncCaptchaClient(base_url="http://localhost:8080") as client:
        slider = await client.get_slider_captcha(width=320, height=160)
        print(f"Session ID: {slider.session_id}")

        result = await client.verify_slider_captcha(
            session_id=slider.session_id,
            x=150,
            y=slider.secret_y,
        )
        print(f"Success: {result.success}")


if __name__ == "__main__":
    asyncio.run(async_example())
