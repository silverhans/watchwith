import { useEffect, useRef } from 'react';

interface ScreenViewProps {
  stream: MediaStream;
}

export function ScreenView({ stream }: ScreenViewProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  return (
    <div className="screen-view">
      <video ref={videoRef} autoPlay playsInline style={{ width: '100%', height: '100%' }} />
    </div>
  );
}
