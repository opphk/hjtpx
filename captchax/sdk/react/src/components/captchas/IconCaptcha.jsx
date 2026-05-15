import { useState, useCallback, useMemo, useEffect } from 'react';
import './IconCaptcha.css';

const ICONS = [
  { id: 'car', name: '汽车', path: 'M19 9h-4V3H9v6H5l3 4 3-4zM5 18v2h14v-2H5z' },
  { id: 'house', name: '房子', path: 'M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z' },
  { id: 'star', name: '星星', path: 'M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z' },
  { id: 'heart', name: '爱心', path: 'M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z' },
  { id: 'sun', name: '太阳', path: 'M6.76 4.84l-1.8-1.79-1.41 1.41 1.79 1.79 1.42-1.41zM4 10.5H1v2h3v-2zm9-9.95h-2V3.5h2V.55zm7.45 3.91l-1.41-1.41-1.79 1.79 1.41 1.41 1.79-1.79zm-3.21 13.7l1.79 1.8 1.41-1.41-1.8-1.79-1.4 1.4zM20 10.5v2h3v-2h-3zm-8-5c-3.31 0-6 2.69-6 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6zm-1 16.95h2V19.5h-2v2.95zm-7.45-3.91l1.41 1.41 1.79-1.8-1.41-1.41-1.79 1.8z' },
  { id: 'music', name: '音乐', path: 'M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z' },
  { id: 'phone', name: '手机', path: 'M17 1.01L7 1c-1.1 0-2 .9-2 2v18c0 1.1.9 2 2 2h10c1.1 0 2-.9 2-2V3c0-1.1-.9-1.99-2-1.99zM17 19H7V5h10v14z' },
  { id: 'tree', name: '树木', path: 'M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71L12 2z' }
];

/**
 * @typedef {Object} IconCaptchaProps
 * @property {Object} [data]
 * @property {Function} [onSuccess]
 * @property {Function} [onRefresh]
 * @property {Function} [onError]
 * @property {string} [status='ready']
 * @property {'light' | 'dark'} [theme='light']
 */

/**
 * IconCaptcha - 图标验证码组件
 * 选择正确的图标完成验证
 * @param {IconCaptchaProps} props
 * @returns {JSX.Element}
 */
export const IconCaptcha = ({
  data,
  onSuccess,
  onRefresh,
  onError,
  status = 'ready',
  theme = 'light'
}) => {
  const [selected, setSelected] = useState([]);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState(null);

  const targetIcons = data?.targetIcons || ['car', 'house'];
  const targetNames = data?.targetNames || ['汽车', '房子'];
  const instruction = data?.instruction || `请依次点击: ${targetNames.join(' 和 ')}`;

  const handleIconClick = useCallback((iconId) => {
    if (isVerified) return;

    const newSelected = selected.includes(iconId)
      ? selected.filter(id => id !== iconId)
      : [...selected, iconId];

    setSelected(newSelected);

    if (newSelected.length === targetIcons.length) {
      const isCorrect = targetIcons.every(id => newSelected.includes(id));

      if (isCorrect) {
        setIsVerified(true);
        onSuccess?.({
          type: 'icon',
          selected: newSelected,
          targets: targetIcons
        });
      } else {
        setError('选择错误，请重试');
        setSelected([]);
        onError?.({
          selected: newSelected,
          targets: targetIcons
        });
      }
    }
  }, [isVerified, selected, targetIcons, onSuccess, onError]);

  useEffect(() => {
    if (status === 'ready') {
      setSelected([]);
      setIsVerified(false);
      setError(null);
    }
  }, [status]);

  const shuffledIcons = useMemo(() => {
    const shuffled = [...ICONS];
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    return shuffled;
  }, []);

  const containerClasses = useMemo(() => {
    const classes = [
      'icon-captcha',
      `icon-captcha--${theme}`
    ];

    if (isVerified) classes.push('icon-captcha--verified');

    return classes.join(' ');
  }, [theme, isVerified]);

  return (
    <div className={containerClasses}>
      <div className="icon-captcha__header">
        <span className="icon-captcha__instruction">{instruction}</span>
        <span className="icon-captcha__progress">
          {selected.length} / {targetIcons.length}
        </span>
      </div>

      <div className="icon-captcha__grid">
        {shuffledIcons.map((icon) => {
          const isSelected = selected.includes(icon.id);
          const isTarget = targetIcons.includes(icon.id);

          return (
            <button
              key={icon.id}
              type="button"
              className={`icon-captcha__item ${isSelected ? 'selected' : ''}`}
              onClick={() => handleIconClick(icon.id)}
              aria-label={icon.name}
            >
              <svg viewBox="0 0 24 24" width="32" height="32">
                <path fill="currentColor" d={icon.path} />
              </svg>
              <span className="icon-captcha__item-name">{icon.name}</span>

              {isSelected && (
                <span className="icon-captcha__checkmark">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                  </svg>
                </span>
              )}
            </button>
          );
        })}
      </div>

      <div className="icon-captcha__footer">
        <button
          type="button"
          className="icon-captcha__refresh"
          onClick={() => {
            setSelected([]);
            setError(null);
            onRefresh?.();
          }}
        >
          <svg viewBox="0 0 24 24" width="16" height="16">
            <path fill="currentColor" d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
          </svg>
          换一组
        </button>
      </div>

      {error && (
        <div className="icon-captcha__error">
          {error}
        </div>
      )}
    </div>
  );
};

export default IconCaptcha;
