import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSession } from '../auth/session';
import { GameSettings } from '../games/GameSettings';
import { useGameSelection } from '../games/selection';
import type { Game } from '../games/types';
import { CreateGameDialog, FindGameDialog, MyGamesDialog } from './dialogs';

type Dialog = 'create' | 'find' | 'mine';

const menuAction = {
  create: 'Создать игру',
  find: 'Найти игру',
  mine: 'Мои игры',
} as const;

export function LobbyPage() {
  const { player, logout, error } = useSession();
  const { game, select } = useGameSelection();
  const navigate = useNavigate();
  const [dialog, setDialog] = useState<Dialog | null>(null);

  function enterGame(next: Game) {
    select(next);
    navigate(`/play/${next.id}`);
  }

  return (
    <main className="screen">
      <section className="panel">
        <h1>Привет, {player?.login}</h1>
        {error ? (
          <p className="form-error" role="alert">
            {error}
          </p>
        ) : null}
        <div className="menu">
          <button type="button" onClick={() => setDialog('create')}>
            {menuAction.create}
          </button>
          <button type="button" onClick={() => setDialog('find')}>
            {menuAction.find}
          </button>
          <button type="button" onClick={() => setDialog('mine')}>
            {menuAction.mine}
          </button>
          <button type="button" className="secondary" onClick={() => void logout()}>
            Выйти
          </button>
        </div>
      </section>
      {dialog === 'create' ? (
        <CreateGameDialog
          onClose={() => setDialog(null)}
          onCreated={(created) => {
            select(created);
            setDialog(null);
          }}
        />
      ) : null}
      {dialog === 'find' ? (
        <FindGameDialog onClose={() => setDialog(null)} onJoin={enterGame} />
      ) : null}
      {dialog === 'mine' ? (
        <MyGamesDialog onClose={() => setDialog(null)} onEnter={enterGame} />
      ) : null}
      {game && dialog === null ? (
        <div className="corner-stack">
          <GameSettings
            game={game}
            onRenamed={(next) => {
              select(next);
            }}
          />
        </div>
      ) : null}
    </main>
  );
}
