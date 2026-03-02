import { useCallback, useRef, useState } from 'react';
import type { WSMessage, PeerInfo } from '../types';

const ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
  ],
};

interface UseScreenShareOptions {
  send: (type: WSMessage['type'], payload?: unknown, to?: string) => void;
  myId: string;
  peers: PeerInfo[];
}

export function useScreenShare({ send, myId, peers }: UseScreenShareOptions) {
  const [sharing, setSharing] = useState(false);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [sharingPeerId, setSharingPeerId] = useState<string | null>(null);
  const localStream = useRef<MediaStream | null>(null);
  const peerConnections = useRef<Map<string, RTCPeerConnection>>(new Map());

  const createPC = useCallback((peerId: string, stream?: MediaStream): RTCPeerConnection => {
    const pc = new RTCPeerConnection(ICE_SERVERS);

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        send('rtc:ice', { candidate: event.candidate, screen: true }, peerId);
      }
    };

    pc.ontrack = (event) => {
      setRemoteStream(event.streams[0]);
    };

    if (stream) {
      stream.getTracks().forEach((track) => pc.addTrack(track, stream));
    }

    peerConnections.current.set(peerId, pc);
    return pc;
  }, [send]);

  const startSharing = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: true,
        audio: true,
      });
      localStream.current = stream;
      setSharing(true);
      send('screen:start', { from: myId });

      // Send stream to all peers
      for (const peer of peers) {
        if (peer.id === myId) continue;
        const pc = createPC(peer.id, stream);
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        send('rtc:offer', { sdp: pc.localDescription, screen: true }, peer.id);
      }

      // Handle user stopping share via browser UI
      stream.getVideoTracks()[0].onended = () => {
        stopSharing();
      };
    } catch (err) {
      console.error('Screen share failed:', err);
    }
  }, [peers, myId, send, createPC]);

  const stopSharing = useCallback(() => {
    localStream.current?.getTracks().forEach((t) => t.stop());
    localStream.current = null;
    peerConnections.current.forEach((pc) => pc.close());
    peerConnections.current.clear();
    setSharing(false);
    setRemoteStream(null);
    setSharingPeerId(null);
    send('screen:stop', {});
  }, [send]);

  const handleWSMessage = useCallback(async (msg: WSMessage) => {
    const payload = msg.payload as Record<string, unknown> | undefined;
    if (!payload) return;

    switch (msg.type) {
      case 'screen:start':
        setSharingPeerId(msg.from ?? null);
        break;
      case 'screen:stop':
        setSharingPeerId(null);
        setRemoteStream(null);
        peerConnections.current.forEach((pc) => pc.close());
        peerConnections.current.clear();
        break;
      case 'rtc:offer': {
        if (!payload.screen) break;
        const pc = createPC(msg.from!, undefined);
        await pc.setRemoteDescription(new RTCSessionDescription(payload.sdp as RTCSessionDescriptionInit));
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        send('rtc:answer', { sdp: pc.localDescription, screen: true }, msg.from!);
        break;
      }
      case 'rtc:answer': {
        if (!payload.screen) break;
        const pc = peerConnections.current.get(msg.from!);
        if (pc) {
          await pc.setRemoteDescription(new RTCSessionDescription(payload.sdp as RTCSessionDescriptionInit));
        }
        break;
      }
      case 'rtc:ice': {
        if (!payload.screen) break;
        const pc = peerConnections.current.get(msg.from!);
        if (pc && payload.candidate) {
          await pc.addIceCandidate(new RTCIceCandidate(payload.candidate as RTCIceCandidateInit));
        }
        break;
      }
    }
  }, [send, createPC]);

  return {
    sharing,
    remoteStream,
    sharingPeerId,
    startSharing,
    stopSharing,
    handleWSMessage,
  };
}
