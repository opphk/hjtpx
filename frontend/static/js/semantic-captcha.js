const SemanticCaptcha = {
  sessionId: null,
  puzzle: null,
  selectedAnswer: null,
  startTime: null,
  
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
      const response = await fetch('/api/v1/captcha/semantic/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          difficulty: this.options.difficulty || 'medium',
          category: this.options.category || '',
          analysis_type: this.options.analysisType || '',
          image_count: this.options.imageCount || 4
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.sessionId = result.data.session_id;
        this.puzzle = result.data.puzzle;
        this.startTime = Date.now();
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
    
    this.container.innerHTML = `
      <div class="semantic-captcha">
        <div class="captcha-hint">${puzzle.question || '请分析图片内容'}</div>
        <div class="image-gallery" id="image-gallery">
          ${this.renderImages(puzzle)}
        </div>
        <div class="options-container" id="options-container">
          ${this.renderOptions(puzzle)}
        </div>
        <div class="confidence-slider" id="confidence-section" style="display: none;">
          <label>您的置信度：</label>
          <input type="range" id="confidence-slider" min="0" max="100" value="50" />
          <span id="confidence-value">50%</span>
        </div>
        <div class="action-buttons">
          <button id="analyze-btn" class="semantic-btn">分析</button>
          <button id="submit-btn" class="semantic-btn primary" disabled>提交</button>
        </div>
      </div>
    `;
    
    this.initInteractions();
  },
  
  renderImages(puzzle) {
    return puzzle.images.map((img, index) => `
      <div class="semantic-image" data-index="${index}">
        <div class="image-placeholder" style="background: ${this.getColorForCategory(img.category)}">
          <span class="category-icon">${this.getCategoryIcon(img.category)}</span>
          <span class="category-label">${img.category}</span>
        </div>
        <div class="image-labels">
          ${img.labels.map(l => `<span class="label">${l}</span>`).join('')}
        </div>
      </div>
    `).join('');
  },
  
  renderOptions(puzzle) {
    return puzzle.options.map((option, index) => `
      <div class="option-item" data-option="${option}" data-index="${index}">
        <span class="option-marker">${String.fromCharCode(65 + index)}</span>
        <span class="option-text">${option}</span>
      </div>
    `).join('');
  },
  
  getColorForCategory(category) {
    const colors = {
      'animal': '#FFE4E1',
      'vehicle': '#E6E6FA',
      'food': '#FFFACD',
      'building': '#E0FFFF',
      'nature': '#F0FFF0',
      'object': '#FFF0F5'
    };
    return colors[category] || '#F5F5F5';
  },
  
  getCategoryIcon(category) {
    const icons = {
      'animal': '🐾',
      'vehicle': '🚗',
      'food': '🍽️',
      'building': '🏛️',
      'nature': '🌿',
      'object': '📦'
    };
    return icons[category] || '🖼️';
  },
  
  initInteractions() {
    const optionsContainer = document.getElementById('options-container');
    const analyzeBtn = document.getElementById('analyze-btn');
    const submitBtn = document.getElementById('submit-btn');
    const confidenceSlider = document.getElementById('confidence-slider');
    
    optionsContainer?.addEventListener('click', (e) => {
      const option = e.target.closest('.option-item');
      if (!option) return;
      
      document.querySelectorAll('.option-item').forEach(o => o.classList.remove('selected'));
      option.classList.add('selected');
      
      this.selectedAnswer = {
        text: option.dataset.option,
        index: parseInt(option.dataset.index)
      };
      
      submitBtn.disabled = false;
    });
    
    analyzeBtn?.addEventListener('click', () => {
      document.getElementById('confidence-section').style.display = 'block';
      this.analyze();
    });
    
    submitBtn?.addEventListener('click', () => {
      this.submit();
    });
    
    confidenceSlider?.addEventListener('input', (e) => {
      document.getElementById('confidence-value').textContent = e.target.value + '%';
    });
  },
  
  analyze() {
    const images = document.querySelectorAll('.semantic-image');
    const hints = [];
    
    images.forEach((img, index) => {
      const labels = this.puzzle.images[index].labels;
      if (labels.length > 0) {
        hints.push(labels[0]);
      }
    });
    
    const hint = this.container.querySelector('.captcha-hint');
    if (hint && hints.length > 0) {
      hint.innerHTML = `提示：图片关键词包括 <b>${hints.slice(0, 3).join('、')}</b> 等`;
      hint.classList.add('analysis-hint');
    }
  },
  
  async submit() {
    if (!this.selectedAnswer) {
      this.showFeedback('请选择一个答案');
      return;
    }
    
    const endTime = Date.now();
    const responseTime = endTime - this.startTime;
    const confidenceSlider = document.getElementById('confidence-slider');
    const confidence = confidenceSlider ? parseInt(confidenceSlider.value) / 100 : 0.5;
    
    try {
      const response = await fetch('/api/v1/captcha/semantic/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          answer: this.selectedAnswer.text,
          answer_index: this.selectedAnswer.index,
          confidence_score: confidence,
          response_time: responseTime,
          analysis_method: 'manual',
          keywords: this.puzzle.images.flatMap(img => img.labels),
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
      this.showFeedback(data.message || '验证失败');
      
      const hint = this.container.querySelector('.captcha-hint');
      if (hint && data.analysis_feedback) {
        hint.innerHTML = data.analysis_feedback;
        hint.classList.add('feedback');
      }
    }
  },
  
  showFeedback(message) {
    const hint = this.container.querySelector('.captcha-hint');
    if (hint) {
      hint.textContent = message;
      hint.classList.add('error');
      setTimeout(() => {
        hint.classList.remove('error');
        hint.textContent = this.puzzle?.question || '请分析图片内容';
      }, 3000);
    }
  },
  
  showError(message) {
    this.container.innerHTML = `
      <div class="captcha-error">
        <span>${message}</span>
        <button onclick="SemanticCaptcha.generate()">重试</button>
      </div>
    `;
  },
  
  onSuccess(data) {
    if (this.options.onSuccess) {
      this.options.onSuccess(data);
    }
    this.container.querySelector('.captcha-hint').textContent = '语义验证成功！';
    this.container.querySelector('.captcha-hint').classList.add('success');
  }
};

window.SemanticCaptcha = SemanticCaptcha;
