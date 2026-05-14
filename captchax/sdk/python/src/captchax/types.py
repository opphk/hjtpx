"""Type definitions for CaptchaX Python SDK."""

from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any, Literal
from enum import Enum


class ApiVersion(Enum):
    """API version enum."""
    V1 = 'v1'
    V2 = 'v2'


@dataclass
class CaptchaConfig:
    """Configuration for CaptchaX client."""
    base_url: str
    app_id: Optional[str] = None
    timeout: int = 10000
    retry_times: int = 3
    api_version: ApiVersion = ApiVersion.V1


@dataclass
class SliderCaptchaResult:
    """Slider captcha generation result."""
    id: str
    background_b64: str
    slider_b64: str
    target_x: int
    target_y: int


@dataclass
class SliderVerifyResult:
    """Slider captcha verification result."""
    success: bool
    message: str


@dataclass
class CharPosition:
    """Character position for click captcha."""
    char: str
    x: int
    y: int


@dataclass
class ClickCaptchaResult:
    """Click captcha generation result."""
    id: str
    image: str
    target_chars: List[str]
    char_positions: List[CharPosition]


@dataclass
class ClickVerifyResult:
    """Click captcha verification result."""
    success: bool
    score: float
    message: str


@dataclass
class PuzzleCaptchaResult:
    """Puzzle captcha generation result."""
    id: str
    background_b64: str
    puzzle_b64: str
    target_x: int
    target_y: int


@dataclass
class PuzzleVerifyResult:
    """Puzzle captcha verification result."""
    success: bool
    message: str


@dataclass
class Scenario:
    """Captcha verification scenario."""
    id: Optional[str] = None
    name: Optional[str] = None
    description: Optional[str] = None
    difficulty: Optional[str] = None
    config: Optional[Dict[str, Any]] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


@dataclass
class Webhook:
    """Webhook configuration."""
    id: Optional[str] = None
    app_id: Optional[str] = None
    url: Optional[str] = None
    secret: Optional[str] = None
    events: List[str] = field(default_factory=list)
    headers: Optional[Dict[str, str]] = None
    enabled: Optional[bool] = True
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


@dataclass
class BatchVerifyItem:
    """Batch verification item."""
    captcha_id: str
    type: Literal['slider', 'click', 'puzzle']
    target_x: int
    target_y: Optional[int] = None
    clicks: Optional[List[CharPosition]] = None


@dataclass
class BatchVerifyResult:
    """Single batch verification result."""
    captcha_id: str
    success: bool
    message: str
    score: Optional[float] = None


@dataclass
class BatchVerifySummary:
    """Batch verification summary."""
    total: int
    success: int
    failed: int
    skipped: int


@dataclass
class BatchVerifyResponse:
    """Batch verification response."""
    results: List[BatchVerifyResult]
    summary: BatchVerifySummary


@dataclass
class HealthStatus:
    """Health check response."""
    status: str
    service: str
    timestamp: str
    version: str


@dataclass
class CaptchaResponse:
    """Generic API response wrapper."""
    code: int
    message: str
    data: Optional[Any] = None
