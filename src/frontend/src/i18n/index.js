import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const resources = {
  en: {
    translation: {
      common: {
        appName: 'HJTPX',
        loading: 'Loading...',
        save: 'Save',
        cancel: 'Cancel',
        delete: 'Delete',
        edit: 'Edit',
        add: 'Add',
        search: 'Search',
        filter: 'Filter',
        export: 'Export',
        import: 'Import',
        refresh: 'Refresh',
        back: 'Back',
        next: 'Next',
        previous: 'Previous',
        submit: 'Submit',
        confirm: 'Confirm',
        close: 'Close',
        yes: 'Yes',
        no: 'No',
        success: 'Success',
        error: 'Error',
        warning: 'Warning',
        info: 'Info'
      },
      nav: {
        dashboard: 'Dashboard',
        users: 'Users',
        settings: 'Settings',
        analytics: 'Analytics',
        notifications: 'Notifications',
        search: 'Search',
        help: 'Help'
      },
      auth: {
        login: 'Login',
        logout: 'Logout',
        register: 'Register',
        forgotPassword: 'Forgot Password?',
        email: 'Email',
        password: 'Password',
        confirmPassword: 'Confirm Password',
        rememberMe: 'Remember Me',
        noAccount: "Don't have an account?",
        hasAccount: 'Already have an account?'
      },
      dashboard: {
        welcome: 'Welcome',
        overview: 'Overview',
        recentActivity: 'Recent Activity',
        quickActions: 'Quick Actions',
        totalUsers: 'Total Users',
        activeUsers: 'Active Users',
        newUsers: 'New Users',
        totalRevenue: 'Total Revenue'
      },
      analytics: {
        title: 'Analytics Dashboard',
        userStats: 'User Statistics',
        businessStats: 'Business Statistics',
        trendAnalysis: 'Trend Analysis',
        dateRange: 'Date Range',
        last7Days: 'Last 7 Days',
        last30Days: 'Last 30 Days',
        last90Days: 'Last 90 Days',
        custom: 'Custom Range',
        chart: {
          line: 'Line Chart',
          bar: 'Bar Chart',
          pie: 'Pie Chart',
          area: 'Area Chart'
        },
        metrics: {
          dailyActiveUsers: 'Daily Active Users',
          monthlyActiveUsers: 'Monthly Active Users',
          retention: 'Retention Rate',
          conversion: 'Conversion Rate'
        }
      },
      notifications: {
        title: 'Notifications',
        markAllRead: 'Mark All as Read',
        noNotifications: 'No notifications',
        settings: {
          title: 'Notification Settings',
          email: 'Email Notifications',
          sms: 'SMS Notifications',
          push: 'Push Notifications',
          inApp: 'In-App Notifications'
        },
        types: {
          system: 'System',
          user: 'User',
          security: 'Security',
          promotion: 'Promotion'
        }
      },
      search: {
        title: 'Advanced Search',
        placeholder: 'Search...',
        filters: 'Filters',
        results: 'Results',
        noResults: 'No results found',
        suggestions: 'Suggestions',
        recentSearches: 'Recent Searches',
        popularSearches: 'Popular Searches',
        advanced: {
          exactMatch: 'Exact Match',
          fuzzyMatch: 'Fuzzy Match',
          rangeFilter: 'Range Filter',
          multiSelect: 'Multi-Select'
        }
      },
      export: {
        title: 'Export Data',
        format: 'Format',
        csv: 'CSV',
        excel: 'Excel',
        pdf: 'PDF',
        json: 'JSON',
        selectedFields: 'Selected Fields',
        allFields: 'All Fields',
        startDate: 'Start Date',
        endDate: 'End Date',
        download: 'Download',
        preparing: 'Preparing export...'
      }
    }
  },
  zh: {
    translation: {
      common: {
        appName: 'HJTPX',
        loading: '加载中...',
        save: '保存',
        cancel: '取消',
        delete: '删除',
        edit: '编辑',
        add: '添加',
        search: '搜索',
        filter: '筛选',
        export: '导出',
        import: '导入',
        refresh: '刷新',
        back: '返回',
        next: '下一个',
        previous: '上一个',
        submit: '提交',
        confirm: '确认',
        close: '关闭',
        yes: '是',
        no: '否',
        success: '成功',
        error: '错误',
        warning: '警告',
        info: '信息'
      },
      nav: {
        dashboard: '仪表板',
        users: '用户管理',
        settings: '设置',
        analytics: '数据分析',
        notifications: '通知',
        search: '搜索',
        help: '帮助'
      },
      auth: {
        login: '登录',
        logout: '退出登录',
        register: '注册',
        forgotPassword: '忘记密码？',
        email: '邮箱',
        password: '密码',
        confirmPassword: '确认密码',
        rememberMe: '记住我',
        noAccount: '还没有账号？',
        hasAccount: '已有账号？'
      },
      dashboard: {
        welcome: '欢迎',
        overview: '概览',
        recentActivity: '最近活动',
        quickActions: '快捷操作',
        totalUsers: '总用户数',
        activeUsers: '活跃用户',
        newUsers: '新增用户',
        totalRevenue: '总收入'
      },
      analytics: {
        title: '数据分析仪表板',
        userStats: '用户统计',
        businessStats: '业务统计',
        trendAnalysis: '趋势分析',
        dateRange: '日期范围',
        last7Days: '最近7天',
        last30Days: '最近30天',
        last90Days: '最近90天',
        custom: '自定义范围',
        chart: {
          line: '折线图',
          bar: '柱状图',
          pie: '饼图',
          area: '面积图'
        },
        metrics: {
          dailyActiveUsers: '日活跃用户',
          monthlyActiveUsers: '月活跃用户',
          retention: '留存率',
          conversion: '转化率'
        }
      },
      notifications: {
        title: '通知中心',
        markAllRead: '全部标为已读',
        noNotifications: '暂无通知',
        settings: {
          title: '通知设置',
          email: '邮件通知',
          sms: '短信通知',
          push: '推送通知',
          inApp: '应用内通知'
        },
        types: {
          system: '系统',
          user: '用户',
          security: '安全',
          promotion: '促销'
        }
      },
      search: {
        title: '高级搜索',
        placeholder: '搜索...',
        filters: '筛选条件',
        results: '结果',
        noResults: '未找到结果',
        suggestions: '建议',
        recentSearches: '最近搜索',
        popularSearches: '热门搜索',
        advanced: {
          exactMatch: '精确匹配',
          fuzzyMatch: '模糊匹配',
          rangeFilter: '范围筛选',
          multiSelect: '多选'
        }
      },
      export: {
        title: '导出数据',
        format: '格式',
        csv: 'CSV',
        excel: 'Excel',
        pdf: 'PDF',
        json: 'JSON',
        selectedFields: '选择字段',
        allFields: '所有字段',
        startDate: '开始日期',
        endDate: '结束日期',
        download: '下载',
        preparing: '正在准备导出...'
      }
    }
  }
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: 'en',
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false
    },
    react: {
      useSuspense: false
    }
  });

export default i18n;
