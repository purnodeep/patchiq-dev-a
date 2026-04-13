import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { PatchesTimeline } from '../../../pages/dashboard/PatchesTimeline';

const sampleData = [
  { date: 'Jan 1', critical: 5, high: 10, medium: 15 },
  { date: 'Jan 8', critical: 3, high: 8, medium: 12 },
  { date: 'Jan 15', critical: 7, high: 14, medium: 20 },
];

describe('PatchesTimeline', () => {
  it('renders card title "Patches Over Time"', () => {
    render(<PatchesTimeline data={sampleData} />);
    expect(screen.getByText('Patches Over Time')).toBeInTheDocument();
  });

  it('renders subtitle text', () => {
    render(<PatchesTimeline data={sampleData} />);
    expect(screen.getByText('Last 90 days • severity stacked')).toBeInTheDocument();
  });

  it('renders legend labels Critical, High, Medium', () => {
    render(<PatchesTimeline data={sampleData} />);
    expect(screen.getByText('Critical')).toBeInTheDocument();
    expect(screen.getByText('High')).toBeInTheDocument();
    expect(screen.getByText('Medium')).toBeInTheDocument();
  });

  it('renders empty state message when data is empty', () => {
    render(<PatchesTimeline data={[]} />);
    expect(screen.getByText('No patch data available')).toBeInTheDocument();
  });

  it('renders without crashing with valid data', () => {
    const { container } = render(<PatchesTimeline data={sampleData} />);
    expect(container.firstChild).toBeTruthy();
  });
});
