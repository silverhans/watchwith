import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { getRoom } from '../services/api';
import { useWebSocket } from '../hooks/useWebSocket';
import { usePlayer } from '../hooks/usePlayer';
import { useVoice } from '../hooks/useVoice';
import { useScreenShare } from '../hooks/useScreenShare';
import { VideoPlayer } from '../components/Player/VideoPlayer';
import { ChatPanel } from '../components/Chat/ChatPanel';
import { ParticipantList } from '../components/Room/ParticipantList';
import { Controls } from '../components/Room/Controls';
import { ScreenView } from '../components/ScreenShare/ScreenView';
import type { ChatMessage, PeerInfo, RoomState, WSMessage, PeersPayload } from '../types';

export function Room() {
  const { slug } = useParams<{ slug: string }>();
  const [searchParams] = useSearchParams();
  const username = searchParams.get('username') || 'Guest';

  const [roomName, setRoomName] = useState('');
  const [roomNotFound, setRoomNotFound] = useState(false);
  const [peers, setPeers] = useState<PeerInfo[]>([]);
  const [myId, setMyId] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [videoInput, setVideoInput] = useState('');

  const syncIntervalRef = useRef<ReturnType<typeof setInterval>>();
  const isHostRef = useRef(false);

  const { send, connected } = useWebSocket({
    slug: slug!,
    username,
    onMessage: useCallback((msg: WSMessage) => {
      handleMessage(msg);
    }, []),
  });

  const player = usePlayer({ send });
  const voice = useVoice({ send, myId, peers });
  const screenShare = useScreenShare({ send, myId, peers });

  // We need a stable reference to handleMessage that can access latest state
  const handleMessageRef = useRef<(msg: WSMessage) => void>();
  handleMessageRef.current = (msg: WSMessage) => {
    switch (msg.type) {
      case 'room:state': {
        const state = msg.payload as RoomState;
        setMyId(state.yourId);
        setPeers(state.peers);
        isHostRef.current = state.hostId === state.yourId;
        player.handleWSMessage(msg);

        // Start sync interval if host
        if (isHostRef.current) {
          clearInterval(syncIntervalRef.current);
          syncIntervalRef.current = setInterval(() => {
            const time = player.playerRef.current?.getCurrentTime() ?? 0;
            send('player:sync', { time });
          }, 5000);
        }
        break;
      }
      case 'room:peers': {
        const payload = msg.payload as PeersPayload;
        setPeers(payload.peers);
        break;
      }
      case 'chat:message': {
        const chat = msg.payload as ChatMessage;
        setMessages((prev) => [...prev, chat]);
        break;
      }
      case 'chat:history': {
        const history = msg.payload as ChatMessage[];
        setMessages(history);
        break;
      }
      case 'player:play':
      case 'player:pause':
      case 'player:seek':
      case 'player:source':
      case 'player:sync':
        player.handleWSMessage(msg);
        break;
      case 'rtc:offer':
      case 'rtc:answer':
      case 'rtc:ice': {
        const payload = msg.payload as Record<string, unknown>;
        if (payload?.screen) {
          screenShare.handleWSMessage(msg);
        } else {
          voice.handleWSMessage(msg);
        }
        break;
      }
      case 'screen:start':
      case 'screen:stop':
        screenShare.handleWSMessage(msg);
        break;
    }
  };

  function handleMessage(msg: WSMessage) {
    handleMessageRef.current?.(msg);
  }

  useEffect(() => {
    if (!slug) return;
    getRoom(slug)
      .then((room) => setRoomName(room.name))
      .catch(() => setRoomNotFound(true));
  }, [slug]);

  useEffect(() => {
    return () => {
      clearInterval(syncIntervalRef.current);
    };
  }, []);

  const handleSendChat = useCallback((text: string) => {
    send('chat:message', { text, time: Date.now() });
  }, [send]);

  const handleChangeSource = (e: React.FormEvent) => {
    e.preventDefault();
    if (videoInput.trim()) {
      player.changeSource(videoInput.trim());
      setVideoInput('');
    }
  };

  if (roomNotFound) {
    return (
      <div className="room-error">
        <h2>Room not found</h2>
        <a href="/">Back to home</a>
      </div>
    );
  }

  const showScreenShare = screenShare.remoteStream && screenShare.sharingPeerId;

  return (
    <div className="room">
      <header className="room-header">
        <h1>{roomName || 'Loading...'}</h1>
        <div className="room-info">
          <span className={`connection-status ${connected ? 'connected' : ''}`}>
            {connected ? 'Connected' : 'Connecting...'}
          </span>
          <span className="room-slug">Room: {slug}</span>
        </div>
      </header>

      <div className="room-content">
        <div className="main-area">
          <form className="video-url-form" onSubmit={handleChangeSource}>
            <input
              type="text"
              value={videoInput}
              onChange={(e) => setVideoInput(e.target.value)}
              placeholder="Paste YouTube, Vimeo, Twitch, or direct video URL..."
            />
            <button type="submit" disabled={!videoInput.trim()}>
              Load
            </button>
          </form>

          <div className="video-area">
            {showScreenShare && screenShare.remoteStream ? (
              <ScreenView stream={screenShare.remoteStream} />
            ) : null}
            <VideoPlayer
              playerRef={player.playerRef}
              url={player.url}
              playing={player.playing}
              onPlay={player.onPlay}
              onPause={player.onPause}
              onReady={player.onReady}
              hidden={!!showScreenShare}
            />
          </div>

          <Controls
            voiceActive={voice.voiceActive}
            muted={voice.muted}
            sharing={screenShare.sharing}
            onToggleVoice={voice.voiceActive ? voice.stopVoice : voice.startVoice}
            onToggleMute={voice.toggleMute}
            onToggleScreenShare={
              screenShare.sharing ? screenShare.stopSharing : screenShare.startSharing
            }
          />
        </div>

        <aside className="sidebar">
          <ParticipantList peers={peers} myId={myId} />
          <ChatPanel messages={messages} onSend={handleSendChat} username={username} />
        </aside>
      </div>
    </div>
  );
}
