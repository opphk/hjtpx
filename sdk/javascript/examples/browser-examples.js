(function() {
  'use strict';

  const CaptchaBrowserExamples = {
    simpleSliderExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) return;

      fetch('http://localhost:8080/api/v1/captcha/slider?width=320&height=160')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            const captcha = data.data;
            container.innerHTML = `
              <div class="captcha-slider">
                <img src="${captcha.image_url}" alt="Background" />
                <input type="range" id="slider-input" min="0" max="280" value="0" />
                <button id="verify-btn">Verify</button>
              </div>
            `;

            document.getElementById('verify-btn').addEventListener('click', function() {
              const x = document.getElementById('slider-input').value;
              fetch('http://localhost:8080/api/v1/captcha/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  session_id: captcha.session_id,
                  type: 'slider',
                  x: parseInt(x),
                  y: captcha.secret_y
                })
              }).then(response => response.json())
                .then(result => {
                  alert(result.data.success ? 'Success!' : 'Failed');
                });
            });
          }
        });
    },

    clickCaptchaExample: function() {
      const container = document.getElementById('captcha-container');
      if (!container) return;

      fetch('http://localhost:8080/api/v1/captcha/click?mode=number&points=3')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            const captcha = data.data;
            container.innerHTML = `
              <div class="captcha-click">
                <img src="${captcha.image_url}" alt="Captcha" style="cursor: crosshair;" />
                <p>Hint: ${captcha.hint}</p>
                <button id="verify-click-btn">Verify</button>
              </div>
            `;

            const clicks = [];
            container.querySelector('img').addEventListener('click', function(e) {
              const rect = this.getBoundingClientRect();
              clicks.push([e.clientX - rect.left, e.clientY - rect.top]);
            });

            document.getElementById('verify-click-btn').addEventListener('click', function() {
              fetch('http://localhost:8080/api/v1/captcha/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  session_id: captcha.session_id,
                  type: 'click',
                  points: clicks,
                  click_sequence: captcha.hint_order
                })
              }).then(response => response.json())
                .then(result => {
                  alert(result.data.success ? 'Success!' : 'Failed');
                });
            });
          }
        });
    }
  };

  if (typeof window !== 'undefined') {
    window.CaptchaBrowserExamples = CaptchaBrowserExamples;
  }

})();
