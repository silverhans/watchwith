export type MessageType =
  | 'room:join'
  | 'room:leave'
  | 'room:state'
  | 'room:peers'
  | 'player:play'
  | 'player:pause'
  | 'player:seek'
  | 'player:source'
  | 'player:sync'
  | 'chat:message'
  | 'chat:history'
  | 'rtc:offer'
  | 'rtc:answer'
  | 'rtc:ice'
  | 'screen:start'
  | 'screen:stop';

export interface WSMessage {
  type: MessageType;
  payload?: unknown;
  from?: string;
  to?: string;
}

export interface PeerInfo {
  id: string;
  username: string;
  isHost: boolean;
}

export interface RoomState {
  videoUrl: string;
  playing: boolean;
  time: number;
  peers: PeerInfo[];
  hostId: string;
  yourId: string;
}

export interface PlayerPayload {
  time: number;
  url?: string;
}

export interface ChatMessage {
  text: string;
  username: string;
  time: number;
}

export interface PeersPayload {
  peers: PeerInfo[];
}

export interface RoomInfo {
  slug: string;
  name: string;
  peerCount: number;
}
