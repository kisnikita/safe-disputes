export async function apiFetch(input: RequestInfo, init?: RequestInit) {
  const initDataRaw = localStorage.getItem('initDataRaw');
  if (!initDataRaw) throw new Error('initDataRaw отсутствует');

  // создаём заголовки, включая авторизацию
  const headers = new Headers(init?.headers);
  headers.set('Authorization', `tma ${initDataRaw}`);

  const isFormData = init?.body instanceof FormData;

  const response = await fetch(input, {
    ...init,
    // если это не FormData, тогда ставим JSON
    headers: isFormData ? headers : (() => {
      headers.set('Content-Type','application/json');
      return headers;
    })(),
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(`HTTP ${response.status}: ${text}`);
  }
  return response;
}
