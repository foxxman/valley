import { useNavigate } from 'react-router-dom';
import { useSession } from '../auth/session';

export function LobbyPage() {
  const { player, logout, error } = useSession();
  const navigate = useNavigate();

  return (
    <main className="screen">
      <section className="panel">
        <h1>Привет, {player?.login}</h1>
        <p>Карта откроется после входа в игру.</p>
        {error ? (
          <p className="form-error" role="alert">
            {error}
          </p>
        ) : null}
        <div className="actions">
          <button type="button" onClick={() => navigate('/play')}>
            В игру
          </button>
          <button type="button" className="secondary" onClick={() => void logout()}>
            Выйти
          </button>
        </div>
      </section>
    </main>
  );
}
