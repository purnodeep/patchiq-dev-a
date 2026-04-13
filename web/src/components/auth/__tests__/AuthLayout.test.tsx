import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { AuthLayout } from '../AuthLayout';

describe('AuthLayout', () => {
  it('renders children in right panel', () => {
    render(
      <AuthLayout>
        <p>Test Form</p>
      </AuthLayout>,
    );
    expect(screen.getByText('Test Form')).toBeInTheDocument();
  });

  it('renders branding text in left panel', () => {
    render(
      <AuthLayout>
        <div />
      </AuthLayout>,
    );
    expect(screen.getByText(/Enterprise patch management/i)).toBeInTheDocument();
  });

  it('renders PatchIQ logo text', () => {
    render(
      <AuthLayout>
        <div />
      </AuthLayout>,
    );
    // PatchIQ appears in both the desktop left panel h1 and the mobile logo span
    expect(screen.getAllByText('PatchIQ').length).toBeGreaterThanOrEqual(1);
  });
});
