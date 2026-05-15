import React from 'react';

const StructuredData = ({ type, data }) => {
  const renderJSONLD = () => {
    switch (type) {
      case 'organization':
        return {
          '@context': 'https://schema.org',
          '@type': 'Organization',
          name: data.name || 'HJTPX',
          url: data.url || 'https://hjtpx.com',
          logo: data.logo || 'https://hjtpx.com/favicon.png',
          sameAs: data.sameAs || [
            'https://github.com/hjtpx',
            'https://twitter.com/hjtpx'
          ],
          contactPoint: {
            '@type': 'ContactPoint',
            contactType: 'customer service',
            availableLanguage: ['Chinese', 'English']
          }
        };

      case 'website':
        return {
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          name: data.name || 'HJTPX 系统',
          url: data.url || 'https://hjtpx.com',
          description: data.description || '现代化全栈应用程序',
          inLanguage: data.language || 'zh-CN',
          potentialAction: {
            '@type': 'SearchAction',
            target: {
              '@type': 'EntryPoint',
              urlTemplate: 'https://hjtpx.com/search?q={search_term_string}'
            },
            'query-input': 'required name=search_term_string'
          },
          publisher: {
            '@type': 'Organization',
            name: 'HJTPX',
            url: 'https://hjtpx.com',
            logo: {
              '@type': 'ImageObject',
              url: 'https://hjtpx.com/favicon.png',
              width: 512,
              height: 512
            }
          }
        };

      case 'breadcrumb':
        return {
          '@context': 'https://schema.org',
          '@type': 'BreadcrumbList',
          itemListElement: data.items.map((item, index) => ({
            '@type': 'ListItem',
            position: index + 1,
            name: item.name,
            item: item.url
          }))
        };

      case 'webpage':
        return {
          '@context': 'https://schema.org',
          '@type': 'WebPage',
          name: data.name || 'HJTPX 系统',
          description: data.description || '现代化全栈应用程序',
          url: data.url || 'https://hjtpx.com',
          inLanguage: data.language || 'zh-CN',
          isPartOf: {
            '@type': 'WebSite',
            name: 'HJTPX 系统',
            url: 'https://hjtpx.com'
          },
          about: {
            '@type': 'SoftwareApplication',
            name: 'HJTPX 系统',
            applicationCategory: 'BusinessApplication',
            operatingSystem: 'Web'
          }
        };

      case 'softwareapplication':
        return {
          '@context': 'https://schema.org',
          '@type': 'SoftwareApplication',
          name: data.name || 'HJTPX 系统',
          alternateName: data.alternateName || 'HJTPX',
          description: data.description || '现代化全栈应用程序',
          url: data.url || 'https://hjtpx.com',
          applicationCategory: 'BusinessApplication',
          operatingSystem: 'Web',
          browserRequirements: 'Requires modern web browser with JavaScript enabled',
          softwareVersion: data.version || '1.0.0',
          author: {
            '@type': 'Organization',
            name: data.authorName || 'HJTPX Team',
            url: data.authorUrl || 'https://hjtpx.com'
          },
          inLanguage: data.language || 'zh-CN',
          isAccessibleForFree: true,
          offers: {
            '@type': 'Offer',
            price: data.price || '0',
            priceCurrency: 'CNY'
          },
          aggregateRating: data.rating ? {
            '@type': 'AggregateRating',
            ratingValue: data.rating.value || '4.8',
            ratingCount: data.rating.count || '156'
          } : undefined,
          featureList: data.features || [
            '用户管理',
            '数据分析',
            '审计追踪',
            '实时通知',
            '多语言支持',
            '响应式设计',
            '权限管理',
            '日志记录'
          ]
        };

      default:
        return data;
    }
  };

  const jsonld = renderJSONLD();
  const jsonString = JSON.stringify(jsonld, null, 2);

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: jsonString }}
    />
  );
};

export const OrganizationSchema = ({ data }) => (
  <StructuredData type="organization" data={data} />
);

export const WebsiteSchema = ({ data }) => (
  <StructuredData type="website" data={data} />
);

export const BreadcrumbSchema = ({ items }) => (
  <StructuredData type="breadcrumb" data={{ items }} />
);

export const WebpageSchema = ({ data }) => (
  <StructuredData type="webpage" data={data} />
);

export const SoftwareApplicationSchema = ({ data }) => (
  <StructuredData type="softwareapplication" data={data} />
);

export default StructuredData;
