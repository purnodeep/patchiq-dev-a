import { useQuery } from '@tanstack/react-query';

export interface AgentBinaryInfo {
  os: string;
  arch: string;
  filename: string;
  url: string;
  size: number;
}

interface AgentBinaryRaw {
  name: string;
  size: number;
}

function parseBinary(raw: AgentBinaryRaw): AgentBinaryInfo | null {
  const m = /^patchiq-agent-(linux|darwin|windows)-(amd64|arm64)\.(tar\.gz|zip|exe)$/.exec(raw.name);
  if (!m) return null;
  const [, os, arch] = m;
  return {
    os,
    arch,
    filename: raw.name,
    size: raw.size,
    url: `/api/v1/agent-binaries/${encodeURIComponent(raw.name)}/download`,
  };
}

export function useAgentBinaries() {
  return useQuery({
    queryKey: ['agent-binaries'],
    queryFn: async () => {
      const res = await fetch('/api/v1/agent-binaries', {
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Failed to fetch agent binaries: ${res.status}`);
      const body = (await res.json()) as { data: AgentBinaryRaw[] };
      return body.data.map(parseBinary).filter((b): b is AgentBinaryInfo => b !== null);
    },
  });
}
