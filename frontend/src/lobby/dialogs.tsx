import { useEffect, useState, type FormEvent, type ReactNode } from 'react';
import { createGame, findGame, listMyGames } from '../games/api';
import type { Game } from '../games/types';
import { maxGameNameLength } from '../games/types';

type Closeable = {
  onClose: () => void;
};

export function CreateGameDialog({ onClose, onCreated }: Closeable & { onCreated: (game: Game) => void }) {
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError('');
    const result = await createGame(name);
    setPending(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    onCreated(result.data);
  }

  return (
    <Modal title="Создать игру" onClose={onClose}>
      <form className="dialog-form" onSubmit={onSubmit}>
        <label>
          Название
          <input
            value={name}
            maxLength={maxGameNameLength}
            onChange={(event) => setName(event.target.value)}
            autoFocus
          />
        </label>
        {error ? <Alert text={error} /> : null}
        <DialogActions pending={pending} submitLabel="Создать" onClose={onClose} />
      </form>
    </Modal>
  );
}

export function FindGameDialog({ onClose }: Closeable) {
  const [id, setId] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [found, setFound] = useState<Game | null>(null);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError('');
    setFound(null);
    const result = await findGame(id.trim());
    setPending(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    setFound(result.data);
  }

  return (
    <Modal title="Найти игру" onClose={onClose}>
      <form className="dialog-form" onSubmit={onSubmit}>
        <label>
          ID
          <input value={id} onChange={(event) => setId(event.target.value)} autoFocus />
        </label>
        {error ? <Alert text={error} /> : null}
        {found ? (
          <div role="status">
            <p>Игра найдена!</p>
            <p>{found.name}</p>
            <p className="game-id">{found.id}</p>
            <button type="button">Присоединиться</button>
          </div>
        ) : null}
        <DialogActions pending={pending} submitLabel="Поиск" onClose={onClose} />
      </form>
    </Modal>
  );
}

export function MyGamesDialog({ onClose, onEnter }: Closeable & { onEnter: (game: Game) => void }) {
  const [games, setGames] = useState<Game[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    listMyGames().then((result) => {
      if (cancelled) {
        return;
      }
      if (!result.ok) {
        setError(result.message);
      } else {
        setGames(result.data);
      }
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Modal title="Мои игры" onClose={onClose}>
      {loading ? <p>Загрузка…</p> : null}
      {error ? <Alert text={error} /> : null}
      {!loading && !error && games.length === 0 ? <p>Вы ещё не создали игр.</p> : null}
      <ul className="game-list">
        {games.map((game) => (
          <li key={game.id}>
            <div>
              <strong>{game.name}</strong>
              <span className="game-id">{game.id}</span>
            </div>
            <button type="button" onClick={() => onEnter(game)}>
              Войти
            </button>
          </li>
        ))}
      </ul>
      <div className="actions">
        <button type="button" className="secondary" onClick={onClose}>
          Закрыть
        </button>
      </div>
    </Modal>
  );
}

function Modal({ title, onClose, children }: Closeable & { title: string; children: ReactNode }) {
  return (
    <div className="modal-backdrop" role="presentation" onClick={onClose}>
      <section
        className="panel"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(event) => event.stopPropagation()}
      >
        <h2>{title}</h2>
        {children}
      </section>
    </div>
  );
}

function DialogActions({
  pending,
  submitLabel,
  onClose,
}: {
  pending: boolean;
  submitLabel: string;
  onClose: () => void;
}) {
  return (
    <div className="actions">
      <button type="submit" disabled={pending}>
        {submitLabel}
      </button>
      <button type="button" className="secondary" onClick={onClose}>
        Закрыть
      </button>
    </div>
  );
}

function Alert({ text }: { text: string }) {
  return (
    <p className="form-error" role="alert">
      {text}
    </p>
  );
}
