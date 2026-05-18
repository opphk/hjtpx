const MFASetup = {
    currentStep: 1,
    selectedMethod: 'totp',
    totpSecret: '',
    backupCodes: [],
    smsCode: '',
    emailCode: '',

    init() {
        this.bindEvents();
        this.updateStepIndicator();
    },

    bindEvents() {
        const setupPage = document.getElementById('step1');
        if (setupPage) {
            this.initSetupPage();
        }

        const verifyPage = document.querySelector('.verify-container');
        if (verifyPage) {
            this.initVerifyPage();
        }
    },

    initSetupPage() {
        document.querySelectorAll('.method-card').forEach(card => {
            card.addEventListener('click', () => {
                document.querySelectorAll('.method-card').forEach(c => c.classList.remove('selected'));
                card.classList.add('selected');
                this.selectedMethod = card.dataset.method;
            });
        });

        document.getElementById('nextStep1')?.addEventListener('click', () => this.goToStep(2));
        document.getElementById('prevStep2')?.addEventListener('click', () => this.goToStep(1));
        document.getElementById('nextStep2')?.addEventListener('click', () => this.goToStep(3));
        document.getElementById('prevStep3')?.addEventListener('click', () => this.goToStep(2));

        document.getElementById('sendSmsCode')?.addEventListener('click', () => this.sendSmsCode());
        document.getElementById('sendEmailCode')?.addEventListener('click', () => this.sendEmailCode());
        document.getElementById('enableMFA')?.addEventListener('click', () => this.enableMFA());
    },

    initVerifyPage() {
        document.querySelectorAll('.method-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.method-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.switchVerifyMethod(btn.dataset.method);
            });
        });

        document.getElementById('resendSms')?.addEventListener('click', () => this.resendSms());
        document.getElementById('resendEmail')?.addEventListener('click', () => this.resendEmail());
        document.getElementById('verifyBtn')?.addEventListener('click', () => this.verify());
    },

    goToStep(step) {
        this.currentStep = step;
        this.updateStepIndicator();

        document.querySelectorAll('.step-content').forEach(el => el.style.display = 'none');
        document.getElementById(`step${step}`).style.display = 'block';

        if (step === 2) {
            this.showMethodSetup();
        }

        if (step === 3) {
            this.generateBackupCodes();
        }
    },

    updateStepIndicator() {
        document.querySelectorAll('.step-indicator .step').forEach((step, index) => {
            step.classList.remove('active', 'completed');
            if (index + 1 < this.currentStep) {
                step.classList.add('completed');
            } else if (index + 1 === this.currentStep) {
                step.classList.add('active');
            }
        });
    },

    showMethodSetup() {
        document.getElementById('totpSetup').style.display = 'none';
        document.getElementById('smsSetup').style.display = 'none';
        document.getElementById('emailSetup').style.display = 'none';

        const setupId = this.selectedMethod + 'Setup';
        const setupEl = document.getElementById(setupId);
        if (setupEl) {
            setupEl.style.display = 'block';
        }

        if (this.selectedMethod === 'totp') {
            this.generateTOTP();
        }
    },

    async generateTOTP() {
        try {
            const response = await fetch('/api/v1/mfa/totp/generate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    account_name: 'user@example.com',
                    issuer: 'HJTPX'
                })
            });
            const data = await response.json();
            if (data.success) {
                this.totpSecret = data.data.secret;
                document.getElementById('totpSecret').value = data.data.secret;
            }
        } catch (error) {
            console.error('Error generating TOTP:', error);
            this.showToast('生成密钥失败，请重试', 'error');
        }
    },

    async sendSmsCode() {
        const phone = document.getElementById('smsPhone').value;
        if (!phone) {
            this.showToast('请输入手机号码', 'error');
            return;
        }

        try {
            const response = await fetch('/api/v1/mfa/sms/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ phone })
            });
            const data = await response.json();
            if (data.success) {
                this.smsCode = data.code;
                this.showToast('验证码已发送到您的手机');
            }
        } catch (error) {
            console.error('Error sending SMS:', error);
            this.showToast('发送失败，请重试', 'error');
        }
    },

    async sendEmailCode() {
        const email = document.getElementById('emailAddress').value;
        if (!email) {
            this.showToast('请输入邮箱地址', 'error');
            return;
        }

        try {
            const response = await fetch('/api/v1/mfa/email/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email })
            });
            const data = await response.json();
            if (data.success) {
                this.emailCode = data.code;
                this.showToast('验证码已发送到您的邮箱');
            }
        } catch (error) {
            console.error('Error sending email:', error);
            this.showToast('发送失败，请重试', 'error');
        }
    },

    generateBackupCodes() {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
        this.backupCodes = [];
        for (let i = 0; i < 10; i++) {
            let code = '';
            for (let j = 0; j < 8; j++) {
                code += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            this.backupCodes.push(code);
        }

        const container = document.getElementById('backupCodes');
        container.innerHTML = '';
        this.backupCodes.forEach(code => {
            const div = document.createElement('div');
            div.className = 'bg-light p-2 text-center font-mono border rounded';
            div.textContent = code;
            container.appendChild(div);
        });

        document.getElementById('backupCodesSection').style.display = 'block';
    },

    async enableMFA() {
        const code = document.getElementById('verifyCode').value;
        if (!code) {
            this.showToast('请输入验证码', 'error');
            return;
        }

        try {
            let response;
            if (this.selectedMethod === 'totp') {
                response = await fetch('/api/v1/mfa/totp/verify', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ secret: this.totpSecret, code })
                });
            } else {
                response = await fetch('/api/v1/mfa/code/verify', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ code })
                });
            }

            const data = await response.json();
            if (data.success) {
                await fetch('/api/v1/mfa/enable', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        mfa_type: this.selectedMethod,
                        phone: document.getElementById('smsPhone')?.value,
                        email: document.getElementById('emailAddress')?.value
                    })
                });

                this.showToast('MFA 启用成功！');
                setTimeout(() => window.location.href = '/', 1500);
            }
        } catch (error) {
            console.error('Error enabling MFA:', error);
            this.showToast('启用失败，请重试', 'error');
        }
    },

    switchVerifyMethod(method) {
        document.getElementById('totpVerify').style.display = 'none';
        document.getElementById('smsVerify').style.display = 'none';
        document.getElementById('emailVerify').style.display = 'none';
        document.getElementById('backupVerify').style.display = 'none';

        const verifyId = method + 'Verify';
        const verifyEl = document.getElementById(verifyId);
        if (verifyEl) {
            verifyEl.style.display = 'block';
        }
    },

    async resendSms() {
        const btn = document.getElementById('resendSms');
        btn.disabled = true;
        let countdown = 60;
        const countdownEl = document.getElementById('smsCountdown');
        countdownEl.classList.remove('d-none');

        const interval = setInterval(() => {
            countdown--;
            countdownEl.textContent = `(${countdown}秒后可重发)`;
            if (countdown <= 0) {
                clearInterval(interval);
                btn.disabled = false;
                countdownEl.classList.add('d-none');
            }
        }, 1000);

        await this.sendSmsCode();
    },

    async resendEmail() {
        const btn = document.getElementById('resendEmail');
        btn.disabled = true;
        let countdown = 60;
        const countdownEl = document.getElementById('emailCountdown');
        countdownEl.classList.remove('d-none');

        const interval = setInterval(() => {
            countdown--;
            countdownEl.textContent = `(${countdown}秒后可重发)`;
            if (countdown <= 0) {
                clearInterval(interval);
                btn.disabled = false;
                countdownEl.classList.add('d-none');
            }
        }, 1000);

        await this.sendEmailCode();
    },

    async verify() {
        const activeMethod = document.querySelector('.method-btn.active').dataset.method;
        let code;

        switch (activeMethod) {
            case 'totp':
                code = document.getElementById('totpCode').value;
                break;
            case 'sms':
                code = document.getElementById('smsCode').value;
                break;
            case 'email':
                code = document.getElementById('emailCode').value;
                break;
            case 'backup':
                code = document.getElementById('backupCode').value;
                break;
        }

        if (!code) {
            this.showToast('请输入验证码', 'error');
            return;
        }

        try {
            let endpoint = '/api/v1/mfa/code/verify';
            if (activeMethod === 'totp') {
                endpoint = '/api/v1/mfa/totp/verify';
            } else if (activeMethod === 'backup') {
                endpoint = '/api/v1/mfa/backup-codes/verify';
            }

            const response = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ code })
            });
            const data = await response.json();

            if (data.success) {
                this.showToast('验证成功！');
                setTimeout(() => window.location.href = '/', 1000);
            } else {
                this.showToast(data.error || '验证失败', 'error');
            }
        } catch (error) {
            console.error('Error verifying:', error);
            this.showToast('验证失败，请重试', 'error');
        }
    },

    showToast(message, type = 'success') {
        const toastEl = document.getElementById('toast');
        const messageEl = document.getElementById('toastMessage');
        messageEl.textContent = message;

        if (type === 'error') {
            toastEl.classList.add('bg-danger', 'text-white');
        } else {
            toastEl.classList.remove('bg-danger', 'text-white');
        }

        const toast = new bootstrap.Toast(toastEl);
        toast.show();
    }
};

document.addEventListener('DOMContentLoaded', () => {
    MFASetup.init();
});
