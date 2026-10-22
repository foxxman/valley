import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { logout as logoutRequest, restoreSession } from './api';
import type { Player } from './types';

type SessionStatus = 'loading' | 'ready' | 'error';

type SessionValue = {
  status: SessionStatus;
  player: Player | null;
  error: string;
  setPlayer: (player: Player) => void;
  logout: () => Promise<void>;
  retry: () => void;
};

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>('loading');
  const [player, setPlayerState] = useState<Player | null>(null);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setStatus('loading');
    restoreSession()
      .then((restored) => {
        if (cancelled) {
          return;
        }
        setPlayerState(restored);
        setError('');
        setStatus('ready');
      })
      .catch((reason: unknown) => {
        if (cancelled) {
          return;
        }
        setPlayerState(null);
        setError(reason instanceof Error ? reason.message : 'Нет связи с сервером');
        setStatus('error');
      });
    return () => {
      cancelled = true;
    };
  }, [attempt]);

  const value = useMemo<SessionValue>(
    () => ({
      status,
      player,
      error,
      setPlayer: (next) => {
        setPlayerState(next);
        setStatus('ready');
        setError('');
      },
      logout: async () => {
        const result = await logoutRequest();
        if (!result.ok) {
          setError(result.message);
          return;
        }
        setPlayerState(null);
        setStatus('ready');
      },
      retry: () => setAttempt((current) => current + 1),
    }),
    [status, player, error],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionValue {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error('useSession must be used inside SessionProvider');
  }
  return value;
}
