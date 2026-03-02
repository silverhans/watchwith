import ReactPlayer from 'react-player';
import type { RefObject } from 'react';
import type ReactPlayerType from 'react-player';

interface VideoPlayerProps {
  playerRef: RefObject<ReactPlayerType | null>;
  url: string;
  playing: boolean;
  onPlay: () => void;
  onPause: () => void;
  onReady?: () => void;
  hidden?: boolean;
}

export function VideoPlayer({
  playerRef,
  url,
  playing,
  onPlay,
  onPause,
  onReady,
  hidden,
}: VideoPlayerProps) {
  if (!url) {
    return (
      <div className="player-placeholder">
        <p>No video loaded. Paste a URL above to start watching.</p>
      </div>
    );
  }

  return (
    <div
      className="player-wrapper"
      style={{ display: hidden ? 'none' : undefined }}
    >
      <ReactPlayer
        ref={playerRef}
        url={url}
        playing={playing}
        controls
        width="100%"
        height="100%"
        onPlay={onPlay}
        onPause={onPause}
        onReady={onReady}
        onError={(e) => console.error('ReactPlayer error:', e)}
      />
    </div>
  );
}
