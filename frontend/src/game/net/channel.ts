export type RemotePlayer = {
  playerId: string;
  login: string;
  x: number;
  y: number;
  color: number;
};

export type PresenceView = {
  selfColor: number | null;
  players: RemotePlayer[];
};

type Payload = Partial<RemotePlayer> & {
  players?: RemotePlayer[];
  code?: string;
};

type Incoming = {
  type?: string;
  room?: string;
  payload?: Payload;
};

type ChannelOptions = {
  gameId: string;
  selfId: string;
  x: number;
  y: number;
  onPresence: (view: PresenceView) => void;
};

export type GameChannel = {
  move: (x: number, y: number) => void;
  close: () => void;
};

function finiteColor(value: number | undefined): number | undefined {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return undefined;
  }
  return value;
}

/** Room prefix matches the backend presence bus (`game:<id>`). */
function gameRoom(gameId: string): string {
  return `game:${gameId}`;
}

export function openGameChannel({ gameId, selfId, x, y, onPresence }: ChannelOptions): GameChannel {
  const room = gameRoom(gameId);
  const place = { x, y };
  const remotes = new Map<string, RemotePlayer>();
  let selfColor: number | null = null;
  let notify = onPresence;
  let socket: WebSocket | null = null;
  let closed = false;
  let replaced = false;
  let retry = 0;
  let retryTimer = 0;

  const remember = (player: Partial<RemotePlayer>) => {
    if (!player.playerId || player.playerId === selfId) {
      return;
    }
    if (typeof player.x !== 'number' || typeof player.y !== 'number') {
      return;
    }
    if (!Number.isFinite(player.x) || !Number.isFinite(player.y)) {
      return;
    }
    const previous = remotes.get(player.playerId);
    const color = finiteColor(player.color) ?? previous?.color;
    if (color === undefined) {
      return;
    }
    remotes.set(player.playerId, {
      playerId: player.playerId,
      login: player.login || previous?.login || '',
      x: player.x,
      y: player.y,
      color,
    });
  };

  const emit = () => {
    notify({ selfColor, players: [...remotes.values()] });
  };

  const handlers: Record<string, (payload: Payload) => void> = {
    snapshot(payload) {
      const color = finiteColor(payload.color);
      if (color !== undefined) {
        selfColor = color;
      }
      remotes.clear();
      for (const player of payload.players ?? []) {
        remember(player);
      }
      emit();
    },
    joined(payload) {
      remember(payload);
      emit();
    },
    moved(payload) {
      remember(payload);
      emit();
    },
    left(payload) {
      if (!payload.playerId || payload.playerId === selfId) {
        return;
      }
      remotes.delete(payload.playerId);
      emit();
    },
    error(payload) {
      if (payload.code !== 'replaced') {
        return;
      }
      replaced = true;
      closed = true;
      socket?.close();
    },
  };

  const connect = () => {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const next = new WebSocket(`${protocol}//${location.host}/api/ws`);
    socket = next;
    next.onopen = () => {
      if (socket !== next) {
        return;
      }
      retry = 0;
      next.send(
        JSON.stringify({
          type: 'subscribe',
          room,
          payload: { x: place.x, y: place.y },
        }),
      );
    };
    next.onmessage = (event) => {
      if (socket !== next || typeof event.data !== 'string') {
        return;
      }
      let message: Incoming;
      try {
        message = JSON.parse(event.data) as Incoming;
      } catch {
        return;
      }
      if (message.room && message.room !== room) {
        return;
      }
      handlers[message.type ?? '']?.(message.payload ?? {});
    };
    next.onclose = () => {
      if (socket !== next) {
        return;
      }
      remotes.clear();
      selfColor = null;
      emit();
      if (closed || replaced) {
        return;
      }
      retry += 1;
      retryTimer = window.setTimeout(connect, Math.min(5000, 500 * retry));
    };
  };

  connect();

  return {
    move(nextX, nextY) {
      place.x = nextX;
      place.y = nextY;
      if (socket?.readyState !== WebSocket.OPEN) {
        return;
      }
      socket.send(JSON.stringify({ type: 'position', room, payload: { x: nextX, y: nextY } }));
    },
    close() {
      closed = true;
      notify = () => undefined;
      window.clearTimeout(retryTimer);
      socket?.close();
      socket = null;
    },
  };
}
