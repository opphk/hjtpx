"""
Python SDK 完整测试套件
"""

import pytest
import asyncio
from unittest.mock import Mock, patch, AsyncMock
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from captcha import (
    CaptchaClient, CaptchaType, ClickMode, CaptchaError,
    CaptchaAPIError, CaptchaNetworkError, CaptchaTimeoutError,
    CaptchaValidationError, CaptchaSessionExpiredError,
    SliderCaptchaResponse, ClickCaptchaResponse, VerifyResult,
    TrajectoryPoint, JigsawPiece
)
from async_captcha import (
    AsyncCaptchaClient, AsyncCaptchaType, AsyncClickMode,
    AsyncCaptchaError, AsyncCaptchaAPIError, AsyncCaptchaNetworkError,
    AsyncCaptchaTimeoutError, AsyncCaptchaValidationError,
    AsyncSliderCaptchaResponse, AsyncTrajectoryPoint, AsyncVerifyResult
)


class TestCaptchaClient:
    """测试同步CaptchaClient"""

    def setup_method(self):
        """测试前准备"""
        self.client = CaptchaClient("http://localhost:8080")

    def teardown_method(self):
        """测试后清理"""
        self.client.close()

    def test_client_initialization(self):
        """测试客户端初始化"""
        assert self.client.base_url == "http://localhost:8080"
        assert self.client.api_key is None
        assert self.client.timeout == 30

    def test_client_with_api_key(self):
        """测试带API密钥的客户端"""
        client = CaptchaClient("http://localhost:8080", api_key="test-key")
        assert client.api_key == "test-key"

    def test_client_with_timeout(self):
        """测试带超时的客户端"""
        client = CaptchaClient("http://localhost:8080", timeout=60)
        assert client.timeout == 60

    def test_get_headers_without_token(self):
        """测试无令牌的请求头"""
        headers = self.client._get_headers()
        assert headers['Content-Type'] == 'application/json'
        assert 'Authorization' not in headers

    def test_get_headers_with_api_key(self):
        """测试带API密钥的请求头"""
        client = CaptchaClient("http://localhost:8080", api_key="test-key")
        headers = client._get_headers()
        assert headers['X-API-Key'] == 'test-key'

    def test_get_headers_with_token(self):
        """测试带令牌的请求头"""
        self.client._token = "test-token"
        headers = self.client._get_headers()
        assert headers['Authorization'] == 'Bearer test-token'

    def test_trajectory_point_creation(self):
        """测试轨迹点创建"""
        point = TrajectoryPoint(100, 200, 1234567890)
        assert point.x == 100
        assert point.y == 200
        assert point.t == 1234567890

    def test_trajectory_point_to_dict(self):
        """测试轨迹点转字典"""
        point = TrajectoryPoint(100, 200, 1234567890)
        data = point.to_dict()
        assert data['x'] == 100
        assert data['y'] == 200
        assert data['t'] == 1234567890

    def test_trajectory_point_from_dict(self):
        """测试从字典创建轨迹点"""
        data = {'x': 100, 'y': 200, 't': 1234567890}
        point = TrajectoryPoint.from_dict(data)
        assert point.x == 100
        assert point.y == 200
        assert point.t == 1234567890

    def test_jigsaw_piece_creation(self):
        """测试拼图碎片创建"""
        piece = JigsawPiece(
            index=0,
            original_x=0,
            original_y=0,
            current_x=50,
            current_y=50,
            width=100,
            height=100,
            rotation=0
        )
        assert piece.index == 0
        assert piece.current_x == 50
        assert piece.current_y == 50

    def test_jigsaw_piece_to_dict(self):
        """测试拼图碎片转字典"""
        piece = JigsawPiece(0, 0, 0, 50, 50, 100, 100, 0)
        data = piece.to_dict()
        assert data['index'] == 0
        assert data['current_x'] == 50

    def test_context_manager(self):
        """测试上下文管理器"""
        with CaptchaClient("http://localhost:8080") as client:
            assert client.base_url == "http://localhost:8080"

    @patch('captcha.requests.Session.request')
    def test_get_slider_captcha_success(self, mock_request):
        """测试获取滑块验证码成功"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            'code': 0,
            'message': 'success',
            'data': {
                'session_id': 'test-session-123',
                'image_url': 'http://example.com/image.png',
                'puzzle_url': 'http://example.com/puzzle.png',
                'secret_y': 80,
                'image_width': 320,
                'image_height': 160
            }
        }
        mock_request.return_value = mock_response

        result = self.client.get_slider_captcha(320, 160, 8)
        assert isinstance(result, SliderCaptchaResponse)
        assert result.session_id == 'test-session-123'
        assert result.secret_y == 80

    @patch('captcha.requests.Session.request')
    def test_verify_slider_captcha_success(self, mock_request):
        """测试验证滑块验证码成功"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            'code': 0,
            'message': 'success',
            'data': {
                'success': True,
                'message': 'Verification successful',
                'remaining_attempts': 3,
                'risk_score': 0.95
            }
        }
        mock_request.return_value = mock_response

        result = self.client.verify_slider_captcha('test-session', 150, 80)
        assert isinstance(result, VerifyResult)
        assert result.success is True
        assert result.risk_score == 0.95

    @patch('captcha.requests.Session.request')
    def test_verify_slider_captcha_with_trajectory(self, mock_request):
        """测试带轨迹的验证"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            'code': 0,
            'message': 'success',
            'data': {'success': True, 'message': 'success'}
        }
        mock_request.return_value = mock_response

        trajectory = [
            TrajectoryPoint(0, 80, 1000),
            TrajectoryPoint(75, 82, 500),
            TrajectoryPoint(150, 80, 0)
        ]

        result = self.client.verify_slider_captcha('test-session', 150, 80, trajectory)
        assert result.success is True

    @patch('captcha.requests.Session.request')
    def test_api_error_handling(self, mock_request):
        """测试API错误处理"""
        mock_response = Mock()
        mock_response.status_code = 400
        mock_response.json.return_value = {
            'code': 400,
            'message': 'Invalid parameters'
        }
        mock_response.raise_for_status.side_effect = Exception("HTTP 400")
        mock_request.return_value = mock_response

        with pytest.raises(CaptchaAPIError) as exc_info:
            self.client.get_slider_captcha(320, 160, 8)
        assert exc_info.value.code == 400

    def test_captcha_type_enum(self):
        """测试验证码类型枚举"""
        assert CaptchaType.SLIDER.value == "slider"
        assert CaptchaType.CLICK.value == "click"
        assert CaptchaType.IMAGE.value == "image"

    def test_click_mode_enum(self):
        """测试点击模式枚举"""
        assert ClickMode.NUMBER.value == "number"
        assert ClickMode.LETTER.value == "letter"
        assert ClickMode.CHINESE.value == "chinese"


class TestAsyncCaptchaClient:
    """测试异步CaptchaClient"""

    def setup_method(self):
        """测试前准备"""
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)

    def teardown_method(self):
        """测试后清理"""
        self.loop.close()

    def test_async_client_initialization(self):
        """测试异步客户端初始化"""
        client = AsyncCaptchaClient("http://localhost:8080")
        assert client.base_url == "http://localhost:8080"
        assert client.timeout.total == 30

    def test_async_client_with_config(self):
        """测试异步客户端配置"""
        client = AsyncCaptchaClient(
            "http://localhost:8080",
            api_key="test-key",
            timeout=60,
            max_connections=100
        )
        assert client.api_key == "test-key"
        assert client.timeout.total == 60
        assert client.max_connections == 100

    @pytest.mark.asyncio
    async def test_async_get_slider_captcha(self):
        """测试异步获取滑块验证码"""
        client = AsyncCaptchaClient("http://localhost:8080")

        with patch.object(client, '_request', new_callable=AsyncMock) as mock_request:
            mock_request.return_value = {
                'session_id': 'test-session-async',
                'image_url': 'http://example.com/image.png',
                'puzzle_url': 'http://example.com/puzzle.png',
                'secret_y': 80,
                'image_width': 320,
                'image_height': 160
            }

            result = await client.get_slider_captcha(320, 160, 8)
            assert isinstance(result, AsyncSliderCaptchaResponse)
            assert result.session_id == 'test-session-async'
            assert result.secret_y == 80

        await client.close()

    @pytest.mark.asyncio
    async def test_async_verify_slider_captcha(self):
        """测试异步验证滑块验证码"""
        client = AsyncCaptchaClient("http://localhost:8080")

        with patch.object(client, '_request', new_callable=AsyncMock) as mock_request:
            mock_request.return_value = {
                'success': True,
                'message': 'Verification successful',
                'remaining_attempts': 3,
                'risk_score': 0.95
            }

            result = await client.verify_slider_captcha('test-session', 150, 80)
            assert isinstance(result, AsyncVerifyResult)
            assert result.success is True
            assert result.risk_score == 0.95

        await client.close()

    @pytest.mark.asyncio
    async def test_async_verify_with_trajectory(self):
        """测试异步验证带轨迹"""
        client = AsyncCaptchaClient("http://localhost:8080")

        with patch.object(client, '_request', new_callable=AsyncMock) as mock_request:
            mock_request.return_value = {
                'success': True,
                'message': 'success'
            }

            trajectory = [
                AsyncTrajectoryPoint(0, 80, 1000),
                AsyncTrajectoryPoint(75, 82, 500),
                AsyncTrajectoryPoint(150, 80, 0)
            ]

            result = await client.verify_slider_captcha('test-session', 150, 80, trajectory)
            assert result.success is True

        await client.close()

    def test_async_trajectory_point(self):
        """测试异步轨迹点"""
        point = AsyncTrajectoryPoint(100, 200, 1234567890)
        assert point.x == 100
        assert point.y == 200
        assert point.t == 1234567890

        data = point.to_dict()
        assert data['x'] == 100
        assert data['y'] == 200

    def test_async_context_manager(self):
        """测试异步上下文管理器"""
        async def test():
            async with AsyncCaptchaClient("http://localhost:8080") as client:
                assert client.base_url == "http://localhost:8080"

        self.loop.run_until_complete(test())


class TestErrorHandling:
    """测试错误处理"""

    def test_captcha_error(self):
        """测试基础错误"""
        error = CaptchaError("Test error")
        assert str(error) == "Test error"

    def test_captcha_api_error(self):
        """测试API错误"""
        error = CaptchaAPIError("API error", code=400, data={'key': 'value'})
        assert error.message == "API error"
        assert error.code == 400
        assert error.data == {'key': 'value'}

    def test_captcha_network_error(self):
        """测试网络错误"""
        error = CaptchaNetworkError("Network error")
        assert "Network error" in str(error)

    def test_captcha_timeout_error(self):
        """测试超时错误"""
        error = CaptchaTimeoutError()
        assert "timed out" in str(error)

    def test_captcha_validation_error(self):
        """测试验证错误"""
        error = CaptchaValidationError("Validation failed")
        assert "Validation failed" in str(error)

    def test_captcha_session_expired_error(self):
        """测试会话过期错误"""
        error = CaptchaSessionExpiredError()
        assert "expired" in str(error)

    def test_async_captcha_error(self):
        """测试异步基础错误"""
        error = AsyncCaptchaError("Async error")
        assert str(error) == "Async error"

    def test_async_captcha_api_error(self):
        """测试异步API错误"""
        error = AsyncCaptchaAPIError("Async API error", code=500)
        assert error.message == "Async API error"
        assert error.code == 500


class TestDataModels:
    """测试数据模型"""

    def test_slider_captcha_response(self):
        """测试滑块验证码响应"""
        response = SliderCaptchaResponse(
            session_id='test-123',
            image_url='http://example.com/image.png',
            puzzle_url='http://example.com/puzzle.png',
            hint_url='http://example.com/hint.png',
            shape=1,
            secret_y=80,
            image_width=320,
            image_height=160,
            tolerance=8
        )

        assert response.session_id == 'test-123'
        assert response.image_width == 320
        assert response.secret_y == 80

    def test_click_captcha_response(self):
        """测试点击验证码响应"""
        response = ClickCaptchaResponse(
            session_id='test-123',
            image_url='http://example.com/image.png',
            hint='Click the numbers 1, 2, 3',
            hint_order=[1, 2, 3],
            max_points=3,
            mode='number',
            allow_shuffle=True,
            points=[[100, 100], [200, 200]]
        )

        assert response.session_id == 'test-123'
        assert response.max_points == 3
        assert len(response.hint_order) == 3

    def test_verify_result(self):
        """测试验证结果"""
        result = VerifyResult(
            success=True,
            message='Verification successful',
            remaining_attempts=3,
            trajectory_result={'score': 0.95, 'passed': True},
            risk_score=0.95,
            captcha_pass=True,
            fail_reason=None
        )

        assert result.success is True
        assert result.risk_score == 0.95
        assert result.trajectory_result is not None

    def test_login_response(self):
        """测试登录响应"""
        from captcha import LoginResponse

        response = LoginResponse(
            access_token='access-token-123',
            refresh_token='refresh-token-456',
            expires_in=3600,
            user={'id': 1, 'username': 'testuser', 'email': 'test@example.com'}
        )

        assert response.access_token == 'access-token-123'
        assert response.expires_in == 3600
        assert response.user['username'] == 'testuser'


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
