module.exports = {
  env: {
    browser: true,
    es2021: true,
    node: true,
    jest: true,
    commonjs: true,
    es6: true
  },
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'plugin:jsx-a11y/recommended',
    'plugin:import/errors',
    'plugin:import/warnings',
    'prettier'
  ],
  parserOptions: {
    ecmaFeatures: {
      jsx: true
    },
    ecmaVersion: 12,
    sourceType: 'module'
  },
  plugins: [
    'react',
    'react-hooks',
    'jsx-a11y',
    'import',
    'prettier'
  ],
  rules: {
    'prettier/prettier': 'error',
    'react/react-in-jsx-scope': 'off',
    'react/prop-types': 'warn',
    'react/jsx-uses-react': 'off',
    'react/jsx-uses-vars': 'error',
    'no-unused-vars': ['warn', {
      argsIgnorePattern: '^_',
      varsIgnorePattern: '^_'
    }],
    'no-console': ['warn', {
      allow: ['warn', 'error', 'info']
    }],
    'no-debugger': 'error',
    'no-duplicate-imports': 'error',
    'no-eval': 'error',
    'no-implied-eval': 'error',
    'no-new-func': 'error',
    'no-new-require': 'error',
    'no-path-concat': 'error',
    'no-process-env': 'off',
    'no-restricted-globals': ['error', {
      name: 'isFinite',
      message: 'Use Number.isFinite instead'
    }, {
      name: 'isNaN',
      message: 'Use Number.isNaN instead'
    }],
    'no-return-assign': ['error', 'except-parens'],
    'no-script-url': 'error',
    'no-shadow': ['warn', {
      builtinGlobals: true,
      hoist: 'functions',
      allow: ['resolve', 'reject', 'done', 'next', 'err', 'error']
    }],
    'no-throw-literal': 'error',
    'no-undef': 'error',
    'no-undef-init': 'error',
    'no-unreachable': 'error',
    'no-unsafe-finally': 'error',
    'no-with': 'error',
    'accessor-pairs': 'error',
    'array-callback-return': 'warn',
    'block-scoped-var': 'error',
    'consistent-return': 'warn',
    'curly': ['error', 'multi-line'],
    'default-case': 'warn',
    'default-case-last': 'error',
    'dot-notation': ['error', {
      allowKeywords: true,
      allowPattern: ''
    }],
    'dot-location': ['error', 'property'],
    'eqeqeq': ['error', 'always'],
    'grouped-accessor-pairs': 'error',
    'guard-for-in': 'error',
    'max-classes-per-file': ['warn', 1],
    'no-alert': 'warn',
    'no-caller': 'error',
    'no-case-declarations': 'error',
    'no-constructor-return': 'error',
    'no-div-regex': 'warn',
    'no-else-return': ['warn', {
      allowElseIf: false
    }],
    'no-empty-function': ['warn', {
      allow: ['arrowFunctions', 'functions', 'methods']
    }],
    'no-empty-pattern': 'error',
    'no-eq-null': 'warn',
    'no-extra-semi': 'error',
    'no-floating-decimal': 'error',
    'no-global-assign': 'error',
    'no-native-reassign': 'error',
    'no-invalid-this': 'warn',
    'no-lone-blocks': 'error',
    'no-loop-func': 'warn',
    'no-multi-spaces': ['error', {
      ignoreEOLComments: true
    }],
    'no-new-wrappers': 'error',
    'no-octal': 'error',
    'no-octal-escape': 'error',
    'no-proto': 'error',
    'no-redeclare': 'error',
    'no-return-await': 'error',
    'no-self-assign': 'error',
    'no-self-compare': 'error',
    'no-sequences': 'error',
    'no-useless-catch': 'error',
    'no-useless-escape': 'error',
    'no-useless-return': 'warn',
    'no-void': 'error',
    'no-with': 'error',
    'prefer-promise-reject-errors': ['error', {
      allowEmptyReject: true
    }],
    'require-await': 'warn',
    'yoda': 'error',
    'jsx-a11y/alt-text': 'warn',
    'jsx-a11y/anchor-has-content': 'warn',
    'jsx-a11y/aria-props': 'warn',
    'jsx-a11y/aria-proptypes': 'warn',
    'jsx-a11y/aria-unsupported-elements': 'warn',
    'jsx-a11y/click-events-have-key-events': 'warn',
    'jsx-a11y/heading-has-content': 'warn',
    'jsx-a11y/html-has-lang': 'warn',
    'jsx-a11y/img-redundant-alt': 'warn',
    'jsx-a11y/interactive-supports-focus': 'warn',
    'jsx-a11y/label-has-associated-control': 'warn',
    'jsx-a11y/mouse-events-have-key-events': 'warn',
    'jsx-a11y/no-autofocus': 'warn',
    'jsx-a11y/no-distracting-elements': 'warn',
    'jsx-a11y/no-noninteractive-element-interactions': 'warn',
    'jsx-a11y/no-redundant-roles': 'warn',
    'jsx-a11y/role-has-required-aria-props': 'warn',
    'jsx-a11y/role-supports-aria-props': 'warn',
    'jsx-a11y/scope': 'warn',
    'jsx-a11y/tabindex-no-positive': 'warn',
    'import/no-unresolved': 'error',
    'import/named': 'error',
    'import/default': 'error',
    'import/namespace': 'error',
    'import/export': 'error',
    'import/no-named-as-default': 'warn',
    'import/no-named-as-default-member': 'warn',
    'import/no-deprecated': 'warn',
    'import/no-extraneous-dependencies': ['error', {
      devDependencies: true,
      optionalDependencies: false
    }],
    'import/no-commonjs': 'off',
    'import/order': ['error', {
      groups: [
        'builtin',
        'external',
        'internal',
        'parent',
        'sibling',
        'index'
      ],
      'newlines-between': 'always',
      alphabetize: {
        order: 'asc',
        caseInsensitive: true
      }
    }],
    'import/newline-after-import': 'error',
    'import/no-duplicates': 'error'
  },
  settings: {
    react: {
      version: 'detect'
    },
    'import/resolver': {
      node: {
        extensions: ['.js', '.jsx', '.json']
      }
    }
  },
  overrides: [
    {
      files: ['*.test.js', '*.spec.js'],
      env: {
        jest: true
      },
      rules: {
        'no-unused-expressions': 'off',
        'max-nested-callbacks': 'off'
      }
    },
    {
      files: ['*.config.js'],
      rules: {
        'import/no-extraneous-dependencies': 'off'
      }
    }
  ]
};
