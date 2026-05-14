"""CaptchaX Python SDK - Python client for CaptchaX verification service."""

__version__ = '1.0.0'
__author__ = 'CaptchaX Team'
__license__ = 'MIT'

from .client import CaptchaXClient, CaptchaXError
from .types import (
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
    BatchVerifyResponse,
    HealthStatus,
    CharPosition,
)

__all__ = [
    'CaptchaXClient',
    'CaptchaXError',
    'CaptchaConfig',
    'SliderCaptchaResult',
    'SliderVerifyResult',
    'ClickCaptchaResult',
    'ClickVerifyResult',
    'PuzzleCaptchaResult',
    'PuzzleVerifyResult',
    'Scenario',
    'Webhook',
    'BatchVerifyItem',
    'BatchVerifyResponse',
    'HealthStatus',
    'CharPosition',
]
