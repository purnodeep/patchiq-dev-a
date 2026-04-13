import { render, screen } from '@testing-library/react';

let mockCan: (resource: string, action: string) => boolean;

vi.mock('react-router', () => ({
  NavLink: ({
    to,
    children,
    style,
    'aria-label': ariaLabel,
  }: {
    to: string;
    children: (props: { isActive: boolean }) => React.ReactNode;
    style?: React.CSSProperties;
    'aria-label'?: string;
    end?: boolean;
  }) => (
    <a href={to} style={style} aria-label={ariaLabel}>
      {children({ isActive: false })}
    </a>
  ),
  useLocation: () => ({ pathname: '/endpoints' }),
}));

vi.mock('../../../app/auth/AuthContext', () => ({
  useAuth: () => ({ user: { name: 'Test User', email: 'test@test.com' } }),
  useCan: () => (resource: string, action: string) => mockCan(resource, action),
}));

vi.mock('../../../api/hooks/useAlerts', () => ({
  useAlertCount: () => ({ data: undefined }),
}));

vi.mock('../../../api/hooks/useAuth', () => ({
  useLogout: () => ({ mutate: vi.fn() }),
}));

vi.mock('../../../pages/settings/SettingsSidebar', () => ({
  SettingsSidebar: () => <div data-testid="settings-sidebar">Settings Sidebar</div>,
}));

import { AppSidebar } from '../../../app/layout/AppSidebar';

const allNavLabels = [
  'Dashboard',
  'Endpoints',
  'Patches',
  'CVEs',
  'Policies',
  'Deployments',
  'Workflows',
  'Compliance',
  'Alerts',
  'Audit',
  'Settings',
  'Agent Downloads',
];

describe('AppSidebar RBAC behavior', () => {
  beforeEach(() => {
    mockCan = () => true;
  });

  it('renders all nav items as links for a super admin', () => {
    mockCan = () => true;
    render(<AppSidebar />);

    for (const label of allNavLabels) {
      const link = screen.getByRole('link', { name: label });
      expect(link).toBeInTheDocument();
      expect(link.tagName).toBe('A');
    }

    // No lock icons — restricted divs have title containing "don't have access"
    const restrictedItems = screen.queryAllByTitle(/don't have access/i);
    expect(restrictedItems).toHaveLength(0);
  });

  it('renders restricted items as divs with lock treatment for a help desk user', () => {
    const allowedResources = new Set(['endpoints', 'patches', 'deployments']);
    mockCan = (resource: string) => allowedResources.has(resource);
    render(<AppSidebar />);

    // These should be links (their resource is in allowedResources)
    const expectedLinks = [
      'Dashboard',
      'Endpoints',
      'Patches',
      'CVEs',
      'Deployments',
      'Agent Downloads',
    ];
    for (const label of expectedLinks) {
      const link = screen.getByRole('link', { name: label });
      expect(link.tagName).toBe('A');
    }

    // These should be restricted (divs, not links)
    const expectedRestricted = [
      'Policies',
      'Workflows',
      'Compliance',
      'Alerts',
      'Audit',
      'Settings',
    ];
    for (const label of expectedRestricted) {
      expect(screen.queryByRole('link', { name: label })).toBeNull();

      const restrictedDiv = screen.getByTitle(`You don't have access to ${label}`);
      expect(restrictedDiv).toBeInTheDocument();
      expect(restrictedDiv.tagName).toBe('DIV');
      expect(restrictedDiv.style.cursor).toBe('not-allowed');
      expect(restrictedDiv.style.opacity).toBe('0.4');
    }
  });

  it('renders all items as restricted divs for a fully restricted user', () => {
    mockCan = () => false;
    render(<AppSidebar />);

    // No links at all
    const links = screen.queryAllByRole('link');
    expect(links).toHaveLength(0);

    // All items should be restricted
    for (const label of allNavLabels) {
      const restrictedDiv = screen.getByTitle(`You don't have access to ${label}`);
      expect(restrictedDiv).toBeInTheDocument();
      expect(restrictedDiv.tagName).toBe('DIV');
      expect(restrictedDiv.style.cursor).toBe('not-allowed');
    }
  });
});
