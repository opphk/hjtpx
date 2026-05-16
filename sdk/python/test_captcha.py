#!/usr/bin/env python3
"""
行为验证系统 Python SDK 单元测试
"""

import unittest
from unittest import mock
import sys
import os

# 添加当前目录到路径
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
)


class TestTrajectoryPoint(unittest.TestCase):
    """测试轨迹点类"""

    def test_init(self):
        """测试初始化"""
        point = TrajectoryPoint(x=100, y=200, t=1000)
        self.assertEqual(point.x, 100)
        self.assertEqual(point.y, 200)
        self.assertEqual(point.t, 1000)

    def test_to_dict(self):
        """测试转换为字典"""
        point = TrajectoryPoint(x=100, y=200, t=1000)
        data = point.to_dict()
        self.assertEqual(data["x"], 100)
        self.assertEqual(data["y"], 200)
        self.assertEqual(data["t"], 1000)

    def test_from_dict(self):
        """测试从字典创建"""
        data = {"x": 100, "y": 200, "t": 1000}
        point = TrajectoryPoint.from_dict(data)
        self.assertEqual(point.x, 100)
        self.assertEqual(point.y, 200)
        self.assertEqual(point.t, 1000)


class TestJigsawPiece(unittest.TestCase):
    """测试拼图碎片类"""

    def test_init(self):
        """测试初始化"""
        piece = JigsawPiece(
            index=0,
            original_x=0,
            original_y=0,
            current_x=10,
            current_y=10,
            width=100,
            height=100,
            rotation=0,
        )
        self.assertEqual(piece.index, 0)
        self.assertEqual(piece.original_x, 0)
        self.assertEqual(piece.current_x, 10)

    def test_to_dict(self):
        """测试转换为字典"""
        piece = JigsawPiece(
            index=0,
            original_x=0,
            original_y=0,
            current_x=10,
            current_y=10,
            width=100,
            height=100,
            rotation=90,
        )
        data = piece.to_dict()
        self.assertEqual(data["index"], 0)
        self.assertEqual(data["rotation"], 90)


class TestExceptions(unittest.TestCase):
    """测试异常类"""

    def test_captcha_error(self):
        """测试基础异常"""
        error = CaptchaError("测试错误")
        self.assertEqual(str(error), "测试错误")

    def test_api_error(self):
        """测试 API 错误"""
        error = CaptchaAPIError("API 错误", code=400)
        self.assertEqual(error.message, "API 错误")
        self.assertEqual(error.code, 400)

    def test_network_error(self):
        """测试网络错误"""
        error = CaptchaNetworkError("网络错误")
        self.assertEqual(str(error), "网络错误")


class TestCaptchaClient(unittest.TestCase):
    """测试验证码客户端"""

    def setUp(self):
        """测试前准备"""
        # 直接创建 client，但我们会在测试中 mock 它的 _request 方法
        self.client = CaptchaClient(
            base_url="http://test.example.com",
            timeout=5,
        )

    def tearDown(self):
        """测试后清理"""
        self.client.close()

    def test_get_slider_captcha(self):
        """测试获取滑块验证码"""
        # 直接 mock _request 方法
        mock_response_data = {
            "session_id": "test-session",
            "image_url": "http://test.example.com/image.jpg",
            "puzzle_url": "http://test.example.com/puzzle.jpg",
            "secret_y": 80,
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            captcha = self.client.get_slider_captcha()

            self.assertIsInstance(captcha, SliderCaptchaResponse)
            self.assertEqual(captcha.session_id, "test-session")
            self.assertEqual(captcha.secret_y, 80)

    def test_verify_slider_captcha(self):
        """测试验证滑块验证码"""
        mock_response_data = {
            "success": True,
            "message": "验证成功",
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            result = self.client.verify_slider_captcha(
                session_id="test-session",
                x=150,
                y=80,
            )

            self.assertIsInstance(result, VerifyResult)
            self.assertTrue(result.success)
            self.assertEqual(result.message, "验证成功")

    def test_get_click_captcha(self):
        """测试获取点击验证码"""
        mock_response_data = {
            "session_id": "test-click-session",
            "image_url": "http://test.example.com/click.jpg",
            "hint": "点击数字 1, 2, 3",
            "hint_order": ["1", "2", "3"],
            "max_points": 3,
            "mode": "number",
            "allow_shuffle": True,
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            captcha = self.client.get_click_captcha()

            self.assertIsInstance(captcha, ClickCaptchaResponse)
            self.assertEqual(captcha.session_id, "test-click-session")

    def test_get_image_captcha(self):
        """测试获取图形验证码"""
        mock_response_data = {
            "challenge_id": "test-challenge",
            "image": "base64-image-data",
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            captcha = self.client.get_image_captcha()

            self.assertIsInstance(captcha, ImageCaptchaResponse)
            self.assertEqual(captcha.challenge_id, "test-challenge")

    def test_get_gesture_captcha(self):
        """测试获取手势验证码"""
        mock_response_data = {
            "session_id": "test-gesture-session",
            "pattern": "1-2-3",
            "grid_size": 3,
            "hint": "按顺序连接",
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            captcha = self.client.get_gesture_captcha()

            self.assertIsInstance(captcha, GestureCaptchaResponse)
            self.assertEqual(captcha.session_id, "test-gesture-session")

    def test_get_jigsaw_captcha(self):
        """测试获取拼图验证码"""
        mock_response_data = {
            "session_id": "test-jigsaw-session",
            "grid_size": 3,
            "pieces": [
                {
                    "index": 0,
                    "original_x": 0,
                    "original_y": 0,
                    "current_x": 100,
                    "current_y": 100,
                    "width": 100,
                    "height": 100,
                    "rotation": 0,
                }
            ],
            "piece_images": [],
            "piece_width": 100,
            "piece_height": 100,
            "image_width": 300,
            "image_height": 300,
        }
        
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.return_value = mock_response_data
            
            captcha = self.client.get_jigsaw_captcha()

            self.assertIsInstance(captcha, JigsawCaptchaResponse)
            self.assertEqual(captcha.session_id, "test-jigsaw-session")
            self.assertEqual(len(captcha.pieces), 1)

    def test_api_error(self):
        """测试 API 错误处理"""
        # 模拟抛出 API 错误
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.side_effect = CaptchaAPIError("Bad Request", code=400)
            
            with self.assertRaises(CaptchaAPIError) as context:
                self.client.get_slider_captcha()

            self.assertEqual(context.exception.code, 400)

    def test_network_error(self):
        """测试网络错误处理"""
        with mock.patch.object(self.client, '_request') as mock_request:
            mock_request.side_effect = CaptchaNetworkError("Connection refused")
            
            with self.assertRaises(CaptchaNetworkError):
                self.client.get_slider_captcha()

    def test_context_manager(self):
        """测试上下文管理器"""
        with CaptchaClient(base_url="http://test.example.com") as client:
            self.assertIsNotNone(client)


class TestCaptchaTypes(unittest.TestCase):
    """测试验证码类型枚举"""

    def test_slider(self):
        """测试滑块类型"""
        self.assertEqual(CaptchaType.SLIDER.value, "slider")

    def test_click(self):
        """测试点击类型"""
        self.assertEqual(CaptchaType.CLICK.value, "click")

    def test_image(self):
        """测试图形类型"""
        self.assertEqual(CaptchaType.IMAGE.value, "image")

    def test_rotation(self):
        """测试旋转类型"""
        self.assertEqual(CaptchaType.ROTATION.value, "rotation")

    def test_gesture(self):
        """测试手势类型"""
        self.assertEqual(CaptchaType.GESTURE.value, "gesture")

    def test_jigsaw(self):
        """测试拼图类型"""
        self.assertEqual(CaptchaType.JIGSAW.value, "jigsaw")


class TestClickMode(unittest.TestCase):
    """测试点击模式枚举"""

    def test_number(self):
        """测试数字模式"""
        self.assertEqual(ClickMode.NUMBER.value, "number")

    def test_letter(self):
        """测试字母模式"""
        self.assertEqual(ClickMode.LETTER.value, "letter")

    def test_chinese(self):
        """测试中文模式"""
        self.assertEqual(ClickMode.CHINESE.value, "chinese")

    def test_mixed(self):
        """测试混合模式"""
        self.assertEqual(ClickMode.MIXED.value, "mixed")

    def test_icon(self):
        """测试图标模式"""
        self.assertEqual(ClickMode.ICON.value, "icon")


def run_tests():
    """运行所有测试"""
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()

    # 添加所有测试
    suite.addTests(loader.loadTestsFromTestCase(TestTrajectoryPoint))
    suite.addTests(loader.loadTestsFromTestCase(TestJigsawPiece))
    suite.addTests(loader.loadTestsFromTestCase(TestExceptions))
    suite.addTests(loader.loadTestsFromTestCase(TestCaptchaClient))
    suite.addTests(loader.loadTestsFromTestCase(TestCaptchaTypes))
    suite.addTests(loader.loadTestsFromTestCase(TestClickMode))

    # 运行测试
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    return result


if __name__ == "__main__":
    result = run_tests()
    sys.exit(0 if result.wasSuccessful() else 1)
