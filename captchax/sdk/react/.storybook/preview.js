import React from 'react';
import { CaptchaProvider } from '../src/CaptchaProvider';

export const decorators = [
  (Story) => (
    <CaptchaProvider config={{ apiServer: 'http://localhost:8080' }}>
      <div style={{
        padding: '20px',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
      }}>
        <Story />
      </div>
    </CaptchaProvider>
  )
];

export const parameters = {
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/i
    }
  },
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#f5f5f5' },
      { name: 'dark', value: '#262626' },
      { name: 'white', value: '#ffffff' }
    ]
  }
};
