import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ComplianceGauges } from '../../../pages/dashboard/ComplianceGauges';

const frameworks = [
  { name: 'NIST CSF', rate: 82 },
  { name: 'PCI DSS', rate: 67 },
  { name: 'HIPAA', rate: 91 },
];

describe('ComplianceGauges', () => {
  it('renders the card title "Compliance Frameworks"', () => {
    render(<ComplianceGauges complianceRate={80} frameworks={frameworks} />);
    expect(screen.getByText('Compliance Frameworks')).toBeInTheDocument();
  });

  it('renders the overall compliance rate', () => {
    render(<ComplianceGauges complianceRate={80} frameworks={frameworks} />);
    // RingGauge renders "80%" inside an SVG text element
    expect(screen.getByText('80%')).toBeInTheDocument();
  });

  it('renders framework names as labels', () => {
    render(<ComplianceGauges complianceRate={80} frameworks={frameworks} />);
    expect(screen.getByText('NIST CSF')).toBeInTheDocument();
    expect(screen.getByText('PCI DSS')).toBeInTheDocument();
    expect(screen.getByText('HIPAA')).toBeInTheDocument();
  });

  it('renders percentage values for frameworks', () => {
    render(<ComplianceGauges complianceRate={80} frameworks={frameworks} />);
    expect(screen.getByText('82%')).toBeInTheDocument();
    expect(screen.getByText('67%')).toBeInTheDocument();
    expect(screen.getByText('91%')).toBeInTheDocument();
  });

  it('handles empty frameworks array gracefully', () => {
    render(<ComplianceGauges complianceRate={0} frameworks={[]} />);
    expect(screen.getByText('Compliance Frameworks')).toBeInTheDocument();
    expect(screen.getByText('Not Configured')).toBeInTheDocument();
    // No framework labels
    expect(screen.queryByText('NIST CSF')).not.toBeInTheDocument();
  });
});
