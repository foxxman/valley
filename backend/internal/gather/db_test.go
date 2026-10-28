package gather

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"game/backend/internal/db"
	"game/backend/internal/game"
	"game/backend/internal/inventory"
	"game/backend/internal/mapstate"
	"game/backend/internal/presence"
	"game/backend/internal/realtime"
)

func TestGatherSameCellGrantsOnce(t *testing.T) {
	world := newWorld(t)
	first := world.player(t)
	second := world.player(t)
	const tileX, tileY = 5, 6
	world.stand(t, first, tileX, tileY)
	world.stand(t, second, tileX, tileY)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ids := []string{first, second}
	outcomes := make([]struct {
		result Result
		err    error
	}, len(ids))
	var ready, done sync.WaitGroup
	ready.Add(len(ids))
	done.Add(len(ids))
	start := make(chan struct{})
	for i := range ids {
		go func(i int) {
			defer done.Done()
			ready.Done()
			<-start
			outcomes[i].result, outcomes[i].err = world.svc.Gather(ctx, world.gameID, ids[i], tileX, tileY, "flower-0")
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()

	won := 0
	taken := 0
	var winner Result
	var winnerID string
	for i, item := range outcomes {
		if item.err == nil {
			won++
			winner = item.result
			winnerID = ids[i]
			continue
		}
		if errors.Is(item.err, ErrAlreadyTaken) {
			taken++
			continue
		}
		t.Fatalf("player %s: %v", ids[i], item.err)
	}
	if won != 1 || taken != 1 {
		t.Fatalf("won %d taken %d", won, taken)
	}
	if len(winner.Inventory.Slots) != 1 || winner.Inventory.Slots[0].Quantity != 1 || winner.Inventory.Slots[0].Item.Slug != "flower-0" {
		t.Fatalf("bag %+v", winner.Inventory)
	}
	if winner.Cell.X != tileX || winner.Cell.Y != tileY || winner.Cell.Layers[mapstate.LayerFlora] != nil {
		t.Fatalf("cell %+v", winner.Cell)
	}
	world.assertFlora(t, tileX, tileY, nil)
	if got := world.slotCount(t, winnerID); got != 1 {
		t.Fatalf("winner slots %d", got)
	}
	loser := first
	if loser == winnerID {
		loser = second
	}
	if got := world.slotCount(t, loser); got != 0 {
		t.Fatalf("loser slots %d", got)
	}
	if got := world.cellCount(t); got != 1 {
		t.Fatalf("cells %d", got)
	}
}

func TestGatherBagFullRollsBackCell(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 2, 2)
	world.fillBag(t, playerID)

	ctx := context.Background()
	_, err := world.svc.Gather(ctx, world.gameID, playerID, 2, 2, "flower-1")
	if !errors.Is(err, ErrBagFull) {
		t.Fatalf("fresh cell %v", err)
	}
	if got := world.cellCount(t); got != 0 {
		t.Fatalf("cells after fresh claim %d", got)
	}

	if _, err := world.pool.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, 3, 2, 'flora', 'flower-3')
	`, world.gameID); err != nil {
		t.Fatal(err)
	}
	_, err = world.svc.Gather(ctx, world.gameID, playerID, 3, 2, "flower-0")
	if !errors.Is(err, ErrBagFull) {
		t.Fatalf("stored flower %v", err)
	}
	world.assertFlora(t, 3, 2, strPtr("flower-3"))
}

func TestGatherNotInReachWritesNothing(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 0, 0)
	ctx := context.Background()

	_, err := world.svc.Gather(ctx, world.gameID, playerID, 2, 0, "flower-0")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("far %v", err)
	}
	_, err = world.svc.Gather(ctx, world.gameID, playerID, 1, 1, "flower-0")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("diagonal %v", err)
	}
	other := world.player(t)
	_, err = world.svc.Gather(ctx, world.gameID, other, 0, 0, "flower-0")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("absent %v", err)
	}
	if got := world.cellCount(t); got != 0 {
		t.Fatalf("cells %d", got)
	}
}

func TestGatherGrantsStoredSlug(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 8, 8)
	ctx := context.Background()
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, 8, 8, 'flora', 'flower-2')
	`, world.gameID); err != nil {
		t.Fatal(err)
	}

	result, err := world.svc.Gather(ctx, world.gameID, playerID, 8, 8, "flower-0")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Inventory.Slots) != 1 || result.Inventory.Slots[0].Item.Slug != "flower-2" || result.Inventory.Slots[0].Quantity != 1 {
		t.Fatalf("bag %+v", result.Inventory)
	}
	world.assertFlora(t, 8, 8, nil)
}

type worldFix struct {
	pool     *pgxpool.Pool
	svc      *Service
	gameID   string
	presence *presence.Service
	hub      *realtime.Hub
	router   *realtime.Router
}

func newWorld(t *testing.T) *worldFix {
	t.Helper()
	pool := openDB(t)
	owner := insertPlayer(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM players WHERE id = $1::uuid`, owner)
	})
	var gameID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO games (name, owner_id, world_seed)
		VALUES ('Gather', $1::uuid, 1)
		RETURNING id::text
	`, owner).Scan(&gameID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM games WHERE id = $1::uuid`, gameID)
	})

	games := game.NewService(game.NewRepository(pool))
	hub := realtime.NewHub()
	router := realtime.NewRouter()
	standing := presence.New(hub, games)
	standing.Register(router)
	svc := NewService(pool, mapstate.NewRepository(pool), inventory.NewRepository(pool), games, standing)
	return &worldFix{pool: pool, svc: svc, gameID: gameID, presence: standing, hub: hub, router: router}
}

func openDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func (w *worldFix) player(t *testing.T) string {
	t.Helper()
	id := insertPlayer(t, w.pool)
	t.Cleanup(func() {
		_, _ = w.pool.Exec(context.Background(), `DELETE FROM players WHERE id = $1::uuid`, id)
	})
	return id
}

func (w *worldFix) stand(t *testing.T, playerID string, tileX, tileY int) {
	t.Helper()
	client := w.hub.Add(playerID, "player")
	env, err := realtime.NewEnvelope("subscribe", presence.Room(w.gameID), map[string]float64{
		"x": float64(tileX*tileSize + tileSize/2),
		"y": float64(tileY*tileSize + tileSize/2),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.router.Dispatch(context.Background(), client, env); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := w.presence.Position(w.gameID, playerID); !ok {
		t.Fatal("player has no position")
	}
}

func (w *worldFix) fillBag(t *testing.T, playerID string) {
	t.Helper()
	tag, err := w.pool.Exec(context.Background(), `
		INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
		SELECT $1::uuid, $2::uuid, slot.index::smallint, items.id, items.max_stack
		FROM generate_series(0, 19) AS slot(index)
		JOIN items ON items.slug = 'flower-0'
	`, w.gameID, playerID)
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 20 {
		t.Fatalf("filled %d", tag.RowsAffected())
	}
}

func (w *worldFix) cellCount(t *testing.T) int {
	t.Helper()
	var count int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM map_cells WHERE game_id = $1::uuid
	`, w.gameID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func (w *worldFix) slotCount(t *testing.T, playerID string) int {
	t.Helper()
	var count int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM inventory_slots WHERE game_id = $1::uuid AND player_id = $2::uuid
	`, w.gameID, playerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func (w *worldFix) assertFlora(t *testing.T, x, y int, want *string) {
	t.Helper()
	var got *string
	err := w.pool.QueryRow(context.Background(), `
		SELECT value FROM map_cells
		WHERE game_id = $1::uuid AND x = $2 AND y = $3 AND layer = 'flora'
	`, w.gameID, x, y).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want == nil {
		if got != nil {
			t.Fatalf("flora %q", *got)
		}
		return
	}
	if got == nil || *got != *want {
		t.Fatalf("flora %v want %s", got, *want)
	}
}

func insertPlayer(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO players (login, password_hash)
		VALUES ($1, 'test')
		RETURNING id::text
	`, testLogin()).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func testLogin() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return "fl" + hex.EncodeToString(buf)
}

func strPtr(value string) *string {
	return &value
}
