const SocialCaptcha = {
  sessionId: null,
  puzzle: null,
  traceData: [],
  startTime: null,
  isDrawing: false,
  
  async init(options = {}) {
    this.options = options;
    this.container = document.getElementById(options.containerId || 'captcha-container');
    
    if (!this.container) {
      console.error('Container not found');
      return;
    }
    
    await this.generate();
  },
  
  async generate() {
    try {
      const response = await fetch('/api/v1/captcha/social/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          difficulty: this.options.difficulty || 'medium',
          behavior_type: this.options.behaviorType || '',
          pattern_count: this.options.patternCount || 1
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.sessionId = result.data.session_id;
        this.puzzle = result.data.puzzle;
        this.traceData = [];
        this.render(result.data);
      } else {
        this.showError(result.message);
      }
    } catch (error) {
      console.error('Generate error:', error);
      this.showError('生成验证码失败');
    }
  },
  
  render(data) {
    const puzzle = data.puzzle;
    const pattern = puzzle.patterns[0];
    
    this.container.innerHTML = `
      <div class="social-captcha">
        <div class="captcha-hint">${puzzle.instructions || '请沿轨迹绘制图形'}</div>
        <div class="social-canvas-container">
          <canvas id="social-canvas" width="320" height="320"></canvas>
          <div class="guide-overlay" id="guide-overlay">
            ${this.renderGuidePattern(pattern)}
          </div>
        </div>
        <div class="trace-info">
          <span id="trace-status">开始绘制</span>
          <span id="trace-points">点数: 0</span>
        </div>
        <div class="social-controls">
          <button id="clear-btn" class="social-btn">清除</button>
          <button id="submit-trace-btn" class="social-btn primary" disabled>提交</button>
        </div>
      </div>
    `;
    
    this.initCanvas();
    this.initControls();
  },
  
  renderGuidePattern(pattern) {
    if (!pattern || !pattern.control_points || pattern.control_points.length === 0) {
      return '<div class="default-guide"></div>';
    }
    
    const points = pattern.control_points;
    const svgPath = this.generateSVGPath(points, pattern.target_shape);
    
    return `
      <svg class="guide-svg" viewBox="0 0 320 320">
        <path class="guide-path" d="${svgPath}" />
        <circle class="guide-start" cx="${points[0].X}" cy="${points[0].Y}" r="8" />
        <circle class="guide-end" cx="${points[points.length-1].X}" cy="${points[points.length-1].Y}" r="8" />
        ${points.slice(1, -1).map((p, i) => 
          `<circle class="guide-point" cx="${p.X}" cy="${p.Y}" r="4" />`
        ).join('')}
      </svg>
    `;
  },
  
  generateSVGPath(points, shapeType) {
    if (points.length < 2) return '';
    
    let path = `M ${points[0].X} ${points[0].Y}`;
    
    if (points.length === 2) {
      path += ` L ${points[1].X} ${points[1].Y}`;
    } else {
      for (let i = 1; i < points.length - 1; i++) {
        const xc = (points[i].X + points[i + 1].X) / 2;
        const yc = (points[i].Y + points[i + 1].Y) / 2;
        path += ` Q ${points[i].X} ${points[i].Y} ${xc} ${yc}`;
      }
      path += ` L ${points[points.length-1].X} ${points[points.length-1].Y}`;
    }
    
    return path;
  },
  
  initCanvas() {
    const canvas = document.getElementById('social-canvas');
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.strokeStyle = '#2196F3';
    ctx.lineWidth = 4;
    
    this.canvas = canvas;
    this.ctx = ctx;
    this.traceData = [];
    
    const getPos = (e) => {
      const rect = canvas.getBoundingClientRect();
      const clientX = e.touches ? e.touches[0].clientX : e.clientX;
      const clientY = e.touches ? e.touches[0].clientY : e.clientY;
      return {
        x: clientX - rect.left,
        y: clientY - rect.top
      };
    };
    
    const onStart = (e) => {
      if (!this.startTime) {
        this.startTime = Date.now();
      }
      
      this.isDrawing = true;
      const pos = getPos(e);
      ctx.beginPath();
      ctx.moveTo(pos.x, pos.y);
      
      this.traceData.push({
        X: pos.x,
        Y: pos.y,
        Timestamp: Date.now(),
        Pressure: 0.5,
        Angle: 0
      });
      
      this.updateTraceStatus();
      e.preventDefault();
    };
    
    const onMove = (e) => {
      if (!this.isDrawing) return;
      
      const pos = getPos(e);
      ctx.lineTo(pos.x, pos.y);
      ctx.stroke();
      
      this.traceData.push({
        X: pos.x,
        Y: pos.y,
        Timestamp: Date.now(),
        Pressure: 0.5,
        Angle: 0
      });
      
      this.updateTraceStatus();
      e.preventDefault();
    };
    
    const onEnd = () => {
      this.isDrawing = false;
      document.getElementById('submit-trace-btn').disabled = this.traceData.length < 10;
    };
    
    canvas.addEventListener('mousedown', onStart);
    canvas.addEventListener('touchstart', onStart, { passive: false });
    canvas.addEventListener('mousemove', onMove);
    canvas.addEventListener('touchmove', onMove, { passive: false });
    canvas.addEventListener('mouseup', onEnd);
    canvas.addEventListener('touchend', onEnd);
    canvas.addEventListener('mouseleave', onEnd);
  },
  
  initControls() {
    document.getElementById('clear-btn')?.addEventListener('click', () => {
      this.clearCanvas();
    });
    
    document.getElementById('submit-trace-btn')?.addEventListener('click', () => {
      this.submit();
    });
  },
  
  clearCanvas() {
    if (!this.ctx || !this.canvas) return;
    
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    this.traceData = [];
    this.startTime = null;
    
    this.updateTraceStatus();
    document.getElementById('submit-trace-btn').disabled = true;
  },
  
  updateTraceStatus() {
    const status = document.getElementById('trace-status');
    const points = document.getElementById('trace-points');
    
    if (status) {
      status.textContent = this.isDrawing ? '绘制中...' : '完成绘制';
    }
    if (points) {
      points.textContent = `点数: ${this.traceData.length}`;
    }
  },
  
  async submit() {
    if (this.traceData.length < 10) {
      this.showFeedback('请绘制更完整的轨迹');
      return;
    }
    
    const endTime = Date.now();
    
    try {
      const response = await fetch('/api/v1/captcha/social/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          trace_data: this.traceData,
          pattern_type: this.puzzle.patterns[0].target_shape,
          start_time: this.startTime,
          end_time: endTime,
          touch_points: this.traceData.map(p => ({
            X: p.X,
            Y: p.Y,
            Pressure: p.Pressure,
            Timestamp: p.Timestamp
          })),
          risk_score: 0.5
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.handleResult(result.data);
      }
    } catch (error) {
      console.error('Submit error:', error);
      this.showError('提交失败');
    }
  },
  
  handleResult(data) {
    if (data.success) {
      this.onSuccess(data);
    } else {
      this.showFeedback(data.feedback || '验证失败');
    }
  },
  
  showFeedback(message) {
    const hint = this.container.querySelector('.captcha-hint');
    if (hint) {
      hint.textContent = message;
      hint.classList.add('error');
      setTimeout(() => {
        hint.classList.remove('error');
        hint.textContent = this.puzzle?.instructions || '请沿轨迹绘制图形';
      }, 2000);
    }
  },
  
  showError(message) {
    this.container.innerHTML = `
      <div class="captcha-error">
        <span>${message}</span>
        <button onclick="SocialCaptcha.generate()">重试</button>
      </div>
    `;
  },
  
  onSuccess(data) {
    if (this.options.onSuccess) {
      this.options.onSuccess(data);
    }
    this.container.querySelector('.captcha-hint').textContent = '社交行为验证成功！';
    this.container.querySelector('.captcha-hint').classList.add('success');
  }
};

window.SocialCaptcha = SocialCaptcha;
