#!/usr/bin/env python3
"""
行为验证系统 Python SDK 使用示例

本文件包含了 SDK 的各种使用示例。
"""

import sys
import os

# 添加当前目录到路径，方便直接运行
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
)


def example_slider_captcha():
    """滑块验证码使用示例"""
    print("\n" + "="*50)
    print("滑块验证码示例")
    print("="*50)
    
    try:
        # 创建客户端
        client = CaptchaClient(
            base_url="http://localhost:8080",
            timeout=30,
            max_retries=3,
        )
        
        # 获取验证码
        print("1. 获取验证码...")
        captcha = client.get_slider_captcha(
            width=320,
            height=160,
            tolerance=8,
        )
        print(f"   会话ID: {captcha.session_id}")
        print(f"   图片URL: {captcha.image_url[:80]}...")
        print(f"   谜题URL: {captcha.puzzle_url[:80]}...")
        
        # 模拟用户滑动轨迹
        print("\n2. 生成模拟轨迹...")
        secret_x = 150  # 实际应该通过前端获取
        trajectory = [
            TrajectoryPoint(x=0, y=captcha.secret_y or 80, t=0),
            TrajectoryPoint(x=50, y=captcha.secret_y or 80 + 5, t=200),
            TrajectoryPoint(x=100, y=captcha.secret_y or 80 - 3, t=400),
            TrajectoryPoint(x=150, y=captcha.secret_y or 80 + 2, t=600),
            TrajectoryPoint(x=secret_x, y=captcha.secret_y or 80, t=800),
        ]
        
        # 验证
        print("\n3. 验证验证码...")
        result = client.verify_slider_captcha(
            session_id=captcha.session_id,
            x=secret_x,
            y=captcha.secret_y,
            trajectory=trajectory,
        )
        
        print(f"   成功: {result.success}")
        print(f"   消息: {result.message}")
        if result.remaining_attempts is not None:
            print(f"   剩余尝试次数: {result.remaining_attempts}")
        if result.trajectory_result:
            print(f"   轨迹分析: {result.trajectory_result}")
        
        client.close()
        
    except CaptchaNetworkError as e:
        print(f"   网络错误: {e}")
    except CaptchaAPIError as e:
        print(f"   API错误: {e} (code: {e.code})")
    except CaptchaError as e:
        print(f"   错误: {e}")
    except Exception as e:
        print(f"   未知错误: {e}")


def example_click_captcha():
    """点击验证码使用示例"""
    print("\n" + "="*50)
    print("点击验证码示例")
    print("="*50)
    
    try:
        # 使用上下文管理器
        with CaptchaClient(base_url="http://localhost:8080") as client:
            # 获取验证码
            print("1. 获取验证码...")
            captcha = client.get_click_captcha(
                mode=ClickMode.NUMBER,
                max_points=3,
                allow_shuffle=True,
            )
            print(f"   会话ID: {captcha.session_id}")
            print(f"   提示: {captcha.hint}")
            print(f"   提示顺序: {captcha.hint_order}")
            
            # 模拟点击点
            print("\n2. 准备点击数据...")
            points = [[100, 100], [200, 100], [150, 200]]  # 实际应该从用户输入获取
            
            # 验证
            print("\n3. 验证验证码...")
            result = client.verify_click_captcha(
                session_id=captcha.session_id,
                points=points,
                click_sequence=[0, 1, 2],
            )
            
            print(f"   成功: {result.success}")
            print(f"   消息: {result.message}")
            if result.risk_score is not None:
                print(f"   风险评分: {result.risk_score}")
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_image_captcha():
    """图形验证码使用示例"""
    print("\n" + "="*50)
    print("图形验证码示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        
        print("1. 获取验证码...")
        captcha = client.get_image_captcha(
            type_="mixed",
            count=4,
        )
        print(f"   挑战ID: {captcha.challenge_id}")
        print(f"   图片: {captcha.image[:80]}...")
        
        print("\n2. 输入验证码...")
        answer = "ABCD"  # 实际应该从用户输入获取
        
        print("\n3. 验证...")
        result = client.verify_image_captcha(
            challenge_id=captcha.challenge_id,
            answer=answer,
        )
        
        print(f"   成功: {result.success}")
        print(f"   消息: {result.message}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_gesture_captcha():
    """手势验证码使用示例"""
    print("\n" + "="*50)
    print("手势验证码示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        
        print("1. 获取验证码...")
        captcha = client.get_gesture_captcha()
        print(f"   会话ID: {captcha.session_id}")
        if captcha.hint:
            print(f"   提示: {captcha.hint}")
        
        print("\n2. 准备手势模式...")
        pattern = [1, 2, 3, 5, 7]  # 实际应该从用户输入获取
        
        print("\n3. 验证...")
        result = client.verify_gesture_captcha(
            session_id=captcha.session_id,
            pattern=pattern,
        )
        
        print(f"   成功: {result.success}")
        print(f"   消息: {result.message}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_jigsaw_captcha():
    """拼图验证码使用示例"""
    print("\n" + "="*50)
    print("拼图验证码示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        
        print("1. 获取验证码...")
        captcha = client.get_jigsaw_captcha(
            width=300,
            height=300,
            grid_size=3,
        )
        print(f"   会话ID: {captcha.session_id}")
        print(f"   网格大小: {captcha.grid_size}x{captcha.grid_size}")
        print(f"   碎片数量: {len(captcha.pieces)}")
        
        print("\n2. 准备答案...")
        # 实际应用中需要用户移动拼图到正确位置
        pieces = []
        for piece in captcha.pieces:
            # 模拟用户将碎片放到正确位置
            correct_piece = JigsawPiece(
                index=piece.index,
                original_x=piece.original_x,
                original_y=piece.original_y,
                current_x=piece.original_x,
                current_y=piece.original_y,
                width=piece.width,
                height=piece.height,
                rotation=0,
            )
            pieces.append(correct_piece)
        
        print("\n3. 验证...")
        result = client.verify_jigsaw_captcha(
            session_id=captcha.session_id,
            pieces=pieces,
        )
        
        print(f"   成功: {result.success}")
        print(f"   消息: {result.message}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_generic_verify():
    """通用验证方法示例"""
    print("\n" + "="*50)
    print("通用验证方法示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        
        # 使用通用方法
        print("1. 获取滑块验证码...")
        captcha = client.get_slider_captcha()
        
        print("\n2. 使用通用方法验证...")
        result = client.verify_captcha(
            captcha_type=CaptchaType.SLIDER,
            session_id=captcha.session_id,
            x=150,
            y=captcha.secret_y,
        )
        
        print(f"   成功: {result.success}")
        print(f"   消息: {result.message}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_error_handling():
    """错误处理示例"""
    print("\n" + "="*50)
    print("错误处理示例")
    print("="*50)
    
    try:
        # 连接到不存在的服务来演示错误
        client = CaptchaClient(
            base_url="http://nonexistent.example.com",
            timeout=5,
        )
        
        print("尝试连接到不存在的服务...")
        captcha = client.get_slider_captcha()
        print(f"会话ID: {captcha.session_id}")
        
    except CaptchaTimeoutError as e:
        print(f"超时错误: {e}")
    except CaptchaNetworkError as e:
        print(f"网络错误: {e}")
    except CaptchaAPIError as e:
        print(f"API错误: {e}, 代码: {e.code}")
    except CaptchaSessionExpiredError as e:
        print(f"会话过期: {e}")
    except CaptchaValidationError as e:
        print(f"验证错误: {e}")
    except CaptchaError as e:
        print(f"验证码错误: {e}")


def example_user_auth():
    """用户认证示例"""
    print("\n" + "="*50)
    print("用户认证示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        auth = client.auth()
        
        print("1. 登录...")
        login_result = auth.login(
            username="testuser",
            password="password123",
        )
        print(f"   登录成功!")
        print(f"   访问令牌: {login_result.access_token[:50]}...")
        print(f"   过期时间: {login_result.expires_in}秒")
        
        print("\n2. 刷新令牌...")
        refresh_result = auth.refresh_token()
        print(f"   刷新成功!")
        
        print("\n3. 登出...")
        auth.logout()
        print("   登出成功!")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_environment():
    """环境检测示例"""
    print("\n" + "="*50)
    print("环境检测示例")
    print("="*50)
    
    try:
        client = CaptchaClient(base_url="http://localhost:8080")
        env = client.env()
        
        print("1. 获取检测脚本...")
        script = env.get_detection_script(callback="onDetectReady")
        print(f"   脚本长度: {len(script)}字符")
        
        print("\n2. 提交检测数据...")
        detection_data = {
            "detection_id": "test123",
            "risk_score": 0.1,
            "fingerprint": "user-fingerprint-hash",
            "timestamp": 1234567890,
        }
        result = env.submit_detection(detection_data)
        print(f"   结果: {result}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"   错误: {e}")


def example_connection_pool():
    """连接池配置示例"""
    print("\n" + "="*50)
    print("连接池配置示例")
    print("="*50)
    
    try:
        # 自定义连接池配置
        client = CaptchaClient(
            base_url="http://localhost:8080",
            timeout=30,
            max_retries=5,
            retry_backoff_factor=0.3,
            pool_connections=20,
            pool_maxsize=20,
        )
        print("客户端已创建，使用自定义连接池配置")
        
        # 测试连接
        captcha = client.get_slider_captcha()
        print(f"验证码获取成功: {captcha.session_id}")
        
        client.close()
        
    except CaptchaError as e:
        print(f"错误: {e}")


def main():
    """主函数，运行所有示例"""
    print("="*50)
    print("行为验证系统 Python SDK 示例")
    print("="*50)
    
    # 运行示例
    examples = [
        ("滑块验证码", example_slider_captcha),
        ("点击验证码", example_click_captcha),
        ("图形验证码", example_image_captcha),
        ("手势验证码", example_gesture_captcha),
        ("拼图验证码", example_jigsaw_captcha),
        ("通用验证方法", example_generic_verify),
        ("错误处理", example_error_handling),
        ("用户认证", example_user_auth),
        ("环境检测", example_environment),
        ("连接池配置", example_connection_pool),
    ]
    
    # 显示菜单
    print("\n可用示例:")
    for i, (name, _) in enumerate(examples, 1):
        print(f"  {i}. {name}")
    print("  0. 运行全部")
    
    choice = input("\n请选择要运行的示例 (0-{}): ".format(len(examples))).strip()
    
    if choice == "0":
        # 运行全部示例
        for name, func in examples:
            try:
                func()
            except Exception as e:
                print(f"示例 '{name}' 执行失败: {e}")
    else:
        # 运行指定示例
        try:
            index = int(choice) - 1
            if 0 <= index < len(examples):
                examples[index][1]()
            else:
                print("无效的选择")
        except ValueError:
            print("无效的输入")
    
    print("\n" + "="*50)
    print("示例运行完成")
    print("="*50)


if __name__ == "__main__":
    main()
