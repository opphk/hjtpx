const seoConfig = {
  site: {
    name: 'HJTPX',
    title: 'HJTPX - Modern Web Application Platform',
    description: 'A comprehensive web application platform with real-time features, user management, and analytics.',
    url: process.env.SITE_URL || 'https://hjtpx.example.com',
    logo: '/logo.png',
    logoAlt: 'HJTPX Logo',
    social: {
      twitter: '@hjtpx',
      facebook: 'hjtpx',
    },
  },

  defaultMeta: {
    keywords: 'web application, real-time, user management, analytics, platform',
    author: 'HJTPX Team',
    copyright: 'Copyright © 2024 HJTPX. All rights reserved.',
    robots: 'index, follow',
    language: 'en-US',
    charset: 'UTF-8',
  },

  openGraph: {
    type: 'website',
    locale: 'en_US',
    siteName: 'HJTPX',
    image: {
      url: '/og-image.png',
      width: 1200,
      height: 630,
      alt: 'HJTPX Platform',
    },
    twitter: {
      card: 'summary_large_image',
      site: '@hjtpx',
      creator: '@hjtpx',
    },
  },

  structuredData: {
    organization: {
      type: 'Organization',
      name: 'HJTPX',
      url: process.env.SITE_URL || 'https://hjtpx.example.com',
      logo: '/logo.png',
      sameAs: [
        'https://twitter.com/hjtpx',
        'https://facebook.com/hjtpx',
        'https://github.com/hjtpx',
      ],
    },
    webSite: {
      type: 'WebSite',
      name: 'HJTPX',
      url: process.env.SITE_URL || 'https://hjtpx.example.com',
      potentialAction: {
        type: 'SearchAction',
        target: {
          type: 'EntryPoint',
          urlTemplate: '/search?q={search_term_string}',
        },
        'query-input': 'required name=search_term_string',
      },
    },
  },

  pages: {
    home: {
      title: 'HJTPX - Modern Web Application Platform',
      description: 'A comprehensive web application platform with real-time features, user management, and analytics.',
      path: '/',
      ogType: 'website',
    },
    about: {
      title: 'About Us - HJTPX',
      description: 'Learn more about HJTPX and our mission to build modern web applications.',
      path: '/about',
      ogType: 'website',
    },
    contact: {
      title: 'Contact Us - HJTPX',
      description: 'Get in touch with the HJTPX team.',
      path: '/contact',
      ogType: 'website',
    },
    blog: {
      title: 'Blog - HJTPX',
      description: 'Latest news, tutorials, and updates from the HJTPX team.',
      path: '/blog',
      ogType: 'website',
    },
  },
};

function generateMetaTags(config = {}) {
  const { page, customMeta = {} } = config;
  const pageConfig = page ? seoConfig.pages[page] : {};

  return {
    title: customMeta.title || pageConfig.title || seoConfig.site.title,
    description: customMeta.description || pageConfig.description || seoConfig.site.description,
    keywords: customMeta.keywords || seoConfig.defaultMeta.keywords,
    author: customMeta.author || seoConfig.defaultMeta.author,
    robots: customMeta.robots || seoConfig.defaultMeta.robots,
    canonical: customMeta.canonical || `${seoConfig.site.url}${pageConfig.path || '/'}`,
  };
}

function generateOpenGraph(config = {}) {
  const { page, customOG = {} } = config;
  const pageConfig = page ? seoConfig.pages[page] : {};

  return {
    'og:title': customOG.title || pageConfig.title || seoConfig.site.title,
    'og:description': customOG.description || pageConfig.description || seoConfig.site.description,
    'og:url': customOG.url || `${seoConfig.site.url}${pageConfig.path || '/'}`,
    'og:type': customOG.type || pageConfig.ogType || seoConfig.openGraph.type,
    'og:locale': customOG.locale || seoConfig.openGraph.locale,
    'og:site_name': customOG.siteName || seoConfig.openGraph.siteName,
    'og:image': customOG.image || `${seoConfig.site.url}${seoConfig.openGraph.image.url}`,
    'og:image:width': customOG.imageWidth || seoConfig.openGraph.image.width,
    'og:image:height': customOG.imageHeight || seoConfig.openGraph.image.height,
    'og:image:alt': customOG.imageAlt || seoConfig.openGraph.image.alt,
  };
}

function generateTwitterCards(config = {}) {
  const { customTwitter = {} } = config;

  return {
    'twitter:card': customTwitter.card || seoConfig.openGraph.twitter.card,
    'twitter:site': customTwitter.site || seoConfig.openGraph.twitter.site,
    'twitter:creator': customTwitter.creator || seoConfig.openGraph.twitter.creator,
    'twitter:title': customTwitter.title || seoConfig.site.title,
    'twitter:description': customTwitter.description || seoConfig.site.description,
    'twitter:image': customTwitter.image || `${seoConfig.site.url}${seoConfig.openGraph.image.url}`,
    'twitter:image:alt': customTwitter.imageAlt || seoConfig.openGraph.image.alt,
  };
}

function generateStructuredData(config = {}) {
  const { type = 'webSite', customData = {} } = config;

  const dataGenerators = {
    organization: () => ({
      '@context': 'https://schema.org',
      '@type': seoConfig.structuredData.organization.type,
      name: customData.name || seoConfig.structuredData.organization.name,
      url: customData.url || seoConfig.structuredData.organization.url,
      logo: customData.logo || seoConfig.structuredData.organization.logo,
      sameAs: customData.sameAs || seoConfig.structuredData.organization.sameAs,
    }),

    webSite: () => ({
      '@context': 'https://schema.org',
      '@type': seoConfig.structuredData.webSite.type,
      name: customData.name || seoConfig.structuredData.webSite.name,
      url: customData.url || seoConfig.structuredData.webSite.url,
      potentialAction: seoConfig.structuredData.webSite.potentialAction,
    }),

    article: (customData) => ({
      '@context': 'https://schema.org',
      '@type': 'Article',
      headline: customData.headline || 'Untitled Article',
      description: customData.description || '',
      author: {
        '@type': 'Person',
        name: customData.authorName || seoConfig.site.name,
      },
      datePublished: customData.datePublished || new Date().toISOString(),
      dateModified: customData.dateModified || new Date().toISOString(),
      image: customData.image || `${seoConfig.site.url}${seoConfig.openGraph.image.url}`,
      publisher: {
        '@type': 'Organization',
        name: seoConfig.structuredData.organization.name,
        logo: {
          '@type': 'ImageObject',
          url: `${seoConfig.site.url}${seoConfig.structuredData.organization.logo}`,
        },
      },
    }),

    breadcrumb: (customData) => ({
      '@context': 'https://schema.org',
      '@type': 'BreadcrumbList',
      itemListElement: customData.items || [],
    }),
  };

  return dataGenerators[type] ? dataGenerators[type](customData) : dataGenerators.webSite();
}

module.exports = {
  seoConfig,
  generateMetaTags,
  generateOpenGraph,
  generateTwitterCards,
  generateStructuredData,
};
