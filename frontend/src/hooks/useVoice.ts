import { useCallback, useEffect, useRef, useState } from 'react';
import type { WSMessage, PeerInfo } from '../types';

const ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
  ],
};

interface UseVoiceOptions {
  send: (type: WSMessage['type'], payload?: unknown, to?: string) => void;
  myId: string;
  peers: PeerInfo[];
}

export function useVoice({ send, myId, peers }: UseVoiceOptions) {
  const [muted, setMuted] = useState(true);
  const [voiceActive, setVoiceActive] = useState(false);
  const localStream = useRef<MediaStream | null>(null);
  const peerConnections = useRef<Map<string, RTCPeerConnection>>(new Map());
  const remoteAudios = useRef<Map<string, HTMLAudioElement>>(new Map());

  const createPeerConnection = useCallback((peerId: string): RTCPeerConnection => {
    const pc = new RTCPeerConnection(ICE_SERVERS);

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        send('rtc:ice', { candidate: event.candidate }, peerId);
      }
    };

    pc.ontrack = (event) => {
      let audio = remoteAudios.current.get(peerId);
      if (!audio) {
        audio = new Audio();
        audio.autoplay = true;
        remoteAudios.current.set(peerId, audio);
      }
      audio.srcObject = event.streams[0];
    };

    if (localStream.current) {
      localStream.current.getTracks().forEach((track) => {
        pc.addTrack(track, localStream.current!);
      });
    }

    peerConnections.current.set(peerId, pc);
    return pc;
  }, [send]);

  const startVoice = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      localStream.current = stream;
      setVoiceActive(true);
      setMuted(false);

      // Initiate connections with all existing peers
      for (const peer of peers) {
        if (peer.id === myId) continue;
        const pc = createPeerConnection(peer.id);
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        send('rtc:offer', { sdp: pc.localDescription }, peer.id);
      }
    } catch (err) {
      console.error('Failed to get microphone:', err);
    }
  }, [peers, myId, send, createPeerConnection]);

  const stopVoice = useCallback(() => {
    localStream.current?.getTracks().forEach((t) => t.stop());
    localStream.current = null;
    peerConnections.current.forEach((pc) => pc.close());
    peerConnections.current.clear();
    remoteAudios.current.forEach((a) => { a.srcObject = null; });
    remoteAudios.current.clear();
    setVoiceActive(false);
    setMuted(true);
  }, []);

  const toggleMute = useCallback(() => {
    if (!localStream.current) return;
    const audioTrack = localStream.current.getAudioTracks()[0];
    if (audioTrack) {
      audioTrack.enabled = !audioTrack.enabled;
      setMuted(!audioTrack.enabled);
    }
  }, []);

  const handleWSMessage = useCallback(async (msg: WSMessage) => {
    const payload = msg.payload as Record<string, unknown> | undefined;
    if (!payload || !msg.from) return;

    switch (msg.type) {
      case 'rtc:offer': {
        let pc = peerConnections.current.get(msg.from);
        if (!pc) pc = createPeerConnection(msg.from);
        await pc.setRemoteDescription(new RTCSessionDescription(payload.sdp as RTCSessionDescriptionInit));
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        send('rtc:answer', { sdp: pc.localDescription }, msg.from);
        break;
      }
      case 'rtc:answer': {
        const pc = peerConnections.current.get(msg.from);
        if (pc) {
          await pc.setRemoteDescription(new RTCSessionDescription(payload.sdp as RTCSessionDescriptionInit));
        }
        break;
      }
      case 'rtc:ice': {
        const pc = peerConnections.current.get(msg.from);
        if (pc && payload.candidate) {
          await pc.addIceCandidate(new RTCIceCandidate(payload.candidate as RTCIceCandidateInit));
        }
        break;
      }
    }
  }, [send, createPeerConnection]);

  useEffect(() => {
    return () => {
      stopVoice();
    };
  }, [stopVoice]);

  return {
    muted,
    voiceActive,
    startVoice,
    stopVoice,
    toggleMute,
    handleWSMessage,
  };
}
