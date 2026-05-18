document.addEventListener('DOMContentLoaded', function() {
    console.log('管理后台已加载');
    
    initAdminPageProgress();
    initAdminPerformanceMonitoring();
    initAdminEnhancedAnimations();
    
    const sidebarLinks = document.querySelectorAll('.sidebar-nav a');
    sidebarLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            sidebarLinks.forEach(l => l.classList.remove('active'));
            this.classList.add('active');
        });
    });
    
    const logoutBtn = document.querySelector('.logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', function(e) {
            e.preventDefault();
            if (confirm('确定要退出登录吗？')) {
                window.location.href = '/admin/logout';
            }
        });
    }
    
    const statCards = document.querySelectorAll('.stat-card');
    statCards.forEach(card => {
        card.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-5px)';
            this.style.boxShadow = '0 10px 25px rgba(0, 0, 0, 0.15)';
        });
        card.addEventListener('mouseleave', function() {
            this.style.transform = 'translateY(0)';
            this.style.boxShadow = '0 1px 3px rgba(0, 0, 0, 0.1)';
        });
    });
    
    initDataTableEnhancements();
    initFormEnhancements();
});

function initAdminPageProgress() {
    const progressBar = document.createElement('div');
    progressBar.id = 'admin-page-progress';
    progressBar.innerHTML = '<div class="admin-progress-bar"></div>';
    progressBar.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:3px;z-index:10000;background:rgba(0,123,255,0.1);';
    const innerBar = progressBar.querySelector('.admin-progress-bar');
    innerBar.style.cssText = 'height:100%;background:linear-gradient(90deg,#007bff,#0056b3);width:0%;transition:width 0.3s ease;';
    document.body.appendChild(progressBar);
    
    let progress = 0;
    const interval = setInterval(() => {
        progress += Math.random() * 20;
        if (progress >= 90) {
            clearInterval(interval);
            innerBar.style.width = '90%';
        } else {
            innerBar.style.width = progress + '%';
        }
    }, 80);
    
    window.addEventListener('load', function() {
        clearInterval(interval);
        innerBar.style.width = '100%';
        setTimeout(() => {
            progressBar.style.opacity = '0';
            setTimeout(() => progressBar.remove(), 300);
        }, 200);
    });
}

function initAdminPerformanceMonitoring() {
    if (!window.PerformanceObserver) return;
    
    try {
        const observer = new PerformanceObserver((list) => {
            list.getEntries().forEach((entry) => {
                if (entry.entryType === 'navigation') {
                    const loadTime = Math.round(entry.loadEventEnd - entry.startTime);
                    console.log('管理后台加载时间:', loadTime + 'ms');
                }
            });
        });
        observer.observe({ entryTypes: ['navigation'] });
    } catch (e) {
        console.log('性能监控不可用');
    }
}

function initAdminEnhancedAnimations() {
    const animStyle = document.createElement('style');
    animStyle.textContent = `
        .stat-card {
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }
        .stat-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
        }
        .nav-link {
            transition: all 0.2s ease;
        }
        .btn {
            transition: all 0.2s ease;
        }
        .card {
            transition: all 0.3s ease;
        }
        .fade-in-up {
            animation: fadeInUp 0.4s ease forwards;
        }
        @keyframes fadeInUp {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .loading-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255,255,255,0.9);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 100;
            border-radius: inherit;
        }
        .loading-spinner {
            width: 30px;
            height: 30px;
            border: 3px solid rgba(0,123,255,0.2);
            border-top-color: #007bff;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    `;
    document.head.appendChild(animStyle);
}

function initDataTableEnhancements() {
    document.querySelectorAll('table').forEach(table => {
        table.classList.add('table-hover');
        const rows = table.querySelectorAll('tbody tr');
        rows.forEach((row, index) => {
            row.style.animationDelay = (index * 30) + 'ms';
        });
    });
}

function initFormEnhancements() {
    document.querySelectorAll('form').forEach(form => {
        const inputs = form.querySelectorAll('input, select, textarea');
        inputs.forEach(input => {
            input.addEventListener('focus', function() {
                this.closest('.form-group')?.classList.add('focused');
            });
            input.addEventListener('blur', function() {
                this.closest('.form-group')?.classList.remove('focused');
            });
        });
    });
}
