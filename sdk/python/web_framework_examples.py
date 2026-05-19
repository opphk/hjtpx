"""
Web框架集成示例

展示如何在FastAPI和Django中使用异步SDK。
"""

import asyncio
import sys
import os
from typing import Optional
from dataclasses import dataclass

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from async_captcha import AsyncCaptchaClient, AsyncCaptchaType, AsyncClickMode, AsyncCaptchaError


@dataclass
class CaptchaResponse:
    """验证码响应封装"""
    success: bool
    session_id: str
    data: Optional[dict] = None
    error: Optional[str] = None


class CaptchaService:
    """验证码服务"""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self._client: Optional[AsyncCaptchaClient] = None

    async def get_client(self) -> AsyncCaptchaClient:
        """获取或创建客户端"""
        if self._client is None:
            self._client = AsyncCaptchaClient(self.base_url)
        return self._client

    async def generate_slider(self, width: int = 320, height: int = 160) -> CaptchaResponse:
        """生成滑块验证码"""
        try:
            client = await self.get_client()
            slider = await client.get_slider_captcha(width, height)

            return CaptchaResponse(
                success=True,
                session_id=slider.session_id,
                data={
                    "image_url": slider.image_url,
                    "puzzle_url": slider.puzzle_url,
                    "secret_y": slider.secret_y,
                }
            )
        except AsyncCaptchaError as e:
            return CaptchaResponse(
                success=False,
                session_id="",
                error=str(e)
            )

    async def verify_slider(
        self,
        session_id: str,
        x: int,
        y: Optional[int] = None,
        trajectory: Optional[list] = None
    ) -> CaptchaResponse:
        """验证滑块验证码"""
        try:
            client = await self.get_client()
            result = await client.verify_slider_captcha(
                session_id=session_id,
                x=x,
                y=y,
                trajectory=trajectory
            )

            return CaptchaResponse(
                success=result.success,
                session_id=session_id,
                data={
                    "message": result.message,
                    "risk_score": result.risk_score,
                }
            )
        except AsyncCaptchaError as e:
            return CaptchaResponse(
                success=False,
                session_id=session_id,
                error=str(e)
            )

    async def generate_click(self, mode: str = "number", max_points: int = 3) -> CaptchaResponse:
        """生成点击验证码"""
        try:
            client = await self.get_client()
            click = await client.get_click_captcha(mode=mode, max_points=max_points)

            return CaptchaResponse(
                success=True,
                session_id=click.session_id,
                data={
                    "image_url": click.image_url,
                    "hint": click.hint,
                    "hint_order": click.hint_order,
                }
            )
        except AsyncCaptchaError as e:
            return CaptchaResponse(
                success=False,
                session_id="",
                error=str(e)
            )

    async def verify_click(
        self,
        session_id: str,
        points: list,
        click_sequence: Optional[list] = None
    ) -> CaptchaResponse:
        """验证点击验证码"""
        try:
            client = await self.get_client()
            result = await client.verify_click_captcha(
                session_id=session_id,
                points=points,
                click_sequence=click_sequence
            )

            return CaptchaResponse(
                success=result.success,
                session_id=session_id,
                data={
                    "message": result.message,
                    "risk_score": result.risk_score,
                }
            )
        except AsyncCaptchaError as e:
            return CaptchaResponse(
                success=False,
                session_id=session_id,
                error=str(e)
            )

    async def close(self):
        """关闭服务"""
        if self._client:
            await self._client.close()
            self._client = None


async def fastapi_example():
    """
    FastAPI 集成示例

    这是一个完整的 FastAPI 应用示例，展示了如何集成异步验证码SDK。
    """
    print("\n" + "="*60)
    print("FastAPI 集成示例")
    print("="*60)

    try:
        from fastapi import FastAPI, HTTPException
        from fastapi.responses import JSONResponse
        from pydantic import BaseModel
        import uvicorn
    except ImportError:
        print("请安装 FastAPI 和 uvicorn: pip install fastapi uvicorn")
        return

    app = FastAPI(title="验证码服务 API")
    service = CaptchaService()

    class SliderRequest(BaseModel):
        width: int = 320
        height: int = 160

    class SliderVerifyRequest(BaseModel):
        session_id: str
        x: int
        y: Optional[int] = None
        trajectory: Optional[list] = None

    class ClickRequest(BaseModel):
        mode: str = "number"
        max_points: int = 3

    class ClickVerifyRequest(BaseModel):
        session_id: str
        points: list
        click_sequence: Optional[list] = None

    @app.post("/captcha/slider", response_model=dict)
    async def create_slider_captcha(request: SliderRequest):
        """创建滑块验证码"""
        result = await service.generate_slider(request.width, request.height)

        if not result.success:
            raise HTTPException(status_code=500, detail=result.error)

        return {
            "success": True,
            "session_id": result.session_id,
            "data": result.data,
        }

    @app.post("/captcha/slider/verify", response_model=dict)
    async def verify_slider_captcha(request: SliderVerifyRequest):
        """验证滑块验证码"""
        result = await service.verify_slider_captcha(
            request.session_id,
            request.x,
            request.y,
            request.trajectory
        )

        return {
            "success": result.success,
            "data": result.data,
        }

    @app.post("/captcha/click", response_model=dict)
    async def create_click_captcha(request: ClickRequest):
        """创建点击验证码"""
        result = await service.generate_click(request.mode, request.max_points)

        if not result.success:
            raise HTTPException(status_code=500, detail=result.error)

        return {
            "success": True,
            "session_id": result.session_id,
            "data": result.data,
        }

    @app.post("/captcha/click/verify", response_model=dict)
    async def verify_click_captcha(request: ClickVerifyRequest):
        """验证点击验证码"""
        result = await service.verify_click_captcha(
            request.session_id,
            request.points,
            request.click_sequence
        )

        return {
            "success": result.success,
            "data": result.data,
        }

    @app.get("/health")
    async def health_check():
        """健康检查"""
        return {"status": "healthy"}

    @app.on_event("shutdown")
    async def shutdown_event():
        """关闭事件"""
        await service.close()

    print("FastAPI 应用已创建")
    print("启动服务器...")
    print("API 端点:")
    print("  POST /captcha/slider - 创建滑块验证码")
    print("  POST /captcha/slider/verify - 验证滑块验证码")
    print("  POST /captcha/click - 创建点击验证码")
    print("  POST /captcha/click/verify - 验证点击验证码")
    print("  GET /health - 健康检查")
    print("\n启动 uvicorn 服务器...")

    config = uvicorn.Config(app, host="0.0.0.0", port=8000, log_level="info")
    server = uvicorn.Server(config)

    asyncio.create_task(server.serve())
    await asyncio.sleep(2)

    print("\n测试 API 端点...")

    import httpx

    async with httpx.AsyncClient() as client:
        response = await client.post(
            "http://localhost:8000/captcha/slider",
            json={"width": 320, "height": 160}
        )
        print(f"  POST /captcha/slider: {response.status_code}")

        if response.status_code == 200:
            data = response.json()
            print(f"    Session ID: {data.get('session_id', 'N/A')[:20]}...")

        response = await client.get("http://localhost:8000/health")
        print(f"  GET /health: {response.status_code}")

    print("\n关闭服务器...")
    server.should_exit = True

    await service.close()


async def django_example():
    """
    Django 集成示例

    这是一个 Django 视图示例，展示了如何在 Django 中使用异步验证码SDK。
    """
    print("\n" + "="*60)
    print("Django 集成示例")
    print("="*60)

    print("\n注意: Django 使用同步视图，需要在视图中使用 async_to_sync")
    print("或者使用 ASGI 服务器（如 uvicorn）运行异步视图")

    print("\n示例 Django 视图代码:")

    example_code = '''
# views.py
import asyncio
from django.http import JsonResponse
from asgiref.sync import async_to_sync
from .services import CaptchaService

service = CaptchaService()

@async_to_sync
async def slider_captcha(request):
    if request.method == "POST":
        width = int(request.POST.get("width", 320))
        height = int(request.POST.get("height", 160))

        result = await service.generate_slider(width, height)

        return JsonResponse({
            "success": result.success,
            "session_id": result.session_id,
            "data": result.data,
            "error": result.error,
        })

    return JsonResponse({"error": "Method not allowed"}, status=405)

@async_to_sync
async def verify_slider(request):
    if request.method == "POST":
        import json
        data = json.loads(request.body)

        result = await service.verify_slider(
            data["session_id"],
            data["x"],
            data.get("y"),
            data.get("trajectory"),
        )

        return JsonResponse({
            "success": result.success,
            "data": result.data,
            "error": result.error,
        })

    return JsonResponse({"error": "Method not allowed"}, status=405)
'''

    print(example_code)

    print("\nDjango URL 配置示例:")

    url_example = '''
# urls.py
from django.urls import path
from . import views

urlpatterns = [
    path('captcha/slider', views.slider_captcha, name='slider_captcha'),
    path('captcha/slider/verify', views.verify_slider, name='verify_slider'),
]
'''

    print(url_example)


async def main():
    """主函数"""
    print("="*60)
    print("Web 框架集成示例")
    print("="*60)

    await fastapi_example()
    await django_example()

    print("\n" + "="*60)
    print("Web 框架示例完成")
    print("="*60)


if __name__ == "__main__":
    asyncio.run(main())
