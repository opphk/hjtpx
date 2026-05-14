const fs = require('fs');
const path = require('path');

const sitemapConfig = {
  siteUrl: process.env.SITE_URL || 'https://hjtpx.example.com',
  outputDir: path.join(__dirname, '../../dist'),
  changefreq: 'weekly',
  priority: {
    home: 1.0,
    about: 0.8,
    contact: 0.7,
    blog: 0.9,
    default: 0.6,
  },
  routes: [
    { path: '/', changefreq: 'daily', priority: 1.0 },
    { path: '/about', changefreq: 'monthly', priority: 0.8 },
    { path: '/contact', changefreq: 'monthly', priority: 0.7 },
    { path: '/blog', changefreq: 'weekly', priority: 0.9 },
    { path: '/login', changefreq: 'yearly', priority: 0.3 },
    { path: '/register', changefreq: 'yearly', priority: 0.3 },
    { path: '/dashboard', changefreq: 'daily', priority: 0.7 },
    { path: '/profile', changefreq: 'weekly', priority: 0.6 },
    { path: '/settings', changefreq: 'monthly', priority: 0.5 },
  ],
};

function generateSitemap() {
  const { siteUrl, routes, changefreq, priority } = sitemapConfig;

  let urls = '';

  routes.forEach((route) => {
    const routePriority = route.priority || priority[route.path.replace('/', '')] || priority.default;
    const routeChangefreq = route.changefreq || changefreq;
    const lastmod = new Date().toISOString().split('T')[0];

    urls += `
  <url>
    <loc>${siteUrl}${route.path}</loc>
    <lastmod>${lastmod}</lastmod>
    <changefreq>${routeChangefreq}</changefreq>
    <priority>${routePriority.toFixed(1)}</priority>
  </url>`;
  });

  const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9
        http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd">
${urls}
</urlset>`;

  if (!fs.existsSync(sitemapConfig.outputDir)) {
    fs.mkdirSync(sitemapConfig.outputDir, { recursive: true });
  }

  const outputPath = path.join(sitemapConfig.outputDir, 'sitemap.xml');
  fs.writeFileSync(outputPath, sitemap);

  console.log(`Sitemap generated at: ${outputPath}`);
  return outputPath;
}

function generateSitemapIndex() {
  const { siteUrl, outputDir } = sitemapConfig;

  const sitemapIndex = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>${siteUrl}/sitemap.xml</loc>
    <lastmod>${new Date().toISOString().split('T')[0]}</lastmod>
  </sitemap>
</sitemapindex>`;

  const outputPath = path.join(outputDir, 'sitemap-index.xml');
  fs.writeFileSync(outputPath, sitemapIndex);

  console.log(`Sitemap index generated at: ${outputPath}`);
  return outputPath;
}

if (require.main === module) {
  generateSitemap();
  generateSitemapIndex();
}

module.exports = {
  generateSitemap,
  generateSitemapIndex,
  sitemapConfig,
};
