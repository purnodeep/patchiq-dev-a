import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

interface MockTenant {
  id: string;
  name: string;
  slug: string;
}

let mockUser: {
  user_id: string;
  organization?: { id: string; name: string; slug: string; type: string };
  active_tenant_id?: string;
  accessible_tenants?: MockTenant[];
};

vi.mock('../../../app/auth/AuthContext', () => ({
  useAuth: () => ({ user: mockUser }),
}));

const mockGet = vi.fn<() => string | null>();
const mockSet = vi.fn<(id: string | null) => void>();
const mockSubscribe = vi.fn<(cb: () => void) => () => void>(() => () => {});

vi.mock('../../../api/activeTenantStore', () => ({
  getActiveTenantId: () => mockGet(),
  setActiveTenantId: (id: string | null) => mockSet(id),
  subscribeActiveTenant: (cb: () => void) => mockSubscribe(cb),
}));

const mockClear = vi.fn();
vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({ clear: mockClear }),
}));

import { TenantSwitcher } from '../../../app/layout/TenantSwitcher';

const twoTenants: MockTenant[] = [
  { id: 'tenant-a', name: 'Acme Corp', slug: 'acme' },
  { id: 'tenant-b', name: 'Globex Inc', slug: 'globex' },
];

describe('TenantSwitcher', () => {
  beforeEach(() => {
    mockGet.mockReturnValue('tenant-a');
    mockSet.mockReset();
    mockClear.mockReset();
    mockUser = {
      user_id: 'user-1',
      organization: { id: 'org-1', name: 'MSP Org', slug: 'msp', type: 'msp' },
      active_tenant_id: 'tenant-a',
      accessible_tenants: twoTenants,
    };
  });

  afterEach(() => {
    cleanup();
  });

  it('renders the current tenant name in the trigger button', () => {
    render(<TenantSwitcher />);
    const trigger = screen.getByRole('button', { name: /Acme Corp/ });
    expect(trigger).toBeInTheDocument();
  });

  it('renders nothing when accessible_tenants length is 1', () => {
    mockUser = {
      user_id: 'user-1',
      organization: { id: 'org-1', name: 'Solo', slug: 'solo', type: 'direct' },
      active_tenant_id: 'tenant-a',
      accessible_tenants: [twoTenants[0]],
    };
    render(<TenantSwitcher />);
    expect(screen.queryByRole('button', { name: /tenant/i })).toBeNull();
  });

  it('opens dropdown on trigger click and closes on outside click', async () => {
    const user = userEvent.setup();
    render(<TenantSwitcher />, { container: document.body.appendChild(document.createElement('div')) });

    const trigger = screen.getByRole('button', { name: /Acme Corp/ });
    await user.click(trigger);

    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(screen.getByText('Globex Inc')).toBeInTheDocument();

    fireEvent.mouseDown(document.body);

    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('stays open after clicking inside the menu, then closes on outside click', async () => {
    // Regression test for the original onBlur bug: clicking inside the menu
    // (e.g., the menu container itself or an inert region) used to close the
    // dropdown because focus moved off the parent <div>. The mousedown-based
    // listener must NOT close the menu when the click target is contained
    // within the switcher.
    const user = userEvent.setup();
    render(<TenantSwitcher />);

    const trigger = screen.getByRole('button', { name: /Acme Corp/ });
    await user.click(trigger);

    const menu = screen.getByRole('menu');
    expect(menu).toBeInTheDocument();

    // Click inside the menu container — anywhere that isn't a menuitem.
    fireEvent.mouseDown(menu);

    // Menu must still be open.
    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(screen.getByText('Globex Inc')).toBeInTheDocument();

    // Now click truly outside.
    fireEvent.mouseDown(document.body);

    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('closes on Escape', async () => {
    const user = userEvent.setup();
    render(<TenantSwitcher />);

    const trigger = screen.getByRole('button', { name: /Acme Corp/ });
    await user.click(trigger);
    expect(screen.getByRole('menu')).toBeInTheDocument();

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(screen.queryByRole('menu')).toBeNull();
  });
});
