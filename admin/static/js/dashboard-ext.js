// 仪表盘扩展功能
let currentTheme = 'default';
let isEditingLayout = false;
let widgets = [];
let notificationFilter = 'all';

// 主题配置
const themes = {
    default: {
        bodyClass: '',
        navbarClass: 'navbar-light bg-white',
        sidebarClass: 'bg-light'
    },
    dark: {
        bodyClass: 'bg-dark text-light',
        navbarClass: 'navbar-dark bg-dark',
        sidebarClass: 'bg-dark',
        cardClass: 'bg-dark border-secondary'
    },
    blue: {
        bodyClass: '',
        navbarClass: 'navbar-light bg-primary text-white',
        sidebarClass: 'bg-primary',
        cardClass: 'bg-light'
    },
    green: {
        bodyClass: '',
        navbarClass: 'navbar-light bg-success text-white',
        sidebarClass: 'bg-success',
        cardClass: 'bg-light'
    }
};

// 初始化扩展功能
document.addEventListener('DOMContentLoaded', function() {
    initTheme();
    initNotifications();
    initLayoutEditor();
    checkUnreadCount();
    setInterval(checkUnreadCount, 30000); // 每30秒检查一次
});

// ==================== 主题功能 ====================
function initTheme() {
    const savedTheme = localStorage.getItem('dashboardTheme') || 'default';
    setTheme(savedTheme, false);
}

function setTheme(theme, save = true) {
    currentTheme = theme;
    if (save) {
        localStorage.setItem('dashboardTheme', theme);
        // 保存到后端
        fetch('/admin/api/dashboard/theme', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ theme: theme })
        });
    }
    applyTheme(theme);
}

function applyTheme(theme) {
    const config = themes[theme] || themes.default;
    
    // 移除之前的主题类
    document.body.className = document.body.className.replace(/theme-\w+/g, '');
    
    // 应用新主题
    if (theme !== 'default') {
        document.body.classList.add(`theme-${theme}`);
    }
    
    // 更新主题下拉菜单选中状态
    document.querySelectorAll('#themeDropdown .dropdown-item').forEach(item => {
        item.classList.remove('active');
        if (item.onclick.toString().includes(`'${theme}'`)) {
            item.classList.add('active');
        }
    });
}

// ==================== 通知功能 ====================
function initNotifications() {
    document.getElementById('notificationsBtn').addEventListener('click', openNotifications);
    document.getElementById('filterAllBtn').addEventListener('click', () => filterNotifications('all'));
    document.getElementById('filterUnreadBtn').addEventListener('click', () => filterNotifications('unread'));
    document.getElementById('markAllReadBtn').addEventListener('click', markAllAsRead);
}

async function checkUnreadCount() {
    try {
        const response = await fetch('/admin/api/notifications/unread-count');
        const result = await response.json();
        if (result.code === 0) {
            const badge = document.getElementById('notificationBadge');
            if (result.data.count > 0) {
                badge.textContent = result.data.count;
                badge.style.display = 'block';
            } else {
                badge.style.display = 'none';
            }
        }
    } catch (error) {
        console.error('检查未读通知失败:', error);
    }
}

async function openNotifications() {
    $('#notificationsModal').modal('show');
    await loadNotifications();
}

async function loadNotifications() {
    try {
        const url = notificationFilter === 'unread' 
            ? '/admin/api/notifications?only_unread=true' 
            : '/admin/api/notifications';
        
        const response = await fetch(url);
        const result = await response.json();
        
        if (result.code === 0) {
            renderNotifications(result.data.items);
        }
    } catch (error) {
        console.error('加载通知失败:', error);
    }
}

function renderNotifications(notifications) {
    const container = document.getElementById('notificationsList');
    
    if (!notifications || notifications.length === 0) {
        container.innerHTML = '<div class="text-center text-muted py-4">暂无通知</div>';
        return;
    }
    
    container.innerHTML = notifications.map(notification => `
        <div class="list-group-item ${notification.is_read ? '' : 'list-group-item-warning'}">
            <div class="d-flex w-100 justify-content-between">
                <h6 class="mb-1">
                    <i class="fas ${getNotificationIcon(notification.type)}"></i>
                    ${escapeHtml(notification.title)}
                </h6>
                <small>${formatTime(notification.created_at)}</small>
            </div>
            <p class="mb-1">${escapeHtml(notification.content)}</p>
            <div class="d-flex justify-content-between align-items-center">
                <small class="text-muted">${notification.is_read ? '已读' : '未读'}</small>
                <div class="btn-group btn-group-sm">
                    ${!notification.is_read ? `
                        <button class="btn btn-outline-primary" onclick="markAsRead(${notification.id})">
                            <i class="fas fa-check"></i> 标为已读
                        </button>
                    ` : ''}
                    <button class="btn btn-outline-danger" onclick="deleteNotification(${notification.id})">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

function getNotificationIcon(type) {
    const icons = {
        info: 'fa-info-circle text-info',
        warning: 'fa-exclamation-triangle text-warning',
        error: 'fa-exclamation-circle text-danger',
        success: 'fa-check-circle text-success'
    };
    return icons[type] || icons.info;
}

function filterNotifications(filter) {
    notificationFilter = filter;
    document.getElementById('filterAllBtn').className = filter === 'all' ? 'btn btn-sm btn-primary' : 'btn btn-sm btn-outline-secondary';
    document.getElementById('filterUnreadBtn').className = filter === 'unread' ? 'btn btn-sm btn-primary' : 'btn btn-sm btn-outline-secondary';
    loadNotifications();
}

async function markAsRead(id) {
    try {
        await fetch(`/admin/api/notifications/${id}/read`, { method: 'PUT' });
        checkUnreadCount();
        loadNotifications();
    } catch (error) {
        console.error('标记已读失败:', error);
    }
}

async function markAllAsRead() {
    try {
        await fetch('/admin/api/notifications/read-all', { method: 'PUT' });
        checkUnreadCount();
        loadNotifications();
    } catch (error) {
        console.error('全部标记已读失败:', error);
    }
}

async function deleteNotification(id) {
    if (!confirm('确定要删除这条通知吗？')) return;
    
    try {
        await fetch(`/admin/api/notifications/${id}`, { method: 'DELETE' });
        checkUnreadCount();
        loadNotifications();
    } catch (error) {
        console.error('删除通知失败:', error);
    }
}

// ==================== 布局编辑功能 ====================
function initLayoutEditor() {
    document.getElementById('editLayoutBtn').addEventListener('click', openLayoutEditor);
    document.getElementById('saveLayoutBtn').addEventListener('click', saveLayout);
}

async function openLayoutEditor() {
    $('#layoutModal').modal('show');
    await loadWidgets();
    initDragAndDrop();
}

async function loadWidgets() {
    try {
        const response = await fetch('/admin/api/dashboard/widgets');
        const result = await response.json();
        
        if (result.code === 0) {
            widgets = result.data;
            renderWidgetContainer();
        }
    } catch (error) {
        console.error('加载组件失败:', error);
        // 使用默认组件
        loadDefaultWidgets();
    }
}

function loadDefaultWidgets() {
    widgets = [
        { widget_type: 'stat', title: '今日验证', position_x: 0, position_y: 0, width: 3, height: 1 },
        { widget_type: 'stat', title: '通过率', position_x: 3, position_y: 0, width: 3, height: 1 },
        { widget_type: 'stat', title: '拦截率', position_x: 6, position_y: 0, width: 3, height: 1 },
        { widget_type: 'stat', title: '平均响应', position_x: 9, position_y: 0, width: 3, height: 1 },
        { widget_type: 'chart', title: '24小时趋势', position_x: 0, position_y: 1, width: 8, height: 3 },
        { widget_type: 'list', title: '最近验证', position_x: 8, position_y: 1, width: 4, height: 3 }
    ];
    renderWidgetContainer();
}

function renderWidgetContainer() {
    const container = document.getElementById('widgetContainer');
    container.innerHTML = widgets.map((widget, index) => `
        <div class="col-md-${widget.width * 2} mb-3 widget-card" data-index="${index}" draggable="true">
            <div class="card ${themes[currentTheme].cardClass || ''}">
                <div class="card-header d-flex justify-content-between align-items-center">
                    <span>${widget.title}</span>
                    <div class="btn-group btn-group-sm">
                        <button class="btn btn-outline-secondary" onclick="editWidget(${index})">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-outline-danger" onclick="removeWidget(${index})">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                </div>
                <div class="card-body" style="min-height: 100px;">
                    <div class="text-muted text-center">
                        <i class="fas ${getWidgetIcon(widget.widget_type)} fa-2x"></i>
                        <p class="mt-2">${widget.widget_type} 组件</p>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

function getWidgetIcon(type) {
    const icons = {
        stat: 'fa-chart-bar',
        chart: 'fa-chart-line',
        list: 'fa-list',
        progress: 'fa-tasks'
    };
    return icons[type] || 'fa-box';
}

function initDragAndDrop() {
    const widgetItems = document.querySelectorAll('.widget-item');
    const widgetCards = document.querySelectorAll('.widget-card');
    
    widgetItems.forEach(item => {
        item.addEventListener('dragstart', handleDragStart);
    });
    
    widgetCards.forEach(card => {
        card.addEventListener('dragstart', handleDragStart);
        card.addEventListener('dragover', handleDragOver);
        card.addEventListener('drop', handleDrop);
    });
    
    document.getElementById('widgetContainer').addEventListener('dragover', handleDragOver);
    document.getElementById('widgetContainer').addEventListener('drop', handleDrop);
}

let draggedItem = null;
let draggedFromSidebar = false;

function handleDragStart(e) {
    draggedItem = this;
    draggedFromSidebar = this.classList.contains('widget-item');
    e.dataTransfer.effectAllowed = 'move';
}

function handleDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
}

function handleDrop(e) {
    e.preventDefault();
    
    if (draggedFromSidebar) {
        // 从侧边栏添加新组件
        const widgetType = draggedItem.dataset.widgetType;
        const widgetTitle = draggedItem.dataset.widgetTitle;
        
        widgets.push({
            widget_type: widgetType,
            title: widgetTitle,
            position_x: widgets.length,
            position_y: Math.floor(widgets.length / 4),
            width: 3,
            height: 1
        });
        
        renderWidgetContainer();
        initDragAndDrop();
    }
    
    draggedItem = null;
}

function editWidget(index) {
    const widget = widgets[index];
    const newTitle = prompt('组件标题:', widget.title);
    if (newTitle) {
        widget.title = newTitle;
        renderWidgetContainer();
    }
}

function removeWidget(index) {
    if (confirm('确定要删除这个组件吗？')) {
        widgets.splice(index, 1);
        renderWidgetContainer();
    }
}

async function saveLayout() {
    try {
        await fetch('/admin/api/dashboard/widgets', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(widgets)
        });
        
        $('#layoutModal').modal('hide');
        alert('布局保存成功！');
        location.reload();
    } catch (error) {
        console.error('保存布局失败:', error);
        alert('保存失败，请重试');
    }
}

// ==================== 工具函数 ====================
function escapeHtml(text) {
    if (text === null || text === undefined) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

function formatTime(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 60000) return '刚刚';
    if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前';
    if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前';
    if (diff < 604800000) return Math.floor(diff / 86400000) + '天前';
    
    return date.toLocaleDateString('zh-CN');
}
