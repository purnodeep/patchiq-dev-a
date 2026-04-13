import { render, screen } from '@testing-library/react';
import { CVSSScore } from '../../components/CVSSScore';

describe('CVSSScore', () => {
  it('renders score value', () => {
    render(<CVSSScore score={9.8} />);
    expect(screen.getByText('9.8')).toBeInTheDocument();
  });

  it('renders dash for null score', () => {
    render(<CVSSScore score={null} />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('applies red color for critical score (>= 9.0)', () => {
    const { container } = render(<CVSSScore score={9.8} />);
    const el = container.firstChild as HTMLElement;
    expect(el.textContent).toBe('9.8');
    expect(el.style.color).toBe('var(--signal-critical)');
  });

  it('applies warning color for high score (>= 7.0)', () => {
    const { container } = render(<CVSSScore score={7.5} />);
    const el = container.firstChild as HTMLElement;
    expect(el.style.color).toBe('var(--signal-warning)');
  });

  it('applies text-secondary color for medium score (>= 4.0)', () => {
    const { container } = render(<CVSSScore score={5.0} />);
    const el = container.firstChild as HTMLElement;
    expect(el.style.color).toBe('var(--text-secondary)');
  });

  it('applies text-muted color for low score (< 4.0)', () => {
    const { container } = render(<CVSSScore score={2.0} />);
    const el = container.firstChild as HTMLElement;
    expect(el.style.color).toBe('var(--text-muted)');
  });
});
