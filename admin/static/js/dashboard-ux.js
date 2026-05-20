let uxEnhancements = {
    animations: {
        enabled: true,
        duration: 300,
        easing: 'ease-out'
    },
    feedback: {
        haptics: false,
        sounds: false
    },
    accessibility: {
        reducedMotion: false,
        highContrast: false
    }
};

function initUXEnhancements() {
    checkAccessibilityPreferences();
    initKeyboardShortcuts();
    initQuickActions();
    initContextMenus();
    initDragAndDrop();
    initTouchGestures();
    enhanceCards();
    enhanceButtons();
    enhanceInputs();
    initNotificationCenter();
    initSearchFunctionality();
}

function checkAccessibilityPreferences() {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
        uxEnhancements.animations.enabled = false;
        uxEnhancements.accessibility.reducedMotion = true;
        document.documentElement.style.setProperty('--animation-duration', '0.01ms');
    }

    if (window.matchMedia('(prefers-contrast: high)').matches) {
        uxEnhancements.accessibility.highContrast = true;
        document.body.classList.add('high-contrast');
    }

    window.matchMedia('(prefers-reduced-motion: reduce)').addEventListener('change', (e) => {
        uxEnhancements.animations.enabled = !e.matches;
        document.documentElement.style.setProperty('--animation-duration', e.matches ? '0.01ms' : '300ms');
    });
}

function initKeyboardShortcuts() {
    const shortcuts = {
        'r': { action: 'refresh', description: '刷新数据' },
        'f': { action: 'fullscreen', description: '全屏模式' },
        't': { action: 'theme', description: '切换主题' },
        'e': { action: 'export', description: '导出数据' },
        '?': { action: 'help', description: '显示帮助' }
    };

    document.addEventListener('keydown', (e) => {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

        const key = e.key.toLowerCase();
        if (shortcuts[key]) {
            e.preventDefault();
            executeShortcut(shortcuts[key].action);
            showShortcutFeedback(shortcuts[key].description);
        }
    });
}

function executeShortcut(action) {
    switch (action) {
        case 'refresh':
            if (typeof refreshAllStats === 'function') {
                refreshAllStats();
            }
            break;
        case 'fullscreen':
            toggleFullscreen();
            break;
        case 'theme':
            toggleTheme();
            break;
        case 'export':
            showExportMenu();
            break;
        case 'help':
            showHelpModal();
            break;
    }
}

function showShortcutFeedback(description) {
    const feedback = document.createElement('div');
    feedback.className = 'shortcut-feedback position-fixed';
    feedback.innerHTML = `
        <i class="fas fa-keyboard mr-2"></i>
        <span>${description}</span>
    `;
    feedback.style.cssText = `
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: rgba(0, 0, 0, 0.8);
        color: white;
        padding: 1rem 2rem;
        border-radius: 8px;
        z-index: 10000;
        animation: fadeInOut 1s ease forwards;
    `;

    const style = document.createElement('style');
    style.textContent = `
        @keyframes fadeInOut {
            0% { opacity: 0; transform: translate(-50%, -50%) scale(0.9); }
            20% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
            80% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
            100% { opacity: 0; transform: translate(-50%, -50%) scale(0.9); }
        }
    `;

    document.head.appendChild(style);
    document.body.appendChild(feedback);

    setTimeout(() => {
        feedback.remove();
        style.remove();
    }, 1000);
}

function initQuickActions() {
    const quickActions = [
        { id: 'refresh', icon: 'sync', label: '刷新', shortcut: 'R' },
        { id: 'export', icon: 'download', label: '导出', shortcut: 'E' },
        { id: 'fullscreen', icon: 'expand', label: '全屏', shortcut: 'F' },
        { id: 'help', icon: 'question-circle', label: '帮助', shortcut: '?' }
    ];

    const toolbar = document.createElement('div');
    toolbar.className = 'quick-actions-toolbar';
    toolbar.id = 'quickActionsToolbar';
    toolbar.innerHTML = quickActions.map(action => `
        <button class="btn btn-sm btn-outline-secondary quick-action-btn"
                data-action="${action.id}"
                title="${action.label} (${action.shortcut})"
                aria-label="${action.label}">
            <i class="fas fa-${action.icon}"></i>
            <span class="d-none d-md-inline ml-1">${action.label}</span>
        </button>
    `).join('');

    const headerActions = document.querySelector('.content-header .col-sm-6');
    if (headerActions) {
        headerActions.insertBefore(toolbar, headerActions.firstChild);
    }

    toolbar.querySelectorAll('.quick-action-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            executeShortcut(btn.dataset.action);
        });
    });
}

function initContextMenus() {
    document.addEventListener('contextmenu', (e) => {
        const target = e.target.closest('.card, .small-box, .info-box');
        if (target) {
            e.preventDefault();
            showContextMenu(e.clientX, e.clientY, target);
        }
    });

    document.addEventListener('click', () => {
        hideContextMenu();
    });
}

function showContextMenu(x, y, target) {
    hideContextMenu();

    const menu = document.createElement('div');
    menu.className = 'context-menu';
    menu.id = 'contextMenu';
    menu.innerHTML = `
        <div class="context-menu-item" data-action="refresh">
            <i class="fas fa-sync mr-2"></i>刷新
        </div>
        <div class="context-menu-item" data-action="expand">
            <i class="fas fa-expand mr-2"></i>展开详情
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item" data-action="export">
            <i class="fas fa-download mr-2"></i>导出数据
        </div>
        <div class="context-menu-item" data-action="copy">
            <i class="fas fa-copy mr-2"></i>复制数据
        </div>
    `;

    menu.style.cssText = `
        position: fixed;
        top: ${y}px;
        left: ${x}px;
        background: white;
        border: 1px solid rgba(0,0,0,0.1);
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        padding: 0.5rem 0;
        z-index: 10000;
        min-width: 150px;
    `;

    const itemStyle = document.createElement('style');
    itemStyle.textContent = `
        .context-menu-item {
            padding: 0.5rem 1rem;
            cursor: pointer;
            transition: background 0.2s;
        }
        .context-menu-item:hover {
            background: rgba(0, 123, 255, 0.1);
        }
        .context-menu-divider {
            height: 1px;
            background: rgba(0,0,0,0.1);
            margin: 0.5rem 0;
        }
    `;

    document.head.appendChild(itemStyle);
    document.body.appendChild(menu);

    menu.querySelectorAll('.context-menu-item').forEach(item => {
        item.addEventListener('click', () => {
            handleContextAction(item.dataset.action, target);
            hideContextMenu();
        });
    });
}

function hideContextMenu() {
    const menu = document.getElementById('contextMenu');
    if (menu) {
        menu.remove();
    }
}

function handleContextAction(action, target) {
    switch (action) {
        case 'refresh':
            if (typeof refreshAllStats === 'function') {
                refreshAllStats();
            }
            break;
        case 'expand':
            const cardWidget = target.closest('.card');
            if (cardWidget) {
                const collapseBtn = cardWidget.querySelector('[data-card-widget="collapse"]');
                if (collapseBtn) {
                    collapseBtn.click();
                }
            }
            break;
        case 'export':
            showExportMenu();
            break;
        case 'copy':
            copyToClipboard(target.textContent);
            break;
    }
}

function initDragAndDrop() {
    const draggableElements = document.querySelectorAll('.card, .small-box');
    draggableElements.forEach(el => {
        el.setAttribute('draggable', 'true');
        el.style.cursor = 'move';

        el.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/plain', el.id || '');
            el.style.opacity = '0.5';
        });

        el.addEventListener('dragend', (e) => {
            el.style.opacity = '1';
        });
    });
}

function initTouchGestures() {
    if ('ontouchstart' in window) {
        let touchStartX = 0;
        let touchStartY = 0;

        document.addEventListener('touchstart', (e) => {
            touchStartX = e.touches[0].clientX;
            touchStartY = e.touches[0].clientY;
        });

        document.addEventListener('touchend', (e) => {
            const touchEndX = e.changedTouches[0].clientX;
            const touchEndY = e.changedTouches[0].clientY;
            const deltaX = touchEndX - touchStartX;
            const deltaY = touchEndY - touchStartY;

            if (Math.abs(deltaX) > 100 && Math.abs(deltaX) > Math.abs(deltaY)) {
                if (deltaX > 0) {
                    showNotification('右滑手势', '从左向右滑动');
                } else {
                    showNotification('左滑手势', '从右向左滑动');
                }
            }
        });
    }
}

function enhanceCards() {
    const cards = document.querySelectorAll('.card, .small-box, .info-box');
    cards.forEach(card => {
        card.classList.add('enhanced-card');

        card.addEventListener('mouseenter', () => {
            if (uxEnhancements.animations.enabled) {
                card.style.transform = 'translateY(-5px)';
                card.style.boxShadow = '0 8px 25px rgba(0, 0, 0, 0.15)';
            }
        });

        card.addEventListener('mouseleave', () => {
            if (uxEnhancements.animations.enabled) {
                card.style.transform = '';
                card.style.boxShadow = '';
            }
        });

        card.addEventListener('click', (e) => {
            if (!e.target.closest('a, button')) {
                card.classList.add('clicked');
                setTimeout(() => {
                    card.classList.remove('clicked');
                }, 200);
            }
        });
    });
}

function enhanceButtons() {
    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(btn => {
        if (uxEnhancements.animations.enabled) {
            btn.addEventListener('mouseenter', () => {
                btn.style.transform = 'translateY(-2px)';
            });

            btn.addEventListener('mouseleave', () => {
                btn.style.transform = '';
            });

            btn.addEventListener('mousedown', () => {
                btn.style.transform = 'translateY(0) scale(0.98)';
            });

            btn.addEventListener('mouseup', () => {
                btn.style.transform = 'translateY(-2px)';
            });
        }
    });
}

function enhanceInputs() {
    const inputs = document.querySelectorAll('input, select');
    inputs.forEach(input => {
        input.classList.add('enhanced-input');

        input.addEventListener('focus', () => {
            input.parentElement.classList.add('focused');
        });

        input.addEventListener('blur', () => {
            input.parentElement.classList.remove('focused');
        });
    });
}

function initNotificationCenter() {
    const notificationBell = document.getElementById('notificationsBtn');
    if (!notificationBell) return;

    const badge = document.createElement('span');
    badge.className = 'notification-badge';
    badge.id = 'notificationBadgeCount';
    badge.style.cssText = `
        position: absolute;
        top: -5px;
        right: -5px;
        background: #dc3545;
        color: white;
        border-radius: 50%;
        width: 18px;
        height: 18px;
        font-size: 11px;
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.3s;
    `;

    notificationBell.style.position = 'relative';
    notificationBell.appendChild(badge);

    setTimeout(() => {
        updateNotificationCount(Math.floor(Math.random() * 10));
    }, 2000);
}

function updateNotificationCount(count) {
    const badge = document.getElementById('notificationBadgeCount');
    if (badge) {
        badge.textContent = count > 9 ? '9+' : count;
        badge.style.opacity = count > 0 ? '1' : '0';
    }
}

function initSearchFunctionality() {
    const searchInput = document.createElement('input');
    searchInput.type = 'search';
    searchInput.className = 'form-control dashboard-search';
    searchInput.placeholder = '搜索仪表盘内容... (Ctrl+K)';
    searchInput.id = 'dashboardSearchInput';
    searchInput.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        width: 300px;
        z-index: 9999;
        opacity: 0;
        pointer-events: none;
        transition: opacity 0.3s;
    `;

    document.body.appendChild(searchInput);

    document.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            searchInput.style.opacity = searchInput.style.opacity === '1' ? '0' : '1';
            searchInput.style.pointerEvents = searchInput.style.opacity === '1' ? 'auto' : 'none';
            if (searchInput.style.opacity === '1') {
                searchInput.focus();
            }
        }

        if (e.key === 'Escape' && searchInput.style.opacity === '1') {
            searchInput.style.opacity = '0';
            searchInput.style.pointerEvents = 'none';
            searchInput.blur();
        }
    });

    searchInput.addEventListener('input', debounce((e) => {
        performSearch(e.target.value);
    }, 300));
}

function performSearch(query) {
    if (!query) return;

    const searchableElements = document.querySelectorAll('.card-title, .small-box p, .info-box-text');
    const results = [];

    searchableElements.forEach(el => {
        if (el.textContent.toLowerCase().includes(query.toLowerCase())) {
            results.push(el);
            el.classList.add('search-highlight');
        } else {
            el.classList.remove('search-highlight');
        }
    });

    if (results.length > 0) {
        showNotification(`找到 ${results.length} 个匹配项`, '搜索结果');
        results[0].scrollIntoView({ behavior: 'smooth', block: 'center' });
    } else {
        showNotification('未找到匹配项', '搜索结果');
    }
}

function toggleFullscreen() {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen().catch(err => {
            showNotification('无法进入全屏模式', '错误');
        });
    } else {
        document.exitFullscreen();
    }
}

function toggleTheme() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', newTheme);
    localStorage.setItem('adminTheme', newTheme);
    showNotification(`已切换到${newTheme === 'dark' ? '深色' : '浅色'}模式`, '主题切换');
}

function showExportMenu() {
    const menu = document.createElement('div');
    menu.className = 'export-menu dropdown-menu show';
    menu.style.cssText = `
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 10000;
        min-width: 200px;
    `;
    menu.innerHTML = `
        <h6 class="dropdown-header">导出数据</h6>
        <a class="dropdown-item" href="#" onclick="exportData('csv'); return false;">
            <i class="fas fa-file-csv mr-2"></i>导出为 CSV
        </a>
        <a class="dropdown-item" href="#" onclick="exportData('excel'); return false;">
            <i class="fas fa-file-excel mr-2"></i>导出为 Excel
        </a>
        <a class="dropdown-item" href="#" onclick="exportData('json'); return false;">
            <i class="fas fa-file-code mr-2"></i>导出为 JSON
        </a>
        <div class="dropdown-divider"></div>
        <a class="dropdown-item" href="#" onclick="this.closest('.export-menu').remove(); return false;">
            <i class="fas fa-times mr-2"></i>关闭
        </a>
    `;

    document.body.appendChild(menu);

    setTimeout(() => {
        if (!menu.matches(':hover')) {
            menu.remove();
        }
    }, 3000);
}

function showHelpModal() {
    const modal = document.createElement('div');
    modal.className = 'modal fade';
    modal.id = 'helpModal';
    modal.innerHTML = `
        <div class="modal-dialog">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title"><i class="fas fa-keyboard mr-2"></i>键盘快捷键</h5>
                    <button type="button" class="close" data-dismiss="modal">
                        <span>&times;</span>
                    </button>
                </div>
                <div class="modal-body">
                    <table class="table table-sm">
                        <thead>
                            <tr>
                                <th>快捷键</th>
                                <th>功能</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr><td><kbd>R</kbd></td><td>刷新数据</td></tr>
                            <tr><td><kbd>F</kbd></td><td>全屏模式</td></tr>
                            <tr><td><kbd>T</kbd></td><td>切换主题</td></tr>
                            <tr><td><kbd>E</kbd></td><td>导出数据</td></tr>
                            <tr><td><kbd>?</kbd></td><td>显示帮助</td></tr>
                            <tr><td><kbd>Ctrl+K</kbd></td><td>搜索</td></tr>
                            <tr><td><kbd>Esc</kbd></td><td>关闭弹窗</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(modal);
    const bsModal = new bootstrap.Modal(modal);
    bsModal.show();

    modal.addEventListener('hidden.bs.modal', () => {
        modal.remove();
    });
}

function showNotification(title, message) {
    const notification = document.createElement('div');
    notification.className = 'ux-notification alert alert-info';
    notification.innerHTML = `<strong>${title}</strong> ${message}`;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 10000;
        min-width: 250px;
        animation: slideInRight 0.3s ease;
    `;

    document.body.appendChild(notification);

    setTimeout(() => {
        notification.classList.add('fade');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showNotification('已复制到剪贴板', '');
    }).catch(err => {
        console.error('复制失败:', err);
        showNotification('复制失败', '请手动复制');
    });
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

document.addEventListener('DOMContentLoaded', function() {
    if (document.getElementById('dashboardContent')) {
        initUXEnhancements();
    }
});

window.uxEnhancements = uxEnhancements;
window.showNotification = showNotification;
window.toggleFullscreen = toggleFullscreen;
window.toggleTheme = toggleTheme;
window.showExportMenu = showExportMenu;
window.showHelpModal = showHelpModal;
