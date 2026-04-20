import React, { ChangeEvent, CSSProperties, MouseEvent, useEffect, useRef, useState } from 'react';
import './FileInput.css';
import { compressImageIfNeeded } from '../../utils/imageCompression';
import { UploadItem, Props } from './Types'
import { CloseIcon, FailedIcon, FileIcon, PaperclipIcon } from './icons';
import { parseAcceptTokens, isCountedItem, revokePreview, getFileTypeLabel, isFileAccepted } from './helpers';
import { getFileTooLargeMessage, getUnsupportedFormatMessage, LIMIT_EXCEEDED_MESSAGE } from './messages';
import { FILE_INPUT_DEFAULT_ACCEPT, FILE_INPUT_DEFAULT_MAX_FILES, FILE_INPUT_DEFAULT_MAX_FILE_SIZE_MB, FILE_PROGRESS_INTERVAL_MS, INITIAL_PROGRESS, IMAGE_COMPRESSION_OPTIONS } from './config';


export const FileInput: React.FC<Props> = ({
  accept = FILE_INPUT_DEFAULT_ACCEPT,
  className,
  label = 'Attach file',
  maxFiles = FILE_INPUT_DEFAULT_MAX_FILES,
  maxFileSizeMb = FILE_INPUT_DEFAULT_MAX_FILE_SIZE_MB,
  multiple = true,
  onHasErrorChange,
  onPrimaryFileChange,
}) => {
  const [items, setItems] = useState<UploadItem[]>([]);
  const itemsRef = useRef<UploadItem[]>([]);
  const nextIdRef = useRef(1);
  const normalizedMaxFiles = Math.max(1, maxFiles);
  const maxFileSizeBytes = maxFileSizeMb * 1024 * 1024;
  const acceptTokens = parseAcceptTokens(accept);
  const unsupportedFormatMessage = getUnsupportedFormatMessage(acceptTokens);
  const countedItems = items.filter(isCountedItem).length;
  const hasUploadingItems = items.some(item => item.status === 'uploading');
  const isLimitReached = multiple && countedItems >= normalizedMaxFiles;

  useEffect(() => {
    itemsRef.current = items;
  }, [items]);

  useEffect(() => {
    if (!hasUploadingItems) return;

    const id = window.setInterval(() => {
      setItems(prev =>
        prev.map(item => {
          if (item.status !== 'uploading') return item;
          const delta = item.progress < 78
            ? 7 + Math.floor(Math.random() * 13)
            : 2 + Math.floor(Math.random() * 5);
          const nextProgress = Math.min(100, item.progress + delta);
          if (nextProgress >= 100) {
            return { ...item, progress: 100, status: 'ready' };
          }
          return { ...item, progress: nextProgress };
        }),
      );
    }, FILE_PROGRESS_INTERVAL_MS);

    return () => {
      clearInterval(id);
    };
  }, [hasUploadingItems]);

  useEffect(() => {
    return () => {
      itemsRef.current.forEach(revokePreview);
    };
  }, []);

  useEffect(() => {
    if (!onPrimaryFileChange) return;
    const primaryImage = [...items]
      .reverse()
      .find(item => item.status === 'ready' && item.file.type.startsWith('image/'));
    onPrimaryFileChange(primaryImage?.file ?? null);
  }, [items, onPrimaryFileChange]);

  useEffect(() => {
    if (!onHasErrorChange) return;
    onHasErrorChange(items.some(item => item.status === 'failed'));
  }, [items, onHasErrorChange]);

  const createFailedItem = (file: File, errorMessage: string): UploadItem => ({
    id: `upload-${nextIdRef.current++}`,
    file,
    previewUrl: null,
    status: 'failed',
    progress: 0,
    typeLabel: getFileTypeLabel(file),
    errorMessage,
  });

  const createUploadingItem = (file: File): UploadItem => ({
    id: `upload-${nextIdRef.current++}`,
    file,
    previewUrl: file.type.startsWith('image/') ? URL.createObjectURL(file) : null,
    status: 'uploading',
    progress: INITIAL_PROGRESS,
    typeLabel: getFileTypeLabel(file),
  });

  const handleChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(event.target.files ?? []);
    event.target.value = '';

    if (selected.length === 0) return;

    const validationItems: UploadItem[] = [];
    const filesForProcessing: File[] = [];
    let remainingSlots = Math.max(
      0,
      normalizedMaxFiles - itemsRef.current.filter(isCountedItem).length,
    );

    selected.forEach(file => {
      if (!isFileAccepted(file, acceptTokens)) {
        validationItems.push(createFailedItem(file, unsupportedFormatMessage));
        return;
      }

      if (remainingSlots <= 0) {
        validationItems.push(createFailedItem(file, LIMIT_EXCEEDED_MESSAGE));
        return;
      }

      remainingSlots -= 1;
      filesForProcessing.push(file);
    });

    const prepared = await Promise.all(
      filesForProcessing.map(async file => {
        try {
          return await compressImageIfNeeded(file, IMAGE_COMPRESSION_OPTIONS);
        } catch {
          return file;
        }
      }),
    );

    const processedItems = prepared.map(file => (
      file.size > maxFileSizeBytes
        ? createFailedItem(file, getFileTooLargeMessage(file, maxFileSizeMb))
        : createUploadingItem(file)
    ));

    setItems(prev => [...prev, ...validationItems, ...processedItems]);
  };

  const removeItem = (id: string, event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setItems(prev => {
      const target = prev.find(item => item.id === id);
      if (target) revokePreview(target);
      return prev.filter(item => item.id !== id);
    });
  };

  const getSubtitle = (item: UploadItem): string => {
    if (item.status === 'uploading') return `${item.progress}%`;
    if (item.status === 'failed') return item.errorMessage ?? 'Failed';
    return item.typeLabel;
  };

  const rootClassName = `file-input${className ? ` ${className}` : ''}`;

  return (
    <div className={rootClassName}>
      {items.length > 0 && (
        <ul className="file-input-list">
          {items.map(item => {
            const progressStyle = { ['--fi-progress' as string]: item.progress } as CSSProperties;
            return (
              <li key={item.id} className="file-input-item">
                <div className="file-input-item-icon-box">
                  {item.status === 'failed' ? (
                    <div className="file-input-item-icon file-input-item-icon-failed">
                      <FailedIcon />
                    </div>
                  ) : item.status === 'uploading' ? (
                    <div
                      className="file-input-item-icon file-input-item-icon-progress"
                      style={progressStyle}
                    />
                  ) : item.previewUrl ? (
                    <div className="file-input-item-icon file-input-item-icon-image">
                      <img src={item.previewUrl} alt="" />
                    </div>
                  ) : (
                    <div className="file-input-item-icon file-input-item-icon-file">
                      <FileIcon />
                    </div>
                  )}
                </div>

                <div className="file-input-item-content">
                  <div className="file-input-item-name" title={item.file.name}>
                    {item.file.name}
                  </div>
                  <div
                    className={`file-input-item-subtitle${item.status === 'failed' ? ' is-failed' : ''
                      }`}
                  >
                    {getSubtitle(item)}
                  </div>
                </div>

                <button
                  type="button"
                  className="file-input-item-remove"
                  onMouseDown={event => {
                    event.preventDefault();
                    event.stopPropagation();
                  }}
                  onClick={event => removeItem(item.id, event)}
                  aria-label={`Удалить файл ${item.file.name}`}
                >
                  <CloseIcon />
                </button>
              </li>
            );
          })}
        </ul>
      )}

      <div className="file-input-attach-row">
        <label className={`file-input-attach${isLimitReached ? ' is-disabled' : ''}`}>
          <input
            className="file-input-native"
            type="file"
            accept={accept}
            onChange={handleChange}
            multiple={multiple && normalizedMaxFiles > 1}
            disabled={isLimitReached}
          />
          <span className="file-input-attach-icon">
            <PaperclipIcon />
          </span>
          <span className="file-input-attach-label">{label}</span>
        </label>
        {multiple && (
          <span className="file-input-counter">
            {countedItems}/{normalizedMaxFiles}
          </span>
        )}
      </div>
    </div>
  );
};
