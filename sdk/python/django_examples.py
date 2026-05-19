"""
Django集成示例

展示如何在Django项目中使用Python SDK进行验证码验证
"""

from django.http import JsonResponse
from django.views import View
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator

from captcha import CaptchaClient, CaptchaError, CaptchaValidationError


class CaptchaConfig:
    """验证码配置"""
    BASE_URL = "http://localhost:8080"
    API_KEY = "your-api-key"
    TIMEOUT = 30


def get_captcha_client():
    """获取验证码客户端实例"""
    return CaptchaClient(
        base_url=CaptchaConfig.BASE_URL,
        api_key=CaptchaConfig.API_KEY,
        timeout=CaptchaConfig.TIMEOUT,
    )


@method_decorator(csrf_exempt, name='dispatch')
class SliderCaptchaView(View):
    """滑块验证码API"""

    def get(self, request):
        """获取滑块验证码"""
        try:
            with get_captcha_client() as client:
                captcha = client.get_slider_captcha(
                    width=320,
                    height=160,
                    tolerance=8
                )

                return JsonResponse({
                    'success': True,
                    'session_id': captcha.session_id,
                    'image_url': captcha.image_url,
                    'puzzle_url': captcha.puzzle_url,
                    'secret_y': captcha.secret_y,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)

    def post(self, request):
        """验证滑块验证码"""
        import json

        try:
            data = json.loads(request.body)
            session_id = data.get('session_id')
            x = data.get('x')
            y = data.get('y')
            trajectory = data.get('trajectory')

            if not session_id or x is None:
                return JsonResponse({
                    'success': False,
                    'error': 'Missing required parameters',
                }, status=400)

            with get_captcha_client() as client:
                result = client.verify_slider_captcha(
                    session_id=session_id,
                    x=x,
                    y=y,
                    trajectory=trajectory,
                )

                return JsonResponse({
                    'success': result.success,
                    'message': result.message,
                    'risk_score': result.risk_score,
                    'captcha_pass': result.captcha_pass,
                })

        except CaptchaValidationError as e:
            return JsonResponse({
                'success': False,
                'error': 'Validation error',
                'details': str(e),
            }, status=400)

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)


@method_decorator(csrf_exempt, name='dispatch')
class ClickCaptchaView(View):
    """点击验证码API"""

    def get(self, request):
        """获取点击验证码"""
        mode = request.GET.get('mode', 'number')
        max_points = int(request.GET.get('points', 3))

        try:
            with get_captcha_client() as client:
                captcha = client.get_click_captcha(
                    mode=mode,
                    max_points=max_points,
                    allow_shuffle=True
                )

                return JsonResponse({
                    'success': True,
                    'session_id': captcha.session_id,
                    'image_url': captcha.image_url,
                    'hint': captcha.hint,
                    'hint_order': captcha.hint_order,
                    'mode': captcha.mode,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)

    def post(self, request):
        """验证点击验证码"""
        import json

        try:
            data = json.loads(request.body)
            session_id = data.get('session_id')
            points = data.get('points')
            click_sequence = data.get('click_sequence')

            if not session_id or not points:
                return JsonResponse({
                    'success': False,
                    'error': 'Missing required parameters',
                }, status=400)

            with get_captcha_client() as client:
                result = client.verify_click_captcha(
                    session_id=session_id,
                    points=points,
                    click_sequence=click_sequence,
                )

                return JsonResponse({
                    'success': result.success,
                    'message': result.message,
                    'risk_score': result.risk_score,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)


@method_decorator(csrf_exempt, name='dispatch')
class ImageCaptchaView(View):
    """图形验证码API"""

    def get(self, request):
        """获取图形验证码"""
        captcha_type = request.GET.get('type', 'mixed')
        count = int(request.GET.get('count', 4))

        try:
            with get_captcha_client() as client:
                captcha = client.get_image_captcha(
                    type_=captcha_type,
                    count=count,
                )

                return JsonResponse({
                    'success': True,
                    'challenge_id': captcha.challenge_id,
                    'image': captcha.image,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)

    def post(self, request):
        """验证图形验证码"""
        import json

        try:
            data = json.loads(request.body)
            challenge_id = data.get('challenge_id')
            answer = data.get('answer')

            if not challenge_id or not answer:
                return JsonResponse({
                    'success': False,
                    'error': 'Missing required parameters',
                }, status=400)

            with get_captcha_client() as client:
                result = client.verify_image_captcha(
                    challenge_id=challenge_id,
                    answer=answer,
                )

                return JsonResponse({
                    'success': result.success,
                    'message': result.message,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=500)


class LoginView(View):
    """登录视图（带验证码验证）"""

    def post(self, request):
        """处理登录请求"""
        import json

        try:
            data = json.loads(request.body)
            username = data.get('username')
            password = data.get('password')
            captcha_token = data.get('captcha_token')

            if not username or not password:
                return JsonResponse({
                    'success': False,
                    'error': 'Missing credentials',
                }, status=400)

            with get_captcha_client() as client:
                auth = client.auth()

                login_result = auth.login(
                    username=username,
                    password=password,
                    captcha_token=captcha_token,
                )

                return JsonResponse({
                    'success': True,
                    'access_token': login_result.access_token,
                    'refresh_token': login_result.refresh_token,
                    'expires_in': login_result.expires_in,
                })

        except CaptchaError as e:
            return JsonResponse({
                'success': False,
                'error': str(e),
            }, status=401)


class CaptchaService:
    """验证码服务类"""

    @staticmethod
    def get_slider():
        """获取滑块验证码"""
        with get_captcha_client() as client:
            return client.get_slider_captcha()

    @staticmethod
    def verify_slider(session_id, x, y=None, trajectory=None):
        """验证滑块验证码"""
        with get_captcha_client() as client:
            return client.verify_slider_captcha(
                session_id=session_id,
                x=x,
                y=y,
                trajectory=trajectory,
            )

    @staticmethod
    def get_click(mode='number', points=3):
        """获取点击验证码"""
        with get_captcha_client() as client:
            return client.get_click_captcha(
                mode=mode,
                max_points=points,
            )

    @staticmethod
    def verify_click(session_id, points, click_sequence=None):
        """验证点击验证码"""
        with get_captcha_client() as client:
            return client.verify_click_captcha(
                session_id=session_id,
                points=points,
                click_sequence=click_sequence,
            )

    @staticmethod
    def get_image(type_='mixed', count=4):
        """获取图形验证码"""
        with get_captcha_client() as client:
            return client.get_image_captcha(type_=type_, count=count)

    @staticmethod
    def verify_image(challenge_id, answer):
        """验证图形验证码"""
        with get_captcha_client() as client:
            return client.verify_image_captcha(
                challenge_id=challenge_id,
                answer=answer,
            )


if __name__ == '__main__':
    print("Django集成示例")
    print("-" * 50)

    print("\n使用 CaptchaService:")
    print("captcha = CaptchaService.get_slider()")
    print("result = CaptchaService.verify_slider(session_id, x, y)")
    print("-" * 50)
