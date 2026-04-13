import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { DeploymentPipelineWidget } from '../DeploymentPipelineWidget';

const STAGES = ['Scan', 'Approve', 'Stage', 'Deploy', 'Verify'];

describe('DeploymentPipelineWidget', () => {
  it('renders title', () => {
    render(<DeploymentPipelineWidget currentStage={2} stages={STAGES} />);
    expect(screen.getByText('Deployment Pipeline')).toBeInTheDocument();
  });

  it('renders all stages', () => {
    render(<DeploymentPipelineWidget currentStage={2} stages={STAGES} />);
    STAGES.forEach((s) => expect(screen.getByText(s)).toBeInTheDocument());
  });
});
