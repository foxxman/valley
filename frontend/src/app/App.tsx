import { Navigate, Route, Routes } from 'react-router-dom';
import { AuthPage } from '../auth/AuthPage';
import { useSession } from '../auth/session';
import { GamePage } from '../game/GamePage';
import { LobbyPage } from '../lobby/LobbyPage';

export function App() {
  const { status, player, error, retry } = useSession();

  if (status === 'loading') {
    return (
      <main className="screen">
        <p>Проверяем сессию…</p>
      </main>
    );
  }

  if (status === 'error') {
    return (
      <main className="screen">
        <section className="panel">
          <p className="form-error" role="alert">
            {error || 'Нет связи с сервером'}
          </p>
          <button type="button" onClick={retry}>
            Повторить
          </button>
        </section>
      </main>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={player ? <Navigate to="/lobby" replace /> : <AuthPage />} />
      <Route path="/lobby" element={player ? <LobbyPage /> : <Navigate to="/login" replace />} />
      <Route path="/play" element={player ? <GamePage /> : <Navigate to="/login" replace />} />
      <Route path="*" element={<Navigate to={player ? '/lobby' : '/login'} replace />} />
    </Routes>
  );
}
