import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import type { Game } from './types';

type SelectionValue = {
  game: Game | null;
  select: (game: Game) => void;
};

const SelectionContext = createContext<SelectionValue | null>(null);

export function GameSelectionProvider({
  playerId,
  children,
}: {
  playerId: string | null;
  children: ReactNode;
}) {
  const [game, setGame] = useState<Game | null>(null);

  useEffect(() => {
    setGame(null);
  }, [playerId]);

  const select = useCallback((next: Game) => {
    setGame(next);
  }, []);

  const value = useMemo(() => ({ game, select }), [game, select]);
  return <SelectionContext.Provider value={value}>{children}</SelectionContext.Provider>;
}

export function useGameSelection(): SelectionValue {
  const value = useContext(SelectionContext);
  if (!value) {
    throw new Error('useGameSelection must be used inside GameSelectionProvider');
  }
  return value;
}
