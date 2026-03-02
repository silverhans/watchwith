import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { createRoom } from '../services/api';

export function Home() {
  const navigate = useNavigate();
  const [roomName, setRoomName] = useState('');
  const [joinSlug, setJoinSlug] = useState('');
  const [username, setUsername] = useState('');
  const [loading, setLoading] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) return;
    setLoading(true);
    try {
      const room = await createRoom(roomName || 'Watch Party');
      navigate(`/room/${room.slug}?username=${encodeURIComponent(username.trim())}`);
    } catch (err) {
      console.error('Failed to create room:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleJoin = (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !joinSlug.trim()) return;
    // Extract slug from URL if full URL pasted
    let slug = joinSlug.trim();
    const match = slug.match(/\/room\/([a-z0-9]+)/);
    if (match) slug = match[1];
    navigate(`/room/${slug}?username=${encodeURIComponent(username.trim())}`);
  };

  return (
    <div className="home">
      <div className="home-container">
        <h1>WatchWith</h1>
        <p className="subtitle">Watch videos together in real-time</p>

        <div className="username-section">
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Your name"
            maxLength={30}
            className="input-main"
          />
        </div>

        <div className="home-sections">
          <div className="section">
            <h2>Create a Room</h2>
            <form onSubmit={handleCreate}>
              <input
                type="text"
                value={roomName}
                onChange={(e) => setRoomName(e.target.value)}
                placeholder="Room name (optional)"
                maxLength={50}
              />
              <button type="submit" disabled={loading || !username.trim()}>
                {loading ? 'Creating...' : 'Create Room'}
              </button>
            </form>
          </div>

          <div className="divider">or</div>

          <div className="section">
            <h2>Join a Room</h2>
            <form onSubmit={handleJoin}>
              <input
                type="text"
                value={joinSlug}
                onChange={(e) => setJoinSlug(e.target.value)}
                placeholder="Room link or code"
              />
              <button type="submit" disabled={!username.trim() || !joinSlug.trim()}>
                Join
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
