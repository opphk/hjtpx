"""
Flask集成示例

展示如何在Flask项目中使用Python SDK进行验证码验证
"""

from flask import Flask, request, jsonify
from functools import wraps

from captcha import CaptchaClient, CaptchaError, CaptchaValidationError, CaptchaTimeoutError

app = Flask(__name__)

CAPTCHA_CONFIG = {
    'base_url': 'http://localhost:8080',
    'api_key': 'your-api-key',
    'timeout': 30,
}


def get_captcha_client():
    """获取验证码客户端实例"""
    return CaptchaClient(
        base_url=CAPTCHA_CONFIG['base_url'],
        api_key=CAPTCHA_CONFIG['api_key'],
        timeout=CAPTCHA_CONFIG['timeout'],
    )


class CaptchaService:
    """验证码服务类"""

    @staticmethod
    def get_slider(width=320, height=160, tolerance=8):
        """获取滑块验证码"""
        with get_captcha_client() as client:
            return client.get_slider_captcha(
                width=width,
                height=height,
                tolerance=tolerance
            )

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
    def get_click(mode='number', points=3, shuffle=True):
        """获取点击验证码"""
        with get_captcha_client() as client:
            return client.get_click_captcha(
                mode=mode,
                max_points=points,
                allow_shuffle=shuffle,
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
            return client.get_image_captcha(
                type_=type_,
                count=count,
            )

    @staticmethod
    def verify_image(challenge_id, answer):
        """验证图形验证码"""
        with get_captcha_client() as client:
            return client.verify_image_captcha(
                challenge_id=challenge_id,
                answer=answer,
            )

    @staticmethod
    def user_login(username, password, captcha_token=None):
        """用户登录"""
        with get_captcha_client() as client:
            auth = client.auth()
            return auth.login(
                username=username,
                password=password,
                captcha_token=captcha_token,
            )


@app.route('/api/captcha/slider', methods=['GET'])
def get_slider_captcha():
    """获取滑块验证码"""
    try:
        width = request.args.get('width', 320, type=int)
        height = request.args.get('height', 160, type=int)
        tolerance = request.args.get('tolerance', 8, type=int)

        captcha = CaptchaService.get_slider(width, height, tolerance)

        return jsonify({
            'success': True,
            'session_id': captcha.session_id,
            'image_url': captcha.image_url,
            'puzzle_url': captcha.puzzle_url,
            'secret_y': captcha.secret_y,
            'hint_url': captcha.hint_url,
        })

    except CaptchaTimeoutError:
        return jsonify({
            'success': False,
            'error': 'Request timeout'
        }), 504

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/captcha/slider/verify', methods=['POST'])
def verify_slider_captcha():
    """验证滑块验证码"""
    try:
        data = request.get_json()
        session_id = data.get('session_id')
        x = data.get('x')
        y = data.get('y')
        trajectory = data.get('trajectory')

        if not session_id or x is None:
            return jsonify({
                'success': False,
                'error': 'Missing required parameters'
            }), 400

        result = CaptchaService.verify_slider(session_id, x, y, trajectory)

        return jsonify({
            'success': result.success,
            'message': result.message,
            'risk_score': result.risk_score,
            'captcha_pass': result.captcha_pass,
        })

    except CaptchaValidationError:
        return jsonify({
            'success': False,
            'error': 'Invalid request'
        }), 400

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/captcha/click', methods=['GET'])
def get_click_captcha():
    """获取点击验证码"""
    try:
        mode = request.args.get('mode', 'number')
        points = request.args.get('points', 3, type=int)

        captcha = CaptchaService.get_click(mode, points)

        return jsonify({
            'success': True,
            'session_id': captcha.session_id,
            'image_url': captcha.image_url,
            'hint': captcha.hint,
            'hint_order': captcha.hint_order,
            'max_points': captcha.max_points,
            'mode': captcha.mode,
        })

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/captcha/click/verify', methods=['POST'])
def verify_click_captcha():
    """验证点击验证码"""
    try:
        data = request.get_json()
        session_id = data.get('session_id')
        points = data.get('points')
        click_sequence = data.get('click_sequence')

        if not session_id or not points:
            return jsonify({
                'success': False,
                'error': 'Missing required parameters'
            }), 400

        result = CaptchaService.verify_click(session_id, points, click_sequence)

        return jsonify({
            'success': result.success,
            'message': result.message,
            'risk_score': result.risk_score,
        })

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/captcha/image', methods=['GET'])
def get_image_captcha():
    """获取图形验证码"""
    try:
        type_ = request.args.get('type', 'mixed')
        count = request.args.get('count', 4, type=int)

        captcha = CaptchaService.get_image(type_, count)

        return jsonify({
            'success': True,
            'challenge_id': captcha.challenge_id,
            'image': captcha.image,
        })

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/captcha/image/verify', methods=['POST'])
def verify_image_captcha():
    """验证图形验证码"""
    try:
        data = request.get_json()
        challenge_id = data.get('challenge_id')
        answer = data.get('answer')

        if not challenge_id or not answer:
            return jsonify({
                'success': False,
                'error': 'Missing required parameters'
            }), 400

        result = CaptchaService.verify_image(challenge_id, answer)

        return jsonify({
            'success': result.success,
            'message': result.message,
        })

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/auth/login', methods=['POST'])
def login():
    """用户登录"""
    try:
        data = request.get_json()
        username = data.get('username')
        password = data.get('password')
        captcha_token = data.get('captcha_token')

        if not username or not password:
            return jsonify({
                'success': False,
                'error': 'Missing credentials'
            }), 400

        result = CaptchaService.user_login(username, password, captcha_token)

        return jsonify({
            'success': True,
            'access_token': result.access_token,
            'refresh_token': result.refresh_token,
            'expires_in': result.expires_in,
        })

    except CaptchaError as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 401


def require_captcha_verification(f):
    """验证码验证装饰器"""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        captcha_token = request.headers.get('X-Captcha-Token')

        if not captcha_token:
            return jsonify({
                'success': False,
                'error': 'Captcha verification required'
            }), 403

        return f(*args, **kwargs)

    return decorated_function


@app.route('/api/protected', methods=['POST'])
@require_captcha_verification
def protected_endpoint():
    """需要验证码验证的保护端点"""
    return jsonify({
        'success': True,
        'message': 'Access granted',
    })


if __name__ == '__main__':
    print("Flask集成示例")
    print("-" * 50)
    print("运行Flask应用: flask run")
    print("-" * 50)

    app.run(debug=True, host='0.0.0.0', port=5000)
