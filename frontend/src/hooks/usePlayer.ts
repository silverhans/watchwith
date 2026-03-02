import { useCallback, useRef, useState } from 'react';
import type ReactPlayer from 'react-player';
import type { WSMessage, PlayerPayload } from '../types';

interface UsePlayerOptions {
  send: (type: WSMessage['type'], payload?: unknown) => void;
}

export function usePlayer({ send }: UsePlayerOptions) {
  const playerRef = useRef<ReactPlayer | null>(null);
  const [url, setUrl] = useState('');
  const [playing, setPlaying] = useState(false);
  const isRemoteAction = useRef(false);
  const readyRef = useRef(false);

  const getCurrentTime = (): number => {
    return playerRef.current?.getCurrentTime() ?? 0;
  };

  const seekTo = (time: number) => {
    playerRef.current?.seekTo(time, 'seconds');
  };

  const handleWSMessage = useCallback((msg: WSMessage) => {
    const payload = msg.payload as PlayerPayload | undefined;

    switch (msg.type) {
      case 'room:state': {
        const state = msg.payload as {
          videoUrl: string;
          playing: boolean;
          time: number;
        };
        if (state.videoUrl) setUrl(state.videoUrl);
        setPlaying(state.playing);
        const trySeek = () => {
          if (playerRef.current && readyRef.current) {
            isRemoteAction.current = true;
            playerRef.current.seekTo(state.time, 'seconds');
          } else {
            setTimeout(trySeek, 300);
          }
        };
        setTimeout(trySeek, 500);
        break;
      }
      case 'player:play':
        isRemoteAction.current = true;
        setPlaying(true);
        if (payload?.time !== undefined && playerRef.current) {
          playerRef.current.seekTo(payload.time, 'seconds');
        }
        break;
      case 'player:pause':
        isRemoteAction.current = true;
        setPlaying(false);
        if (payload?.time !== undefined && playerRef.current) {
          playerRef.current.seekTo(payload.time, 'seconds');
        }
        break;
      case 'player:seek':
        isRemoteAction.current = true;
        if (payload?.time !== undefined && playerRef.current) {
          playerRef.current.seekTo(payload.time, 'seconds');
        }
        break;
      case 'player:source':
        if (payload?.url) {
          readyRef.current = false;
          setUrl(payload.url);
          setPlaying(false);
        }
        break;
      case 'player:sync':
        if (payload?.time !== undefined && playerRef.current && readyRef.current) {
          const current = playerRef.current.getCurrentTime();
          if (Math.abs(current - payload.time) > 2) {
            isRemoteAction.current = true;
            playerRef.current.seekTo(payload.time, 'seconds');
          }
        }
        break;
    }
  }, []);

  const onReady = useCallback(() => {
    readyRef.current = true;
  }, []);

  const onPlay = useCallback(() => {
    if (isRemoteAction.current) {
      isRemoteAction.current = false;
      return;
    }
    setPlaying(true);
    const time = getCurrentTime();
    send('player:play', { time });
  }, [send]);

  const onPause = useCallback(() => {
    if (isRemoteAction.current) {
      isRemoteAction.current = false;
      return;
    }
    setPlaying(false);
    const time = getCurrentTime();
    send('player:pause', { time });
  }, [send]);

  const changeSource = useCallback((newUrl: string) => {
    readyRef.current = false;
    setUrl(newUrl);
    setPlaying(false);
    send('player:source', { url: newUrl });
  }, [send]);

  return {
    playerRef,
    url,
    playing,
    onPlay,
    onPause,
    onReady,
    getCurrentTime,
    seekTo,
    changeSource,
    handleWSMessage,
  };
}
