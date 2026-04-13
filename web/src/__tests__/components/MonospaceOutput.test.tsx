import { render, screen } from '@testing-library/react';
import { MonospaceOutput } from '../../components/MonospaceOutput';

describe('MonospaceOutput', () => {
  it('renders content in a pre tag', () => {
    render(<MonospaceOutput content="hello world" />);
    const pre = screen.getByText('hello world');
    expect(pre.tagName).toBe('PRE');
  });

  it('renders empty state when no content', () => {
    render(<MonospaceOutput content="" />);
    expect(screen.getByText('No output')).toBeInTheDocument();
  });
});
