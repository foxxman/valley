import type Phaser from 'phaser';
import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useSession } from '../auth/session';
import { findGame } from '../games/api';
import { GameSettings } from '../games/GameSettings';
import { useGameSelection } from '../games/selection';
import type { Game } from '../games/types';
import { loadInventory } from './inventory/api';
import { InventoryModal, type InventoryView } from './inventory/InventoryModal';
import { inventoryOpenKey } from './inventory/open';
import type { Inventory } from './inventory/types';
import { loadMap } from './map/api';
import type { MapCell } from './map/types';
import { mountGame } from './mountGame';

type JournalState =
  | { gameId: string; status: 'loading' }
  | { gameId: string; status: 'error'; message: string }
  | { gameId: string; status: 'ready'; cells: MapCell[] };

export function GamePage() {
  const { gameId = '' } = useParams();
  const navigate = useNavigate();
  const { player } = useSession();
  const { select } = useGameSelection();
  const hostRef = useRef<HTMLDivElement>(null);
  const gameRef = useRef<Phaser.Game | null>(null);
  const inventoryOpenRef = useRef(false);
  const toggleInventoryRef = useRef<() => void>(() => {});
  const closeInventoryRef = useRef<() => void>(() => {});
  const replaceInventoryRef = useRef<(inventory: Inventory) => void>(() => {});
  const [game, setGame] = useState<Game | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [inventoryOpen, setInventoryOpen] = useState(false);
  const [inventory, setInventory] = useState<InventoryView>({ status: 'loading' });
  const [journal, setJournal] = useState<JournalState>({ gameId: '', status: 'loading' });

  toggleInventoryRef.current = () => {
    const next = !inventoryOpenRef.current;
    inventoryOpenRef.current = next;
    gameRef.current?.registry.set(inventoryOpenKey, next);
    setInventoryOpen(next);
  };
  closeInventoryRef.current = () => {
    if (!inventoryOpenRef.current) {
      return;
    }
    inventoryOpenRef.current = false;
    gameRef.current?.registry.set(inventoryOpenKey, false);
    setInventoryOpen(false);
  };
  replaceInventoryRef.current = (next) => {
    setInventory({
      status: 'ready',
      inventory: { capacity: next.capacity, slots: next.slots ?? [] },
    });
  };

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

  useEffect(() => {
    if (!game || !player) {
      return;
    }
    let cancelled = false;
    setInventory({ status: 'loading' });
    loadInventory(game.id).then((result) => {
      if (cancelled) {
        return;
      }
      if (!result.ok) {
        setInventory({ status: 'error', message: result.message });
        return;
      }
      setInventory({
        status: 'ready',
        inventory: { ...result.data, slots: result.data.slots ?? [] },
      });
    });
    return () => {
      cancelled = true;
    };
  }, [game?.id, player?.id]);

  useEffect(() => {
    if (!game) {
      return;
    }
    let cancelled = false;
    const gameId = game.id;
    loadMap(gameId).then((result) => {
      if (cancelled) {
        return;
      }
      if (!result.ok) {
        setJournal({ gameId, status: 'error', message: result.message });
        return;
      }
      setJournal({ gameId, status: 'ready', cells: result.data.cells ?? [] });
    });
    return () => {
      cancelled = true;
    };
  }, [game?.id]);

  const journalForGame: JournalState =
    journal.gameId === game?.id ? journal : { gameId: game?.id ?? '', status: 'loading' };
  const mapCells = journalForGame.status === 'ready' ? journalForGame.cells : null;
  const mapError = journalForGame.status === 'error' ? journalForGame.message : '';

  useEffect(() => {
    const parent = hostRef.current;
    if (!parent || !game || !player || !mapCells) {
      return;
    }
    inventoryOpenRef.current = false;
    setInventoryOpen(false);
    const world = mountGame(parent, {
      gameId: game.id,
      worldSeed: game.worldSeed,
      playerId: player.id,
      login: player.login,
      cells: mapCells,
      onToggleInventory: () => toggleInventoryRef.current(),
      onCloseInventory: () => closeInventoryRef.current(),
      onInventory: (next) => replaceInventoryRef.current(next),
    });
    gameRef.current = world;
    return () => {
      inventoryOpenRef.current = false;
      gameRef.current = null;
      world.destroy(true);
    };
  }, [game?.id, game?.worldSeed, player?.id, player?.login, mapCells]);

  if (loading || (game && !mapCells && !mapError)) {
    return (
      <main className="screen">
        <p>Загрузка игры…</p>
      </main>
    );
  }

  if (error || mapError || !game || !player) {
    return (
      <main className="screen">
        <section className="panel">
          <p className="form-error" role="alert">
            {error || mapError || 'Игра не найдена'}
          </p>
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
      {inventoryOpen ? <InventoryModal view={inventory} onClose={() => closeInventoryRef.current()} /> : null}
    </div>
  );
}
