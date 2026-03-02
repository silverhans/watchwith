interface ControlsProps {
  voiceActive: boolean;
  muted: boolean;
  sharing: boolean;
  onToggleVoice: () => void;
  onToggleMute: () => void;
  onToggleScreenShare: () => void;
}

export function Controls({
  voiceActive,
  muted,
  sharing,
  onToggleVoice,
  onToggleMute,
  onToggleScreenShare,
}: ControlsProps) {
  return (
    <div className="controls">
      <button
        className={`control-btn ${voiceActive ? 'active' : ''}`}
        onClick={onToggleVoice}
      >
        {voiceActive ? 'Leave Voice' : 'Join Voice'}
      </button>
      {voiceActive && (
        <button
          className={`control-btn ${muted ? 'muted' : ''}`}
          onClick={onToggleMute}
        >
          {muted ? 'Unmute' : 'Mute'}
        </button>
      )}
      <button
        className={`control-btn ${sharing ? 'active' : ''}`}
        onClick={onToggleScreenShare}
      >
        {sharing ? 'Stop Share' : 'Share Screen'}
      </button>
    </div>
  );
}
