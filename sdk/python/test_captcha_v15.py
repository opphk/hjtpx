"""
Python SDK v15.0 单元测试
"""

import unittest
from unittest.mock import Mock, patch, MagicMock
import json
from dataclasses import asdict

from captcha import (
    CaptchaClient,
    CaptchaType,
    ClickMode,
    CaptchaError,
    CaptchaAPIError,
    CaptchaNetworkError,
    CaptchaTimeoutError,
    CaptchaValidationError,
    CaptchaSessionExpiredError,
    TrajectoryPoint,
    SliderCaptchaResponse,
    ClickCaptchaResponse,
    ImageCaptchaResponse,
    VerifyResult,
)
from async_captcha import (
    AsyncCaptchaClient,
    AsyncCaptchaType,
    AsyncCaptchaError,
    AsyncCaptchaAPIError,
    AsyncCaptchaRateLimitError,
    AsyncTrajectoryPoint,
    AsyncSliderCaptchaResponse,
    AsyncVerifyResult,
)


class TestTrajectoryPoint(unittest.TestCase):
    """测试轨迹点类"""

    def test_creation(self):
        point = TrajectoryPoint(x=100, y=50, t=1700000000)
        self.assertEqual(point.x, 100)
        self.assertEqual(point.y, 50)
        self.assertEqual(point.t, 1700000000)

    def test_to_dict(self):
        point = TrajectoryPoint(x=100, y=50, t=1700000000)
        result = point.to_dict()
        self.assertEqual(result, {'x': 100, 'y': 50, 't': 1700000000})

    def test_from_dict(self):
        data = {'x': 100, 'y': 50, 't': 1700000000}
        point = TrajectoryPoint.from_dict(data)
        self.assertEqual(point.x, 100)
        self.assertEqual(point.y, 50)
        self.assertEqual(point.t, 1700000000)


class TestSliderCaptchaResponse(unittest.TestCase):
    """测试滑块验证码响应"""

    def test_creation(self):
        response = SliderCaptchaResponse(
            session_id='test-session',
            image_url='http://example.com/image.png',
            puzzle_url='http://example.com/puzzle.png',
        )
        self.assertEqual(response.session_id, 'test-session')
        self.assertEqual(response.image_url, 'http://example.com/image.png')

    def test_optional_fields(self):
        response = SliderCaptchaResponse(
            session_id='test-session',
            image_url='http://example.com/image.png',
            puzzle_url='http://example.com/puzzle.png',
            secret_y=50,
            image_width=320,
            image_height=160,
        )
        self.assertEqual(response.secret_y, 50)
        self.assertEqual(response.image_width, 320)


class TestClickCaptchaResponse(unittest.TestCase):
    """测试点击验证码响应"""

    def test_creation(self):
        response = ClickCaptchaResponse(
            session_id='test-session',
            image_url='http://example.com/image.png',
            hint='Click 1, 2, 3',
            hint_order=[1, 2, 3],
            max_points=3,
            mode='number',
            allow_shuffle=True,
        )
        self.assertEqual(response.session_id, 'test-session')
        self.assertEqual(response.hint_order, [1, 2, 3])


class TestVerifyResult(unittest.TestCase):
    """测试验证结果"""

    def test_success_result(self):
        result = VerifyResult(
            success=True,
            message='Verification passed',
            remaining_attempts=3,
        )
        self.assertTrue(result.success)
        self.assertEqual(result.message, 'Verification passed')

    def test_failure_result(self):
        result = VerifyResult(
            success=False,
            message='Verification failed',
            fail_reason='Invalid trajectory',
        )
        self.assertFalse(result.success)
        self.assertEqual(result.fail_reason, 'Invalid trajectory')


class TestCaptchaClient(unittest.TestCase):
    """测试同步客户端"""

    def setUp(self):
        self.client = CaptchaClient(
            base_url='http://localhost:8080',
            api_key='test-api-key',
            timeout=30,
        )

    def test_client_creation(self):
        self.assertEqual(self.client.base_url, 'http://localhost:8080')
        self.assertEqual(self.client.api_key, 'test-api-key')
        self.assertEqual(self.client.timeout, 30)

    def test_get_headers(self):
        headers = self.client._get_headers()
        self.assertIn('Content-Type', headers)
        self.assertIn('X-API-Key', headers)
        self.assertEqual(headers['X-API-Key'], 'test-api-key')

    def test_get_headers_with_token(self):
        self.client._token = 'test-token'
        headers = self.client._get_headers()
        self.assertIn('Authorization', headers)
        self.assertEqual(headers['Authorization'], 'Bearer test-token')

    @patch('captcha.requests.Session')
    def test_get_slider_captcha(self, mock_session):
        mock_response = Mock()
        mock_response.json.return_value = {
            'code': 0,
            'message': 'success',
            'data': {
                'session_id': 'test-session',
                'image_url': 'http://example.com/image.png',
                'puzzle_url': 'http://example.com/puzzle.png',
                'secret_y': 50,
            }
        }
        mock_response.raise_for_status = Mock()
        
        mock_session_instance = Mock()
        mock_session_instance.request.return_value = mock_response
        mock_session.return_value = mock_session_instance

        client = CaptchaClient('http://localhost:8080')
        result = client.get_slider_captcha(320, 160, 8)
        
        self.assertEqual(result.session_id, 'test-session')
        self.assertEqual(result.secret_y, 50)


class TestCaptchaExceptions(unittest.TestCase):
    """测试异常类"""

    def test_captcha_error(self):
        error = CaptchaError('Test error')
        self.assertEqual(str(error), 'Test error')

    def test_captcha_api_error(self):
        error = CaptchaAPIError('API error', code=400, data={'test': 'data'})
        self.assertEqual(error.message, 'API error')
        self.assertEqual(error.code, 400)
        self.assertEqual(error.data, {'test': 'data'})

    def test_captcha_network_error(self):
        error = CaptchaNetworkError('Network error')
        self.assertEqual(str(error), 'Network error')

    def test_captcha_timeout_error(self):
        error = CaptchaTimeoutError()
        self.assertIn('timed out', str(error))

    def test_captcha_validation_error(self):
        error = CaptchaValidationError('Validation failed')
        self.assertEqual(str(error), 'Validation failed')

    def test_captcha_session_expired_error(self):
        error = CaptchaSessionExpiredError('Session expired')
        self.assertEqual(str(error), 'Session expired')


class TestCaptchaType(unittest.TestCase):
    """测试验证码类型枚举"""

    def test_enum_values(self):
        self.assertEqual(CaptchaType.SLIDER.value, 'slider')
        self.assertEqual(CaptchaType.CLICK.value, 'click')
        self.assertEqual(CaptchaType.IMAGE.value, 'image')
        self.assertEqual(CaptchaType.ROTATION.value, 'rotation')
        self.assertEqual(CaptchaType.GESTURE.value, 'gesture')
        self.assertEqual(CaptchaType.JIGSAW.value, 'jigsaw')


class TestClickMode(unittest.TestCase):
    """测试点击模式枚举"""

    def test_enum_values(self):
        self.assertEqual(ClickMode.NUMBER.value, 'number')
        self.assertEqual(ClickMode.LETTER.value, 'letter')
        self.assertEqual(ClickMode.CHINESE.value, 'chinese')
        self.assertEqual(ClickMode.MIXED.value, 'mixed')
        self.assertEqual(ClickMode.ICON.value, 'icon')


class TestAsyncCaptchaTypes(unittest.TestCase):
    """测试异步验证码类型"""

    def test_async_trajectory_point(self):
        point = AsyncTrajectoryPoint(x=100, y=50, t=1700000000)
        result = point.to_dict()
        self.assertEqual(result, {'x': 100, 'y': 50, 't': 1700000000})

    def test_async_slider_response(self):
        response = AsyncSliderCaptchaResponse(
            session_id='test-session',
            image_url='http://example.com/image.png',
            puzzle_url='http://example.com/puzzle.png',
        )
        self.assertEqual(response.session_id, 'test-session')

    def test_async_verify_result(self):
        result = AsyncVerifyResult(
            success=True,
            message='OK',
            risk_score=0.15,
        )
        self.assertTrue(result.success)
        self.assertEqual(result.risk_score, 0.15)


class TestAsyncCaptchaExceptions(unittest.TestCase):
    """测试异步异常类"""

    def test_async_captcha_error(self):
        error = AsyncCaptchaError('Async error')
        self.assertEqual(str(error), 'Async error')

    def test_async_captcha_api_error(self):
        error = AsyncCaptchaAPIError('API error', code=500)
        self.assertEqual(error.message, 'API error')
        self.assertEqual(error.code, 500)

    def test_async_captcha_rate_limit_error(self):
        error = AsyncCaptchaRateLimitError('Rate limited')
        self.assertEqual(str(error), 'Rate limited')


class TestContextManagers(unittest.TestCase):
    """测试上下文管理器"""

    def test_sync_client_context_manager(self):
        with CaptchaClient('http://localhost:8080') as client:
            self.assertEqual(client.base_url, 'http://localhost:8080')


if __name__ == '__main__':
    unittest.main()
