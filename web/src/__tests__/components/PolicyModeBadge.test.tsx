import { render, screen } from '@testing-library/react';
import { PolicyModeBadge } from '../../components/PolicyModeBadge';

describe('PolicyModeBadge', () => {
  it('renders all_available as "All Patches"', () => {
    render(<PolicyModeBadge mode="all_available" />);
    expect(screen.getByText('All Patches')).toBeInTheDocument();
  });

  it('renders by_severity', () => {
    render(<PolicyModeBadge mode="by_severity" />);
    expect(screen.getByText('By Severity')).toBeInTheDocument();
  });

  it('renders by_cve_list', () => {
    render(<PolicyModeBadge mode="by_cve_list" />);
    expect(screen.getByText('By CVE')).toBeInTheDocument();
  });

  it('renders by_regex', () => {
    render(<PolicyModeBadge mode="by_regex" />);
    expect(screen.getByText('By Regex')).toBeInTheDocument();
  });
});
