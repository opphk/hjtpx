"""Tests for CaptchaX Python SDK."""

import pytest
from captchax import (
    CaptchaXClient,
    CaptchaXError,
    CaptchaConfig,
    ApiVersion,
    SliderCaptchaResult,
    SliderVerifyResult,
    ClickCaptchaResult,
    ClickVerifyResult,
    PuzzleCaptchaResult,
    PuzzleVerifyResult,
    CharPosition,
    BatchVerifyItem,
)


class TestCaptchaXClient:
    """Test CaptchaXClient."""

    def test_constructor_with_config_object(self):
        """Test constructor with CaptchaConfig object."""
        config = CaptchaConfig(base_url='https://captchax.example.com', app_id='test-app')
        client = CaptchaXClient(config)
        assert client.get_api_version() == 'v1'

    def test_constructor_with_dict(self):
        """Test constructor with dict."""
        config = {'base_url': 'https://captchax.example.com', 'app_id': 'test-app', 'api_version': 'v2'}
        client = CaptchaXClient(config)
        assert client.get_api_version() == 'v2'

    def test_constructor_raises_without_base_url(self):
        """Test constructor raises error without base_url."""
        with pytest.raises(ValueError, match="base_url is required"):
            CaptchaXClient({'app_id': 'test-app'})

    def test_set_app_id(self):
        """Test set_app_id method."""
        config = CaptchaConfig(base_url='https://captchax.example.com')
        client = CaptchaXClient(config)
        client.set_app_id('new-app-id')
        # Verify app_id was set by attempting to generate captcha
        with pytest.raises(CaptchaXError, match="app_id is required"):
            client.generate_slider_captcha()

    def test_set_api_version(self):
        """Test set_api_version method."""
        config = CaptchaConfig(base_url='https://captchax.example.com')
        client = CaptchaXClient(config)
        assert client.get_api_version() == 'v1'
        client.set_api_version('v2')
        assert client.get_api_version() == 'v2'

    def test_set_api_version_with_enum(self):
        """Test set_api_version with ApiVersion enum."""
        config = CaptchaConfig(base_url='https://captchax.example.com')
        client = CaptchaXClient(config)
        client.set_api_version(ApiVersion.V2)
        assert client.get_api_version() == 'v2'

    def test_create_client_info(self):
        """Test create_client_info method."""
        config = CaptchaConfig(base_url='https://captchax.example.com')
        client = CaptchaXClient(config)
        info = client.create_client_info(custom_field='value')
        import json
        parsed = json.loads(info)
        assert 'timestamp' in parsed
        assert parsed['custom_field'] == 'value'


class TestCaptchaXError:
    """Test CaptchaXError exception."""

    def test_error_with_defaults(self):
        """Test error with default values."""
        error = CaptchaXError('Test error')
        assert str(error) == 'CaptchaXError(500): Test error'
        assert error.code == 500
        assert error.status_code == 500

    def test_error_with_custom_values(self):
        """Test error with custom values."""
        error = CaptchaXError('Test error', code=400, status_code=400, details={'field': 'value'})
        assert error.message == 'Test error'
        assert error.code == 400
        assert error.status_code == 400
        assert error.details == {'field': 'value'}

    def test_error_is_exception(self):
        """Test error is instance of Exception."""
        error = CaptchaXError('Test error')
        assert isinstance(error, Exception)


class TestTypes:
    """Test type definitions."""

    def test_captcha_config_defaults(self):
        """Test CaptchaConfig default values."""
        config = CaptchaConfig(base_url='https://example.com')
        assert config.timeout == 10000
        assert config.retry_times == 3
        assert config.api_version == ApiVersion.V1

    def test_char_position(self):
        """Test CharPosition dataclass."""
        pos = CharPosition(char='A', x=100, y=50)
        assert pos.char == 'A'
        assert pos.x == 100
        assert pos.y == 50

    def test_slider_captcha_result(self):
        """Test SliderCaptchaResult dataclass."""
        result = SliderCaptchaResult(
            id='cap-123',
            background_b64='data:image/png;base64,...',
            slider_b64='data:image/png;base64,...',
            target_x=150,
            target_y=40,
        )
        assert result.id == 'cap-123'
        assert result.target_x == 150

    def test_slider_verify_result(self):
        """Test SliderVerifyResult dataclass."""
        result = SliderVerifyResult(success=True, message='Verification successful')
        assert result.success is True
        assert 'successful' in result.message

    def test_click_captcha_result(self):
        """Test ClickCaptchaResult dataclass."""
        result = ClickCaptchaResult(
            id='cap-456',
            image='data:image/png;base64,...',
            target_chars=['中', '国', '人', '文'],
            char_positions=[
                CharPosition(char='中', x=50, y=30),
                CharPosition(char='国', x=120, y=25),
                CharPosition(char='人', x=80, y=60),
                CharPosition(char='文', x=150, y=45),
            ],
        )
        assert len(result.target_chars) == 4
        assert len(result.char_positions) == 4

    def test_batch_verify_item(self):
        """Test BatchVerifyItem dataclass."""
        item = BatchVerifyItem(
            captcha_id='cap-789',
            type='slider',
            target_x=150,
            target_y=40,
        )
        assert item.captcha_id == 'cap-789'
        assert item.type == 'slider'
