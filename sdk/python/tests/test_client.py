"""hjtpx Python SDK - Tests"""

import pytest
import json
import base64
from unittest.mock import Mock, patch, MagicMock
import urllib.request
import urllib.error

from hjtpx import CaptchaClient, Config
from hjtpx import (
    SDKError,
    NetworkError,
    TimeoutError,
    InvalidParamsError,
    RateLimitedError,
    UnauthorizedError,
    InvalidResponseError,
    ServerError,
)
from hjtpx import (
    ImageCaptchaRequest,
    SliderCaptchaRequest,
    ClickCaptchaRequest,
    ClickData,
    CaptchaType,
)


class TestConfig:
    """Test Config class"""

    def test_default_values(self):
        config = Config()
        assert config.base_url == "http://localhost:8080"
        assert config.app_id == ""
        assert config.app_secret == ""
        assert config.timeout == 30.0
        assert config.max_retries == 3
        assert config.retry_delay == 0.1

    def test_custom_values(self):
        config = Config(
            base_url="https://api.example.com",
            app_id="test-id",
            app_secret="test-secret",
            timeout=60.0,
            max_retries=5,
        )
        assert config.base_url == "https://api.example.com"
        assert config.app_id == "test-id"
        assert config.app_secret == "test-secret"
        assert config.timeout == 60.0
        assert config.max_retries == 5


class TestCaptchaClient:
    """Test CaptchaClient class"""

    def test_init_default(self):
        client = CaptchaClient()
        assert client.config.base_url == "http://localhost:8080"

    def test_init_with_config(self):
        config = Config(base_url="https://api.example.com", app_id="test-id")
        client = CaptchaClient(config)
        assert client.config.base_url == "https://api.example.com"
        assert client.config.app_id == "test-id"

    def test_set_debug_mode(self):
        client = CaptchaClient()
        client.set_debug_mode(True)
        assert client.config.debug_mode is True
        client.set_debug_mode(False)
        assert client.config.debug_mode is False

    def test_set_timeout(self):
        client = CaptchaClient()
        client.set_timeout(60.0)
        assert client.config.timeout == 60.0

    def test_set_max_retries(self):
        client = CaptchaClient()
        client.set_max_retries(10)
        assert client.config.max_retries == 10

    def test_set_retry_delay(self):
        client = CaptchaClient()
        client.set_retry_delay(0.5)
        assert client.config.retry_delay == 0.5

    def test_close(self):
        client = CaptchaClient()
        client.close()


class TestImageCaptcha:
    """Test image captcha methods"""

    @patch("urllib.request.urlopen")
    def test_generate_image_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "challenge_id": "test-challenge-id",
                "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        result = client.generate_image_captcha()

        assert result.challenge_id == "test-challenge-id"
        assert result.image.startswith("data:image/png;base64,")

    @patch("urllib.request.urlopen")
    def test_generate_image_captcha_with_params(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "challenge_id": "test-challenge-id",
                "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        request = ImageCaptchaRequest(captcha_type=CaptchaType.NUMBER, count=4)
        result = client.generate_image_captcha(request)

        assert result.challenge_id == "test-challenge-id"

    @patch("urllib.request.urlopen")
    def test_verify_image_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {"success": True},
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        result = client.verify_image_captcha("test-challenge-id", "1234")

        assert result.success is True

    def test_verify_image_captcha_missing_challenge_id(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.verify_image_captcha("", "1234")

    def test_verify_image_captcha_missing_answer(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.verify_image_captcha("test-challenge-id", "")


class TestSliderCaptcha:
    """Test slider captcha methods"""

    @patch("urllib.request.urlopen")
    def test_generate_slider_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "challenge_id": "slider-test-id",
                "background_image": "data:image/png;base64,abc",
                "slider_image": "data:image/png;base64,xyz",
                "slider_width": 50,
                "slider_height": 50,
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        result = client.generate_slider_captcha()

        assert result.challenge_id == "slider-test-id"
        assert result.slider_width == 50
        assert result.slider_height == 50

    @patch("urllib.request.urlopen")
    def test_verify_slider_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "success": True,
                "score": 15.5,
                "risk_level": "low",
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        result = client.verify_slider_captcha("slider-test-id", "120")

        assert result.success is True
        assert result.score == 15.5
        assert result.risk_level == "low"

    def test_verify_slider_captcha_missing_challenge_id(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.verify_slider_captcha("", "120")

    def test_verify_slider_captcha_missing_offset(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.verify_slider_captcha("slider-test-id", "")


class TestClickCaptcha:
    """Test click captcha methods"""

    @patch("urllib.request.urlopen")
    def test_generate_click_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "challenge_id": "click-test-id",
                "background_image": "data:image/png;base64,abc",
                "target_position": [100, 120],
                "target_index": 2,
                "icon_positions": [[50, 60], [100, 120], [150, 180], [200, 220]],
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        result = client.generate_click_captcha()

        assert result.challenge_id == "click-test-id"
        assert result.target_index == 2
        assert len(result.icon_positions) == 4

    @patch("urllib.request.urlopen")
    def test_verify_click_captcha_success(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "success": True,
                "score": 10.0,
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        clicks = [
            ClickData(x=100, y=120, duration=500),
            ClickData(x=150, y=180, duration=300),
        ]
        result = client.verify_click_captcha("click-test-id", clicks)

        assert result.success is True

    def test_verify_click_captcha_missing_clicks(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.verify_click_captcha("click-test-id", [])


class TestExtractBase64Image:
    """Test base64 image extraction"""

    def test_extract_png_image(self):
        client = CaptchaClient()
        test_image = b"fake png data"
        base64_data = base64.b64encode(test_image).decode("utf-8")
        data_uri = f"data:image/png;base64,{base64_data}"

        result = client.extract_base64_image(data_uri)
        assert result == test_image

    def test_extract_jpeg_image(self):
        client = CaptchaClient()
        test_image = b"fake jpeg data"
        base64_data = base64.b64encode(test_image).decode("utf-8")
        data_uri = f"data:image/jpeg;base64,{base64_data}"

        result = client.extract_base64_image(data_uri)
        assert result == test_image

    def test_extract_empty_data_uri(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.extract_base64_image("")

    def test_extract_unsupported_format(self):
        client = CaptchaClient()
        with pytest.raises(InvalidParamsError):
            client.extract_base64_image("data:image/gif;base64,abc")


class TestErrorHandling:
    """Test error handling"""

    @patch("urllib.request.urlopen")
    def test_rate_limited_error(self, mock_urlopen):
        mock_error = urllib.error.HTTPError(
            url="http://localhost:8080",
            code=429,
            msg="Too Many Requests",
            hdrs={"Retry-After": "60"},
            fp=None,
        )
        mock_urlopen.side_effect = mock_error

        client = CaptchaClient()
        with pytest.raises(RateLimitedError) as exc_info:
            client.generate_image_captcha()

        assert exc_info.value.retry_after == 60

    @patch("urllib.request.urlopen")
    def test_unauthorized_error(self, mock_urlopen):
        mock_error = urllib.error.HTTPError(
            url="http://localhost:8080",
            code=401,
            msg="Unauthorized",
            hdrs={},
            fp=None,
        )
        mock_urlopen.side_effect = mock_error

        client = CaptchaClient()
        with pytest.raises(UnauthorizedError):
            client.generate_image_captcha()

    @patch("urllib.request.urlopen")
    def test_server_error(self, mock_urlopen):
        mock_error = urllib.error.HTTPError(
            url="http://localhost:8080",
            code=500,
            msg="Internal Server Error",
            hdrs={},
            fp=None,
        )
        mock_urlopen.side_effect = mock_error

        client = CaptchaClient()
        with pytest.raises(ServerError):
            client.generate_image_captcha()


class TestStats:
    """Test statistics methods"""

    @patch("urllib.request.urlopen")
    def test_get_stats(self, mock_urlopen):
        mock_response = {
            "code": 0,
            "message": "success",
            "data": {
                "challenge_id": "test-id",
                "image": "data:image/png;base64,abc",
            },
        }

        mock_response_obj = MagicMock()
        mock_response_obj.__enter__ = Mock(return_value=mock_response_obj)
        mock_response_obj.__exit__ = Mock(return_value=False)
        mock_response_obj.read.return_value = json.dumps(mock_response).encode("utf-8")
        mock_urlopen.return_value = mock_response_obj

        client = CaptchaClient()
        client.generate_image_captcha()
        stats = client.get_stats()

        assert stats.total_requests == 1
        assert stats.successful_requests == 1
        assert stats.success_rate > 0


class TestModels:
    """Test data models"""

    def test_click_data_to_dict(self):
        click = ClickData(x=100, y=200, duration=500)
        data = click.to_dict()

        assert data["x"] == 100
        assert data["y"] == 200
        assert data["duration"] == 500

    def test_image_captcha_request_to_params(self):
        request = ImageCaptchaRequest(
            captcha_type=CaptchaType.NUMBER,
            count=4,
            noise_mode=2,
            line_mode=1,
        )
        params = request.to_params()

        assert params["type"] == "number"
        assert params["count"] == 4
        assert params["noise_mode"] == 2
        assert params["line_mode"] == 1

    def test_slider_captcha_request_to_params(self):
        request = SliderCaptchaRequest(width=360, height=220)
        params = request.to_params()

        assert params["width"] == 360
        assert params["height"] == 220

    def test_click_captcha_request_to_params(self):
        request = ClickCaptchaRequest(width=360, height=220, icon_count=4)
        params = request.to_params()

        assert params["width"] == 360
        assert params["height"] == 220
        assert params["icon_count"] == 4


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
