import { useState, useMemo } from 'react';
import { useCan } from '../../app/auth/AuthContext';
import { Download, Copy, Check, Terminal, Key, HardDrive, MousePointerClick } from 'lucide-react';
import { Button } from '@patchiq/ui';
import { useAgentBinaries, type AgentBinaryInfo } from '../../api/hooks/useAgentBinaries';
import { useCreateRegistration } from '../../api/hooks/useEndpoints';

interface PlatformGroup {
  os: string;
  label: string;
  icon: string;
  binaries: AgentBinaryInfo[];
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function archLabel(arch: string, os: string): string {
  if (arch === 'amd64') return 'AMD64 (x86_64)';
  if (arch === 'arm64') return os === 'darwin' ? 'ARM64 (Apple Silicon)' : 'ARM64 (aarch64)';
  return arch;
}

function osIcon(os: string): string {
  if (os === 'linux') return '\u{1F427}';
  if (os === 'darwin') return '\u{1F34E}';
  if (os === 'windows') return '\u{1FA9F}';
  return '\u{1F4E6}';
}

function osLabel(os: string): string {
  if (os === 'linux') return 'Linux';
  if (os === 'darwin') return 'macOS';
  if (os === 'windows') return 'Windows';
  return os;
}

function buildInstallCommand(binary: AgentBinaryInfo, serverUrl: string): string | null {
  if (binary.os === 'linux') return null;
  const filename = binary.filename;
  if (binary.os === 'windows') {
    // Windows: server address is baked into the binary at build time.
    // Operator double-clicks the .exe and pastes the token in the wizard.
    return `# Right-click ${filename} → "Run as administrator"\n# Paste your registration token in the wizard.`;
  }
  const binaryName = filename.endsWith('.tar.gz') ? filename.slice(0, -'.tar.gz'.length) : filename;
  if (filename.endsWith('.tar.gz')) {
    return `tar xzf ${filename} && chmod +x ${binaryName} && sudo ./${binaryName} install --server ${serverUrl}`;
  }
  return `chmod +x ${filename} && sudo ./${filename} install --server ${serverUrl}`;
}

function downloadHref(binary: AgentBinaryInfo): string {
  return `/api/v1/agent-binaries/${binary.filename}/download`;
}

export function AgentDownloadsPage() {
  const can = useCan();
  const { data: binaries, isLoading, isError } = useAgentBinaries();
  const createRegistration = useCreateRegistration();
  const [token, setToken] = useState<string | null>(null);
  const [tokenLoading, setTokenLoading] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [selectedBinary, setSelectedBinary] = useState<AgentBinaryInfo | null>(null);

  const serverUrl = `${window.location.protocol}//${window.location.host}`;

  const platformGroups: PlatformGroup[] = useMemo(() => {
    if (!binaries || binaries.length === 0) return [];
    const grouped = new Map<string, AgentBinaryInfo[]>();
    for (const b of binaries) {
      const existing = grouped.get(b.os) ?? [];
      existing.push(b);
      grouped.set(b.os, existing);
    }
    const order = ['linux', 'darwin', 'windows'];
    return order
      .filter((os) => grouped.has(os))
      .map((os) => ({
        os,
        label: osLabel(os),
        icon: osIcon(os),
        binaries: grouped.get(os)!,
      }));
  }, [binaries]);

  const handleGenerateToken = async () => {
    setTokenLoading(true);
    try {
      const result = await createRegistration.mutateAsync();
      setToken(result.registration_token);
    } catch {
      // mutation error is surfaced by TanStack Query
    } finally {
      setTokenLoading(false);
    }
  };

  const copyToClipboard = (text: string, field: string) => {
    void navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const installCommand = selectedBinary ? buildInstallCommand(selectedBinary, serverUrl) : null;

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Page header */}
      <div
        style={{
          borderBottom: '1px solid var(--border)',
          padding: '20px 24px',
        }}
      >
        <h1
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 22,
            fontWeight: 600,
            color: 'var(--text-emphasis)',
            letterSpacing: '-0.02em',
            marginBottom: 4,
          }}
        >
          Agent Downloads
        </h1>
        <p
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 13,
            color: 'var(--text-secondary)',
            margin: 0,
          }}
        >
          Enroll endpoints by following the steps below: generate a one-time registration token,
          download the agent for your target platform, and run the installer on the endpoint.
        </p>
      </div>

      {/* Content */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '24px',
        }}
      >
        {isLoading && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                style={{
                  height: 120,
                  borderRadius: 8,
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  animation: 'pulse 2s ease-in-out infinite',
                }}
              />
            ))}
          </div>
        )}

        {isError && (
          <div
            style={{
              padding: '40px 24px',
              textAlign: 'center',
              color: 'var(--text-secondary)',
              fontSize: 13,
            }}
          >
            Failed to load agent binaries. Please try again.
          </div>
        )}

        {!isLoading && !isError && platformGroups.length === 0 && (
          <div
            style={{
              padding: '60px 24px',
              textAlign: 'center',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 8,
            }}
          >
            <HardDrive
              style={{
                width: 40,
                height: 40,
                color: 'var(--text-muted)',
                marginBottom: 16,
                strokeWidth: 1.5,
              }}
            />
            <p
              style={{
                fontSize: 14,
                fontWeight: 500,
                color: 'var(--text-emphasis)',
                marginBottom: 8,
              }}
            >
              No agent binaries available
            </p>
            <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: 0 }}>
              Run{' '}
              <code
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 12,
                  padding: '2px 6px',
                  borderRadius: 4,
                  background: 'var(--bg-inset)',
                  border: '1px solid var(--border)',
                }}
              >
                make build-agents
              </code>{' '}
              to build them.
            </p>
          </div>
        )}

        {!isLoading && !isError && platformGroups.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 24, maxWidth: 800 }}>
            {/* Platform cards */}
            <div>
              <h2
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 15,
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  marginBottom: 12,
                }}
              >
                Platform Binaries
              </h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {platformGroups.map((group) => (
                  <div
                    key={group.os}
                    style={{
                      background: 'var(--bg-card)',
                      border: '1px solid var(--border)',
                      borderRadius: 8,
                      padding: '16px 20px',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 10,
                        marginBottom: 12,
                      }}
                    >
                      <span style={{ fontSize: 22 }}>{group.icon}</span>
                      <span
                        style={{
                          fontFamily: 'var(--font-sans)',
                          fontSize: 15,
                          fontWeight: 600,
                          color: 'var(--text-emphasis)',
                        }}
                      >
                        {group.label}
                      </span>
                    </div>

                    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                      {group.binaries.map((binary) => {
                        const isSelected = selectedBinary?.filename === binary.filename;
                        return (
                          <div
                            key={binary.filename}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              padding: '10px 14px',
                              borderRadius: 6,
                              border: isSelected
                                ? '1px solid var(--accent)'
                                : '1px solid var(--border)',
                              background: isSelected ? 'var(--bg-inset)' : 'transparent',
                              cursor: 'pointer',
                              transition: 'border-color 0.15s, background 0.15s',
                            }}
                            onClick={() => setSelectedBinary(binary)}
                          >
                            <div>
                              <div
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 12,
                                  color: 'var(--text-emphasis)',
                                  fontWeight: 500,
                                }}
                              >
                                {archLabel(binary.arch, binary.os)}
                              </div>
                              <div
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 11,
                                  color: 'var(--text-muted)',
                                  marginTop: 2,
                                }}
                              >
                                {binary.filename} &middot; {formatSize(binary.size)}
                              </div>
                            </div>
                            <Button variant="outline" size="sm" asChild>
                              <a
                                href={downloadHref(binary)}
                                download
                                onClick={(e) => e.stopPropagation()}
                              >
                                <Download style={{ width: 14, height: 14, marginRight: 6 }} />
                                Download
                              </a>
                            </Button>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Install instructions */}
            <div>
              <h2
                style={{
                  fontFamily: 'var(--font-sans)',
                  fontSize: 15,
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  marginBottom: 12,
                }}
              >
                Install Instructions
              </h2>

              <div
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 8,
                  padding: '20px',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 20,
                }}
              >
                {/* Step 1: Generate token */}
                <div style={{ display: 'flex', gap: 14 }}>
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      background: 'var(--accent)',
                      color: 'var(--text-on-color, #fff)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 13,
                      fontWeight: 600,
                      flexShrink: 0,
                    }}
                  >
                    1
                  </div>
                  <div style={{ flex: 1 }}>
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: 'var(--text-emphasis)',
                        marginBottom: 8,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                      }}
                    >
                      <Key style={{ width: 14, height: 14 }} />
                      Generate a Registration Token
                    </div>
                    <p
                      style={{
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        margin: '0 0 8px 0',
                        lineHeight: 1.5,
                      }}
                    >
                      Click the button below to generate a one-time token. You will need to paste
                      this token on the target machine during installation to authorize the agent
                      to connect to this server. Each token can only be used once.
                    </p>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleGenerateToken}
                        disabled={tokenLoading || !can('endpoints', 'create')}
                        title={
                          !can('endpoints', 'create') ? "You don't have permission" : undefined
                        }
                      >
                        {tokenLoading ? 'Generating...' : 'Generate Token'}
                      </Button>
                      {token && (
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                            flex: 1,
                            minWidth: 0,
                          }}
                        >
                          <code
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 11,
                              padding: '4px 8px',
                              borderRadius: 4,
                              background: 'var(--bg-inset)',
                              border: '1px solid var(--border)',
                              color: 'var(--text-emphasis)',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                              flex: 1,
                              minWidth: 0,
                            }}
                          >
                            {token}
                          </code>
                          <button
                            onClick={() => copyToClipboard(token, 'token')}
                            style={{
                              background: 'none',
                              border: 'none',
                              cursor: 'pointer',
                              padding: 4,
                              color: 'var(--text-secondary)',
                              flexShrink: 0,
                            }}
                            title="Copy token"
                          >
                            {copiedField === 'token' ? (
                              <Check style={{ width: 14, height: 14, color: 'var(--accent)' }} />
                            ) : (
                              <Copy style={{ width: 14, height: 14 }} />
                            )}
                          </button>
                        </div>
                      )}
                    </div>
                    {token && (
                      <p
                        style={{
                          fontSize: 11,
                          color: 'var(--accent)',
                          margin: '6px 0 0 0',
                          fontWeight: 500,
                        }}
                      >
                        Token generated. Copy it now — you will need it in Step 3.
                      </p>
                    )}
                    {createRegistration.isError && (
                      <p
                        style={{
                          fontSize: 12,
                          color: 'var(--signal-critical)',
                          marginTop: 6,
                          marginBottom: 0,
                        }}
                      >
                        Failed to generate token. Please try again.
                      </p>
                    )}
                  </div>
                </div>

                {/* Step 2: Download */}
                <div style={{ display: 'flex', gap: 14 }}>
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      background: 'var(--accent)',
                      color: 'var(--text-on-color, #fff)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 13,
                      fontWeight: 600,
                      flexShrink: 0,
                    }}
                  >
                    2
                  </div>
                  <div>
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: 'var(--text-emphasis)',
                        marginBottom: 4,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                      }}
                    >
                      <Download style={{ width: 14, height: 14 }} />
                      Download the Agent Binary
                    </div>
                    <p
                      style={{
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        margin: 0,
                        lineHeight: 1.5,
                      }}
                    >
                      {selectedBinary
                        ? <>Selected: <strong>{selectedBinary.filename}</strong> ({osLabel(selectedBinary.os)} {archLabel(selectedBinary.arch, selectedBinary.os)}). Transfer this file to the target machine where you want to install the agent.</>
                        : 'Select a platform binary from the list above, then click "Download" to save the installer. Transfer the downloaded file to the target endpoint.'}
                    </p>
                  </div>
                </div>

                {/* Step 3: Install */}
                <div style={{ display: 'flex', gap: 14 }}>
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      background: 'var(--accent)',
                      color: 'var(--text-on-color, #fff)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 13,
                      fontWeight: 600,
                      flexShrink: 0,
                    }}
                  >
                    3
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    {selectedBinary?.os === 'linux' ? (
                      <>
                        <div
                          style={{
                            fontSize: 13,
                            fontWeight: 600,
                            color: 'var(--text-emphasis)',
                            marginBottom: 8,
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                          }}
                        >
                          <MousePointerClick style={{ width: 14, height: 14 }} />
                          Install the Agent (Linux GUI)
                        </div>
                        <p
                          style={{
                            fontSize: 12,
                            color: 'var(--text-secondary)',
                            margin: '0 0 8px 0',
                            lineHeight: 1.5,
                          }}
                        >
                          On the target machine, follow these steps:
                        </p>
                        <ol
                          style={{
                            margin: 0,
                            paddingLeft: 20,
                            fontSize: 13,
                            color: 'var(--text-emphasis)',
                            lineHeight: 2,
                          }}
                        >
                          <li>Extract the downloaded <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11, padding: '1px 4px', borderRadius: 3, background: 'var(--bg-inset)', border: '1px solid var(--border)' }}>.tar.gz</code> archive.</li>
                          <li>
                            Double-click the <strong>Install PatchIQ Agent</strong> shortcut inside the extracted folder.
                          </li>
                          <li>A dialog will open asking for your <strong>registration token</strong> — paste the token you copied in Step 1.</li>
                          <li>The installer will connect to this server, register the endpoint, and start the agent service automatically.</li>
                        </ol>
                        <p
                          style={{
                            fontSize: 12,
                            color: 'var(--text-secondary)',
                            margin: '10px 0 0 0',
                            lineHeight: 1.5,
                          }}
                        >
                          <strong>Headless / SSH install:</strong> Run the following instead:
                        </p>
                        <div
                          style={{
                            position: 'relative',
                            background: 'var(--bg-inset)',
                            border: '1px solid var(--border)',
                            borderRadius: 6,
                            padding: '12px 40px 12px 14px',
                            marginTop: 6,
                          }}
                        >
                          <code
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 12,
                              color: 'var(--text-emphasis)',
                              wordBreak: 'break-all',
                              whiteSpace: 'pre-wrap',
                              lineHeight: 1.6,
                            }}
                          >
                            {`tar xzf ${selectedBinary.filename} && sudo ./${selectedBinary.filename.replace('.tar.gz', '')} install --server ${serverUrl} --token <YOUR_TOKEN> --non-interactive`}
                          </code>
                          <button
                            onClick={() => copyToClipboard(
                              `tar xzf ${selectedBinary.filename} && sudo ./${selectedBinary.filename.replace('.tar.gz', '')} install --server ${serverUrl} --token ${token || '<YOUR_TOKEN>'} --non-interactive`,
                              'command',
                            )}
                            style={{
                              position: 'absolute',
                              top: 8,
                              right: 8,
                              background: 'none',
                              border: 'none',
                              cursor: 'pointer',
                              padding: 4,
                              color: 'var(--text-secondary)',
                            }}
                            title="Copy install command"
                          >
                            {copiedField === 'command' ? (
                              <Check
                                style={{ width: 14, height: 14, color: 'var(--accent)' }}
                              />
                            ) : (
                              <Copy style={{ width: 14, height: 14 }} />
                            )}
                          </button>
                        </div>
                        <p
                          style={{
                            fontSize: 11,
                            color: 'var(--text-muted)',
                            margin: '4px 0 0 0',
                          }}
                        >
                          Replace <code style={{ fontFamily: 'var(--font-mono)', fontSize: 10, padding: '1px 3px', borderRadius: 3, background: 'var(--bg-inset)' }}>&lt;YOUR_TOKEN&gt;</code> with the token from Step 1{token ? ' (or click copy — the token is already included)' : ''}.
                        </p>
                      </>
                    ) : selectedBinary?.os === 'windows' ? (
                      <>
                        <div
                          style={{
                            fontSize: 13,
                            fontWeight: 600,
                            color: 'var(--text-emphasis)',
                            marginBottom: 8,
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                          }}
                        >
                          <MousePointerClick style={{ width: 14, height: 14 }} />
                          Install the Agent (Windows)
                        </div>
                        <p
                          style={{
                            fontSize: 12,
                            color: 'var(--text-secondary)',
                            margin: '0 0 8px 0',
                            lineHeight: 1.5,
                          }}
                        >
                          On the target Windows machine:
                        </p>
                        <ol
                          style={{
                            margin: 0,
                            paddingLeft: 20,
                            fontSize: 13,
                            color: 'var(--text-emphasis)',
                            lineHeight: 2,
                          }}
                        >
                          <li>Right-click the downloaded <strong>{selectedBinary.filename}</strong> and select <strong>&quot;Run as administrator&quot;</strong>.</li>
                          <li>The setup wizard will open — paste the <strong>registration token</strong> from Step 1 when prompted.</li>
                          <li>Click <strong>Install</strong>. The agent will register with this server and start as a Windows service.</li>
                        </ol>
                        <p
                          style={{
                            fontSize: 11,
                            color: 'var(--text-muted)',
                            margin: '8px 0 0 0',
                            lineHeight: 1.5,
                          }}
                        >
                          The server address is pre-configured in the binary — you only need the token.
                        </p>
                      </>
                    ) : (
                      <>
                        <div
                          style={{
                            fontSize: 13,
                            fontWeight: 600,
                            color: 'var(--text-emphasis)',
                            marginBottom: 8,
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                          }}
                        >
                          <Terminal style={{ width: 14, height: 14 }} />
                          Run the Install Command
                        </div>
                        {installCommand ? (
                          <>
                            <p
                              style={{
                                fontSize: 12,
                                color: 'var(--text-secondary)',
                                margin: '0 0 8px 0',
                                lineHeight: 1.5,
                              }}
                            >
                              Open a terminal on the target machine, navigate to where you saved the binary, and run:
                            </p>
                            <div
                              style={{
                                position: 'relative',
                                background: 'var(--bg-inset)',
                                border: '1px solid var(--border)',
                                borderRadius: 6,
                                padding: '12px 40px 12px 14px',
                              }}
                            >
                              <code
                                style={{
                                  fontFamily: 'var(--font-mono)',
                                  fontSize: 12,
                                  color: 'var(--text-emphasis)',
                                  wordBreak: 'break-all',
                                  whiteSpace: 'pre-wrap',
                                  lineHeight: 1.6,
                                }}
                              >
                                {installCommand}
                              </code>
                              <button
                                onClick={() => copyToClipboard(installCommand, 'command')}
                                style={{
                                  position: 'absolute',
                                  top: 8,
                                  right: 8,
                                  background: 'none',
                                  border: 'none',
                                  cursor: 'pointer',
                                  padding: 4,
                                  color: 'var(--text-secondary)',
                                }}
                                title="Copy install command"
                              >
                                {copiedField === 'command' ? (
                                  <Check
                                    style={{ width: 14, height: 14, color: 'var(--accent)' }}
                                  />
                                ) : (
                                  <Copy style={{ width: 14, height: 14 }} />
                                )}
                              </button>
                            </div>
                            <p
                              style={{
                                fontSize: 12,
                                color: 'var(--text-secondary)',
                                margin: '8px 0 0 0',
                                lineHeight: 1.5,
                              }}
                            >
                              The installer will prompt you to paste the <strong>registration token</strong> from Step 1. After pasting, the agent will connect to this server, register itself, and start running as a background service.
                            </p>
                          </>
                        ) : (
                          <p
                            style={{
                              fontSize: 12,
                              color: 'var(--text-muted)',
                              margin: 0,
                              fontStyle: 'italic',
                            }}
                          >
                            Select a platform binary above to see the install command.
                          </p>
                        )}
                      </>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
