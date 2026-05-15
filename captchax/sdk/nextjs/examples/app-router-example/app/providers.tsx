'use client';

import { CaptchaProvider } from '@captchax/nextjs';

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <CaptchaProvider 
      apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
      serverUrl={process.env.NEXT_PUBLIC_CAPTCHA_SERVER_URL}
    >
      {children}
    </CaptchaProvider>
  );
}

export default Providers;
