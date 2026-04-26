const AVATAR_CACHE_BUSTER = Date.now().toString();

export function withAvatarCacheBuster(url?: string | null): string | undefined {
  if (!url) return undefined;
  if (!/^https?:\/\//i.test(url)) return url;
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}v=${AVATAR_CACHE_BUSTER}`;
}
