import React, { useEffect } from 'react';
import { seoConfig, generateMetaTags, generateOpenGraph, generateTwitterCards, generateStructuredData } from '../utils/seo';

function SEO({ page, customMeta = {}, customOG = {}, customTwitter = {}, structuredDataType, structuredData = {} }) {
  const meta = generateMetaTags({ page, customMeta });
  const og = generateOpenGraph({ page, customOG });
  const twitter = generateTwitterCards({ customTwitter });
  const schema = generateStructuredData({ type: structuredDataType, customData: structuredData });

  useEffect(() => {
    const updateMetaTag = (name, content, property = false) => {
      let tag = property
        ? document.querySelector(`meta[property="${name}"]`)
        : document.querySelector(`meta[name="${name}"]`);

      if (!tag) {
        tag = document.createElement('meta');
        if (property) {
          tag.setAttribute('property', name);
        } else {
          tag.setAttribute('name', name);
        }
        document.head.appendChild(tag);
      }
      tag.setAttribute('content', content);
    };

    const setLinkTag = (rel, href) => {
      let tag = document.querySelector(`link[rel="${rel}"]`);
      if (!tag) {
        tag = document.createElement('link');
        tag.setAttribute('rel', rel);
        document.head.appendChild(tag);
      }
      tag.setAttribute('href', href);
    };

    Object.entries(meta).forEach(([key, value]) => {
      if (key !== 'canonical' && key !== 'title') {
        updateMetaTag(key, value);
      }
    });

    if (meta.canonical) {
      setLinkTag('canonical', meta.canonical);
    }

    Object.entries(og).forEach(([key, value]) => {
      updateMetaTag(key, value, true);
    });

    Object.entries(twitter).forEach(([key, value]) => {
      updateMetaTag(key, value);
    });

    if (schema && Object.keys(schema).length > 0) {
      let schemaTag = document.querySelector('script[type="application/ld+json"]');
      if (!schemaTag) {
        schemaTag = document.createElement('script');
        schemaTag.type = 'application/ld+json';
        document.head.appendChild(schemaTag);
      }
      schemaTag.textContent = JSON.stringify(schema);
    }

    document.title = meta.title;

    return () => {
      if (schema) {
        const schemaTag = document.querySelector('script[type="application/ld+json"]');
        if (schemaTag) {
          schemaTag.remove();
        }
      }
    };
  }, [meta, og, twitter, schema]);

  return null;
}

export default SEO;

export function useSEO(props) {
  return <SEO {...props} />;
}

export { generateMetaTags, generateOpenGraph, generateTwitterCards, generateStructuredData };
