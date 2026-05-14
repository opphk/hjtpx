import { addDecorator } from '@storybook/react';
import { withTests } from '@storybook/addon-jest';

export const parameters = {
  actions: { argTypesRegex: '^on[A-Z].*' },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/i
    },
    expanded: true,
    sort: 'requiredFirst'
  },
  docs: {
    toc: {
      contentsSelector: '.sbdocs-content',
      headingSelector: 'h1, h2, h3',
      ignoreSelector: '#primary',
      title: 'On this page',
      disable: false,
      unsafeTocbotOptions: {
        orderedList: false
      }
    }
  },
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#ffffff' },
      { name: 'dark', value: '#1a1a1a' },
      { name: 'primary', value: '#3b82f6' }
    ]
  },
  viewport: {
    viewports: {
      mobile: {
        name: 'Mobile',
        styles: {
          width: '375px',
          height: '667px'
        }
      },
      tablet: {
        name: 'Tablet',
        styles: {
          width: '768px',
          height: '1024px'
        }
      },
      desktop: {
        name: 'Desktop',
        styles: {
          width: '1280px',
          height: '720px'
        }
      }
    }
  },
  a11y: {
    config: {
      rules: [{ id: 'color-contrast', enabled: true }]
    }
  }
};

export const decorators = [
  (Story) => (
    <div style={{ padding: '20px' }}>
      <Story />
    </div>
  )
];
