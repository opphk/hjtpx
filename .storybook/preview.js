import { initialize, mswDecorator } from 'msw-storybook-addon';
import '../src/frontend/src/styles/globals.css';

initialize();

export const decorators = [
  (Story) => ({
    components: { Story },
    template: '<div style="margin: 20px;"><Story /></div>'
  }),
  mswDecorator
];

export const parameters = {
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  docs: {
    description: {
      component: '组件文档描述'
    }
  },
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#ffffff' },
      { name: 'dark', value: '#1a1a1a' },
      { name: 'gray', value: '#f5f5f5' }
    ]
  }
};
