"""CaptchaX Python SDK client implementation."""

import json
import time
from typing import Optional, Dict, Any, List, Union
from dataclasses import dataclass, asdict, field
from urllib.parse import urljoin

import httpx

from .types import (
    ApiVersion,
    CaptchaConfig,
    SliderCaptchaResult,
    SliderVerifyResult,
    ClickCaptchaResult,
    ClickVerifyResult,
    PuzzleCaptchaResult,
    PuzzleVerifyResult,
    Scenario,
    Webhook,
    BatchVerifyItem,
    BatchVerifyResult,
    BatchVerifyResponse,
    BatchVerifySummary,
    HealthStatus,
    CharPosition,
    CaptchaResponse,
)


class CaptchaXError(Exception):
    """CaptchaX SDK exception."""

    def __init__(self, message: str, code: int = 500, status_code: int = 500, details: Optional[Any] = None):
        super().__init__(message)
        self.message = message
        self.code = code
        self.status_code = status_code
        self.details = details

    def __str__(self) -> str:
        return f"CaptchaXError({self.code}): {self.message}"


class HttpClient:
    """HTTP client with retry and timeout support."""

    def __init__(
        self,
        base_url: str,
        timeout: int = 10000,
        retry_times: int = 3,
    ):
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout / 1000.0
        self.retry_times = retry_times
        self._headers: Dict[str, str] = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
        }

    def set_header(self, key: str, value: str) -> None:
        """Set a default header."""
        self._headers[key] = value

    def set_headers(self, headers: Dict[str, str]) -> None:
        """Set multiple default headers."""
        self._headers.update(headers)

    def _request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
        deduplication_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Make HTTP request with retry logic."""
        url = urljoin(self.base_url + '/', endpoint.lstrip('/'))
        headers = dict(self._headers)

        if deduplication_id:
            headers['X-Deduplication-ID'] = deduplication_id

        last_error: Optional[Exception] = None

        for attempt in range(self.retry_times + 1):
            try:
                with httpx.Client(timeout=self.timeout) as client:
                    if method.upper() == 'GET':
                        response = client.get(url, headers=headers, params=params)
                    elif method.upper() == 'POST':
                        response = client.post(url, headers=headers, json=data)
                    elif method.upper() == 'PUT':
                        response = client.put(url, headers=headers, json=data)
                    elif method.upper() == 'DELETE':
                        response = client.delete(url, headers=headers, params=params)
                    else:
                        raise ValueError(f"Unsupported HTTP method: {method}")

                    response.raise_for_status()
                    result = response.json()

                    if result.get('code') and result.get('code') != 200:
                        raise CaptchaXError(
                            message=result.get('message', 'Unknown error'),
                            code=result['code'],
                            status_code=response.status_code,
                            details=result,
                        )

                    return result

            except httpx.HTTPStatusError as e:
                last_error = e
                if e.response.status_code >= 500 and attempt < self.retry_times:
                    time.sleep(2 ** attempt)
                    continue
                try:
                    error_data = e.response.json()
                    raise CaptchaXError(
                        message=error_data.get('message', str(e)),
                        code=error_data.get('code', e.response.status_code),
                        status_code=e.response.status_code,
                        details=error_data,
                    )
                except json.JSONDecodeError:
                    raise CaptchaXError(
                        message=str(e),
                        code=e.response.status_code,
                        status_code=e.response.status_code,
                    )

            except httpx.RequestError as e:
                last_error = e
                if attempt < self.retry_times:
                    time.sleep(2 ** attempt)
                    continue
                raise CaptchaXError(
                    message=f"Request error: {str(e)}",
                    code=0,
                    status_code=0,
                )

            except CaptchaXError:
                raise

            except Exception as e:
                last_error = e
                if attempt < self.retry_times:
                    time.sleep(2 ** attempt)
                    continue
                raise CaptchaXError(
                    message=f"Unexpected error: {str(e)}",
                    code=0,
                    status_code=0,
                )

        raise CaptchaXError(
            message=f"Request failed after {self.retry_times + 1} attempts: {str(last_error)}",
            code=0,
            status_code=0,
        )

    def get(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Send GET request."""
        return self._request('GET', endpoint, params=params)

    def post(self, endpoint: str, data: Optional[Dict[str, Any]] = None, deduplication_id: Optional[str] = None) -> Dict[str, Any]:
        """Send POST request."""
        return self._request('POST', endpoint, data=data, deduplication_id=deduplication_id)

    def put(self, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Send PUT request."""
        return self._request('PUT', endpoint, data=data)

    def delete(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Send DELETE request."""
        return self._request('DELETE', endpoint, params=params)


class CaptchaXClient:
    """Main client for CaptchaX API."""

    def __init__(self, config: Union[CaptchaConfig, Dict[str, Any]]):
        if isinstance(config, dict):
            if 'api_version' in config and isinstance(config['api_version'], str):
                config['api_version'] = ApiVersion(config['api_version'])
            config = CaptchaConfig(**config)

        if not config.base_url:
            raise ValueError("base_url is required")

        self._http = HttpClient(
            base_url=config.base_url,
            timeout=config.timeout,
            retry_times=config.retry_times,
        )
        self._app_id = config.app_id
        self._api_version = config.api_version

        if self._app_id:
            self._http.set_header('X-App-ID', self._app_id)

    def set_app_id(self, app_id: str) -> None:
        """Set the application ID."""
        self._app_id = app_id
        self._http.set_header('X-App-ID', app_id)

    def set_api_version(self, version: Union[str, ApiVersion]) -> None:
        """Set API version."""
        if isinstance(version, str):
            version = ApiVersion(version)
        self._api_version = version

    def get_api_version(self) -> str:
        """Get current API version."""
        return self._api_version.value

    def _get_api_prefix(self) -> str:
        """Get API prefix."""
        return f"/api/{self._api_version.value}"

    def _parse_char_positions(self, data: List[Dict[str, Any]]) -> List[CharPosition]:
        """Parse char positions from dict."""
        return [CharPosition(**pos) for pos in data]

    def health_check(self) -> HealthStatus:
        """Check service health."""
        result = self._http.get('/health')
        return HealthStatus(
            status=result['data']['status'],
            service=result['data']['service'],
            timestamp=result['data']['timestamp'],
            version=result['data']['version'],
        )

    def generate_slider_captcha(
        self,
        width: Optional[int] = None,
        height: Optional[int] = None,
        client_info: Optional[str] = None,
        scenario_id: Optional[str] = None,
    ) -> SliderCaptchaResult:
        """Generate slider captcha."""
        if not self._app_id:
            raise CaptchaXError("app_id is required for captcha generation")

        body: Dict[str, Any] = {'app_id': self._app_id}
        if width is not None:
            body['width'] = width
        if height is not None:
            body['height'] = height
        if client_info:
            body['client_info'] = client_info
        if scenario_id:
            body['scenario_id'] = scenario_id

        result = self._http.post(f"{self._get_api_prefix()}/captcha/slider", body)
        return SliderCaptchaResult(
            id=result['data']['id'],
            background_b64=result['data']['background_b64'],
            slider_b64=result['data']['slider_b64'],
            target_x=result['data']['target_x'],
            target_y=result['data']['target_y'],
        )

    def verify_slider_captcha(
        self,
        captcha_id: str,
        target_x: int,
        target_y: Optional[int] = None,
    ) -> SliderVerifyResult:
        """Verify slider captcha."""
        body: Dict[str, Any] = {
            'captcha_id': captcha_id,
            'target_x': target_x,
        }
        if target_y is not None:
            body['target_y'] = target_y

        result = self._http.post(f"{self._get_api_prefix()}/captcha/slider/verify", body)
        return SliderVerifyResult(
            success=result['data']['success'],
            message=result['data']['message'],
        )

    def generate_click_captcha(
        self,
        char_count: Optional[int] = None,
        client_info: Optional[str] = None,
        scenario_id: Optional[str] = None,
    ) -> ClickCaptchaResult:
        """Generate click captcha."""
        if not self._app_id:
            raise CaptchaXError("app_id is required for captcha generation")

        body: Dict[str, Any] = {'app_id': self._app_id}
        if char_count is not None:
            body['char_count'] = char_count
        if client_info:
            body['client_info'] = client_info
        if scenario_id:
            body['scenario_id'] = scenario_id

        result = self._http.post(f"{self._get_api_prefix()}/captcha/click", body)
        return ClickCaptchaResult(
            id=result['data']['id'],
            image=result['data']['image'],
            target_chars=result['data']['target_chars'],
            char_positions=self._parse_char_positions(result['data']['char_positions']),
        )

    def verify_click_captcha(
        self,
        captcha_id: str,
        clicks: List[CharPosition],
    ) -> ClickVerifyResult:
        """Verify click captcha."""
        body = {
            'captcha_id': captcha_id,
            'clicks': [{'char': c.char, 'x': c.x, 'y': c.y} for c in clicks],
        }

        result = self._http.post(f"{self._get_api_prefix()}/captcha/click/verify", body)
        return ClickVerifyResult(
            success=result['data']['success'],
            score=result['data'].get('score', 0.0),
            message=result['data']['message'],
        )

    def generate_puzzle_captcha(
        self,
        width: Optional[int] = None,
        height: Optional[int] = None,
        client_info: Optional[str] = None,
        scenario_id: Optional[str] = None,
    ) -> PuzzleCaptchaResult:
        """Generate puzzle captcha."""
        if not self._app_id:
            raise CaptchaXError("app_id is required for captcha generation")

        body: Dict[str, Any] = {'app_id': self._app_id}
        if width is not None:
            body['width'] = width
        if height is not None:
            body['height'] = height
        if client_info:
            body['client_info'] = client_info
        if scenario_id:
            body['scenario_id'] = scenario_id

        result = self._http.post(f"{self._get_api_prefix()}/captcha/puzzle", body)
        return PuzzleCaptchaResult(
            id=result['data']['id'],
            background_b64=result['data']['background_b64'],
            puzzle_b64=result['data']['puzzle_b64'],
            target_x=result['data']['target_x'],
            target_y=result['data']['target_y'],
        )

    def verify_puzzle_captcha(
        self,
        captcha_id: str,
        target_x: int,
        target_y: Optional[int] = None,
    ) -> PuzzleVerifyResult:
        """Verify puzzle captcha."""
        body: Dict[str, Any] = {
            'captcha_id': captcha_id,
            'target_x': target_x,
        }
        if target_y is not None:
            body['target_y'] = target_y

        result = self._http.post(f"{self._get_api_prefix()}/captcha/puzzle/verify", body)
        return PuzzleVerifyResult(
            success=result['data']['success'],
            message=result['data']['message'],
        )

    def batch_verify(
        self,
        items: List[BatchVerifyItem],
        deduplication_id: Optional[str] = None,
    ) -> BatchVerifyResponse:
        """Batch verify captchas."""
        body = {
            'items': [
                {
                    'captcha_id': item.captcha_id,
                    'type': item.type,
                    'target_x': item.target_x,
                    'target_y': item.target_y,
                    'clicks': [{'char': c.char, 'x': c.x, 'y': c.y} for c in item.clicks] if item.clicks else None,
                }
                for item in items
            ]
        }

        result = self._http.post(
            f"{self._get_api_prefix()}/captcha/batch/verify",
            body,
            deduplication_id=deduplication_id,
        )

        return BatchVerifyResponse(
            results=[
                BatchVerifyResult(
                    captcha_id=r['captcha_id'],
                    success=r['success'],
                    message=r['message'],
                    score=r.get('score'),
                )
                for r in result['data']['results']
            ],
            summary=BatchVerifySummary(**result['data']['summary']),
        )

    def list_scenarios(self) -> Dict[str, Any]:
        """List all scenarios."""
        result = self._http.get(f"{self._get_api_prefix()}/captcha/scenarios")
        return result['data']

    def create_scenario(
        self,
        name: str,
        description: Optional[str] = None,
        difficulty: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
    ) -> Scenario:
        """Create a new scenario."""
        body: Dict[str, Any] = {'name': name}
        if description:
            body['description'] = description
        if difficulty:
            body['difficulty'] = difficulty
        if config:
            body['config'] = config

        result = self._http.post(f"{self._get_api_prefix()}/captcha/scenarios", body)
        return Scenario(**result['data'])

    def get_scenario(self, scenario_id: str) -> Scenario:
        """Get scenario by ID."""
        result = self._http.get(f"{self._get_api_prefix()}/captcha/scenarios/{scenario_id}")
        return Scenario(**result['data'])

    def update_scenario(
        self,
        scenario_id: str,
        name: Optional[str] = None,
        description: Optional[str] = None,
        difficulty: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
    ) -> Scenario:
        """Update a scenario."""
        body: Dict[str, Any] = {}
        if name is not None:
            body['name'] = name
        if description is not None:
            body['description'] = description
        if difficulty is not None:
            body['difficulty'] = difficulty
        if config is not None:
            body['config'] = config

        result = self._http.put(f"{self._get_api_prefix()}/captcha/scenarios/{scenario_id}", body)
        return Scenario(**result['data'])

    def delete_scenario(self, scenario_id: str) -> Dict[str, bool]:
        """Delete a scenario."""
        result = self._http.delete(f"{self._get_api_prefix()}/captcha/scenarios/{scenario_id}")
        return result['data']

    def register_webhook(
        self,
        app_id: str,
        url: str,
        events: List[str],
        secret: Optional[str] = None,
        headers: Optional[Dict[str, str]] = None,
    ) -> Webhook:
        """Register a webhook."""
        body: Dict[str, Any] = {
            'app_id': app_id,
            'url': url,
            'events': events,
        }
        if secret:
            body['secret'] = secret
        if headers:
            body['headers'] = headers

        result = self._http.post(f"{self._get_api_prefix()}/captcha/webhook/register", body)
        return Webhook(**result['data'])

    def list_webhooks(self, app_id: Optional[str] = None) -> Dict[str, Any]:
        """List webhooks."""
        endpoint = f"{self._get_api_prefix()}/captcha/webhook"
        if app_id:
            endpoint += f"?app_id={app_id}"
        result = self._http.get(endpoint)
        return result['data']

    def update_webhook(
        self,
        webhook_id: str,
        url: Optional[str] = None,
        secret: Optional[str] = None,
        events: Optional[List[str]] = None,
        enabled: Optional[bool] = None,
        headers: Optional[Dict[str, str]] = None,
    ) -> Webhook:
        """Update a webhook."""
        body: Dict[str, Any] = {}
        if url is not None:
            body['url'] = url
        if secret is not None:
            body['secret'] = secret
        if events is not None:
            body['events'] = events
        if enabled is not None:
            body['enabled'] = enabled
        if headers is not None:
            body['headers'] = headers

        result = self._http.put(f"{self._get_api_prefix()}/captcha/webhook/{webhook_id}", body)
        return Webhook(**result['data'])

    def unregister_webhook(self, webhook_id: str) -> Dict[str, bool]:
        """Unregister a webhook."""
        result = self._http.delete(f"{self._get_api_prefix()}/captcha/webhook/{webhook_id}")
        return result['data']

    def create_client_info(self, **kwargs: Any) -> str:
        """Create client info JSON string."""
        import platform
        import uuid
        info = {
            'platform': platform.platform(),
            'timestamp': int(time.time() * 1000),
            'request_id': str(uuid.uuid4()),
        }
        info.update(kwargs)
        return json.dumps(info)

    def generate_and_verify_slider(
        self,
        width: Optional[int] = None,
        height: Optional[int] = None,
        client_info: Optional[str] = None,
        scenario_id: Optional[str] = None,
    ) -> tuple:
        """Generate and verify slider captcha in one call."""
        captcha = self.generate_slider_captcha(width, height, client_info, scenario_id)
        result = self.verify_slider_captcha(captcha.id, captcha.target_x, captcha.target_y)
        return captcha, result

    def generate_and_verify_click(
        self,
        char_count: Optional[int] = None,
        client_info: Optional[str] = None,
        scenario_id: Optional[str] = None,
    ) -> tuple:
        """Generate and verify click captcha in one call."""
        captcha = self.generate_click_captcha(char_count, client_info, scenario_id)
        result = self.verify_click_captcha(captcha.id, captcha.char_positions)
        return captcha, result
