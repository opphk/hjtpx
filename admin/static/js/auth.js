
document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('loginForm');
    const errorMessage = document.getElementById('errorMessage');
    const passwordToggle = document.getElementById('passwordToggle');
    const loginBtn = document.getElementById('loginBtn');
    const loginText = document.getElementById('loginText');
    const loginSpinner = document.getElementById('loginSpinner');
    const passwordInput = document.getElementById('password');
    const usernameInput = document.getElementById('username');
    const rememberMeCheckbox = document.getElementById('rememberMe');

    if (passwordToggle && passwordInput) {
        passwordToggle.addEventListener('click', function() {
            const type = passwordInput.getAttribute('type') === 'password' ? 'text' : 'password';
            passwordInput.setAttribute('type', type);
            const icon = this.querySelector('i');
            icon.classList.toggle('fa-eye');
            icon.classList.toggle('fa-eye-slash');
        });
    }

    if (rememberMeCheckbox) {
        const savedUsername = localStorage.getItem('savedUsername');
        if (savedUsername) {
            usernameInput.value = savedUsername;
            rememberMeCheckbox.checked = true;
        }
    }

    function showError(message) {
        if (errorMessage) {
            errorMessage.querySelector('span').textContent = message;
            errorMessage.classList.add('show');
            setTimeout(() => {
                errorMessage.classList.remove('show');
            }, 5000);
        }
    }

    function hideError() {
        if (errorMessage) {
            errorMessage.classList.remove('show');
        }
    }

    function setLoading(isLoading) {
        if (loginBtn && loginText && loginSpinner) {
            loginBtn.disabled = isLoading;
            if (isLoading) {
                loginText.style.display = 'none';
                loginSpinner.style.display = 'inline';
            } else {
                loginText.style.display = 'inline';
                loginSpinner.style.display = 'none';
            }
        }
    }

    if (loginForm) {
        loginForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            hideError();

            const username = usernameInput.value.trim();
            const password = passwordInput.value;

            if (!username) {
                showError('请输入用户名');
                usernameInput.focus();
                return;
            }

            if (!password) {
                showError('请输入密码');
                passwordInput.focus();
                return;
            }

            setLoading(true);

            try {
                const response = await fetch('/admin/api/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        username: username,
                        password: password,
                        remember: rememberMeCheckbox ? rememberMeCheckbox.checked : false
                    })
                });

                const data = await response.json();

                if (response.ok && data.success) {
                    if (rememberMeCheckbox && rememberMeCheckbox.checked) {
                        localStorage.setItem('savedUsername', username);
                    } else {
                        localStorage.removeItem('savedUsername');
                    }

                    sessionStorage.setItem('adminUser', JSON.stringify(data.user));

                    window.location.href = data.redirect || '/admin/';
                } else {
                    showError(data.message || '用户名或密码错误');
                }
            } catch (error) {
                console.error('登录请求失败:', error);
                showError('网络错误，请稍后重试');
            } finally {
                setLoading(false);
            }
        });
    }

    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', async function(e) {
            e.preventDefault();

            if (!confirm('确定要退出登录吗？')) {
                return;
            }

            try {
                const response = await fetch('/admin/api/logout', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });

                sessionStorage.removeItem('adminUser');
                localStorage.removeItem('savedUsername');

                window.location.href = '/admin/login';
            } catch (error) {
                console.error('登出失败:', error);
                sessionStorage.removeItem('adminUser');
                window.location.href = '/admin/login';
            }
        });
    }
});

function getCurrentUser() {
    const userStr = sessionStorage.getItem('adminUser');
    return userStr ? JSON.parse(userStr) : null;
}

function isAuthenticated() {
    return sessionStorage.getItem('adminUser') !== null;
}

function requireAuth() {
    if (!isAuthenticated()) {
        window.location.href = '/admin/login';
        return false;
    }
    return true;
}

function updateUserDisplay(username) {
    const usernameDisplay = document.getElementById('usernameDisplay');
    const userAvatar = document.getElementById('userAvatar');

    if (usernameDisplay && username) {
        usernameDisplay.textContent = username;
    }

    if (userAvatar && username) {
        userAvatar.textContent = username.charAt(0).toUpperCase();
    }
}

function showToast(message, type = 'info') {
    const toastContainer = document.createElement('div');
    toastContainer.className = 'position-fixed top-0 end-0 p-3';
    toastContainer.style.zIndex = '9999';

    const toastId = 'toast-' + Date.now();
    const bgClass = type === 'success' ? 'bg-success' : type === 'error' ? 'bg-danger' : 'bg-info';

    toastContainer.innerHTML = `
        <div id="${toastId}" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
            <div class="toast-header ${bgClass} text-white">
                <i class="fas ${type === 'success' ? 'fa-check-circle' : type === 'error' ? 'fa-exclamation-circle' : 'fa-info-circle'} me-2"></i>
                <strong class="me-auto">提示</strong>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="toast" aria-label="Close"></button>
            </div>
            <div class="toast-body">
                ${message}
            </div>
        </div>
    `;

    document.body.appendChild(toastContainer);

    const toastElement = document.getElementById(toastId);
    const toast = new bootstrap.Toast(toastElement, {
        delay: 3000
    });
    toast.show();

    toastElement.addEventListener('hidden.bs.toast', function() {
        toastContainer.remove();
    });
}

function showConfirmDialog(title, message, onConfirm) {
    const modalHtml = `
        <div class="modal fade" id="confirmModal" tabindex="-1" aria-labelledby="confirmModalLabel" aria-hidden="true">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="confirmModalLabel">${title}</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        ${message}
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
                        <button type="button" class="btn btn-primary" id="confirmBtn">确定</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    const existingModal = document.getElementById('confirmModal');
    if (existingModal) {
        existingModal.remove();
    }

    document.body.insertAdjacentHTML('beforeend', modalHtml);

    const modalElement = document.getElementById('confirmModal');
    const confirmBtn = document.getElementById('confirmBtn');

    const modal = new bootstrap.Modal(modalElement);

    confirmBtn.addEventListener('click', function() {
        modal.hide();
        if (typeof onConfirm === 'function') {
            onConfirm();
        }
    });

    modalElement.addEventListener('hidden.bs.modal', function() {
        modalElement.remove();
    });

    modal.show();
}

window.Auth = {
    getCurrentUser: getCurrentUser,
    isAuthenticated: isAuthenticated,
    requireAuth: requireAuth,
    updateUserDisplay: updateUserDisplay,
    showToast: showToast,
    showConfirmDialog: showConfirmDialog
};
