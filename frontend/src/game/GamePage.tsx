import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useSession } from '../auth/session';
import { findGame } from '../games/api';
import { GameSettings } from '../games/GameSettings';
import { useGameSelection } from '../games/selection';
import type { Game } from '../games/types';
import { mountGame } from './mountGame';

export function GamePage() {
  const { gameId = '' } = useParams();
  const navigate = useNavigate();
  const { player } = useSession();
  const { select } = useGameSelection();
  const hostRef = useRef<HTMLDivElement>(null);
  const [game, setGame] = useState<Game | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    findGame(gameId).then((result) => {
      if (cancelled) {
        return;
      }
      if (!result.ok) {
        setGame(null);
        setError(result.message);
      } else {
        setGame(result.data);
        setError('');
        if (player?.id === result.data.owner.id) {
          select(result.data);
        }
      }
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [gameId, player?.id, select]);

  const isOwner = Boolean(game && player && game.owner.id === player.id);

  useEffect(() => {
    const parent = hostRef.current;
    if (!parent || !isOwner) {
      return;
    }
    const world = mountGame(parent);
    return () => {
      world.destroy(true);
    };
  }, [isOwner, game?.id]);

  if (loading) {
    return (
      <main className="screen">
        <p>Загрузка игры…</p>
      </main>
    );
  }

  if (error || !game) {
    return (
      <main className="screen">
        <section className="panel">
          <p className="form-error" role="alert">
            {error || 'Игра не найдена'}
          </p>
          <button type="button" onClick={() => navigate('/lobby')}>
            В лобби
          </button>
        </section>
      </main>
    );
  }

  if (!isOwner) {
    return (
      <main className="screen">
        <section className="panel">
          <h1>{game.name}</h1>
          <p className="game-id">{game.id}</p>
          <p>Войти в чужую игру пока нельзя.</p>
          <button type="button" onClick={() => navigate('/lobby')}>
            В лобби
          </button>
        </section>
      </main>
    );
  }

  return (
    <div className="game-page">
      <div ref={hostRef} className="game-host" />
      <div className="corner-stack">
        <GameSettings
          game={game}
          onRenamed={(next) => {
            setGame(next);
            select(next);
          }}
        />
        <button type="button" className="game-exit" onClick={() => navigate('/lobby')}>
          В лобби
        </button>
      </div>
    </div>
  );
}
