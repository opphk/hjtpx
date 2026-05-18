document.addEventListener('DOMContentLoaded', function() {
    const cssSwitchBtn = document.getElementById('cssSwitchBtn');
    if (cssSwitchBtn) {
        loadCSSSource();
        cssSwitchBtn.addEventListener('click', toggleCSSSource);
    }
});

function loadCSSSource() {
    fetch('/api/v1/admin/css-source', {
        headers: {
            'Authorization': 'Bearer ' + auth.getToken()
        }
    })
    .then(res => res.json())
    .then(data => {
        if (data.code === 0) {
            updateCSSSwitchUI(data.data.source);
        }
    })
    .catch(err => console.error('加载CSS来源失败:', err));
}

function toggleCSSSource() {
    const currentSource = document.getElementById('cssSwitchBtn').dataset.source;
    const newSource = currentSource === 'cdn' ? 'local' : 'cdn';

    fetch('/api/v1/admin/css-source', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + auth.getToken()
        },
        body: JSON.stringify({ source: newSource })
    })
    .then(res => res.json())
    .then(data => {
        if (data.code === 0) {
            updateCSSSwitchUI(data.data.source);
            alert('CSS来源已切换为: ' + (data.data.source === 'cdn' ? 'CDN' : '本地'));
            location.reload();
        }
    })
    .catch(err => console.error('切换CSS来源失败:', err));
}

function updateCSSSwitchUI(source) {
    const btn = document.getElementById('cssSwitchBtn');
    if (btn) {
        btn.dataset.source = source;
        btn.innerHTML = source === 'cdn'
            ? '<i class="fas fa-globe me-1"></i> CDN CSS'
            : '<i class="fas fa-server me-1"></i> 本地 CSS';
        btn.className = source === 'cdn' ? 'btn btn-outline-info btn-sm' : 'btn btn-info btn-sm';
    }
}