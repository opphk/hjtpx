import React from 'react';

const Skeleton = ({ 
  width, 
  height, 
  borderRadius = '4px', 
  style = {},
  'aria-hidden': ariaHidden = true,
  ...props
}) => {
  return (
    <div
      className="skeleton"
      style={{
        width: width || '100%',
        height: height || '16px',
        borderRadius,
        ...style
      }}
      aria-hidden={ariaHidden}
      {...props}
    />
  );
};

export const SkeletonText = ({ 
  lines = 3, 
  lastLineWidth = '60%',
  'aria-label': ariaLabel = '文本加载中',
  ...props
}) => {
  return (
    <div 
      style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton
          key={index}
          height="16px"
          width={index === lines - 1 ? lastLineWidth : '100%'}
        />
      ))}
    </div>
  );
};

export const SkeletonTitle = ({ 
  width = '40%',
  'aria-label': ariaLabel = '标题加载中',
  ...props
}) => {
  return (
    <div
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      <Skeleton height="24px" width={width} />
    </div>
  );
};

export const SkeletonAvatar = ({ 
  size = 48,
  'aria-label': ariaLabel = '头像加载中',
  ...props
}) => {
  return (
    <div
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      <Skeleton width={size} height={size} borderRadius="50%" />
    </div>
  );
};

export const SkeletonCard = ({ 
  showAvatar = false, 
  showImage = false,
  'aria-label': ariaLabel = '卡片内容加载中',
  ...props
}) => {
  return (
    <div 
      className="skeleton-card"
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      {showImage && <div className="skeleton-image" aria-hidden="true" />}
      {showAvatar && (
        <div style={{ display: 'flex', gap: '12px', marginBottom: '16px' }}>
          <SkeletonAvatar aria-hidden="true" />
          <div style={{ flex: 1 }}>
            <SkeletonText lines={2} aria-hidden="true" />
          </div>
        </div>
      )}
      {!showAvatar && <SkeletonTitle aria-hidden="true" />}
      <SkeletonText lines={3} aria-hidden="true" />
    </div>
  );
};

export const SkeletonTable = ({ 
  rows = 5, 
  columns = 4,
  'aria-label': ariaLabel = '表格内容加载中',
  ...props
}) => {
  return (
    <div 
      className="skeleton-card" 
      style={{ padding: 0 }}
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      <div style={{ display: 'flex', gap: '16px', padding: '16px', borderBottom: '1px solid #f0f0f0' }}>
        {Array.from({ length: columns }).map((_, colIndex) => (
          <Skeleton key={colIndex} height="20px" style={{ flex: 1 }} />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div
          key={rowIndex}
          style={{
            display: 'flex',
            gap: '16px',
            padding: '16px',
            borderBottom: rowIndex < rows - 1 ? '1px solid #f0f0f0' : 'none'
          }}
        >
          {Array.from({ length: columns }).map((_, colIndex) => (
            <Skeleton key={colIndex} height="16px" style={{ flex: 1 }} />
          ))}
        </div>
      ))}
    </div>
  );
};

export const SkeletonList = ({ 
  items = 3,
  'aria-label': ariaLabel = '列表内容加载中',
  ...props
}) => {
  return (
    <div 
      style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      {Array.from({ length: items }).map((_, index) => (
        <div
          key={index}
          style={{
            display: 'flex',
            gap: '12px',
            padding: '16px',
            background: 'white',
            borderRadius: '6px',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)'
          }}
        >
          <SkeletonAvatar aria-hidden="true" />
          <div style={{ flex: 1 }}>
            <SkeletonText lines={2} aria-hidden="true" />
          </div>
        </div>
      ))}
    </div>
  );
};

export const SkeletonForm = ({ 
  fields = 3,
  'aria-label': ariaLabel = '表单内容加载中',
  ...props
}) => {
  return (
    <div 
      style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
      role="status"
      aria-label={ariaLabel}
      aria-live="polite"
      aria-busy={true}
      {...props}
    >
      {Array.from({ length: fields }).map((_, index) => (
        <div key={index}>
          <Skeleton height="16px" width="100px" style={{ marginBottom: '8px' }} />
          <Skeleton height="40px" />
        </div>
      ))}
      <Skeleton height="44px" width="200px" style={{ marginTop: '8px' }} />
    </div>
  );
};

export default Skeleton;
