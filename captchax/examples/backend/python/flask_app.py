#!/usr/bin/env python3
"""
CaptchaX Flask 后端集成示例
"""

from flask import Flask, request, jsonify
from functools import wraps
import requests
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)

CAPTCHA_CONFIG = {
    'app_id': 'example-app',
    'server_url': 'http://localhost:8080',
    'timeout': 10,
}


class CaptchaXClient:
    def __init__(self, app_id: str, server_url: str, timeout: int = 10):
        self.app_id = app_id
        self.server_url = server_url.rstrip('/')
        self.timeout = timeout
        self.session = requests.Session()

    def create_slider_captcha(self, width: int = 200, height: int = 80, client_info: str = None) -> dict:
        """创建滑块验证码"""
        url = f'{self.server_url}/api/v1/captcha/slider'
        payload = {
            'app_id': self.app_id,
            'width': width,
            'height': height,
        }
        if client_info:
            payload['client_info'] = client_info

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '创建验证码失败'))

        return data.get('data', {})

    def create_click_captcha(self, char_count: int = 4, client_info: str = None) -> dict:
        """创建点选验证码"""
        url = f'{self.server_url}/api/v1/captcha/click'
        payload = {
            'app_id': self.app_id,
            'char_count': char_count,
        }
        if client_info:
            payload['client_info'] = client_info

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '创建验证码失败'))

        return data.get('data', {})

    def create_puzzle_captcha(self, width: int = 300, height: int = 150, client_info: str = None) -> dict:
        """创建拼图验证码"""
        url = f'{self.server_url}/api/v1/captcha/puzzle'
        payload = {
            'app_id': self.app_id,
            'width': width,
            'height': height,
        }
        if client_info:
            payload['client_info'] = client_info

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '创建验证码失败'))

        return data.get('data', {})

    def verify_slider(self, captcha_id: str, target_x: int, target_y: int = 0) -> dict:
        """验证滑块验证码"""
        url = f'{self.server_url}/api/v1/captcha/slider/verify'
        payload = {
            'captcha_id': captcha_id,
            'target_x': target_x,
            'target_y': target_y,
        }

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '验证失败'))

        return data.get('data', {})

    def verify_click(self, captcha_id: str, clicks: list) -> dict:
        """验证点选验证码"""
        url = f'{self.server_url}/api/v1/captcha/click/verify'
        payload = {
            'captcha_id': captcha_id,
            'clicks': clicks,
        }

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '验证失败'))

        return data.get('data', {})

    def verify_puzzle(self, captcha_id: str, target_x: int, target_y: int = 0) -> dict:
        """验证拼图验证码"""
        url = f'{self.server_url}/api/v1/captcha/puzzle/verify'
        payload = {
            'captcha_id': captcha_id,
            'target_x': target_x,
            'target_y': target_y,
        }

        response = self.session.post(url, json=payload, timeout=self.timeout)
        response.raise_for_status()
        data = response.json()

        if data.get('code') != 200:
            raise Exception(data.get('message', '验证失败'))

        return data.get('data', {})

    def verify_token(self, token: str) -> bool:
        """验证 token（通过验证接口间接验证）"""
        try:
            parts = token.split('.')
            if len(parts) != 3:
                return False
            return True
        except Exception:
            return False


captcha_client = CaptchaXClient(
    app_id=CAPTCHA_CONFIG['app_id'],
    server_url=CAPTCHA_CONFIG['server_url'],
    timeout=CAPTCHA_CONFIG['timeout'],
)


def require_captcha(f):
    """验证码验证装饰器"""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        token = request.headers.get('X-Captcha-Token')

        if not token:
            return jsonify({
                'success': False,
                'message': '验证码 Token 不能为空',
                'code': 400,
            }), 400

        try:
            valid = captcha_client.verify_token(token)
            if not valid:
                return jsonify({
                    'success': False,
                    'message': '验证码验证失败',
                    'code': 400,
                }), 400
        except Exception as e:
            logger.error(f'验证码验证异常: {e}')
            return jsonify({
                'success': False,
                'message': '验证码服务错误',
                'code': 500,
            }), 500

        return f(*args, **kwargs)

    return decorated_function


@app.route('/health', methods=['GET'])
def health():
    """健康检查"""
    return jsonify({
        'status': 'healthy',
        'service': 'captcha-example-backend',
    })


@app.route('/api/captcha/create', methods=['POST'])
def create_captcha():
    """创建验证码"""
    data = request.get_json() or {}
    captcha_type = data.get('type', 'slider')

    try:
        if captcha_type == 'slider':
            result = captcha_client.create_slider_captcha(
                width=data.get('width', 200),
                height=data.get('height', 80),
            )
        elif captcha_type == 'click':
            result = captcha_client.create_click_captcha(
                char_count=data.get('char_count', 4),
            )
        elif captcha_type == 'puzzle':
            result = captcha_client.create_puzzle_captcha(
                width=data.get('width', 300),
                height=data.get('height', 150),
            )
        else:
            return jsonify({
                'success': False,
                'message': f'不支持的验证码类型: {captcha_type}',
            }), 400

        return jsonify({
            'success': True,
            'data': result,
        })

    except Exception as e:
        logger.error(f'创建验证码失败: {e}')
        return jsonify({
            'success': False,
            'message': str(e),
        }), 500


@app.route('/api/captcha/verify', methods=['POST'])
def verify_captcha():
    """验证验证码"""
    data = request.get_json()

    if not data:
        return jsonify({
            'success': False,
            'message': '无效的请求',
        }), 400

    captcha_type = data.get('type', 'slider')
    captcha_id = data.get('captcha_id')

    if not captcha_id:
        return jsonify({
            'success': False,
            'message': 'captcha_id 不能为空',
        }), 400

    try:
        if captcha_type == 'slider':
            result = captcha_client.verify_slider(
                captcha_id=captcha_id,
                target_x=data.get('target_x', 0),
                target_y=data.get('target_y', 0),
            )
        elif captcha_type == 'click':
            clicks = data.get('clicks', [])
            if not clicks:
                return jsonify({
                    'success': False,
                    'message': 'clicks 不能为空',
                }), 400
            result = captcha_client.verify_click(
                captcha_id=captcha_id,
                clicks=clicks,
            )
        elif captcha_type == 'puzzle':
            result = captcha_client.verify_puzzle(
                captcha_id=captcha_id,
                target_x=data.get('target_x', 0),
                target_y=data.get('target_y', 0),
            )
        else:
            return jsonify({
                'success': False,
                'message': f'不支持的验证码类型: {captcha_type}',
            }), 400

        return jsonify({
            'success': True,
            'data': result,
        })

    except Exception as e:
        logger.error(f'验证失败: {e}')
        return jsonify({
            'success': False,
            'message': str(e),
        }), 500


@app.route('/api/login', methods=['POST'])
@require_captcha
def login():
    """登录接口"""
    data = request.get_json()

    if not data:
        return jsonify({
            'success': False,
            'message': '无效的请求',
        }), 400

    username = data.get('username')
    password = data.get('password')

    if not username or not password:
        return jsonify({
            'success': False,
            'message': '用户名和密码不能为空',
        }), 400

    if username == 'admin' and password == 'admin123':
        return jsonify({
            'success': True,
            'message': '登录成功',
            'token': 'mock-jwt-token',
        })

    return jsonify({
        'success': False,
        'message': '用户名或密码错误',
    }), 401


@app.route('/api/register', methods=['POST'])
@require_captcha
def register():
    """注册接口"""
    data = request.get_json()

    if not data:
        return jsonify({
            'success': False,
            'message': '无效的请求',
        }), 400

    username = data.get('username')
    password = data.get('password')
    email = data.get('email')

    if not all([username, password, email]):
        return jsonify({
            'success': False,
            'message': '缺少必填字段',
        }), 400

    return jsonify({
        'success': True,
        'message': '注册成功',
    })


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8081, debug=True)
