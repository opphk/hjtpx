import React from 'react';
import { setupWrapper } from './setup';

export const decorators = [
  (Story) => (
    <div style={{ padding: '20px', fontFamily: 'system-ui, sans-serif' }}>
      <Story />
    </div>
  ),
];

export const parameters = {
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/i,
    },
    expanded: true,
    sort: 'requiredFirst',
  },
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#ffffff' },
      { name: 'dark', value: '#1a1a1a' },
      { name: 'gray', value: '#f5f5f5' },
    ],
  },
  viewport: {
    viewports: {
      mobile: { name: 'Mobile', styles: { width: '375px', height: '667px' } },
      tablet: { name: 'Tablet', styles: { width: '768px', height: '1024px' } },
      desktop: { name: 'Desktop', styles: { width: '1280px', height: '800px' } },
    },
  },
  options: {
    storySort: {
      method: 'alphabetical',
      order: ['Introduction', 'Components'],
    },
  },
};
