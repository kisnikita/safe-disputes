import React, { useEffect, useState } from 'react';
import loading0 from '../../../assets/loading0-icon.svg';
import loading45 from '../../../assets/loading45-icon.svg';
import loading90 from '../../../assets/loading90-icon.svg';
import loading135 from '../../../assets/loading135-icon.svg';
import loading180 from '../../../assets/loading180-icon.svg';
import loading235 from '../../../assets/loading235-icon.svg';
import loading270 from '../../../assets/loading270-icon.svg';
import loading315 from '../../../assets/loading315-icon.svg';
import './Loader.css';

const frames = [
  loading0,
  loading45,
  loading90,
  loading135,
  loading180,
  loading235,
  loading270,
  loading315,
];

type LoaderProps = {
  size?: number;
  className?: string;
  label?: string;
};

export const Loader: React.FC<LoaderProps> = ({
  size = 42,
  className,
  label = 'Загрузка',
}) => {
  const [frameIndex, setFrameIndex] = useState(0);

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setFrameIndex(prev => (prev + 1) % frames.length);
    }, 120);
    return () => window.clearInterval(intervalId);
  }, []);

  const classes = ['loader', className].filter(Boolean).join(' ');

  return (
    <div className={classes} role="status" aria-live="polite" aria-label={label}>
      <img
        className="loader__image"
        src={frames[frameIndex]}
        aria-hidden="true"
        width={size}
        height={size}
      />
    </div>
  );
};
