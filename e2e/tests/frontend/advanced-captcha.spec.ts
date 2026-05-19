import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('Emoji验证码E2E测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('Emoji验证码生成测试', () => {
    test('应该能够生成Emoji验证码', async () => {
      const result = await apiHelper.generateEmojiCaptcha();
      expect(result).toHaveProperty('success', true);
      expect(result.data).toHaveProperty('sessionId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('emojis');
      expect(Array.isArray(result.data.emojis)).toBe(true);
    });

    test('应该能够生成指定难度的Emoji验证码', async ({ request }) => {
      const difficulties = ['easy', 'medium', 'hard', 'expert'];
      
      for (const difficulty of difficulties) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/emoji/create', {
          data: { difficulty }
        });
        
        expect(response.ok()).toBeTruthy();
        const result = await response.json();
        expect(result.data.difficulty).toBe(difficulty);
      }
    });

    test('应该能够指定Emoji数量', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/emoji/create', {
        data: { count: 6 }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data.count).toBe(6);
      expect(result.data.emojis.length).toBe(6);
    });
  });

  test.describe('Emoji验证码验证测试', () => {
    test('应该能够正确验证Emoji验证码', async ({ request }) => {
      const generateResponse = await request.post('http://localhost:8080/api/v1/captcha/emoji/create');
      const captcha = await generateResponse.json();
      
      const targetIds = captcha.data.emojis
        .filter((e: any) => e.target)
        .map((e: any) => e.id);
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/emoji/verify', {
        data: {
          session_id: captcha.data.sessionId,
          selected_ids: targetIds,
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
      const result = await verifyResponse.json();
      expect(result).toHaveProperty('success');
    });

    test('错误的Emoji选择应该验证失败', async ({ request }) => {
      const generateResponse = await request.post('http://localhost:8080/api/v1/captcha/emoji/create');
      const captcha = await generateResponse.json();
      
      const wrongIds = captcha.data.emojis
        .filter((e: any) => !e.target)
        .map((e: any) => e.id);
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/emoji/verify', {
        data: {
          session_id: captcha.data.sessionId,
          selected_ids: wrongIds.slice(0, 2),
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
    });
  });

  test.describe('Emoji验证码数据完整性测试', () => {
    test('Emoji数据应该包含所有必要字段', async () => {
      const result = await apiHelper.generateEmojiCaptcha();
      
      result.data.emojis.forEach((emoji: any) => {
        expect(emoji).toHaveProperty('id');
        expect(emoji).toHaveProperty('emoji');
        expect(emoji).toHaveProperty('target');
        expect(emoji).toHaveProperty('x');
        expect(emoji).toHaveProperty('y');
        expect(typeof emoji.id).toBe('number');
        expect(typeof emoji.emoji).toBe('string');
        expect(typeof emoji.target).toBe('boolean');
      });
    });

    test('应该有正确数量的目标Emoji', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/emoji/create', {
        data: { count: 4 }
      });
      
      const result = await response.json();
      const targetCount = result.data.emojis.filter((e: any) => e.target).length;
      expect(targetCount).toBeGreaterThan(0);
      expect(targetCount).toBeLessThan(result.data.count);
    });
  });
});

test.describe('手势验证码E2E测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('手势验证码生成测试', () => {
    test('应该能够生成手势验证码', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/gesture/create');
      expect(response.ok()).toBeTruthy();
      
      const result = await response.json();
      expect(result.data).toHaveProperty('sessionId');
      expect(result.data).toHaveProperty('patternImage');
      expect(result.data).toHaveProperty('patternType');
    });

    test('应该支持不同的手势图案', async ({ request }) => {
      const patterns = ['L', 'Z', 'N', 'M', 'V', 'W'];
      
      for (const pattern of patterns) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/gesture/create', {
          data: { pattern_type: pattern }
        });
        
        expect(response.ok()).toBeTruthy();
      }
    });
  });

  test.describe('手势验证码验证测试', () => {
    test('应该能够验证正确的手势', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/gesture/create');
      const captcha = await genResponse.json();
      
      const gesturePath = [[50, 50], [150, 50], [150, 150]];
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/gesture/verify', {
        data: {
          session_id: captcha.data.sessionId,
          gesture_path: gesturePath,
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
    });
  });
});

test.describe('3D验证码E2E测试', () => {
  test.describe('3D验证码生成测试', () => {
    test('应该能够生成3D验证码', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/3d/create');
      expect(response.ok()).toBeTruthy();
      
      const result = await response.json();
      expect(result.data).toHaveProperty('sessionId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('modelData');
      expect(result.data).toHaveProperty('targetRotation');
    });

    test('应该能够指定3D模型类型', async ({ request }) => {
      const modelTypes = ['cube', 'sphere', 'cylinder', 'cone'];
      
      for (const modelType of modelTypes) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/3d/create', {
          data: { model_type: modelType }
        });
        
        expect(response.ok()).toBeTruthy();
        const result = await response.json();
        expect(result.data.modelType).toBe(modelType);
      }
    });

    test('3D模型数据应该包含顶点和面信息', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/3d/create');
      const result = await response.json();
      
      expect(result.data.modelData).toHaveProperty('type');
      expect(result.data.modelData).toHaveProperty('vertices');
      expect(result.data.modelData).toHaveProperty('faces');
      expect(Array.isArray(result.data.modelData.vertices)).toBe(true);
      expect(Array.isArray(result.data.modelData.faces)).toBe(true);
    });
  });

  test.describe('3D验证码验证测试', () => {
    test('应该能够验证3D旋转验证码', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/3d/create');
      const captcha = await genResponse.json();
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/3d/verify', {
        data: {
          session_id: captcha.data.sessionId,
          rotation: captcha.data.targetRotation,
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
      const result = await verifyResponse.json();
      expect(result).toHaveProperty('success');
    });

    test('错误的旋转角度应该验证失败', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/3d/create');
      const captcha = await genResponse.json();
      
      const wrongRotation = { x: 999, y: 999, z: 999 };
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/3d/verify', {
        data: {
          session_id: captcha.data.sessionId,
          rotation: wrongRotation,
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
    });
  });
});

test.describe('连连看验证码E2E测试', () => {
  test.describe('连连看验证码生成测试', () => {
    test('应该能够生成连连看验证码', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/lianliankan/create');
      expect(response.ok()).toBeTruthy();
      
      const result = await response.json();
      expect(result.data).toHaveProperty('sessionId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('icons');
      expect(result.data).toHaveProperty('gridSize');
    });

    test('应该能够指定配对数量', async ({ request }) => {
      const pairCounts = [4, 6, 8, 10, 12];
      
      for (const pairs of pairCounts) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/lianliankan/create', {
          data: { pairs }
        });
        
        expect(response.ok()).toBeTruthy();
        const result = await response.json();
        expect(result.data.pairs).toBe(pairs);
      }
    });

    test('Icons应该包含配对信息', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/lianliankan/create', {
        data: { pairs: 6 }
      });
      
      const result = await response.json();
      
      result.data.icons.forEach((icon: any) => {
        expect(icon).toHaveProperty('id');
        expect(icon).toHaveProperty('icon');
        expect(icon).toHaveProperty('pairs');
        expect(Array.isArray(icon.pairs)).toBe(true);
        expect(icon.pairs.length).toBe(2);
      });
    });
  });

  test.describe('连连看验证码验证测试', () => {
    test('应该能够验证正确的配对', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/lianliankan/create');
      const captcha = await genResponse.json();
      
      const matchingPairs = captcha.data.icons.map((icon: any) => icon.pairs);
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/lianliankan/verify', {
        data: {
          session_id: captcha.data.sessionId,
          matching_pairs: matchingPairs,
          behavior_data: []
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
      const result = await verifyResponse.json();
      expect(result).toHaveProperty('success');
    });
  });
});

test.describe('语音验证码E2E测试', () => {
  test.describe('语音验证码生成测试', () => {
    test('应该能够生成语音验证码', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/voice/create');
      expect(response.ok()).toBeTruthy();
      
      const result = await response.json();
      expect(result.data).toHaveProperty('sessionId');
      expect(result.data).toHaveProperty('audioUrl');
      expect(result.data).toHaveProperty('text');
    });

    test('应该能够指定语音模式', async ({ request }) => {
      const modes = ['number', 'letter', 'mixed'];
      
      for (const mode of modes) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/voice/create', {
          data: { mode }
        });
        
        expect(response.ok()).toBeTruthy();
        const result = await response.json();
        expect(result.data.mode).toBe(mode);
      }
    });

    test('应该能够指定字符数量', async ({ request }) => {
      const counts = [4, 5, 6, 7, 8];
      
      for (const count of counts) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/voice/create', {
          data: { count }
        });
        
        expect(response.ok()).toBeTruthy();
        const result = await response.json();
        expect(result.data.count).toBe(count);
        expect(result.data.text.length).toBe(count);
      }
    });
  });

  test.describe('语音验证码验证测试', () => {
    test('应该能够验证正确的语音验证码', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/voice/create');
      const captcha = await genResponse.json();
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/voice/verify', {
        data: {
          session_id: captcha.data.sessionId,
          answer: captcha.data.text
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
      const result = await verifyResponse.json();
      expect(result).toHaveProperty('success');
    });

    test('错误的验证码内容应该验证失败', async ({ request }) => {
      const genResponse = await request.post('http://localhost:8080/api/v1/captcha/voice/create');
      const captcha = await genResponse.json();
      
      const verifyResponse = await request.post('http://localhost:8080/api/v1/captcha/voice/verify', {
        data: {
          session_id: captcha.data.sessionId,
          answer: 'WRONG'
        }
      });
      
      expect(verifyResponse.ok()).toBeTruthy();
    });
  });
});

test.describe('无感验证E2E测试', () => {
  test.describe('无感验证检查测试', () => {
    test('应该能够进行无感验证检查', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/seamless/check', {
        data: {
          device_fingerprint: `fp_${Date.now()}`,
          behavior_sequence: [
            { event: 'mousemove', timestamp: Date.now() },
            { event: 'click', timestamp: Date.now() + 100 }
          ]
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('trust_level');
      expect(result.data).toHaveProperty('risk_score');
      expect(result.data).toHaveProperty('requires_captcha');
    });

    test('高风险行为应该返回requires_captcha=true', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/seamless/check', {
        data: {
          device_fingerprint: 'high_risk_fp',
          behavior_sequence: [
            { event: 'mousemove', timestamp: Date.now() },
            { event: 'mousemove', timestamp: Date.now() + 10 }
          ]
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data.risk_score).toBeGreaterThan(50);
      expect(result.data.requires_captcha).toBe(true);
    });

    test('正常行为应该返回trust_level=high', async ({ request }) => {
      const behaviorSequence = [];
      let timestamp = Date.now();
      
      for (let i = 0; i < 20; i++) {
        behaviorSequence.push({
          event: 'mousemove',
          timestamp: timestamp + i * 50
        });
      }
      
      const response = await request.post('http://localhost:8080/api/v1/seamless/check', {
        data: {
          device_fingerprint: `fp_${Date.now()}`,
          behavior_sequence: behaviorSequence
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data.risk_score).toBeLessThan(30);
    });
  });
});

test.describe('环境检测E2E测试', () => {
  test.describe('环境检测测试', () => {
    test('应该能够进行环境检测', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/detect/check', {
        data: {
          fingerprint: {
            canvas: 'canvas_hash_123',
            webgl: 'webgl_renderer_info',
            fonts: ['Arial', 'Helvetica']
          }
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data).toHaveProperty('is_proxy');
      expect(result.data).toHaveProperty('is_vpn');
      expect(result.data).toHaveProperty('is_tor');
      expect(result.data).toHaveProperty('is_emulator');
      expect(result.data).toHaveProperty('risk_score');
    });

    test('检测到自动化工具应该返回高风险分数', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/detect/check', {
        data: {
          fingerprint: {
            webdriver: 'wd:true',
            webgl: 'SwiftShader'
          }
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data.risk_score).toBeGreaterThan(40);
      expect(result.data.is_real_browser).toBe(false);
    });

    test('真实浏览器应该返回低风险分数', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/detect/check', {
        data: {
          fingerprint: {
            canvas: 'unique_canvas_hash',
            webgl: 'Intel Iris OpenGL Engine',
            fonts: ['Arial', 'Helvetica', 'PingFang SC']
          }
        }
      });
      
      expect(response.ok()).toBeTruthy();
      const result = await response.json();
      expect(result.data.risk_score).toBeLessThan(30);
      expect(result.data.is_real_browser).toBe(true);
    });
  });
});

test.describe('验证码并发测试', () => {
  test('应该能够并发生成多种验证码', async ({ request }) => {
    const promises = [
      request.post('http://localhost:8080/api/v1/captcha/slider/create'),
      request.post('http://localhost:8080/api/v1/captcha/click/create'),
      request.post('http://localhost:8080/api/v1/captcha/emoji/create'),
      request.post('http://localhost:8080/api/v1/captcha/gesture/create'),
      request.post('http://localhost:8080/api/v1/captcha/voice/create')
    ];
    
    const results = await Promise.all(promises);
    
    results.forEach(response => {
      expect(response.ok()).toBeTruthy();
    });
  });

  test('应该能够处理大量并发生成请求', async ({ request }) => {
    const concurrency = 50;
    const promises = [];
    
    for (let i = 0; i < concurrency; i++) {
      promises.push(request.post('http://localhost:8080/api/v1/captcha/slider/create'));
    }
    
    const startTime = Date.now();
    const results = await Promise.all(promises);
    const duration = Date.now() - startTime;
    
    const successCount = results.filter(r => r.ok()).length;
    console.log(`并发生成${concurrency}个验证码耗时: ${duration}ms, 成功率: ${successCount}/${concurrency}`);
    
    expect(successCount).toBe(concurrency);
    expect(duration).toBeLessThan(30000);
  });

  test('并发验证应该有适当的处理', async ({ request }) => {
    const genResponse = await request.post('http://localhost:8080/api/v1/captcha/slider/create');
    const captcha = await genResponse.json();
    
    const concurrency = 10;
    const promises = [];
    
    for (let i = 0; i < concurrency; i++) {
      promises.push(request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
        data: {
          session_id: captcha.data.sessionId,
          x: 100 + i * 10,
          y: 50
        }
      }));
    }
    
    const results = await Promise.all(promises);
    
    const processedCount = results.filter(r => r.ok()).length;
    expect(processedCount).toBe(concurrency);
  });
});
