#!/usr/bin/env python3
"""
行为验证系统 Python SDK 集成测试
测试完整的验证码工作流程
"""

import unittest
from unittest import mock
import sys
import os
import json
import time
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading
import asyncio

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from captcha import (
    CaptchaClient,
    CaptchaType,
    ClickMode,
    TrajectoryPoint,
    JigsawPiece,
    CaptchaError,
    CaptchaAPIError,
    CaptchaNetworkError,
    CaptchaTimeoutError,
    CaptchaValidationError,
    CaptchaSessionExpiredError,
    SliderCaptchaResponse,
    ClickCaptchaResponse,
    ImageCaptchaResponse,
    RotationCaptchaResponse,
    GestureCaptchaResponse,
    JigsawCaptchaResponse,
    VerifyResult,
    LoginResponse,
    UserAuth,
    Environment,
)


class TestFullWorkflow(unittest.TestCase):
    """测试完整的验证码工作流程"""

    def setUp(self):
        """测试前准备"""
        self.client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=5,
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_complete_slider_workflow(self):
        """测试完整的滑块验证码工作流程"""
        mock_responses = []

        def mock_request(method, url, **kwargs):
            if 'slider' in url and method == 'GET':
                return {
                    'session_id': 'slider-workflow-test',
                    'image_url': 'http://test.com/slider.jpg',
                    'puzzle_url': 'http://test.com/puzzle.jpg',
                    'secret_y': 80,
                    'image_width': 320,
                    'image_height': 160,
                }
            elif 'verify' in url and method == 'POST':
                mock_responses.append(kwargs.get('json', {}))
                return {
                    'success': True,
                    'message': 'Verification successful',
                    'score': 0.95,
                    'risk_level': 'low',
                }
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            captcha = self.client.get_slider_captcha()
            self.assertEqual(captcha.session_id, 'slider-workflow-test')

            trajectory = [
                TrajectoryPoint(x=0, y=80, t=0),
                TrajectoryPoint(x=50, y=82, t=200),
                TrajectoryPoint(x=100, y=78, t=400),
                TrajectoryPoint(x=150, y=80, t=600),
            ]

            result = self.client.verify_slider_captcha(
                session_id=captcha.session_id,
                x=150,
                y=80,
                trajectory=trajectory,
            )

            self.assertTrue(result.success)
            self.assertEqual(len(mock_responses), 1)
            self.assertIn('trajectory', mock_responses[0])

    def test_complete_click_workflow(self):
        """测试完整的点击验证码工作流程"""
        mock_responses = []

        def mock_request(method, url, **kwargs):
            if 'click' in url and method == 'GET':
                return {
                    'session_id': 'click-workflow-test',
                    'image_url': 'http://test.com/click.jpg',
                    'hint': 'Click 1, 2, 3',
                    'hint_order': [0, 1, 2],
                    'max_points': 3,
                    'mode': 'number',
                    'allow_shuffle': True,
                    'points': [[100, 100], [200, 200], [300, 300]],
                }
            elif 'verify' in url and method == 'POST':
                mock_responses.append(kwargs.get('json', {}))
                return {
                    'success': True,
                    'message': 'Verification successful',
                    'score': 0.92,
                }
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            captcha = self.client.get_click_captcha(
                mode=ClickMode.NUMBER,
                max_points=3,
                allow_shuffle=True,
            )
            self.assertEqual(captcha.session_id, 'click-workflow-test')

            result = self.client.verify_click_captcha(
                session_id=captcha.session_id,
                points=[[100, 100], [200, 200], [300, 300]],
                click_sequence=[0, 1, 2],
            )

            self.assertTrue(result.success)
            self.assertEqual(len(mock_responses), 1)

    def test_complete_image_workflow(self):
        """测试完整的图形验证码工作流程"""
        mock_responses = []

        def mock_request(method, url, **kwargs):
            if 'image' in url and method == 'GET':
                return {
                    'challenge_id': 'image-workflow-test',
                    'image': 'data:image/png;base64,abc123',
                }
            elif 'verify' in url and method == 'POST':
                mock_responses.append(kwargs.get('json', {}))
                return {'success': True, 'message': 'Verification successful'}
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            captcha = self.client.get_image_captcha(
                type_='mixed',
                count=4,
                noise_mode=2,
                line_mode=1,
            )
            self.assertEqual(captcha.challenge_id, 'image-workflow-test')

            result = self.client.verify_image_captcha(
                challenge_id=captcha.challenge_id,
                answer='test',
            )

            self.assertTrue(result.success)

    def test_generic_verify_method(self):
        """测试通用验证方法"""
        mock_responses = []

        def mock_request(method, url, **kwargs):
            if method == 'GET':
                return {
                    'session_id': 'generic-test',
                    'image_url': 'http://test.com/image.jpg',
                    'hint': 'Test hint',
                }
            elif method == 'POST':
                mock_responses.append(kwargs.get('json', {}))
                return {'success': True}
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            result = self.client.verify_captcha(
                captcha_type=CaptchaType.SLIDER,
                session_id='generic-test',
                x=150,
            )

            self.assertTrue(result.success)
            self.assertEqual(mock_responses[0]['type'], 'slider')

    def test_multiple_captcha_types(self):
        """测试多种验证码类型"""
        captcha_types_tested = []

        def mock_request(method, url, **kwargs):
            if 'slider' in url:
                captcha_types_tested.append('slider')
                return {'session_id': 'slider-1', 'image_url': 'test.jpg'}
            elif 'click' in url:
                captcha_types_tested.append('click')
                return {'session_id': 'click-1', 'image_url': 'test.jpg'}
            elif 'image' in url:
                captcha_types_tested.append('image')
                return {'challenge_id': 'img-1', 'image': 'data:test'}
            elif 'rotation' in url:
                captcha_types_tested.append('rotation')
                return {'challenge_id': 'rot-1', 'image': 'data:test'}
            elif 'gesture' in url:
                captcha_types_tested.append('gesture')
                return {'session_id': 'gest-1', 'pattern': '1-2-3'}
            elif 'jigsaw' in url:
                captcha_types_tested.append('jigsaw')
                return {
                    'session_id': 'jigsaw-1',
                    'image_url': 'test.jpg',
                    'pieces': [],
                    'piece_images': [],
                    'grid_size': 3,
                    'piece_width': 100,
                    'piece_height': 100,
                    'image_width': 300,
                    'image_height': 300,
                }
            elif 'verify' in url:
                return {'success': True}
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            self.client.get_slider_captcha()
            self.client.get_click_captcha()
            self.client.get_image_captcha()
            self.client.get_rotation_captcha()
            self.client.get_gesture_captcha()
            self.client.get_jigsaw_captcha()

            self.assertIn('slider', captcha_types_tested)
            self.assertIn('click', captcha_types_tested)
            self.assertIn('image', captcha_types_tested)
            self.assertIn('rotation', captcha_types_tested)
            self.assertIn('gesture', captcha_types_tested)
            self.assertIn('jigsaw', captcha_types_tested)


class TestConnectionPoolAndRetry(unittest.TestCase):
    """测试连接池和重试机制"""

    def test_client_with_custom_pool_settings(self):
        """测试自定义连接池设置"""
        client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=10,
            max_retries=5,
            retry_backoff_factor=0.3,
            pool_connections=20,
            pool_maxsize=20,
        )

        self.assertEqual(client.timeout, 10)
        self.assertEqual(client.session.adapters['http://'].max_retries.total, 5)
        self.assertEqual(client.session.adapters['http://'].pool_connections, 20)
        self.assertEqual(client.session.adapters['http://'].pool_maxsize, 20)

        client.close()

    def test_session_persistence(self):
        """测试会话持久化"""
        client = CaptchaClient(base_url="http://test.example.com")

        session1 = client.session
        session2 = client.session

        self.assertIs(session1, session2)

        client.close()


class TestTokenManagement(unittest.TestCase):
    """测试令牌管理"""

    def setUp(self):
        """测试前准备"""
        self.client = CaptchaClient(
            base_url="http://test.example.com",
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_token_storage_after_login(self):
        """测试登录后令牌存储"""
        def mock_request(method, url, **kwargs):
            return {
                'access_token': 'test-access-token',
                'refresh_token': 'test-refresh-token',
                'expires_in': 3600,
                'user': {'id': 1, 'username': 'testuser'},
            }

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            auth = self.client.auth()
            response = auth.login(username='testuser', password='password')

            self.assertEqual(response.access_token, 'test-access-token')
            self.assertEqual(self.client._token, 'test-access-token')
            self.assertEqual(self.client._refresh_token, 'test-refresh-token')

    def test_token_cleared_after_logout(self):
        """测试登出后令牌清除"""
        self.client._token = 'old-token'
        self.client._refresh_token = 'old-refresh-token'

        def mock_request(method, url, **kwargs):
            return {'success': True}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            auth = self.client.auth()
            auth.logout()

            self.assertIsNone(self.client._token)
            self.assertIsNone(self.client._refresh_token)


class TestEnvironmentDetection(unittest.TestCase):
    """测试环境检测功能"""

    def setUp(self):
        """测试前准备"""
        self.client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=5,
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_submit_detection_data(self):
        """测试提交检测数据"""
        mock_response = {
            'success': True,
            'risk_level': 'low',
            'risk_score': 0.1,
            'checks': {
                'browser': 'passed',
                'device': 'passed',
                'network': 'passed',
            },
        }

        with mock.patch.object(self.client.session, 'request', return_value=mock.Mock(json=lambda: {'data': mock_response})) as mock_request:
            with mock.patch.object(self.client.session, 'get', return_value=mock.Mock(text='{}')):
                env = self.client.env()
                result = env.submit_detection({
                    'fingerprint': 'test-fingerprint',
                    'canvas_hash': 'test-canvas',
                    'webgl_vendor': 'test-vendor',
                })

                self.assertTrue(result.get('success', False))

    def test_check_environment(self):
        """测试环境检查"""
        mock_response = {
            'success': True,
            'risk_level': 'low',
            'risk_score': 0.1,
        }

        with mock.patch.object(self.client.session, 'request', return_value=mock.Mock(json=lambda: {'data': mock_response})) as mock_request:
            with mock.patch.object(self.client.session, 'get', return_value=mock.Mock(text='{}')):
                env = self.client.env()
                result = env.check_environment({
                    'fingerprint': 'test-fingerprint',
                })

                self.assertTrue(result.get('success', False))


class TestErrorScenarios(unittest.TestCase):
    """测试错误场景"""

    def setUp(self):
        """测试前准备"""
        self.client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=1,
            max_retries=0,
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_session_expired_error(self):
        """测试会话过期错误"""
        with mock.patch.object(self.client.session, 'request') as mock_request:
            mock_response = mock.Mock()
            mock_response.json.return_value = {
                'code': 404,
                'message': 'Session not found or expired',
            }
            mock_response.status_code = 200
            mock_request.return_value = mock_response

            with self.assertRaises(CaptchaSessionExpiredError):
                self.client.get_slider_captcha()

    def test_validation_error(self):
        """测试验证错误"""
        with mock.patch.object(self.client.session, 'request') as mock_request:
            mock_response = mock.Mock()
            mock_response.json.return_value = {
                'code': 400,
                'message': 'Invalid parameters',
            }
            mock_response.status_code = 200
            mock_request.return_value = mock_response

            with self.assertRaises(CaptchaValidationError):
                self.client.get_slider_captcha()

    def test_api_error_with_code(self):
        """测试带错误码的API错误"""
        with mock.patch.object(self.client.session, 'request') as mock_request:
            mock_response = mock.Mock()
            mock_response.json.return_value = {
                'code': 500,
                'message': 'Internal server error',
            }
            mock_response.status_code = 200
            mock_request.return_value = mock_response

            with self.assertRaises(CaptchaAPIError) as context:
                self.client.get_slider_captcha()

            self.assertEqual(context.exception.code, 500)


class TestConcurrentRequests(unittest.TestCase):
    """测试并发请求"""

    def setUp(self):
        """测试前准备"""
        self.client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=5,
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_sequential_requests(self):
        """测试顺序请求"""
        request_count = [0]

        def mock_request(method, url, **kwargs):
            request_count[0] += 1
            return {
                'session_id': f'request-{request_count[0]}',
                'image_url': 'http://test.com/image.jpg',
            }

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            for i in range(5):
                captcha = self.client.get_slider_captcha()
                self.assertIn(f'request-{i+1}', captcha.session_id)

    def test_multiple_captcha_types_sequential(self):
        """测试多种验证码顺序请求"""
        results = {}

        def mock_request(method, url, **kwargs):
            if 'slider' in url:
                return {'session_id': 'slider-1', 'image_url': 'test.jpg', 'secret_y': 50}
            elif 'click' in url:
                return {
                    'session_id': 'click-1',
                    'image_url': 'test.jpg',
                    'hint': 'Click 1',
                    'hint_order': [0],
                    'max_points': 1,
                    'mode': 'number',
                    'allow_shuffle': False,
                }
            elif 'image' in url:
                return {'challenge_id': 'img-1', 'image': 'data:test'}
            return {}

        with mock.patch.object(self.client.session, 'request', side_effect=mock_request):
            results['slider'] = self.client.get_slider_captcha()
            results['click'] = self.client.get_click_captcha()
            results['image'] = self.client.get_image_captcha()

            self.assertEqual(results['slider'].session_id, 'slider-1')
            self.assertEqual(results['click'].session_id, 'click-1')
            self.assertEqual(results['image'].challenge_id, 'img-1')


class TestContextManager(unittest.TestCase):
    """测试上下文管理器"""

    def test_context_manager_usage(self):
        """测试上下文管理器使用"""
        with CaptchaClient(base_url="http://test.example.com") as client:
            self.assertIsNotNone(client)

    def test_context_manager_with_resources(self):
        """测试上下文管理器资源管理"""
        client = None
        try:
            with CaptchaClient(base_url="http://test.example.com") as c:
                client = c
                self.assertIsNotNone(client)
        finally:
            if client:
                self.assertIsNone(client._token)


class TestDataConversion(unittest.TestCase):
    """测试数据转换"""

    def test_trajectory_point_conversion(self):
        """测试轨迹点转换"""
        point = TrajectoryPoint(x=100, y=200, t=1000)
        data = point.to_dict()
        restored = TrajectoryPoint.from_dict(data)

        self.assertEqual(point.x, restored.x)
        self.assertEqual(point.y, restored.y)
        self.assertEqual(point.t, restored.t)

    def test_jigsaw_piece_conversion(self):
        """测试拼图碎片转换"""
        piece = JigsawPiece(
            index=0,
            original_x=0,
            original_y=0,
            current_x=100,
            current_y=100,
            width=50,
            height=50,
            rotation=90,
        )
        data = piece.to_dict()
        restored = JigsawPiece.from_dict(data)

        self.assertEqual(piece.index, restored.index)
        self.assertEqual(piece.rotation, restored.rotation)
        self.assertEqual(piece.current_x, restored.current_x)


def run_integration_tests():
    """运行集成测试"""
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()

    suite.addTests(loader.loadTestsFromTestCase(TestFullWorkflow))
    suite.addTests(loader.loadTestsFromTestCase(TestConnectionPoolAndRetry))
    suite.addTests(loader.loadTestsFromTestCase(TestTokenManagement))
    suite.addTests(loader.loadTestsFromTestCase(TestEnvironmentDetection))
    suite.addTests(loader.loadTestsFromTestCase(TestErrorScenarios))
    suite.addTests(loader.loadTestsFromTestCase(TestConcurrentRequests))
    suite.addTests(loader.loadTestsFromTestCase(TestContextManager))
    suite.addTests(loader.loadTestsFromTestCase(TestDataConversion))

    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    return result


if __name__ == "__main__":
    result = run_integration_tests()
    sys.exit(0 if result.wasSuccessful() else 1)
