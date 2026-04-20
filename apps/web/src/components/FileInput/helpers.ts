import { UploadItem } from "./Types";

export const isCountedItem = (item: UploadItem): boolean => item.status !== 'failed';

export const parseAcceptTokens = (accept?: string): string[] => (
  (accept ?? '')
    .split(',')
    .map(token => token.trim().toLowerCase())
    .filter(Boolean)
);

export const isFileAccepted = (file: File, acceptTokens: string[]): boolean => {
  if (acceptTokens.length === 0) return true;

  const fileType = file.type.toLowerCase();
  const fileName = file.name.toLowerCase();

  return acceptTokens.some(token => {
    if (token.startsWith('.')) return fileName.endsWith(token);
    if (token.endsWith('/*')) {
      const mimePrefix = token.slice(0, -1);
      return fileType.startsWith(mimePrefix);
    }
    return fileType === token;
  });
};

export const getFileTypeLabel = (file: File): string => {
  if (file.type.startsWith('image/')) return 'Image';
  const ext = file.name.split('.').pop()?.toUpperCase();
  if (file.type === 'application/pdf' || ext === 'PDF') return 'PDF';
  if (ext) return `${ext} File`;
  return 'File';
};

export const revokePreview = (item: UploadItem) => {
  if (item.previewUrl) {
    URL.revokeObjectURL(item.previewUrl);
  }
};
