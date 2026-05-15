import React, { useState } from 'react';
import SliderCaptcha from './SliderCaptcha';
import './CaptchaX.css';

const CaptchaX = ({
  type = 'slider',
  serviceOptions = {},
  onSuccess,
  onError,
  onRefresh,
  width = 320,
  height = 160,
}) => {
  const [verified, setVerified] = useState(false);
  const [captchaToken, setCaptchaToken] = useState(null);

  const handleSuccess = (result) => {
    setVerified(true);
    setCaptchaToken(result.token || result.captcha_id);
    onSuccess?.(result);
  };

  const handleError = (error) => {
    setVerified(false);
    setCaptchaToken(null);
    onError?.(error);
  };

  const handleRefresh = () => {
    setVerified(false);
    setCaptchaToken(null);
    onRefresh?.();
  };

  const renderCaptcha = () => {
    switch (type) {
      case 'slider':
        return (
          <SliderCaptcha
            serviceOptions={serviceOptions}
            onSuccess={handleSuccess}
            onError={handleError}
            onRefresh={handleRefresh}
            width={width}
            height={height}
          />
        );
      case 'click':
        return (
          <div className="captcha-not-implemented">
            点选验证码即将上线
          </div>
        );
      case 'puzzle':
        return (
          <div className="captcha-not-implemented">
            拼图验证码即将上线
          </div>
        );
      default:
        return (
          <div className="captcha-not-implemented">
            不支持的验证码类型
          </div>
        );
    }
  };

  return (
    <div className="captcha-wrapper">
      {renderCaptcha()}
      {verified && captchaToken && (
        <input
          type="hidden"
          name="captchaToken"
          value={captchaToken}
        />
      )}
    </div>
  );
};

export default CaptchaX;
