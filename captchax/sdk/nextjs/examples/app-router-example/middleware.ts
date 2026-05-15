import { captchaMiddleware } from '@captchax/nextjs/middleware';

export default captchaMiddleware;

export const config = {
  matcher: [
    '/login/:path*',
    '/register/:path*',
    '/checkout/:path*',
    '/comment/:path*',
    '/api/protected/:path*'
  ]
};
