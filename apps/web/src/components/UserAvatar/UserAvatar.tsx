import { CSSProperties } from 'react';
import { Avatar } from '@telegram-apps/telegram-ui';
import { withAvatarCacheBuster } from '../../utils/avatarUrl';
import './UserAvatar.css';

interface UserAvatarProps {
  username: string;
  photoUrl?: string | null;
  size?: 20 | 24 | 28 | 40 | 48 | 96;
  className?: string;
}

function getAvatarAcronym(username: string): string {
  const normalized = username.replace(/^@+/, '').trim();
  if (!normalized) return '?';
  return normalized.slice(0, 1).toUpperCase();
}

function joinClasses(...classes: Array<string | undefined>): string {
  return classes.filter(Boolean).join(' ');
}

export function UserAvatar({ username, photoUrl, size = 24, className }: UserAvatarProps) {
  const sizeStyle: CSSProperties = {
    width: size,
    height: size,
    minWidth: size,
    minHeight: size,
    fontSize: Math.max(10, Math.round(size * 0.5)),
    lineHeight: 1,
  };

  if (photoUrl) {
    return (
      <Avatar
        size={size}
        src={withAvatarCacheBuster(photoUrl)}
        alt={username}
        className={joinClasses('user-avatar', 'user-avatar-image', className)}
      />
    );
  }

  return (
    <span
      className={joinClasses('user-avatar', 'user-avatar-acronym', className)}
      style={sizeStyle}
      aria-label={`Аватар ${username}`}
      title={username}
    >
      {getAvatarAcronym(username)}
    </span>
  );
}
