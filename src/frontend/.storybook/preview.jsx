/** @type { import('@storybook/react-vite').Preview } */
import '../src/styles/global.css';

const preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i
      },
      expanded: true
    },
    a11y: {
      test: 'todo'
    },
    docs: {
      toc: {
        contentsSelector: '.sbdocs-content',
        headingSelector: 'h1, h2, h3',
        ignoreSelector: '#primary',
        title: '目录',
        disable: false,
        unsafeTocbotOptions: {
          orderedList: false
        }
      },
      source: {
        type: 'dynamic'
      },
      description: {
        component: '组件库文档，展示所有UI组件的使用方法和变体。'
      }
    },
    backgrounds: {
      default: 'light',
      values: [
        { name: 'light', value: '#ffffff' },
        { name: 'dark', value: '#1a1a1a' },
        { name: 'gray', value: '#f5f5f5' }
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
            height: '800px'
          }
        }
      }
    }
  },
  decorators: [
    (Story) => (
      <div style={{ 
        padding: '20px',
        minHeight: '100px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}>
        <Story />
      </div>
    )
  ],
  tags: ['autodocs']
};

export default preview;
