"""
HJTPX SDK v2.0 - Python Implementation
Enterprise-grade captcha verification SDK with advanced features
"""

import json
import time
import hashlib
import hmac
import asyncio
from typing import Dict, List, Optional, Any, Callable, Union
from datetime import datetime, timedelta
from dataclasses import dataclass, field
from enum import Enum
from functools import wraps
import logging

logger = logging.getLogger(__name__)

SDK_VERSION = "2.0.0"
API_V2_BASE_URL = "https://api.hjtpx.com/v2"


class CaptchaType(Enum):
    IMAGE = "image"
    SLIDER = "slider"
    VOICE = "voice"
    SMS = "sms"
    EMAIL = "email"
    TOKEN = "token"
    BEHAVIORAL = "behavioral"
    ADAPTIVE = "adaptive"


class SecurityLevel(Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    ENTERPRISE = "enterprise"


@dataclass
class CaptchaRequest:
    app_id: str
    captcha_type: str
    action: str = "create"
    user_id: Optional[str] = None
    session_id: Optional[str] = None
    ip_address: Optional[str] = None
    user_agent: Optional[str] = None
    parameters: Dict[str, Any] = field(default_factory=dict)
    metadata: Dict[str, str] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "app_id": self.app_id,
            "captcha_type": self.captcha_type,
            "action": self.action,
            "user_id": self.user_id,
            "session_id": self.session_id,
            "ip_address": self.ip_address,
            "user_agent": self.user_agent,
            "parameters": self.parameters,
            "metadata": self.metadata
        }


@dataclass
class CaptchaResponse:
    captcha_id: str
    status: str
    type: str
    data: Dict[str, Any]
    expires_at: Optional[datetime] = None
    created_at: datetime = field(default_factory=datetime.utcnow)
    metadata: Dict[str, str] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'CaptchaResponse':
        expires_at = None
        if data.get("expires_at"):
            expires_at = datetime.fromisoformat(data["expires_at"])

        return cls(
            captcha_id=data["captcha_id"],
            status=data["status"],
            type=data["type"],
            data=data.get("data", {}),
            expires_at=expires_at,
            created_at=datetime.fromisoformat(data.get("created_at", datetime.utcnow().isoformat())),
            metadata=data.get("metadata", {})
        )


@dataclass
class VerificationRequest:
    captcha_id: str
    token: str
    solution: Any = None
    user_id: Optional[str] = None
    session_id: Optional[str] = None
    ip_address: Optional[str] = None
    parameters: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "captcha_id": self.captcha_id,
            "token": self.token,
            "solution": self.solution,
            "user_id": self.user_id,
            "session_id": self.session_id,
            "ip_address": self.ip_address,
            "parameters": self.parameters
        }


@dataclass
class VerificationResponse:
    valid: bool
    score: Optional[float] = None
    risk_level: Optional[str] = None
    reasons: List[str] = field(default_factory=list)
    session_id: Optional[str] = None
    remaining_tries: Optional[int] = None
    metadata: Dict[str, str] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'VerificationResponse':
        return cls(
            valid=data["valid"],
            score=data.get("score"),
            risk_level=data.get("risk_level"),
            reasons=data.get("reasons", []),
            session_id=data.get("session_id"),
            remaining_tries=data.get("remaining_tries"),
            metadata=data.get("metadata", {})
        )


@dataclass
class AnalyticsRequest:
    app_id: str
    start_date: datetime
    end_date: datetime
    metrics: List[str]
    dimensions: List[str] = field(default_factory=list)
    filters: Dict[str, Any] = field(default_factory=dict)


@dataclass
class MetricResult:
    metric: str
    value: Any
    breakdown: Dict[str, Any] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'MetricResult':
        return cls(
            metric=data["metric"],
            value=data["value"],
            breakdown=data.get("breakdown")
        )


@dataclass
class AnalyticsResponse:
    results: List[MetricResult]
    summary: Optional[Dict[str, Any]] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AnalyticsResponse':
        results = [MetricResult.from_dict(r) for r in data.get("results", [])]
        return cls(
            results=results,
            summary=data.get("summary")
        )


class Plugin:
    """Base plugin interface"""

    def name(self) -> str:
        raise NotImplementedError

    def version(self) -> str:
        raise NotImplementedError

    async def execute(self, request: CaptchaRequest) -> Optional[CaptchaResponse]:
        return None


class Middleware:
    """Base middleware interface"""

    async def before_request(self, request: Dict[str, Any]) -> None:
        pass

    async def after_response(self, response: Any) -> None:
        pass


class RetryPlugin(Plugin):
    """Plugin for automatic retry with exponential backoff"""

    def __init__(self, max_retries: int = 3, base_delay: float = 1.0):
        self.max_retries = max_retries
        self.base_delay = base_delay

    def name(self) -> str:
        return "retry"

    def version(self) -> str:
        return "1.0.0"

    async def execute(self, request: CaptchaRequest) -> Optional[CaptchaResponse]:
        return None


class CachePlugin(Plugin):
    """Plugin for caching captcha responses"""

    def __init__(self, ttl: int = 300):
        self.cache: Dict[str, tuple] = {}
        self.ttl = ttl

    def name(self) -> str:
        return "cache"

    def version(self) -> str:
        return "1.0.0"

    def get(self, key: str) -> Optional[CaptchaResponse]:
        if key in self.cache:
            response, timestamp = self.cache[key]
            if datetime.utcnow() - timestamp < timedelta(seconds=self.ttl):
                return response
            del self.cache[key]
        return None

    def set(self, key: str, response: CaptchaResponse) -> None:
        self.cache[key] = (response, datetime.utcnow())

    async def execute(self, request: CaptchaRequest) -> Optional[CaptchaResponse]:
        if request.session_id:
            return self.get(request.session_id)
        return None


class RateLimitPlugin(Plugin):
    """Plugin for rate limiting requests"""

    def __init__(self, max_requests: int, window: int):
        self.max_requests = max_requests
        self.window = window
        self.requests: List[float] = []

    def name(self) -> str:
        return "rate_limiter"

    def version(self) -> str:
        return "1.0.0"

    async def execute(self, request: CaptchaRequest) -> Optional[CaptchaResponse]:
        now = time.time()
        cutoff = now - self.window

        self.requests = [t for t in self.requests if t > cutoff]

        if len(self.requests) >= self.max_requests:
            raise RateLimitExceeded(f"Rate limit exceeded: {self.max_requests} requests per {self.window}s")

        self.requests.append(now)
        return None


class MetricsPlugin(Plugin):
    """Plugin for collecting request metrics"""

    def __init__(self):
        self.total_requests = 0
        self.success_count = 0
        self.failure_count = 0
        self.total_latency = 0.0

    def name(self) -> str:
        return "metrics"

    def version(self) -> str:
        return "1.0.0"

    def record_success(self, latency_ms: float) -> None:
        self.total_requests += 1
        self.success_count += 1
        self.total_latency += latency_ms

    def record_failure(self) -> None:
        self.total_requests += 1
        self.failure_count += 1

    def get_metrics(self) -> Dict[str, Any]:
        avg_latency = self.total_latency / self.total_requests if self.total_requests > 0 else 0
        return {
            "total_requests": self.total_requests,
            "success_count": self.success_count,
            "failure_count": self.failure_count,
            "success_rate": self.success_count / self.total_requests if self.total_requests > 0 else 0,
            "avg_latency_ms": avg_latency
        }

    async def execute(self, request: CaptchaRequest) -> Optional[CaptchaResponse]:
        return None


class RateLimitExceeded(Exception):
    pass


class SDKError(Exception):
    def __init__(self, code: str, message: str, details: str = None):
        self.code = code
        self.message = message
        self.details = details
        super().__init__(f"[{code}] {message}: {details}")


class HjtpxSDK:
    """
    HJTPX SDK v2.0 for Python

    Enterprise-grade captcha verification SDK with support for:
    - Multiple captcha types
    - Plugin architecture
    - Middleware support
    - Rate limiting
    - Caching
    - Metrics collection
    - Circuit breaker pattern
    """

    def __init__(
        self,
        api_key: str,
        api_secret: str,
        base_url: str = API_V2_BASE_URL,
        timeout: int = 30,
        retry_attempts: int = 3,
        retry_delay: float = 1.0,
        enable_debug: bool = False
    ):
        self.api_key = api_key
        self.api_secret = api_secret
        self.base_url = base_url
        self.timeout = timeout
        self.retry_attempts = retry_attempts
        self.retry_delay = retry_delay
        self.enable_debug = enable_debug

        self.plugins: List[Plugin] = []
        self.middleware: List[Middleware] = []

        self._circuit_breaker_state = "closed"
        self._circuit_breaker_failures = 0
        self._circuit_breaker_threshold = 5
        self._circuit_breaker_timeout = 60

    def use_plugin(self, plugin: Plugin) -> None:
        """Register a plugin"""
        self.plugins.append(plugin)

    def use_middleware(self, middleware: Middleware) -> None:
        """Register middleware"""
        self.middleware.append(middleware)

    async def create_captcha(self, request: CaptchaRequest) -> CaptchaResponse:
        """Create a new captcha challenge"""
        for plugin in self.plugins:
            if plugin.name() == "preprocessor":
                result = await plugin.execute(request)
                if result:
                    return result

        for mw in self.middleware:
            await mw.before_request(request.to_dict())

        start_time = time.time()
        try:
            response = await self._do_request("POST", "/captcha/create", request.to_dict())
            captcha_response = CaptchaResponse.from_dict(response)

            for plugin in self.plugins:
                if plugin.name() == "cache" and request.session_id:
                    cache_plugin = plugin
                    cache_plugin.set(request.session_id, captcha_response)

            if hasattr(self, '_metrics_plugin'):
                latency_ms = (time.time() - start_time) * 1000
                self._metrics_plugin.record_success(latency_ms)

            return captcha_response
        except Exception as e:
            if hasattr(self, '_metrics_plugin'):
                self._metrics_plugin.record_failure()
            raise

    async def verify(self, request: VerificationRequest) -> VerificationResponse:
        """Verify a captcha solution"""
        start_time = time.time()

        try:
            response = await self._do_request("POST", "/captcha/verify", request.to_dict())
            return VerificationResponse.from_dict(response)
        finally:
            if hasattr(self, '_metrics_plugin'):
                latency_ms = (time.time() - start_time) * 1000
                self._metrics_plugin.record_success(latency_ms)

    async def get_analytics(
        self,
        app_id: str,
        start_date: datetime,
        end_date: datetime,
        metrics: List[str],
        dimensions: List[str] = None,
        filters: Dict[str, Any] = None
    ) -> AnalyticsResponse:
        """Get analytics data"""
        request = {
            "app_id": app_id,
            "start_date": start_date.isoformat(),
            "end_date": end_date.isoformat(),
            "metrics": metrics,
            "dimensions": dimensions or [],
            "filters": filters or {}
        }

        response = await self._do_request("POST", "/analytics/query", request)
        return AnalyticsResponse.from_dict(response)

    async def get_app_config(self, app_id: str) -> Dict[str, Any]:
        """Get application configuration"""
        return await self._do_request("GET", f"/app/{app_id}/config")

    async def update_app_config(self, app_id: str, config: Dict[str, Any]) -> Dict[str, Any]:
        """Update application configuration"""
        return await self._do_request("PUT", f"/app/{app_id}/config", config)

    async def register_webhook(self, app_id: str, event_type: str, url: str) -> Dict[str, Any]:
        """Register a webhook"""
        request = {
            "app_id": app_id,
            "event": event_type,
            "webhook_url": url
        }
        return await self._do_request("POST", "/webhooks/register", request)

    async def list_webhooks(self, app_id: str) -> List[Dict[str, Any]]:
        """List all webhooks for an app"""
        response = await self._do_request("GET", f"/app/{app_id}/webhooks")
        return response.get("webhooks", [])

    async def _do_request(
        self,
        method: str,
        endpoint: str,
        data: Dict[str, Any] = None
    ) -> Dict[str, Any]:
        """Execute HTTP request with retry logic"""
        if self._circuit_breaker_state == "open":
            if time.time() - self._circuit_breaker_failures > self._circuit_breaker_timeout:
                self._circuit_breaker_state = "half-open"
            else:
                raise SDKError("CIRCUIT_OPEN", "Circuit breaker is open")

        url = f"{self.base_url}{endpoint}"

        for attempt in range(self.retry_attempts + 1):
            try:
                headers = self._get_headers(data)

                if self.enable_debug:
                    logger.debug(f"Request: {method} {url} {json.dumps(data, indent=2)}")

                response = await self._make_request(method, url, headers, data)

                if 200 <= response.status_code < 300:
                    if response.status_code == 204:
                        return {}
                    return response.json()

                if response.status_code >= 500 and attempt < self.retry_attempts:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue

                error_data = response.json()
                raise SDKError(
                    error_data.get("code", "UNKNOWN"),
                    error_data.get("message", "Request failed"),
                    error_data.get("details")
                )

            except Exception as e:
                if isinstance(e, SDKError):
                    raise

                if attempt < self.retry_attempts:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    self._circuit_breaker_failures += 1
                    continue

                raise SDKError("NETWORK_ERROR", str(e))

        raise SDKError("MAX_RETRIES", "Maximum retry attempts exceeded")

    async def _make_request(
        self,
        method: str,
        url: str,
        headers: Dict[str, str],
        data: Dict[str, Any] = None
    ) -> Any:
        """Make HTTP request (stub for actual HTTP library)"""
        return MockResponse(200, {"captcha_id": "test-123", "status": "success"})

    def _get_headers(self, data: Dict[str, Any] = None) -> Dict[str, str]:
        """Generate request headers"""
        timestamp = str(int(time.time()))
        headers = {
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-Timestamp": timestamp,
            "X-SDK-Version": SDK_VERSION
        }

        if data:
            payload = json.dumps(data, separators=(',', ':'))
            signature = self._generate_signature(payload, timestamp)
            headers["X-Signature"] = signature

        return headers

    def _generate_signature(self, payload: str, timestamp: str) -> str:
        """Generate HMAC-SHA256 signature"""
        message = f"{timestamp}:{payload}"
        signature = hmac.new(
            self.api_secret.encode(),
            message.encode(),
            hashlib.sha256
        ).hexdigest()
        return signature


class MockResponse:
    """Mock HTTP response for testing"""

    def __init__(self, status_code: int, json_data: Dict[str, Any]):
        self.status_code = status_code
        self._json_data = json_data

    def json(self) -> Dict[str, Any]:
        return self._json_data


class CaptchaBuilder:
    """Fluent builder for CaptchaRequest"""

    def __init__(self, sdk: HjtpxSDK, app_id: str, captcha_type: str):
        self.sdk = sdk
        self.request = CaptchaRequest(
            app_id=app_id,
            captcha_type=captcha_type,
            parameters={},
            metadata={}
        )

    def user_id(self, user_id: str) -> 'CaptchaBuilder':
        self.request.user_id = user_id
        return self

    def session_id(self, session_id: str) -> 'CaptchaBuilder':
        self.request.session_id = session_id
        return self

    def ip_address(self, ip: str) -> 'CaptchaBuilder':
        self.request.ip_address = ip
        return self

    def user_agent(self, ua: str) -> 'CaptchaBuilder':
        self.request.user_agent = ua
        return self

    def parameter(self, key: str, value: Any) -> 'CaptchaBuilder':
        self.request.parameters[key] = value
        return self

    def metadata(self, key: str, value: str) -> 'CaptchaBuilder':
        self.request.metadata[key] = value
        return self

    async def build(self) -> CaptchaResponse:
        return await self.sdk.create_captcha(self.request)


def async_retry(max_attempts: int = 3, delay: float = 1.0):
    """Decorator for async function retry with exponential backoff"""
    def decorator(func: Callable) -> Callable:
        @wraps(func)
        async def wrapper(*args, **kwargs):
            last_exception = None
            for attempt in range(max_attempts):
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    if attempt < max_attempts - 1:
                        await asyncio.sleep(delay * (2 ** attempt))
            raise last_exception
        return wrapper
    return decorator


class CircuitBreaker:
    """Circuit breaker pattern implementation"""

    def __init__(self, failure_threshold: int = 5, timeout: int = 60):
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.failures = 0
        self.last_failure_time = None
        self.state = "closed"

    def call(self, func: Callable, *args, **kwargs):
        if self.state == "open":
            if time.time() - self.last_failure_time > self.timeout:
                self.state = "half-open"
                self.failures = 0
            else:
                raise SDKError("CIRCUIT_OPEN", "Circuit breaker is open")

        try:
            result = func(*args, **kwargs)
            if self.state == "half-open":
                self.state = "closed"
                self.failures = 0
            return result
        except Exception as e:
            self.failures += 1
            self.last_failure_time = time.time()
            if self.failures >= self.failure_threshold:
                self.state = "open"
            raise

    def get_state(self) -> str:
        return self.state


class RateLimiter:
    """Token bucket rate limiter"""

    def __init__(self, rate: int, per: int):
        self.rate = rate
        self.per = per
        self.tokens = rate
        self.last_update = time.time()

    def allow(self) -> bool:
        now = time.time()
        elapsed = now - self.last_update
        self.last_update = now

        self.tokens = min(self.rate, self.tokens + elapsed * (self.rate / self.per))

        if self.tokens >= 1:
            self.tokens -= 1
            return True
        return False


def enable_logging(level: int = logging.INFO) -> None:
    """Enable SDK logging"""
    logging.basicConfig(
        level=level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
