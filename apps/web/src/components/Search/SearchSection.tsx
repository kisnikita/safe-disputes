import { EmptyState } from '../EmptyState/EmptyState';
import './SearchSection.css';

export function SearchSection() {
  return (
    <EmptyState
      className="search-placeholder"
      message="Скоро будет добавлено"
      variant="comingSoon"
    />
  );
}
