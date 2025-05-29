import { useEffect, useState, useRef } from 'react';
import { retrieveRawInitData } from '@telegram-apps/sdk';

export function useTelegramAuth() {
  const [status, setStatus] = useState<'loading' | 'error' | 'ready'>('loading');
  const [error, setError] = useState<string | null>(null);
  const didFetchRef = useRef(false);

  useEffect(() => {
    if (didFetchRef.current) return;
    didFetchRef.current = true; 

    (async () => {
      const initDataRaw = retrieveRawInitData();
      if (!initDataRaw) {
        setError('initDataRaw отсутствует');
        return setStatus('error');
      }
      try {
        const res = await fetch('/api/v1/auth/telegram', {
          method: 'POST',
          headers: { Authorization: `tma ${initDataRaw}` },
        });
        if (!res.ok) throw new Error(`Ошибка ${res.status}`);
        localStorage.setItem('initDataRaw', initDataRaw);
        setStatus('ready');
      } catch (e: any) {
        setError(e.message);
        setStatus('error');
      }
    })();
  }, []);

  return { status, error };
}
