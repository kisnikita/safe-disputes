export const LIMIT_EXCEEDED_MESSAGE = 'Превышено допустимое количество файлов';

export const getFileTooLargeMessage = (file: File, maxFileSizeMb: number): string => (
  `Файл не удалось сжать до ${maxFileSizeMb} МБ\n(получилось ${formatSizeMb(file.size)} МБ)`
);

const formatSizeMb = (bytes: number): string => (
  (bytes / (1024 * 1024))
    .toFixed(2)
    .replace(/\.00$/, '')
    .replace(/(\.\d)0$/, '$1')
);

export const getUnsupportedFormatMessage = (acceptTokens: string[]): string => (
  `Неподдерживаемый формат.\nДопустимы: ${getAllowedFormatsText(acceptTokens)}`
);

export const getAllowedFormatsText = (acceptTokens: string[]): string => {
  const humanReadable = [...new Set(acceptTokens.map(tokenToHumanLabel))].join(', ');
  return humanReadable || 'изображения (включая GIF), PDF';
};

export const tokenToHumanLabel = (token: string): string => {
  if (token.startsWith('.')) {
    return token.slice(1).toUpperCase();
  }

  if (token.endsWith('/*')) {
    const baseType = token.slice(0, -2);
    if (baseType === 'image') return 'изображения (включая GIF)';
    if (baseType === 'video') return 'видео';
    if (baseType === 'audio') return 'аудио';
    return `${baseType} файлы`;
  }

  if (token === 'application/pdf') return 'PDF';
  return token;
};
