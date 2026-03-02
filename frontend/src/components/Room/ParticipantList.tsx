import type { PeerInfo } from '../../types';

interface ParticipantListProps {
  peers: PeerInfo[];
  myId: string;
}

export function ParticipantList({ peers, myId }: ParticipantListProps) {
  return (
    <div className="participant-list">
      <h3>Participants ({peers.length})</h3>
      <ul>
        {peers.map((peer) => (
          <li key={peer.id} className={peer.id === myId ? 'me' : ''}>
            {peer.username}
            {peer.isHost && <span className="host-badge">Host</span>}
            {peer.id === myId && <span className="you-badge">You</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
