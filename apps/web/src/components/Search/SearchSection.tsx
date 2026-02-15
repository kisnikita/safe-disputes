import { EmptyState } from '../EmptyState/EmptyState';
import './SearchSection.css';

export function SearchSection() {
  return (
    <EmptyState
      className="search-placeholder"
      message="Скоро вы сможете искать других пользователей и их пари"
      variant="comingSoon"
    />
  );
}
