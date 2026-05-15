import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import SEO, { getSEOMeta } from '../components/SEO';
import StructuredData, { 
  OrganizationSchema, 
  WebsiteSchema, 
  BreadcrumbSchema 
} from '../components/StructuredData';

describe('SEO Component Tests', () => {
  beforeEach(() => {
    document.title = 'Original Title';
    const existingMetaTags = document.querySelectorAll('meta');
    existingMetaTags.forEach(tag => tag.remove());
  });

  describe('SEO Component', () => {
    it('should update page title with SEO title', () => {
      const testTitle = 'Test Page';
      render(<SEO title={testTitle} />);
      
      expect(document.title).toContain(testTitle);
      expect(document.title).toContain('HJTPX 系统');
    });

    it('should update meta description', () => {
      const testDescription = 'Test description for SEO';
      render(<SEO description={testDescription} />);
      
      const metaDescription = document.querySelector('meta[name="description"]');
      expect(metaDescription).toBeTruthy();
      expect(metaDescription.content).toBe(testDescription);
    });

    it('should update meta keywords', () => {
      const testKeywords = 'test, keywords, seo';
      render(<SEO keywords={testKeywords} />);
      
      const metaKeywords = document.querySelector('meta[name="keywords"]');
      expect(metaKeywords).toBeTruthy();
      expect(metaKeywords.content).toBe(testKeywords);
    });

    it('should update Open Graph tags', () => {
      const testTitle = 'OG Test Title';
      const testDescription = 'OG Test Description';
      render(
        <SEO 
          title={testTitle} 
          description={testDescription}
          ogImage="https://example.com/og-image.png"
        />
      );
      
      const ogTitle = document.querySelector('meta[property="og:title"]');
      const ogDescription = document.querySelector('meta[property="og:description"]');
      const ogImage = document.querySelector('meta[property="og:image"]');
      
      expect(ogTitle).toBeTruthy();
      expect(ogDescription).toBeTruthy();
      expect(ogImage).toBeTruthy();
      expect(ogTitle.content).toContain(testTitle);
      expect(ogDescription.content).toBe(testDescription);
      expect(ogImage.content).toBe('https://example.com/og-image.png');
    });

    it('should update Twitter card tags', () => {
      const testTitle = 'Twitter Test Title';
      const testDescription = 'Twitter Test Description';
      render(
        <SEO 
          title={testTitle} 
          description={testDescription}
          ogImage="https://example.com/twitter-image.png"
        />
      );
      
      const twitterTitle = document.querySelector('meta[name="twitter:title"]');
      const twitterDescription = document.querySelector('meta[name="twitter:description"]');
      const twitterImage = document.querySelector('meta[name="twitter:image"]');
      
      expect(twitterTitle).toBeTruthy();
      expect(twitterDescription).toBeTruthy();
      expect(twitterImage).toBeTruthy();
    });

    it('should update canonical URL', () => {
      const canonicalUrl = 'https://hjtpx.com/test-page';
      render(<SEO canonical={canonicalUrl} />);
      
      const canonical = document.querySelector('link[rel="canonical"]');
      expect(canonical).toBeTruthy();
      expect(canonical.href).toBe(canonicalUrl);
    });

    it('should set noindex for specific pages', () => {
      render(<SEO noIndex={true} />);
      
      const robots = document.querySelector('meta[name="robots"]');
      expect(robots).toBeTruthy();
      expect(robots.content).toBe('noindex, nofollow');
    });

    it('should use default description when not provided', () => {
      render(<SEO />);
      
      const metaDescription = document.querySelector('meta[name="description"]');
      expect(metaDescription).toBeTruthy();
      expect(metaDescription.content).toContain('HJTPX');
    });
  });

  describe('getSEOMeta Function', () => {
    it('should return correct SEO config for login page', () => {
      const meta = getSEOMeta('login');
      
      expect(meta.title).toBe('用户登录');
      expect(meta.description).toBeTruthy();
      expect(meta.keywords).toContain('登录');
      expect(meta.ogType).toBe('website');
      expect(meta.canonical).toBe('https://hjtpx.com/login');
    });

    it('should return correct SEO config for register page', () => {
      const meta = getSEOMeta('register');
      
      expect(meta.title).toBe('用户注册');
      expect(meta.description).toBeTruthy();
      expect(meta.keywords).toContain('注册');
      expect(meta.ogType).toBe('website');
      expect(meta.canonical).toBe('https://hjtpx.com/register');
    });

    it('should return correct SEO config for dashboard page', () => {
      const meta = getSEOMeta('dashboard');
      
      expect(meta.title).toBe('个人仪表板');
      expect(meta.description).toBeTruthy();
      expect(meta.keywords).toContain('仪表板');
    });

    it('should return correct SEO config for settings page', () => {
      const meta = getSEOMeta('settings');
      
      expect(meta.title).toBe('系统设置');
      expect(meta.description).toBeTruthy();
      expect(meta.keywords).toContain('设置');
    });

    it('should return correct SEO config for admin users page', () => {
      const meta = getSEOMeta('adminUsers');
      
      expect(meta.title).toBe('用户管理');
      expect(meta.noIndex).toBe(true);
    });

    it('should return correct SEO config for logs page', () => {
      const meta = getSEOMeta('logs');
      
      expect(meta.title).toBe('日志管理');
      expect(meta.noIndex).toBe(true);
    });

    it('should return correct SEO config for audit page', () => {
      const meta = getSEOMeta('audit');
      
      expect(meta.title).toBe('审计仪表板');
      expect(meta.noIndex).toBe(true);
    });

    it('should return home page config for unknown pages', () => {
      const meta = getSEOMeta('unknown');
      
      expect(meta.title).toBe('首页');
      expect(meta.canonical).toBe('https://hjtpx.com');
    });

    it('should return home page config when no page specified', () => {
      const meta = getSEOMeta();
      
      expect(meta.title).toBe('首页');
      expect(meta.description).toBeTruthy();
    });
  });
});

describe('StructuredData Component Tests', () => {
  describe('OrganizationSchema', () => {
    it('should render Organization JSON-LD', () => {
      render(
        <OrganizationSchema 
          data={{ 
            name: 'HJTPX', 
            url: 'https://hjtpx.com' 
          }} 
        />
      );
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      expect(scriptTag).toBeTruthy();
      
      const jsonContent = JSON.parse(scriptTag.textContent);
      expect(jsonContent['@type']).toBe('Organization');
      expect(jsonContent.name).toBe('HJTPX');
      expect(jsonContent.url).toBe('https://hjtpx.com');
    });

    it('should include sameAs links', () => {
      render(<OrganizationSchema data={{ name: 'HJTPX', url: 'https://hjtpx.com' }} />);
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent.sameAs).toBeTruthy();
      expect(Array.isArray(jsonContent.sameAs)).toBe(true);
    });
  });

  describe('WebsiteSchema', () => {
    it('should render WebSite JSON-LD', () => {
      render(
        <WebsiteSchema 
          data={{ 
            name: 'HJTPX 系统',
            url: 'https://hjtpx.com' 
          }} 
        />
      );
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      expect(scriptTag).toBeTruthy();
      
      const jsonContent = JSON.parse(scriptTag.textContent);
      expect(jsonContent['@type']).toBe('WebSite');
      expect(jsonContent.name).toBe('HJTPX 系统');
      expect(jsonContent.url).toBe('https://hjtpx.com');
    });

    it('should include potentialAction for search', () => {
      render(<WebsiteSchema data={{ name: 'HJTPX 系统', url: 'https://hjtpx.com' }} />);
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent.potentialAction).toBeTruthy();
      expect(jsonContent.potentialAction['@type']).toBe('SearchAction');
    });

    it('should include publisher information', () => {
      render(<WebsiteSchema data={{ name: 'HJTPX 系统', url: 'https://hjtpx.com' }} />);
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent.publisher).toBeTruthy();
      expect(jsonContent.publisher['@type']).toBe('Organization');
      expect(jsonContent.publisher.name).toBe('HJTPX');
    });
  });

  describe('BreadcrumbSchema', () => {
    it('should render BreadcrumbList JSON-LD', () => {
      const items = [
        { name: '首页', url: 'https://hjtpx.com' },
        { name: '登录', url: 'https://hjtpx.com/login' }
      ];
      
      render(<BreadcrumbSchema items={items} />);
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      expect(scriptTag).toBeTruthy();
      
      const jsonContent = JSON.parse(scriptTag.textContent);
      expect(jsonContent['@type']).toBe('BreadcrumbList');
      expect(jsonContent.itemListElement).toHaveLength(2);
      expect(jsonContent.itemListElement[0].name).toBe('首页');
      expect(jsonContent.itemListElement[1].name).toBe('登录');
    });

    it('should set correct position for each item', () => {
      const items = [
        { name: '首页', url: 'https://hjtpx.com' },
        { name: '子页面', url: 'https://hjtpx.com/subpage' },
        { name: '详情页', url: 'https://hjtpx.com/subpage/details' }
      ];
      
      render(<BreadcrumbSchema items={items} />);
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent.itemListElement[0].position).toBe(1);
      expect(jsonContent.itemListElement[1].position).toBe(2);
      expect(jsonContent.itemListElement[2].position).toBe(3);
    });
  });

  describe('StructuredData Component', () => {
    it('should render WebPage JSON-LD', () => {
      render(
        <StructuredData 
          type="webpage" 
          data={{ 
            name: '测试页面',
            description: '测试描述',
            url: 'https://hjtpx.com/test'
          }} 
        />
      );
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent['@type']).toBe('WebPage');
      expect(jsonContent.name).toBe('测试页面');
      expect(jsonContent.description).toBe('测试描述');
    });

    it('should render SoftwareApplication JSON-LD', () => {
      render(
        <StructuredData 
          type="softwareapplication" 
          data={{ 
            name: 'HJTPX 系统',
            description: '测试应用'
          }} 
        />
      );
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent['@type']).toBe('SoftwareApplication');
      expect(jsonContent.name).toBe('HJTPX 系统');
      expect(jsonContent.applicationCategory).toBe('BusinessApplication');
    });

    it('should handle unknown type gracefully', () => {
      render(
        <StructuredData 
          type="unknown" 
          data={{ custom: 'data' }} 
        />
      );
      
      const scriptTag = document.querySelector('script[type="application/ld+json"]');
      const jsonContent = JSON.parse(scriptTag.textContent);
      
      expect(jsonContent.custom).toBe('data');
    });
  });
});

describe('SEO Integration Tests', () => {
  beforeEach(() => {
    document.title = 'Original Title';
    const existingMetaTags = document.querySelectorAll('meta');
    existingMetaTags.forEach(tag => tag.remove());
  });

  it('should handle all pages have SEO meta configured', () => {
    const pages = ['login', 'register', 'dashboard', 'settings', 'adminUsers', 'logs', 'audit', 'home'];
    
    pages.forEach(page => {
      const meta = getSEOMeta(page);
      expect(meta).toBeTruthy();
      expect(meta.title).toBeTruthy();
      expect(meta.description).toBeTruthy();
    });
  });

  it('should maintain SEO meta uniqueness per page', () => {
    const loginMeta = getSEOMeta('login');
    const registerMeta = getSEOMeta('register');
    const dashboardMeta = getSEOMeta('dashboard');
    
    expect(loginMeta.title).not.toBe(registerMeta.title);
    expect(loginMeta.title).not.toBe(dashboardMeta.title);
    expect(registerMeta.title).not.toBe(dashboardMeta.title);
  });

  it('should set noindex for admin and internal pages', () => {
    const adminPages = ['adminUsers', 'logs', 'audit'];
    
    adminPages.forEach(page => {
      const meta = getSEOMeta(page);
      expect(meta.noIndex).toBe(true);
    });
  });

  it('should allow public pages to be indexed', () => {
    const publicPages = ['login', 'register', 'home'];
    
    publicPages.forEach(page => {
      const meta = getSEOMeta(page);
      expect(meta.noIndex).not.toBe(true);
    });
  });
});
