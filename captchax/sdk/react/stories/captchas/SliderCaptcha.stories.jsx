import React from 'react';
import { SliderCaptcha } from '../../src/components/captchas/SliderCaptcha';

export default {
  title: 'Captchas/SliderCaptcha',
  component: SliderCaptcha,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '滑块验证码组件，拖动滑块到目标位置完成验证'
      }
    }
  },
  tags: ['autodocs']
};

export const Default = {
  args: {
    width: 300,
    height: 150,
    theme: 'light',
    status: 'ready'
  },
  render: (args) => (
    <SliderCaptcha
      {...args}
      onSuccess={(result) => console.log('Success:', result)}
      onError={(error) => console.error('Error:', error)}
      onRefresh={() => console.log('Refreshing...')}
    />
  )
};

export const DarkTheme = {
  args: {
    ...Default.args,
    theme: 'dark'
  }
};

export const WithImages = {
  args: {
    ...Default.args,
    data: {
      targetImage: 'https://picsum.photos/300/150?random=1',
      sliderImage: 'https://picsum.photos/40/40?random=2',
      targetX: 200
    }
  }
};

export const CustomSize = {
  args: {
    ...Default.args,
    width: 400,
    height: 200
  }
};

export const SuccessState = {
  args: {
    ...Default.args,
    status: 'success',
    data: {
      targetX: 200
    }
  },
  render: (args) => (
    <SliderCaptcha
      {...args}
      onSuccess={(result) => console.log('Success:', result)}
    />
  )
};

export const AllSizes = {
  render: () => (
    <div style={{ display: 'flex', gap: '20px', flexWrap: 'wrap' }}>
      <div>
        <h4>Small (250x120)</h4>
        <SliderCaptcha
          width={250}
          height={120}
          onSuccess={(result) => console.log('Success:', result)}
        />
      </div>
      <div>
        <h4>Medium (300x150)</h4>
        <SliderCaptcha
          width={300}
          height={150}
          onSuccess={(result) => console.log('Success:', result)}
        />
      </div>
      <div>
        <h4>Large (350x180)</h4>
        <SliderCaptcha
          width={350}
          height={180}
          onSuccess={(result) => console.log('Success:', result)}
        />
      </div>
    </div>
  )
};
