"""hjtpx Python SDK - 主客户端

提供完整的验证码服务封装，支持图片验证码、滑块验证码、点选验证码。
"""

import json
import base64
import urllib.parse
import urllib.request
import urllib.error
import time
from typing import Optional, Dict, Any, List
from dataclasses import asdict

from .exceptions import (
    SDKError,
    NetworkError,
    TimeoutError,
    InvalidResponseError,
    ServerError,
    InvalidParamsError,
    RateLimitedError,
    UnauthorizedError,
)
from .models import (
    ImageCaptchaRequest,
    ImageCaptchaResponse,
    SliderCaptchaResponse,
    ClickCaptchaResponse,
    VerifyCaptchaResponse,
    VerifyImageCaptchaResponse,
    ClickData,
    PoolStats,
    SDKResponse,
    SliderCaptchaRequest,
    ClickCaptchaRequest,
    VerifyCaptchaRequest,
    VerifyImageCaptchaRequest,
)


class Config:
    """SDK配置类"""

    DEFAULT_API_ENDPOINT = "http://localhost:8080"

    def __init__(
        self,
        base_url: str = DEFAULT_API_ENDPOINT,
        app_id: str = "",
        app_secret: str = "",
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 0.1,
        max_idle_conns: int = 10,
        max_open_conns: int = 100,
        debug_mode: bool = False,
    ):
        self.base_url = base_url
        self.app_id = app_id
        self.app_secret = app_secret
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self.max_idle_conns = max_idle_conns
        self.max_open_conns = max_open_conns
        self.debug_mode = debug_mode


class CaptchaClient:
    """验证码客户端

    提供完整的验证码服务封装，支持连接池、自动重试、详细错误处理。

    Example:
        client = CaptchaClient(
            base_url="http://localhost:8080",
            app_id="your-app-id",
            app_secret="your-app-secret"
        )

        # 生成滑块验证码
        slider = client.generate_slider_captcha()
        print(f"Challenge ID: {slider.challenge_id}")

        # 验证
        result = client.verify_slider_captcha(slider.challenge_id, "120")
        print(f"Success: {result.success}")
    """

    API_PATHS = {
        "image_captcha": "/api/v1/captcha/image",
        "image_verify": "/api/v1/captcha/image/verify",
        "slider_captcha": "/api/v1/captcha/slider",
        "click_captcha": "/api/v1/captcha/click",
        "verify": "/api/v1/captcha/verify",
    }

    def __init__(self, config: Optional[Config] = None):
        """初始化客户端

        Args:
            config: SDK配置，如果为None则使用默认配置
        """
        self.config = config or Config()
        self._stats = {
            "total_requests": 0,
            "failed_requests": 0,
            "successful_requests": 0,
            "retried_requests": 0,
            "last_error": None,
            "last_error_time": None,
        }

    def _build_url(self, path: str, params: Optional[Dict[str, Any]] = None) -> str:
        """构建完整URL"""
        url = self.config.base_url.rstrip("/") + path
        if params:
            query_string = urllib.parse.urlencode(params)
            url = f"{url}?{query_string}"
        return url

    def _create_request(
        self,
        method: str,
        url: str,
        data: Optional[Dict[str, Any]] = None,
    ) -> urllib.request.Request:
        """创建请求对象"""
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }

        if self.config.app_id:
            headers["X-App-ID"] = self.config.app_id
        if self.config.app_secret:
            headers["X-App-Secret"] = self.config.app_secret

        if data:
            json_data = json.dumps(data).encode("utf-8")
            request = urllib.request.Request(url, data=json_data, headers=headers, method=method)
        else:
            request = urllib.request.Request(url, headers=headers, method=method)

        return request

    def _do_request(
        self,
        method: str,
        path: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> SDKResponse:
        """执行HTTP请求"""
        url = self._build_url(path, params)
        request = self._create_request(method, url, data)

        self._stats["total_requests"] += 1

        last_error = None
        for attempt in range(self.config.max_retries + 1):
            if attempt > 0:
                self._stats["retried_requests"] += 1
                time.sleep(self.config.retry_delay * attempt)

            try:
                with urllib.request.urlopen(
                    request,
                    timeout=self.config.timeout,
                ) as response:
                    response_body = response.read().decode("utf-8")
                    response_data = json.loads(response_body)

                    if self.config.debug_mode:
                        print(f"[DEBUG] Response: {response_data}")

                    sdk_response = SDKResponse.from_dict(response_data)

                    if sdk_response.code != 0:
                        raise self._create_sdk_error(sdk_response.code, sdk_response.message)

                    self._stats["successful_requests"] += 1
                    return sdk_response

            except urllib.error.HTTPError as e:
                last_error = e
                if e.code == 429:
                    retry_after = None
                    if e.headers.get("Retry-After"):
                        try:
                            retry_after = int(e.headers["Retry-After"])
                        except ValueError:
                            pass
                    raise RateLimitedError(
                        message="Rate limited",
                        retry_after=retry_after,
                    )
                elif e.code == 401:
                    raise UnauthorizedError(message="Unauthorized")
                elif e.code >= 500:
                    if attempt < self.config.max_retries:
                        continue
                    raise ServerError(code=e.code, message=f"Server error: {e.code}")
                else:
                    raise ServerError(code=e.code, message=f"HTTP error: {e.code}")

            except urllib.error.URLError as e:
                last_error = e
                error_message = str(e.reason)
                if "timed out" in error_message.lower():
                    if attempt < self.config.max_retries:
                        continue
                    raise TimeoutError(message="Request timeout")
                elif "connection refused" in error_message.lower():
                    if attempt < self.config.max_retries:
                        continue
                    raise NetworkError(message="Connection refused")
                else:
                    raise NetworkError(message=error_message, cause=e)

        self._stats["failed_requests"] += 1
        self._stats["last_error"] = str(last_error)
        self._stats["last_error_time"] = time.strftime("%Y-%m-%d %H:%M:%S")

        raise NetworkError(message=str(last_error))

    def _create_sdk_error(self, code: int, message: str) -> SDKError:
        """创建SDK错误"""
        if code == 400:
            return InvalidParamsError(message=message)
        elif code == 401:
            return UnauthorizedError(message=message)
        elif code == 429:
            return RateLimitedError(message=message)
        elif code >= 500:
            return ServerError(code=code, message=message)
        else:
            return SDKError(code=code, message=message)

    def generate_image_captcha(
        self,
        request: Optional[ImageCaptchaRequest] = None,
    ) -> ImageCaptchaResponse:
        """生成图片验证码

        Args:
            request: 图片验证码请求参数

        Returns:
            ImageCaptchaResponse: 包含challenge_id和base64编码的图片

        Example:
            client = CaptchaClient()
            captcha = client.generate_image_captcha(
                ImageCaptchaRequest(captcha_type=CaptchaType.NUMBER, count=4)
            )
            print(f"Challenge ID: {captcha.challenge_id}")
        """
        if request is None:
            request = ImageCaptchaRequest()

        params = request.to_params()
        response = self._do_request("GET", self.API_PATHS["image_captcha"], params=params)

        if response.data:
            return ImageCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def verify_image_captcha(
        self,
        challenge_id: str,
        answer: str,
    ) -> VerifyImageCaptchaResponse:
        """验证图片验证码

        Args:
            challenge_id: 验证码ID
            answer: 用户输入的答案

        Returns:
            VerifyImageCaptchaResponse: 验证结果

        Example:
            result = client.verify_image_captcha(captcha.challenge_id, "a1b2")
            print(f"Success: {result.success}")
        """
        if not challenge_id:
            raise InvalidParamsError(message="challenge_id is required")
        if not answer:
            raise InvalidParamsError(message="answer is required")

        request_data = {
            "challenge_id": challenge_id,
            "answer": answer,
        }

        response = self._do_request("POST", self.API_PATHS["image_verify"], data=request_data)

        if response.data:
            return VerifyImageCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def generate_slider_captcha(
        self,
        request: Optional[SliderCaptchaRequest] = None,
    ) -> SliderCaptchaResponse:
        """生成滑块验证码

        Args:
            request: 滑块验证码请求参数

        Returns:
            SliderCaptchaResponse: 包含背景图、滑块图等信息

        Example:
            client = CaptchaClient()
            slider = client.generate_slider_captcha(
                SliderCaptchaRequest(width=360, height=220)
            )
            print(f"Challenge ID: {slider.challenge_id}")
        """
        if request is None:
            request = SliderCaptchaRequest()

        params = request.to_params()
        response = self._do_request("GET", self.API_PATHS["slider_captcha"], params=params)

        if response.data:
            return SliderCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def verify_slider_captcha(
        self,
        challenge_id: str,
        offset: str,
    ) -> VerifyCaptchaResponse:
        """验证滑块验证码

        Args:
            challenge_id: 验证码ID
            offset: 滑块偏移量（字符串形式的数字）

        Returns:
            VerifyCaptchaResponse: 验证结果

        Example:
            result = client.verify_slider_captcha(slider.challenge_id, "120")
            print(f"Success: {result.success}, Risk Score: {result.score}")
        """
        if not challenge_id:
            raise InvalidParamsError(message="challenge_id is required")
        if not offset:
            raise InvalidParamsError(message="offset is required")

        request_data = {
            "challenge_id": challenge_id,
            "action": "slide",
            "data": {
                "offset": offset,
            },
        }

        response = self._do_request("POST", self.API_PATHS["verify"], data=request_data)

        if response.data:
            return VerifyCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def generate_click_captcha(
        self,
        request: Optional[ClickCaptchaRequest] = None,
    ) -> ClickCaptchaResponse:
        """生成点选验证码

        Args:
            request: 点选验证码请求参数

        Returns:
            ClickCaptchaResponse: 包含背景图、图标位置等信息

        Example:
            client = CaptchaClient()
            click = client.generate_click_captcha(
                ClickCaptchaRequest(width=360, height=220, icon_count=4)
            )
            print(f"Challenge ID: {click.challenge_id}")
        """
        if request is None:
            request = ClickCaptchaRequest()

        params = request.to_params()
        response = self._do_request("GET", self.API_PATHS["click_captcha"], params=params)

        if response.data:
            return ClickCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def verify_click_captcha(
        self,
        challenge_id: str,
        clicks: List[ClickData],
    ) -> VerifyCaptchaResponse:
        """验证点选验证码

        Args:
            challenge_id: 验证码ID
            clicks: 点击数据列表

        Returns:
            VerifyCaptchaResponse: 验证结果

        Example:
            clicks = [
                ClickData(x=100, y=120, duration=500),
                ClickData(x=200, y=150, duration=300),
            ]
            result = client.verify_click_captcha(click.challenge_id, clicks)
            print(f"Success: {result.success}")
        """
        if not challenge_id:
            raise InvalidParamsError(message="challenge_id is required")
        if not clicks:
            raise InvalidParamsError(message="clicks is required")

        clicks_data = [c.to_dict() for c in clicks]
        request_data = {
            "challenge_id": challenge_id,
            "action": "click",
            "data": {
                "clicks": clicks_data,
            },
        }

        response = self._do_request("POST", self.API_PATHS["verify"], data=request_data)

        if response.data:
            return VerifyCaptchaResponse.from_dict(response.data)
        raise InvalidResponseError(message="Empty response data")

    def extract_base64_image(self, data_uri: str) -> bytes:
        """从data URI中提取图片数据

        Args:
            data_uri: Base64编码的图片URI

        Returns:
            bytes: 原始图片数据

        Example:
            image_data = client.extract_base64_image(captcha.image)
            with open("captcha.png", "wb") as f:
                f.write(image_data)
        """
        if not data_uri:
            raise InvalidParamsError(message="data_uri is required")

        if data_uri.startswith("data:image/png;base64,"):
            base64_data = data_uri[len("data:image/png;base64,") :]
            return base64.b64decode(base64_data)
        elif data_uri.startswith("data:image/jpeg;base64,"):
            base64_data = data_uri[len("data:image/jpeg;base64,") :]
            return base64.b64decode(base64_data)
        else:
            raise InvalidParamsError(message="Unsupported image format")

    def get_stats(self) -> PoolStats:
        """获取连接池统计信息

        Returns:
            PoolStats: 包含请求统计信息

        Example:
            stats = client.get_stats()
            print(f"Total Requests: {stats.total_requests}")
            print(f"Success Rate: {stats.success_rate:.2f}%")
        """
        total = self._stats["total_requests"]
        success = self._stats["successful_requests"]
        success_rate = (success / total * 100) if total > 0 else 0.0

        return PoolStats(
            active_connections=0,
            idle_connections=self.config.max_idle_conns,
            total_requests=total,
            failed_requests=self._stats["failed_requests"],
            successful_requests=success,
            retried_requests=self._stats["retried_requests"],
            success_rate=success_rate,
            last_error=self._stats["last_error"],
            last_error_time=self._stats["last_error_time"],
        )

    def set_debug_mode(self, enabled: bool) -> None:
        """设置调试模式

        Args:
            enabled: 是否启用调试模式
        """
        self.config.debug_mode = enabled

    def set_timeout(self, timeout: float) -> None:
        """设置请求超时时间

        Args:
            timeout: 超时时间（秒）
        """
        self.config.timeout = timeout

    def set_max_retries(self, max_retries: int) -> None:
        """设置最大重试次数

        Args:
            max_retries: 最大重试次数
        """
        self.config.max_retries = max_retries

    def set_retry_delay(self, delay: float) -> None:
        """设置重试延迟

        Args:
            delay: 延迟时间（秒）
        """
        self.config.retry_delay = delay

    def close(self) -> None:
        """关闭客户端，释放资源"""
        pass


class Client(CaptchaClient):
    """CaptchaClient的别名，保持向后兼容"""

    pass


def new_client(
    base_url: str = Config.DEFAULT_API_ENDPOINT,
    app_id: str = "",
    app_secret: str = "",
    timeout: float = 30.0,
    **kwargs,
) -> CaptchaClient:
    """创建新的验证码客户端的便捷函数

    Args:
        base_url: API端点
        app_id: 应用ID
        app_secret: 应用密钥
        timeout: 超时时间
        **kwargs: 其他配置参数

    Returns:
        CaptchaClient: 配置好的客户端实例

    Example:
        client = new_client(
            base_url="http://localhost:8080",
            app_id="my-app-id",
            app_secret="my-app-secret"
        )
    """
    config = Config(
        base_url=base_url,
        app_id=app_id,
        app_secret=app_secret,
        timeout=timeout,
        **kwargs,
    )
    return CaptchaClient(config)
