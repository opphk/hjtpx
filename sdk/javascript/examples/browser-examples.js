/**
 * 行为验证系统 JavaScript SDK - 浏览器端完整示例
 *
 * 本文件包含多种集成模式和使用示例
 */

(function() {
  'use strict';

  const CaptchaBrowserExamples = {

    /**
     * 模式1: 直接API调用（适用于简单的验证码验证）
     */
    directApiExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      client.getSliderCaptcha({ width: 320, height: 160 })
        .then(function(captcha) {
          console.log('获取验证码成功:', captcha);

          container.innerHTML = `
            <div class="captcha-wrapper">
              <img src="${captcha.image_url}" alt="验证码背景" class="captcha-bg" />
              <p>Session ID: ${captcha.session_id}</p>
              <button onclick="CaptchaBrowserExamples.verifyDirect('${captcha.session_id}', ${captcha.secret_y || 0})">
                模拟验证
              </button>
            </div>
          `;
        })
        .catch(function(error) {
          console.error('获取验证码失败:', error);
          container.innerHTML = '<p class="error">加载失败: ' + error.message + '</p>';
        });
    },

    /**
     * 直接验证（演示用）
     */
    verifyDirect: function(sessionId, secretY) {
      const client = new window.CaptchaClient('http://localhost:8080');

      client.verifySliderCaptcha({
        session_id: sessionId,
        x: 150,
        y: secretY
      })
      .then(function(result) {
        if (result.success) {
          alert('验证成功！');
        } else {
          alert('验证失败: ' + result.message);
        }
      })
      .catch(function(error) {
        alert('验证出错: ' + error.message);
      });
    },

    /**
     * 模式2: 使用UI组件（滑块验证码）
     */
    sliderWidgetExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      if (typeof window.SliderCaptchaWidget !== 'function') {
        console.error('SliderCaptchaWidget not loaded');
        return;
      }

      new window.SliderCaptchaWidget(container, client, {
        width: 320,
        height: 160,
        tolerance: 8,
        onSuccess: function(result) {
          console.log('验证成功:', result);
          alert('验证成功！');
        },
        onFail: function(message) {
          console.log('验证失败:', message);
        }
      });
    },

    /**
     * 模式3: 使用UI组件（点击验证码）
     */
    clickWidgetExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      if (typeof window.ClickCaptchaWidget !== 'function') {
        console.error('ClickCaptchaWidget not loaded');
        return;
      }

      new window.ClickCaptchaWidget(container, client, {
        mode: 'number',
        points: 3,
        onSuccess: function(result) {
          console.log('验证成功:', result);
          alert('验证成功！');
        },
        onFail: function(message) {
          console.log('验证失败:', message);
        }
      });
    },

    /**
     * 模式4: 带轨迹记录的滑块验证
     */
    sliderWithTrajectoryExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      client.getSliderCaptcha({ width: 320, height: 160 })
        .then(function(captcha) {
          console.log('获取验证码成功:', captcha);

          container.innerHTML = `
            <div class="captcha-slider-with-trajectory">
              <img src="${captcha.image_url}" alt="验证码背景" id="slider-bg" />
              <div id="slider-track">
                <div id="slider-thumb"></div>
              </div>
              <div class="captcha-info">
                <p>Session ID: <span id="session-id">${captcha.session_id}</span></p>
                <p>Secret Y: <span id="secret-y">${captcha.secret_y || 0}</span></p>
              </div>
            </div>
          `;

          const thumb = document.getElementById('slider-thumb');
          const track = document.getElementById('slider-track');
          const trajectoryRecorder = client.recordTrajectory(function(points) {
            console.log('当前轨迹点数:', points.length);
          }, track);

          let isDragging = false;
          let startX = 0;
          let currentX = 0;

          thumb.addEventListener('mousedown', function(e) {
            isDragging = true;
            startX = e.clientX;
            trajectoryRecorder.start();
            thumb.classList.add('dragging');
          });

          document.addEventListener('mousemove', function(e) {
            if (!isDragging) return;

            currentX = e.clientX - startX;
            const maxX = track.offsetWidth - thumb.offsetWidth;
            currentX = Math.max(0, Math.min(currentX, maxX));

            thumb.style.left = currentX + 'px';
          });

          document.addEventListener('mouseup', async function() {
            if (!isDragging) return;
            isDragging = false;
            thumb.classList.remove('dragging');

            const trajectory = trajectoryRecorder.stop();
            const targetX = Math.round(currentX);

            try {
              const result = await client.verifySliderCaptcha({
                session_id: document.getElementById('session-id').textContent,
                x: targetX,
                y: parseInt(document.getElementById('secret-y').textContent),
                trajectory: trajectory
              });

              if (result.success) {
                alert('验证成功！轨迹点数量: ' + trajectory.length);
              } else {
                alert('验证失败: ' + result.message);
                thumb.style.left = '0';
              }
            } catch (error) {
              alert('验证出错: ' + error.message);
              thumb.style.left = '0';
            }
          });
        })
        .catch(function(error) {
          console.error('获取验证码失败:', error);
          container.innerHTML = '<p class="error">加载失败: ' + error.message + '</p>';
        });
    },

    /**
     * 模式5: 登录表单集成
     */
    loginFormIntegration: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      container.innerHTML = `
        <div class="captcha-login-form">
          <h2>登录示例</h2>
          <form id="login-form">
            <div class="form-group">
              <label for="username">用户名:</label>
              <input type="text" id="username" name="username" required />
            </div>
            <div class="form-group">
              <label for="password">密码:</label>
              <input type="password" id="password" name="password" required />
            </div>
            <div class="form-group">
              <div id="captcha-area"></div>
            </div>
            <button type="submit">登录</button>
          </form>
          <div id="login-result"></div>
        </div>
      `;

      const captchaArea = document.getElementById('captcha-area');
      let currentCaptchaSession = null;

      function loadCaptcha() {
        client.getSliderCaptcha({ width: 300, height: 150 })
          .then(function(captcha) {
            currentCaptchaSession = captcha.session_id;

            captchaArea.innerHTML = `
              <div class="inline-slider">
                <img src="${captcha.image_url}" alt="验证码" style="max-width: 100%;" />
                <input type="range" id="captcha-slider" min="0" max="260" value="0" />
                <button type="button" id="verify-captcha-btn">验证</button>
              </div>
              <p id="captcha-status"></p>
            `;

            const verifyBtn = document.getElementById('verify-captcha-btn');
            const slider = document.getElementById('captcha-slider');
            const status = document.getElementById('captcha-status');

            verifyBtn.addEventListener('click', function() {
              const x = parseInt(slider.value);

              client.verifySliderCaptcha({
                session_id: currentCaptchaSession,
                x: x,
                y: captcha.secret_y
              })
              .then(function(result) {
                if (result.success) {
                  status.textContent = '验证成功！';
                  status.className = 'success';
                  verifyBtn.disabled = true;
                  slider.disabled = true;
                } else {
                  status.textContent = '验证失败，请重试';
                  status.className = 'error';
                  slider.value = 0;
                  loadCaptcha();
                }
              })
              .catch(function(error) {
                status.textContent = '验证出错: ' + error.message;
                status.className = 'error';
              });
            });
          })
          .catch(function(error) {
            captchaArea.innerHTML = '<p class="error">验证码加载失败</p>';
          });
      }

      loadCaptcha();

      const loginForm = document.getElementById('login-form');
      const loginResult = document.getElementById('login-result');

      loginForm.addEventListener('submit', function(e) {
        e.preventDefault();

        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;

        if (!currentCaptchaSession) {
          loginResult.innerHTML = '<p class="error">请先完成验证码验证</p>';
          return;
        }

        const auth = client.auth();
        auth.login({
          username: username,
          password: password,
          captcha_token: currentCaptchaSession
        })
        .then(function(result) {
          loginResult.innerHTML = '<p class="success">登录成功！</p>';
          console.log('登录结果:', result);
        })
        .catch(function(error) {
          loginResult.innerHTML = '<p class="error">登录失败: ' + error.message + '</p>';
        });
      });
    },

    /**
     * 模式6: 环境检测集成
     */
    environmentDetectionExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');
      const env = client.env();

      container.innerHTML = `
        <div class="captcha-env-detection">
          <h2>环境检测示例</h2>
          <button id="run-detection-btn">运行环境检测</button>
          <div id="detection-results"></div>
        </div>
      `;

      const runBtn = document.getElementById('run-detection-btn');
      const resultsDiv = document.getElementById('detection-results');

      runBtn.addEventListener('click', function() {
        resultsDiv.innerHTML = '<p>正在收集环境数据...</p>';

        const browserData = env.collectBrowserData();

        resultsDiv.innerHTML = `
          <div class="detection-results">
            <h3>收集到的数据:</h3>
            <pre>${JSON.stringify(browserData, null, 2)}</pre>
            <button id="submit-detection-btn">提交检测数据</button>
          </div>
        `;

        const submitBtn = document.getElementById('submit-detection-btn');
        submitBtn.addEventListener('click', function() {
          env.submitDetection(browserData)
            .then(function(result) {
              resultsDiv.innerHTML += `
                <div class="submission-result">
                  <h3>提交结果:</h3>
                  <pre>${JSON.stringify(result, null, 2)}</pre>
                </div>
              `;
            })
            .catch(function(error) {
              resultsDiv.innerHTML += `
                <div class="error">
                  <p>提交失败: ${error.message}</p>
                </div>
              `;
            });
        });
      });
    },

    /**
     * 模式7: React/Vue集成示例（框架无关）
     */
    frameworkIntegrationExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      function createCaptchaComponent(container, options) {
        let sessionId = null;
        let secretY = null;
        let isVerified = false;

        function render() {
          client.getSliderCaptcha(options)
            .then(function(captcha) {
              sessionId = captcha.session_id;
              secretY = captcha.secret_y;

              container.innerHTML = `
                <div class="react-style-captcha" data-session="${sessionId}">
                  <div class="captcha-header">
                    <span>安全验证</span>
                    <button class="refresh-btn">↻</button>
                  </div>
                  <div class="captcha-body">
                    <img src="${captcha.image_url}" alt="验证码" />
                    <div class="slider-container">
                      <div class="slider-track">
                        <div class="slider-fill"></div>
                        <div class="slider-handle"></div>
                      </div>
                      <span class="slider-tip">拖动滑块完成验证</span>
                    </div>
                  </div>
                  <div class="captcha-footer"></div>
                </div>
              `;

              const refreshBtn = container.querySelector('.refresh-btn');
              const sliderHandle = container.querySelector('.slider-handle');
              const sliderFill = container.querySelector('.slider-fill');
              const footer = container.querySelector('.captcha-footer');

              refreshBtn.addEventListener('click', function() {
                render();
              });

              let isDragging = false;
              let startX = 0;
              let currentX = 0;
              const trackWidth = 260;

              sliderHandle.addEventListener('mousedown', function(e) {
                if (isVerified) return;
                isDragging = true;
                startX = e.clientX;
                sliderHandle.classList.add('active');
              });

              document.addEventListener('mousemove', function(e) {
                if (!isDragging) return;

                currentX = e.clientX - startX;
                currentX = Math.max(0, Math.min(currentX, trackWidth));

                sliderHandle.style.left = currentX + 'px';
                sliderFill.style.width = currentX + 'px';
              });

              document.addEventListener('mouseup', function() {
                if (!isDragging) return;
                isDragging = false;
                sliderHandle.classList.remove('active');

                const targetX = Math.round(currentX);

                client.verifySliderCaptcha({
                  session_id: sessionId,
                  x: targetX,
                  y: secretY
                })
                .then(function(result) {
                  if (result.success) {
                    isVerified = true;
                    sliderHandle.classList.add('success');
                    footer.textContent = '验证成功！';
                    footer.className = 'captcha-footer success';
                  } else {
                    footer.textContent = '验证失败，请重试';
                    footer.className = 'captcha-footer error';
                    sliderHandle.style.left = '0';
                    sliderFill.style.width = '0';
                    setTimeout(render, 1500);
                  }
                })
                .catch(function(error) {
                  footer.textContent = '验证出错';
                  footer.className = 'captcha-footer error';
                });
              });
            })
            .catch(function(error) {
              container.innerHTML = '<p class="error">加载失败</p>';
            });
        }

        return {
          render: render,
          reload: function() {
            isVerified = false;
            render();
          }
        };
      }

      const captchaComponent = createCaptchaComponent(container, { width: 300, height: 150 });
      captchaComponent.render();
    },

    /**
     * 模式8: 批量获取验证码示例
     */
    batchCaptchaExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) {
        console.error('Container element not found');
        return;
      }

      const client = new window.CaptchaClient('http://localhost:8080');

      container.innerHTML = `
        <div class="batch-captcha-example">
          <h2>批量验证码示例</h2>
          <button id="batch-load-btn">加载5个验证码</button>
          <div id="batch-results"></div>
        </div>
      `;

      const loadBtn = document.getElementById('batch-load-btn');
      const resultsDiv = document.getElementById('batch-results');

      loadBtn.addEventListener('click', function() {
        loadBtn.disabled = true;
        loadBtn.textContent = '加载中...';
        resultsDiv.innerHTML = '<p>正在并发获取验证码...</p>';

        const promises = [];
        for (let i = 0; i < 5; i++) {
          promises.push(client.getSliderCaptcha({ width: 200, height: 100 }));
        }

        Promise.allSettled(promises)
          .then(function(results) {
            let successCount = 0;
            let html = '<div class="batch-list">';

            results.forEach(function(result, index) {
              if (result.status === 'fulfilled') {
                successCount++;
                html += `
                  <div class="batch-item">
                    <img src="${result.value.image_url}" alt="验证码 ${index + 1}" />
                    <p>${result.value.session_id.substring(0, 20)}...</p>
                  </div>
                `;
              } else {
                html += `
                  <div class="batch-item error">
                    <p>验证码 ${index + 1} 加载失败</p>
                  </div>
                `;
              }
            });

            html += '</div>';
            html += `<p>成功: ${successCount}/${results.length}</p>`;

            resultsDiv.innerHTML = html;
            loadBtn.disabled = false;
            loadBtn.textContent = '重新加载';
          })
          .catch(function(error) {
            resultsDiv.innerHTML = '<p class="error">批量加载失败: ' + error.message + '</p>';
            loadBtn.disabled = false;
            loadBtn.textContent = '重新加载';
          });
      });
    }
  };

  if (typeof window !== 'undefined') {
    window.CaptchaBrowserExamples = CaptchaBrowserExamples;
  }

})();
