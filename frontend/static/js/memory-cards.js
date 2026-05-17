class MemoryCardsCaptcha {
    constructor() {
        this.sessionId = null;
        this.board = null;
        this.boardData = null;
        this.cardIcons = [];
        this.selectedCard = null;
        this.matchedCards = new Set();
        this.matches = [];
        this.timerInterval = null;
        this.seconds = 0;
        this.isLocked = false;
        this.showTime = 5;
        this.init();
    }

    init() {
        this.bindEvents();
        this.refresh();
    }

    bindEvents() {
        document.getElementById('refresh-btn').addEventListener('click', () => this.refresh());
        document.getElementById('submit-btn').addEventListener('click', () => this.submit());
    }

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();

        try {
            const response = await fetch('/api/v1/captcha/memory-cards/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    width: 4,
                    height: 4,
                    card_types: 8,
                    show_time: 5
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.session_id;
                this.boardData = result.data.board;
                this.cardIcons = result.data.card_icons;
                this.showTime = result.data.show_time || 5;
                await this.renderBoard();
                await this.showInitialCards();
            } else {
                this.showResult(false, result.message || '生成验证码失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    renderBoard() {
        this.selectedCard = null;
        this.matchedCards = new Set();
        this.matches = [];
        this.isLocked = false;
        document.getElementById('match-count').textContent = '0';
        document.getElementById('total-pairs').textContent = this.boardData.pair_count;

        const boardEl = document.getElementById('board');
        boardEl.innerHTML = '';
        boardEl.style.gridTemplateColumns = `repeat(${this.boardData.width}, 70px)`;

        for (let y = 0; y < this.boardData.height; y++) {
            for (let x = 0; x < this.boardData.width; x++) {
                const card = this.boardData.cards[y][x];
                const cardEl = document.createElement('div');
                cardEl.className = 'card';
                cardEl.dataset.x = x;
                cardEl.dataset.y = y;
                cardEl.dataset.index = card.index;
                cardEl.dataset.type = card.type;

                cardEl.innerHTML = `
                    <div class="card-inner">
                        <div class="card-front">
                            <i class="fas fa-question-circle"></i>
                        </div>
                        <div class="card-back">
                            ${this.cardIcons[card.type]}
                        </div>
                    </div>
                `;

                cardEl.addEventListener('click', () => this.handleCardClick(card, cardEl));
                boardEl.appendChild(cardEl);
            }
        }
    }

    async showInitialCards() {
        // Show all cards
        const cards = document.querySelectorAll('.card');
        cards.forEach(card => {
            card.classList.add('flipped');
        });

        // Countdown
        const countdownOverlay = document.getElementById('countdown-overlay');
        const countdownNumber = document.getElementById('countdown-number');
        countdownOverlay.classList.add('show');

        for (let i = this.showTime; i > 0; i--) {
            countdownNumber.textContent = i;
            await this.sleep(1000);
        }

        countdownOverlay.classList.remove('show');

        // Hide all cards
        cards.forEach(card => {
            card.classList.remove('flipped');
        });

        // Start timer
        this.startTimer();
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    async handleCardClick(card, cardEl) {
        if (this.isLocked) return;
        if (this.matchedCards.has(card.index)) return;
        if (this.selectedCard && this.selectedCard.index === card.index) return;

        cardEl.classList.add('flipped');

        if (this.selectedCard) {
            this.isLocked = true;

            if (this.selectedCard.type === card.type) {
                await this.matchCards(this.selectedCard, card);
            } else {
                await this.wrongMatch(this.selectedCard, card);
            }

            this.selectedCard = null;
            this.isLocked = false;
        } else {
            this.selectedCard = card;
        }
    }

    async matchCards(card1, card2) {
        const card1El = document.querySelector(`.card[data-index="${card1.index}"]`);
        const card2El = document.querySelector(`.card[data-index="${card2.index}"]`);

        await this.sleep(300);

        card1El.classList.add('matched');
        card2El.classList.add('matched');

        this.matchedCards.add(card1.index);
        this.matchedCards.add(card2.index);
        this.matches.push({
            card1: card1,
            card2: card2
        });

        document.getElementById('match-count').textContent = this.matches.length;

        if (this.matches.length === this.boardData.pair_count) {
            setTimeout(() => this.submit(), 500);
        }
    }

    async wrongMatch(card1, card2) {
        const card1El = document.querySelector(`.card[data-index="${card1.index}"]`);
        const card2El = document.querySelector(`.card[data-index="${card2.index}"]`);

        await this.sleep(1000);

        card1El.classList.remove('flipped');
        card2El.classList.remove('flipped');
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);
        this.stopTimer();

        try {
            const response = await fetch('/api/v1/captcha/memory-cards/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.sessionId,
                    board: this.boardData,
                    matches: this.matches,
                    time_used: this.seconds
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.showResult(result.data.success, result.data.message || (result.data.success ? '验证成功！' : '验证失败'));
            } else {
                this.showResult(false, result.message || '验证失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    startTimer() {
        this.seconds = 0;
        this.updateTimerDisplay();
        this.timerInterval = setInterval(() => {
            this.seconds++;
            this.updateTimerDisplay();
        }, 1000);
    }

    stopTimer() {
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }
    }

    resetTimer() {
        this.stopTimer();
        this.seconds = 0;
        this.updateTimerDisplay();
    }

    updateTimerDisplay() {
        const minutes = Math.floor(this.seconds / 60).toString().padStart(2, '0');
        const seconds = (this.seconds % 60).toString().padStart(2, '0');
        document.getElementById('timer').textContent = `${minutes}:${seconds}`;
    }

    showResult(success, message) {
        const banner = document.getElementById('result-banner');
        banner.className = `result-banner show ${success ? 'success' : 'error'}`;
        banner.innerHTML = `
            <i class="fas fa-${success ? 'check-circle' : 'times-circle'} me-2"></i>
            <strong>${message}</strong>
        `;
    }

    hideResult() {
        const banner = document.getElementById('result-banner');
        banner.classList.remove('show');
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('show');
        } else {
            overlay.classList.remove('show');
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new MemoryCardsCaptcha();
});
