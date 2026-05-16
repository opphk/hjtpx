"""hjtpx Python SDK - 异常定义"""

from typing import Optional


class SDKError(Exception):
    """SDK基础异常类"""

    def __init__(self, code: int, message: str, cause: Optional[Exception] = None):
        self.code = code
        self.message = message
        self.cause = cause
        super().__init__(f"SDKError(code={code}, message={message})")

    def __repr__(self) -> str:
        return f"SDKError(code={self.code}, message='{self.message}')"

    def __str__(self) -> str:
        if self.cause:
            return f"SDKError(code={self.code}, message='{self.message}'): {self.cause}"
        return f"SDKError(code={self.code}, message='{self.message}')"


class NetworkError(SDKError):
    """网络错误"""

    def __init__(self, message: str = "Network error", cause: Optional[Exception] = None):
        super().__init__(code=0, message=message, cause=cause)


class TimeoutError(SDKError):
    """请求超时"""

    def __init__(self, message: str = "Request timeout", cause: Optional[Exception] = None):
        super().__init__(code=408, message=message, cause=cause)


class InvalidResponseError(SDKError):
    """无效响应"""

    def __init__(self, message: str = "Invalid response", cause: Optional[Exception] = None):
        super().__init__(code=500, message=message, cause=cause)


class ServerError(SDKError):
    """服务器错误"""

    def __init__(self, code: int, message: str = "Server error", cause: Optional[Exception] = None):
        super().__init__(code=code, message=message, cause=cause)


class InvalidParamsError(SDKError):
    """参数错误"""

    def __init__(self, message: str = "Invalid parameters", cause: Optional[Exception] = None):
        super().__init__(code=400, message=message, cause=cause)


class VerificationFailedError(SDKError):
    """验证失败"""

    def __init__(self, message: str = "Verification failed", cause: Optional[Exception] = None):
        super().__init__(code=2002, message=message, cause=cause)


class RateLimitedError(SDKError):
    """请求频率限制"""

    def __init__(self, message: str = "Rate limited", cause: Optional[Exception] = None, retry_after: Optional[int] = None):
        super().__init__(code=429, message=message, cause=cause)
        self.retry_after = retry_after


class UnauthorizedError(SDKError):
    """未授权"""

    def __init__(self, message: str = "Unauthorized", cause: Optional[Exception] = None):
        super().__init__(code=401, message=message, cause=cause)


class InternalError(SDKError):
    """内部错误"""

    def __init__(self, message: str = "Internal error", cause: Optional[Exception] = None):
        super().__init__(code=500, message=message, cause=cause)


def is_sdk_error(error: Exception) -> bool:
    """判断是否为SDK错误"""
    return isinstance(error, SDKError)


def get_error_code(error: Exception) -> int:
    """获取错误码"""
    if isinstance(error, SDKError):
        return error.code
    return 0
