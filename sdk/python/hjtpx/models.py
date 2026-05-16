"""hjtpx Python SDK - 数据模型定义"""

from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any
from enum import Enum
import json


class CaptchaType(str, Enum):
    """验证码类型"""
    NUMBER = "number"
    LETTER = "letter"
    MIXED = "mixed"


@dataclass
class ClickData:
    """点击数据"""
    x: int
    y: int
    duration: int = 0

    def to_dict(self) -> Dict[str, Any]:
        return {
            "x": self.x,
            "y": self.y,
            "duration": self.duration,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ClickData":
        return cls(
            x=data.get("x", 0),
            y=data.get("y", 0),
            duration=data.get("duration", 0),
        )


@dataclass
class ImageCaptchaRequest:
    """图片验证码请求"""
    captcha_type: CaptchaType = CaptchaType.MIXED
    count: int = 4
    custom_set: Optional[str] = None
    noise_mode: int = 0
    line_mode: int = 0

    def to_params(self) -> Dict[str, Any]:
        params = {
            "type": self.captcha_type.value,
            "count": self.count,
        }
        if self.custom_set:
            params["custom_set"] = self.custom_set
        if self.noise_mode > 0:
            params["noise_mode"] = self.noise_mode
        if self.line_mode > 0:
            params["line_mode"] = self.line_mode
        return params


@dataclass
class ImageCaptchaResponse:
    """图片验证码响应"""
    challenge_id: str
    image: str

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ImageCaptchaResponse":
        return cls(
            challenge_id=data.get("challenge_id", ""),
            image=data.get("image", ""),
        )


@dataclass
class SliderCaptchaResponse:
    """滑块验证码响应"""
    challenge_id: str
    background_image: str
    slider_image: str
    slider_width: int = 0
    slider_height: int = 0

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "SliderCaptchaResponse":
        return cls(
            challenge_id=data.get("challenge_id", ""),
            background_image=data.get("background_image", ""),
            slider_image=data.get("slider_image", ""),
            slider_width=data.get("slider_width", 0),
            slider_height=data.get("slider_height", 0),
        )


@dataclass
class ClickCaptchaResponse:
    """点选验证码响应"""
    challenge_id: str
    background_image: str
    target_position: List[int] = field(default_factory=list)
    target_index: int = 0
    icon_positions: List[List[int]] = field(default_factory=list)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ClickCaptchaResponse":
        return cls(
            challenge_id=data.get("challenge_id", ""),
            background_image=data.get("background_image", ""),
            target_position=data.get("target_position", []),
            target_index=data.get("target_index", 0),
            icon_positions=data.get("icon_positions", []),
        )


@dataclass
class VerifyCaptchaResponse:
    """验证响应"""
    success: bool
    score: float = 0.0
    message: str = ""
    risk_level: str = ""

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "VerifyCaptchaResponse":
        return cls(
            success=data.get("success", False),
            score=data.get("score", 0.0),
            message=data.get("message", ""),
            risk_level=data.get("risk_level", ""),
        )


@dataclass
class VerifyImageCaptchaResponse:
    """图片验证码验证响应"""
    success: bool

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "VerifyImageCaptchaResponse":
        return cls(
            success=data.get("success", False),
        )


@dataclass
class PoolStats:
    """连接池统计"""
    active_connections: int = 0
    idle_connections: int = 0
    total_requests: int = 0
    failed_requests: int = 0
    successful_requests: int = 0
    retried_requests: int = 0
    success_rate: float = 0.0
    last_error: Optional[str] = None
    last_error_time: Optional[str] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "PoolStats":
        return cls(
            active_connections=data.get("active_connections", 0),
            idle_connections=data.get("idle_connections", 0),
            total_requests=data.get("total_requests", 0),
            failed_requests=data.get("failed_requests", 0),
            successful_requests=data.get("successful_requests", 0),
            retried_requests=data.get("retried_requests", 0),
            success_rate=data.get("success_rate", 0.0),
            last_error=data.get("last_error"),
            last_error_time=data.get("last_error_time"),
        )


@dataclass
class SDKResponse:
    """SDK统一响应"""
    code: int
    message: str
    data: Optional[Dict[str, Any]] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "SDKResponse":
        json_data = data.get("data")
        parsed_data = None
        if json_data:
            if isinstance(json_data, dict):
                parsed_data = json_data
            else:
                parsed_data = json.loads(json_data) if isinstance(json_data, str) else json_data
        
        return cls(
            code=data.get("code", 0),
            message=data.get("message", ""),
            data=parsed_data,
        )


@dataclass
class BehaviorData:
    """行为数据"""
    x: int
    y: int
    timestamp: int
    event: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "x": self.x,
            "y": self.y,
            "timestamp": self.timestamp,
            "event": self.event,
        }


@dataclass
class MouseTrajectory:
    """鼠标轨迹"""
    points: List[BehaviorData] = field(default_factory=list)
    total_distance: float = 0.0
    average_speed: float = 0.0
    max_speed: float = 0.0
    min_speed: float = 0.0
    path_efficiency: float = 0.0
    direction_changes: int = 0

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "MouseTrajectory":
        points = [BehaviorData(**p) if isinstance(p, dict) else p for p in data.get("points", [])]
        return cls(
            points=points,
            total_distance=data.get("total_distance", 0.0),
            average_speed=data.get("average_speed", 0.0),
            max_speed=data.get("max_speed", 0.0),
            min_speed=data.get("min_speed", 0.0),
            path_efficiency=data.get("path_efficiency", 0.0),
            direction_changes=data.get("direction_changes", 0),
        )


@dataclass
class ClickPattern:
    """点击模式"""
    clicks: List[BehaviorData] = field(default_factory=list)
    click_count: int = 0
    average_interval: float = 0.0
    click_speed: float = 0.0
    regularity: float = 0.0

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ClickPattern":
        clicks = [BehaviorData(**c) if isinstance(c, dict) else c for c in data.get("clicks", [])]
        return cls(
            clicks=clicks,
            click_count=data.get("click_count", 0),
            average_interval=data.get("average_interval", 0.0),
            click_speed=data.get("click_speed", 0.0),
            regularity=data.get("regularity", 0.0),
        )


@dataclass
class SpeedAnalysis:
    """速度分析"""
    speeds: List[float] = field(default_factory=list)
    average_speed: float = 0.0
    median_speed: float = 0.0
    max_speed: float = 0.0
    min_speed: float = 0.0
    speed_variance: float = 0.0
    speed_std_dev: float = 0.0
    is_speed_consistent: bool = False
    speed_outliers: int = 0

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "SpeedAnalysis":
        return cls(
            speeds=data.get("speeds", []),
            average_speed=data.get("average_speed", 0.0),
            median_speed=data.get("median_speed", 0.0),
            max_speed=data.get("max_speed", 0.0),
            min_speed=data.get("min_speed", 0.0),
            speed_variance=data.get("speed_variance", 0.0),
            speed_std_dev=data.get("speed_std_dev", 0.0),
            is_speed_consistent=data.get("is_speed_consistent", False),
            speed_outliers=data.get("speed_outliers", 0),
        )


@dataclass
class AnalysisResult:
    """分析结果"""
    trajectory: MouseTrajectory
    click_pattern: ClickPattern
    speed_analysis: SpeedAnalysis
    risk_score: float = 0.0
    risk_indicators: List[str] = field(default_factory=list)
    is_bot_likely: bool = False
    confidence: float = 0.0
    risk_factors: Dict[str, float] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "AnalysisResult":
        return cls(
            trajectory=MouseTrajectory.from_dict(data.get("trajectory", {})),
            click_pattern=ClickPattern.from_dict(data.get("click_pattern", {})),
            speed_analysis=SpeedAnalysis.from_dict(data.get("speed_analysis", {})),
            risk_score=data.get("risk_score", 0.0),
            risk_indicators=data.get("risk_indicators", []),
            is_bot_likely=data.get("is_bot_likely", False),
            confidence=data.get("confidence", 0.0),
            risk_factors=data.get("risk_factors", {}),
        )


@dataclass
class SliderCaptchaRequest:
    """滑块验证码请求"""
    width: int = 360
    height: int = 220

    def to_params(self) -> Dict[str, Any]:
        return {
            "width": self.width,
            "height": self.height,
        }


@dataclass
class ClickCaptchaRequest:
    """点选验证码请求"""
    width: int = 360
    height: int = 220
    icon_count: int = 4

    def to_params(self) -> Dict[str, Any]:
        return {
            "width": self.width,
            "height": self.height,
            "icon_count": self.icon_count,
        }


@dataclass
class VerifyCaptchaRequest:
    """验证请求"""
    challenge_id: str
    action: str
    data: Optional[Dict[str, Any]] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "challenge_id": self.challenge_id,
            "action": self.action,
        }
        if self.data:
            result["data"] = self.data
        return result


@dataclass
class VerifyImageCaptchaRequest:
    """图片验证码验证请求"""
    challenge_id: str
    answer: str

    def to_dict(self) -> Dict[str, Any]:
        return {
            "challenge_id": self.challenge_id,
            "answer": self.answer,
        }
