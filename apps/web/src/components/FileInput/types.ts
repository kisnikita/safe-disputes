export type UploadStatus = 'ready' | 'uploading' | 'failed';

export interface UploadItem {
  id: string;
  file: File;
  previewUrl: string | null;
  status: UploadStatus;
  progress: number;
  typeLabel: string;
  errorMessage?: string;
}

export interface Props {
  accept?: string;
  className?: string;
  label?: string;
  maxFiles?: number;
  maxFileSizeMb?: number;
  multiple?: boolean;
  onHasErrorChange?: (hasError: boolean) => void;
  onPrimaryFileChange?: (file: File | null) => void;
}