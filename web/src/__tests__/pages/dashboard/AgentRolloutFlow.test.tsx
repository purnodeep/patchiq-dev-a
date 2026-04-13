import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { AgentRolloutFlow } from '../../../pages/dashboard/AgentRolloutFlow';

const defaultRollout = {
  total: 100,
  installed: 87,
  enrolled: 80,
  healthy: 75,
  scanning: 70,
};

describe('AgentRolloutFlow', () => {
  it('renders card title "Agent Rollout"', () => {
    render(<AgentRolloutFlow rollout={defaultRollout} />);
    expect(screen.getByText('Agent Rollout')).toBeDefined();
  });

  it('renders all 5 stage labels', () => {
    render(<AgentRolloutFlow rollout={defaultRollout} />);
    expect(screen.getByText('Total')).toBeDefined();
    expect(screen.getByText('Installed')).toBeDefined();
    expect(screen.getByText('Enrolled')).toBeDefined();
    expect(screen.getByText('Healthy')).toBeDefined();
    expect(screen.getByText('Scanning')).toBeDefined();
  });

  it('renders count values', () => {
    render(<AgentRolloutFlow rollout={defaultRollout} />);
    // aria-label attributes carry the count values
    expect(screen.getByLabelText('Total: 100')).toBeDefined();
    expect(screen.getByLabelText('Installed: 87')).toBeDefined();
    expect(screen.getByLabelText('Enrolled: 80')).toBeDefined();
    expect(screen.getByLabelText('Healthy: 75')).toBeDefined();
    expect(screen.getByLabelText('Scanning: 70')).toBeDefined();
  });

  it('shows drop-off values between stages', () => {
    render(<AgentRolloutFlow rollout={defaultRollout} />);
    expect(screen.getByText('−13 not installed')).toBeDefined();
    expect(screen.getByText('−7 not enrolled')).toBeDefined();
    expect(screen.getByText('−5 not healthy')).toBeDefined();
    expect(screen.getByText('−5 not scanning')).toBeDefined();
  });

  it('handles zero total gracefully without division by zero', () => {
    const zeroRollout = { total: 0, installed: 0, enrolled: 0, healthy: 0, scanning: 0 };
    // Should render without throwing
    expect(() => render(<AgentRolloutFlow rollout={zeroRollout} />)).not.toThrow();
    expect(screen.getByText('Agent Rollout')).toBeDefined();
    // All percentages should be 0%
    const pctElements = screen.getAllByText('0%');
    expect(pctElements.length).toBe(5);
  });
});
