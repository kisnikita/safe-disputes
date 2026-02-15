import React from 'react';
import { createPortal } from 'react-dom';
import { HIDE_THRESHOLD } from '../../utils/constants';
import { useScrollVisibility } from '../../hooks/useScrollVisibility';
import './RatingButton.css';

interface Props {
  onClick: () => void;
}

export const RatingButton: React.FC<Props> = ({ onClick }) => {
  const visible = useScrollVisibility(HIDE_THRESHOLD);

  const button = (
    <button className={`rating-button${visible ? '' : ' hidden'}`} onClick={onClick}>
      Рейтинг
    </button>
  );

  return createPortal(button, document.body);
};
