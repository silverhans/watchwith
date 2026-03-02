const API_BASE = '/api';

export async function createRoom(name: string): Promise<{ slug: string; name: string }> {
  const res = await fetch(`${API_BASE}/rooms`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) throw new Error('Failed to create room');
  return res.json();
}

export async function getRoom(slug: string): Promise<{ slug: string; name: string; peerCount: number }> {
  const res = await fetch(`${API_BASE}/rooms/${slug}`);
  if (!res.ok) throw new Error('Room not found');
  return res.json();
}
