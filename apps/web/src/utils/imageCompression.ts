export interface ImageCompressionOptions {
  maxWidth?: number;
  maxHeight?: number;
  quality?: number;
  minBytes?: number;
}

const DEFAULT_MAX_WIDTH = 1920;
const DEFAULT_MAX_HEIGHT = 1920;
const DEFAULT_QUALITY = 0.82;
const DEFAULT_MIN_BYTES = 400 * 1024;

const COMPRESSIBLE_TYPES = new Set([
  'image/jpeg',
  'image/jpg',
  'image/png',
  'image/webp',
]);

const LOSSY_TYPES = new Set([
  'image/jpeg',
  'image/jpg',
  'image/webp',
]);

const getOutputType = (inputType: string): string => {
  if (inputType === 'image/jpg') return 'image/jpeg';
  if (COMPRESSIBLE_TYPES.has(inputType)) return inputType;
  return 'image/jpeg';
};

const loadImage = (file: File): Promise<HTMLImageElement> =>
  new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const image = new Image();

    image.onload = () => {
      URL.revokeObjectURL(url);
      resolve(image);
    };
    image.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to decode image'));
    };

    image.src = url;
  });

const toBlob = (
  canvas: HTMLCanvasElement,
  type: string,
  quality?: number,
): Promise<Blob | null> =>
  new Promise(resolve => {
    canvas.toBlob(resolve, type, quality);
  });

export const compressImageIfNeeded = async (
  file: File,
  options: ImageCompressionOptions = {},
): Promise<File> => {
  if (!file.type.startsWith('image/')) return file;
  if (!COMPRESSIBLE_TYPES.has(file.type)) return file;

  const maxWidth = options.maxWidth ?? DEFAULT_MAX_WIDTH;
  const maxHeight = options.maxHeight ?? DEFAULT_MAX_HEIGHT;
  const quality = options.quality ?? DEFAULT_QUALITY;
  const minBytes = options.minBytes ?? DEFAULT_MIN_BYTES;

  if (file.size < minBytes) return file;

  const image = await loadImage(file);
  const width = image.naturalWidth || image.width;
  const height = image.naturalHeight || image.height;
  if (!width || !height) return file;

  const scale = Math.min(1, maxWidth / width, maxHeight / height);
  const targetWidth = Math.max(1, Math.round(width * scale));
  const targetHeight = Math.max(1, Math.round(height * scale));

  const canvas = document.createElement('canvas');
  canvas.width = targetWidth;
  canvas.height = targetHeight;

  const context = canvas.getContext('2d');
  if (!context) return file;

  context.drawImage(image, 0, 0, targetWidth, targetHeight);

  const outputType = getOutputType(file.type);
  const blob = await toBlob(
    canvas,
    outputType,
    LOSSY_TYPES.has(outputType) ? quality : undefined,
  );

  if (!blob) return file;
  if (blob.size >= file.size) return file;

  return new File([blob], file.name, {
    type: outputType,
    lastModified: file.lastModified,
  });
};
