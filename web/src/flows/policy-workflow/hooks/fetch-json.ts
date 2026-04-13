const headers = { 'Content-Type': 'application/json' };

async function parseErrorMessage(res: Response): Promise<string> {
  const text = await res.text();
  try {
    const json = JSON.parse(text);
    if (json.details && Array.isArray(json.details)) {
      const msgs = json.details.map((d: { message?: string }) => d.message).filter(Boolean);
      if (msgs.length > 0) return msgs.join('; ');
    }
    return json.message || `Request failed: ${res.status}`;
  } catch {
    return text || `Request failed: ${res.status}`;
  }
}

export async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    credentials: 'include',
    headers: { ...headers, ...((init?.headers as Record<string, string>) ?? {}) },
  });
  if (!res.ok) {
    throw new Error(await parseErrorMessage(res));
  }
  const text = await res.text();
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(`Expected JSON response but got: ${text.slice(0, 200)}`);
  }
}

export async function fetchVoid(url: string, init?: RequestInit): Promise<void> {
  const res = await fetch(url, {
    ...init,
    credentials: 'include',
    headers: { ...headers, ...((init?.headers as Record<string, string>) ?? {}) },
  });
  if (!res.ok) {
    throw new Error(await parseErrorMessage(res));
  }
}
