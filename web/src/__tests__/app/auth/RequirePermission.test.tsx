import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

const mockNavigate = vi.fn();

vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return { ...actual, useNavigate: () => mockNavigate };
});

let mockCan = (_resource: string, _action: string) => true;

vi.mock('../../../app/auth/AuthContext', () => ({
  useAuth: () => ({
    can: (resource: string, action: string) => mockCan(resource, action),
  }),
}));

import { RequirePermission } from '../../../app/auth/RequirePermission';

describe('RequirePermission', () => {
  beforeEach(() => {
    mockCan = () => true;
    mockNavigate.mockClear();
  });

  it('renders children when user has permission', () => {
    mockCan = () => true;
    render(
      <RequirePermission resource="endpoints" action="read">
        <div>Protected Content</div>
      </RequirePermission>,
    );
    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });

  it('shows "Access Restricted" when user lacks permission', () => {
    mockCan = () => false;
    render(
      <RequirePermission resource="endpoints" action="read">
        <div>Protected Content</div>
      </RequirePermission>,
    );
    expect(screen.getByText('Access Restricted')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('shows "Go to Dashboard" button when restricted', () => {
    mockCan = () => false;
    render(
      <RequirePermission resource="endpoints" action="read">
        <div>Protected Content</div>
      </RequirePermission>,
    );
    expect(screen.getByText('Go to Dashboard')).toBeInTheDocument();
  });

  it('"Go to Dashboard" navigates to "/" when clicked', async () => {
    mockCan = () => false;
    const user = userEvent.setup();
    render(
      <RequirePermission resource="endpoints" action="read">
        <div>Protected Content</div>
      </RequirePermission>,
    );
    await user.click(screen.getByText('Go to Dashboard'));
    expect(mockNavigate).toHaveBeenCalledWith('/');
  });
});
