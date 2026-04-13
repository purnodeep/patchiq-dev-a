import { render, screen } from '@testing-library/react';
import { OrgSettingsPage } from '../../../pages/organizations/OrgSettingsPage';

vi.mock('../../../app/auth/AuthContext', () => ({
  useAuth: () => ({
    user: {
      user_id: 'u1',
      organization: {
        id: 'org-123',
        name: 'Acme Holdings',
        slug: 'acme',
        type: 'msp' as const,
      },
    },
  }),
}));

describe('OrgSettingsPage', () => {
  it('renders the current organization name', () => {
    render(<OrgSettingsPage />);
    // Name appears in both CardTitle and the Name field row
    expect(screen.getAllByText('Acme Holdings').length).toBeGreaterThan(0);
    expect(screen.getByText('acme')).toBeInTheDocument();
    expect(screen.getByText('MSP')).toBeInTheDocument();
    expect(screen.getByText('org-123')).toBeInTheDocument();
  });
});
