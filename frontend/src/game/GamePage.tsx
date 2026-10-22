import { useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { mountGame } from './mountGame';

export function GamePage() {
  const hostRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const parent = hostRef.current;
    if (!parent) {
      return;
    }
    const game = mountGame(parent);
    return () => {
      game.destroy(true);
    };
  }, []);

  return (
    <div className="game-page">
      <div ref={hostRef} className="game-host" />
      <button type="button" className="game-exit" onClick={() => navigate('/lobby')}>
        В лобби
      </button>
    </div>
  );
}
