import { useEffect, useState, type FormEvent } from 'react';
import { useSession } from '../auth/session';
import { renameGame } from './api';
import type { Game } from './types';
import { maxGameNameLength } from './types';

type GameSettingsProps = {
  game: Game;
  onRenamed: (game: Game) => void;
};

const copyLabel = {
  idle: 'Скопировать',
  done: 'Скопировано',
} as const;

export function GameSettings({ game, onRenamed }: GameSettingsProps) {
  const { player } = useSession();
  const canRename = player?.id === game.owner.id;
  const [open, setOpen] = useState(false);
  const [name, setName] = useState(game.name);
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setName(game.name);
    setError('');
  }, [game]);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!canRename) {
      return;
    }
    setPending(true);
    setError('');
    const result = await renameGame(game.id, name);
    setPending(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    onRenamed(result.data);
  }

  async function copyID() {
    try {
      await navigator.clipboard.writeText(game.id);
      setCopied(true);
    } catch {
      setCopied(false);
      setError('Не удалось скопировать ID');
    }
  }

  return (
    <div className="settings-anchor">
      {open ? (
        <form className="panel corner-panel" onSubmit={onSubmit}>
          <h2>Настройки игры</h2>
          <label>
            Название
            {canRename ? (
              <input
                value={name}
                maxLength={maxGameNameLength}
                onChange={(event) => setName(event.target.value)}
              />
            ) : (
              <input value={game.name} readOnly />
            )}
          </label>
          <label>
            ID
            <input value={game.id} readOnly />
          </label>
          <p>Владелец: {game.owner.login}</p>
          {error ? (
            <p className="form-error" role="alert">
              {error}
            </p>
          ) : null}
          <div className="actions">
            <button type="button" className="secondary" onClick={() => void copyID()}>
              {copied ? copyLabel.done : copyLabel.idle}
            </button>
            {canRename ? (
              <button type="submit" disabled={pending}>
                Сохранить
              </button>
            ) : null}
          </div>
        </form>
      ) : null}
      <button type="button" onClick={() => setOpen((value) => !value)}>
        Настройки игры
      </button>
    </div>
  );
}
