const GridCaptcha = {
  sessionId: null,
  puzzle: null,
  selectedOrder: [],
  
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
      const response = await fetch('/api/v1/captcha/grid/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          grid_size: this.options.gridSize || 3,
          target_count: this.options.targetCount || 3,
          difficulty: this.options.difficulty || 'medium',
          icon_type: this.options.iconType || ''
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.sessionId = result.data.session_id;
        this.puzzle = result.data.puzzle;
        this.selectedOrder = [];
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
    const gridSize = puzzle.grid_size;
    const cellSize = 80;
    
    this.container.innerHTML = `
      <div class="grid-captcha">
        <div class="captcha-hint">${data.hint_text || '请按正确顺序点击目标'}</div>
        <div class="grid-container" id="grid-container" 
             style="width: ${gridSize * cellSize}px; height: ${gridSize * cellSize}px;">
        </div>
        <div class="grid-progress">
          <span id="progress-text">已选择: 0/${puzzle.required_count}</span>
        </div>
        <div class="grid-order-display" id="order-display"></div>
      </div>
    `;
    
    this.renderGrid(puzzle, cellSize);
    this.initClickInteraction();
  },
  
  renderGrid(puzzle, cellSize) {
    const container = document.getElementById('grid-container');
    if (!container) return;
    
    container.innerHTML = '';
    
    puzzle.cells.forEach((cell, index) => {
      const cellEl = document.createElement('div');
      cellEl.className = 'grid-cell';
      cellEl.dataset.index = cell.index;
      cellEl.dataset.row = cell.row;
      cellEl.dataset.col = cell.col;
      cellEl.style.cssText = `
        left: ${cell.col * cellSize}px;
        top: ${cell.row * cellSize}px;
        width: ${cellSize}px;
        height: ${cellSize}px;
        background-color: ${cell.color};
      `;
      
      const iconEl = document.createElement('span');
      iconEl.className = 'cell-icon';
      iconEl.textContent = this.getIconEmoji(cell.icon_type, cell.icon_id);
      cellEl.appendChild(iconEl);
      
      if (cell.is_target) {
        cellEl.classList.add('target');
      }
      
      container.appendChild(cellEl);
    });
  },
  
  getIconEmoji(iconType, iconId) {
    const icons = {
      'animal': ['🐱', '🐕', '🐦', '🐟', '🐴', '🐘', '🦁', '🐰'],
      'food': ['🍕', '🍔', '🍣', '🍜', '🍦', '🍰'],
      'vehicle': ['🚗', '🚌', '🚲', '✈️', '🚂', '🚢'],
      'fruit': ['🍎', '🍊', '🍇', '🍓', '🍉', '🥭'],
      'object': ['📚', '📱', '☕', '🪑', '💡', '📷']
    };
    
    const typeIcons = icons[iconType] || icons['object'];
    return typeIcons[iconId % typeIcons.length] || '❓';
  },
  
  initClickInteraction() {
    const container = document.getElementById('grid-container');
    if (!container) return;
    
    this.startTime = Date.now();
    
    container.addEventListener('click', async (e) => {
      const cell = e.target.closest('.grid-cell');
      if (!cell) return;
      
      const index = parseInt(cell.dataset.index);
      
      if (this.selectedOrder.includes(index)) {
        this.showFeedback('已选择，请选择其他');
        return;
      }
      
      this.selectedOrder.push(index);
      cell.classList.add('selected');
      
      this.updateProgress();
      this.updateOrderDisplay();
      
      if (this.selectedOrder.length === this.puzzle.required_count) {
        await this.verify();
      }
    });
  },
  
  updateProgress() {
    const progressText = document.getElementById('progress-text');
    if (progressText) {
      progressText.textContent = `已选择: ${this.selectedOrder.length}/${this.puzzle.required_count}`;
    }
  },
  
  updateOrderDisplay() {
    const display = document.getElementById('order-display');
    if (display) {
      display.innerHTML = this.selectedOrder
        .map((_, i) => `<span class="order-num">${i + 1}</span>`)
        .join('');
    }
  },
  
  async verify() {
    const endTime = Date.now();
    
    try {
      const response = await fetch('/api/v1/captcha/grid/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          selected_order: this.selectedOrder,
          time_spent: endTime - this.startTime,
          click_pattern: this.selectedOrder.map((idx, i) => ({
            row: this.puzzle.cells[idx].row,
            col: this.puzzle.cells[idx].col,
            timestamp: this.startTime + i * 1000
          })),
          risk_score: 0.5
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.handleResult(result.data);
      }
    } catch (error) {
      console.error('Verify error:', error);
      this.showError('验证失败');
    }
  },
  
  handleResult(data) {
    if (data.success) {
      this.onSuccess(data);
    } else {
      this.showFeedback(data.message || '验证失败');
      this.reset();
    }
  },
  
  showFeedback(message) {
    const hint = this.container.querySelector('.captcha-hint');
    if (hint) {
      hint.textContent = message;
      hint.classList.add('error');
      setTimeout(() => {
        hint.classList.remove('error');
        hint.textContent = this.puzzle?.hint_text || '请按正确顺序点击目标';
      }, 2000);
    }
  },
  
  showError(message) {
    this.container.innerHTML = `
      <div class="captcha-error">
        <span>${message}</span>
        <button onclick="GridCaptcha.generate()">重试</button>
      </div>
    `;
  },
  
  reset() {
    this.selectedOrder = [];
    const cells = this.container.querySelectorAll('.grid-cell');
    cells.forEach(cell => cell.classList.remove('selected', 'correct', 'incorrect'));
    this.updateProgress();
    this.updateOrderDisplay();
  },
  
  onSuccess(data) {
    if (this.options.onSuccess) {
      this.options.onSuccess(data);
    }
    
    const cells = this.container.querySelectorAll('.grid-cell');
    cells.forEach((cell, i) => {
      if (this.selectedOrder.includes(i)) {
        cell.classList.add('correct');
      }
    });
    
    this.container.querySelector('.captcha-hint').textContent = '验证成功！';
    this.container.querySelector('.captcha-hint').classList.add('success');
  }
};

window.GridCaptcha = GridCaptcha;
