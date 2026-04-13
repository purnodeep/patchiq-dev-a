import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AgentDownloadsPage } from '../../../pages/agent-downloads/AgentDownloadsPage';
import type { AgentBinaryInfo } from '../../../api/hooks/useAgentBinaries';

const linuxBinary: AgentBinaryInfo = {
  os: 'linux',
  arch: 'amd64',
  filename: 'patchiq-agent-linux-amd64.tar.gz',
  url: '',
  size: 10_000_000,
};

const windowsBinary: AgentBinaryInfo = {
  os: 'windows',
  arch: 'amd64',
  filename: 'patchiq-agent-windows-amd64.exe',
  url: '',
  size: 12_000_000,
};

const darwinBinary: AgentBinaryInfo = {
  os: 'darwin',
  arch: 'arm64',
  filename: 'patchiq-agent-darwin-arm64.tar.gz',
  url: '',
  size: 11_000_000,
};

const mockMutateAsync = vi.fn();

vi.mock('../../../api/hooks/useAgentBinaries', () => ({
  useAgentBinaries: vi.fn(),
}));

vi.mock('../../../api/hooks/useEndpoints', () => ({
  useCreateRegistration: () => ({
    mutateAsync: mockMutateAsync,
    isError: false,
  }),
}));

import { useAgentBinaries } from '../../../api/hooks/useAgentBinaries';

const mockedUseAgentBinaries = vi.mocked(useAgentBinaries);

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <AgentDownloadsPage />
    </QueryClientProvider>,
  );
}

describe('AgentDownloadsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows GUI instructions when a Linux binary is selected', () => {
    mockedUseAgentBinaries.mockReturnValue({
      data: [linuxBinary],
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useAgentBinaries>);

    renderPage();

    // Click the linux binary row to select it
    const row = screen.getByText('AMD64 (x86_64)').closest('div[style]')!;
    fireEvent.click(row);

    // Should show GUI install steps
    expect(screen.getByText('Install the Agent')).toBeInTheDocument();
    expect(screen.getByText('Extract the downloaded archive.')).toBeInTheDocument();
    expect(screen.getByText('Install PatchIQ Agent')).toBeInTheDocument();
    expect(
      screen.getByText('Paste your registration token in the window that opens.'),
    ).toBeInTheDocument();

    // Should NOT show a code block with shell command
    expect(screen.queryByText('Run the Install Command')).not.toBeInTheDocument();
    const codeElements = document.querySelectorAll('code');
    // Only code element should be the token area (not visible yet) or none related to install
    for (const el of codeElements) {
      expect(el.textContent).not.toMatch(/tar xzf|chmod|sudo/);
    }
  });

  it('shows shell command when a Windows binary is selected', () => {
    mockedUseAgentBinaries.mockReturnValue({
      data: [windowsBinary],
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useAgentBinaries>);

    renderPage();

    const row = screen.getByText('AMD64 (x86_64)').closest('div[style]')!;
    fireEvent.click(row);

    // Should show terminal command
    expect(screen.getByText('Run the Install Command')).toBeInTheDocument();
    expect(screen.getByText(/\\patchiq-agent-windows-amd64\.exe install/)).toBeInTheDocument();

    // Should NOT show GUI instructions
    expect(screen.queryByText('Install the Agent')).not.toBeInTheDocument();
    expect(screen.queryByText('Install PatchIQ Agent')).not.toBeInTheDocument();
  });

  it('uses API endpoint for Linux download links and /repo/files for others', () => {
    mockedUseAgentBinaries.mockReturnValue({
      data: [linuxBinary, windowsBinary, darwinBinary],
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useAgentBinaries>);

    renderPage();

    const downloadLinks = screen.getAllByRole('link');

    // Find Linux download link
    const linuxLink = downloadLinks.find((a) =>
      a.getAttribute('href')?.includes('patchiq-agent-linux'),
    );
    expect(linuxLink).toBeDefined();
    expect(linuxLink!.getAttribute('href')).toBe(
      '/api/v1/agent-binaries/patchiq-agent-linux-amd64.tar.gz/download',
    );

    // Find Windows download link
    const windowsLink = downloadLinks.find((a) =>
      a.getAttribute('href')?.includes('patchiq-agent-windows'),
    );
    expect(windowsLink).toBeDefined();
    expect(windowsLink!.getAttribute('href')).toBe(
      '/repo/files/windows/patchiq-agent-windows-amd64.exe',
    );

    // Find macOS download link
    const darwinLink = downloadLinks.find((a) =>
      a.getAttribute('href')?.includes('patchiq-agent-darwin'),
    );
    expect(darwinLink).toBeDefined();
    expect(darwinLink!.getAttribute('href')).toBe(
      '/repo/files/darwin/patchiq-agent-darwin-arm64.tar.gz',
    );
  });
});
