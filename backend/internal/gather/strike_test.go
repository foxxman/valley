package gather

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"game/backend/internal/mapstate"
)

func TestStrikeTenHitsGrantWood(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	const tileX, tileY = 3, 4
	world.stand(t, playerID, tileX, tileY)
	ctx := context.Background()

	for hit := 1; hit <= 9; hit++ {
		result, err := world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "tree")
		if err != nil || result.Broken {
			t.Fatalf("hit %d err %v broken %v", hit, err, result.Broken)
		}
		want := strconv.Itoa(10 - hit)
		got := result.Cell.Layers[mapstate.LayerDurability]
		if got == nil || *got != want || len(result.Inventory.Slots) != 0 {
			t.Fatalf("hit %d cell %+v slots %d", hit, result.Cell.Layers, len(result.Inventory.Slots))
		}
	}
	if got := world.woodQuantity(t, playerID); got != 0 {
		t.Fatalf("wood before break %d", got)
	}
	if present, _ := world.layerValue(t, tileX, tileY, mapstate.LayerTree); present {
		t.Fatal("tree row appeared before the break")
	}

	result, err := world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "tree")
	if err != nil || !result.Broken {
		t.Fatalf("break err %v broken %v", err, result.Broken)
	}
	if len(result.Inventory.Slots) != 1 || result.Inventory.Slots[0].Quantity != 4 || result.Inventory.Slots[0].Item.Slug != "wood" {
		t.Fatalf("bag %+v", result.Inventory)
	}
	if result.Cell.Layers[mapstate.LayerTree] != nil || result.Cell.Layers[mapstate.LayerDurability] != nil {
		t.Fatalf("cell %+v", result.Cell.Layers)
	}
	world.assertLayer(t, tileX, tileY, mapstate.LayerTree, nil)
	world.assertLayer(t, tileX, tileY, mapstate.LayerDurability, nil)

	_, err = world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "tree")
	if !errors.Is(err, ErrAlreadyTaken) {
		t.Fatalf("eleventh %v", err)
	}
	if got := world.woodQuantity(t, playerID); got != 4 {
		t.Fatalf("wood %d", got)
	}
}

func TestStrikeStoneBecomesGrassAndGrantsFive(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	const tileX, tileY = 4, 9
	world.stand(t, playerID, tileX, tileY)
	ctx := context.Background()

	result, err := world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "stone")
	if err != nil || result.Broken {
		t.Fatalf("first err %v broken %v", err, result.Broken)
	}
	got := result.Cell.Layers[mapstate.LayerDurability]
	if got == nil || *got != "19" {
		t.Fatalf("durability %+v", result.Cell.Layers)
	}
	if present, _ := world.layerValue(t, tileX, tileY, mapstate.LayerTerrain); present {
		t.Fatal("terrain row appeared before the break")
	}

	if _, err := world.pool.Exec(ctx, `
		UPDATE map_cells SET value = '1'
		WHERE game_id = $1::uuid AND x = $2 AND y = $3 AND layer = 'durability'
	`, world.gameID, tileX, tileY); err != nil {
		t.Fatal(err)
	}
	result, err = world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "stone")
	if err != nil || !result.Broken {
		t.Fatalf("break err %v broken %v", err, result.Broken)
	}
	if len(result.Inventory.Slots) != 1 || result.Inventory.Slots[0].Quantity != 5 || result.Inventory.Slots[0].Item.Slug != "stone" {
		t.Fatalf("bag %+v", result.Inventory)
	}
	terrain := result.Cell.Layers[mapstate.LayerTerrain]
	if terrain == nil || *terrain != "grass" || result.Cell.Layers[mapstate.LayerDurability] != nil {
		t.Fatalf("cell %+v", result.Cell.Layers)
	}
	world.assertLayer(t, tileX, tileY, mapstate.LayerTerrain, strPtr("grass"))
	world.assertLayer(t, tileX, tileY, mapstate.LayerDurability, nil)

	_, err = world.svc.Strike(ctx, world.gameID, playerID, tileX, tileY, "stone")
	if !errors.Is(err, ErrAlreadyTaken) {
		t.Fatalf("again %v", err)
	}
	if got := world.stoneQuantity(t, playerID); got != 5 {
		t.Fatalf("stone %d", got)
	}
}

func TestStrikeFullBagStillDamagesUntilBreak(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 2, 2)
	world.fillBag(t, playerID)
	ctx := context.Background()

	result, err := world.svc.Strike(ctx, world.gameID, playerID, 2, 2, "tree")
	if err != nil || result.Broken {
		t.Fatalf("first err %v broken %v", err, result.Broken)
	}
	world.assertLayer(t, 2, 2, mapstate.LayerDurability, strPtr("9"))
	if got := world.woodQuantity(t, playerID); got != 0 {
		t.Fatalf("wood %d", got)
	}

	if _, err := world.pool.Exec(ctx, `
		UPDATE map_cells SET value = '1'
		WHERE game_id = $1::uuid AND x = 2 AND y = 2 AND layer = 'durability'
	`, world.gameID); err != nil {
		t.Fatal(err)
	}
	_, err = world.svc.Strike(ctx, world.gameID, playerID, 2, 2, "tree")
	if !errors.Is(err, ErrBagFull) {
		t.Fatalf("last %v", err)
	}
	world.assertLayer(t, 2, 2, mapstate.LayerDurability, strPtr("1"))
	if present, _ := world.layerValue(t, 2, 2, mapstate.LayerTree); present {
		t.Fatal("tree was cleared")
	}
	if got := world.woodQuantity(t, playerID); got != 0 {
		t.Fatalf("wood %d", got)
	}
}

func TestStrikeNotInReachWritesNothing(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 0, 0)
	ctx := context.Background()

	_, err := world.svc.Strike(ctx, world.gameID, playerID, 2, 0, "tree")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("far %v", err)
	}
	_, err = world.svc.Strike(ctx, world.gameID, playerID, 1, 1, "tree")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("diagonal %v", err)
	}
	if got := world.cellCount(t); got != 0 {
		t.Fatalf("cells %d", got)
	}
}

func TestStrikeConcurrentHitsStack(t *testing.T) {
	world := newWorld(t)
	first := world.player(t)
	second := world.player(t)
	const tileX, tileY = 6, 1
	world.stand(t, first, tileX, tileY)
	world.stand(t, second, tileX, tileY)
	ctx := context.Background()

	ids := []string{first, second}
	errs := make([]error, len(ids))
	var ready, done sync.WaitGroup
	ready.Add(len(ids))
	done.Add(len(ids))
	start := make(chan struct{})
	for i := range ids {
		go func(i int) {
			defer done.Done()
			ready.Done()
			<-start
			_, errs[i] = world.svc.Strike(ctx, world.gameID, ids[i], tileX, tileY, "tree")
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("player %s: %v", ids[i], err)
		}
	}
	world.assertLayer(t, tileX, tileY, mapstate.LayerDurability, strPtr("8"))
	if got := world.woodQuantity(t, first) + world.woodQuantity(t, second); got != 0 {
		t.Fatalf("wood %d", got)
	}
}

func TestStrikeConcurrentLastHitGrantsOnce(t *testing.T) {
	world := newWorld(t)
	first := world.player(t)
	second := world.player(t)
	const tileX, tileY = 7, 7
	world.stand(t, first, tileX, tileY)
	world.stand(t, second, tileX, tileY)
	ctx := context.Background()
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, $2, $3, 'durability', '1')
	`, world.gameID, tileX, tileY); err != nil {
		t.Fatal(err)
	}

	ids := []string{first, second}
	outcomes := make([]struct {
		result StrikeResult
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
			outcomes[i].result, outcomes[i].err = world.svc.Strike(ctx, world.gameID, ids[i], tileX, tileY, "tree")
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()

	won := 0
	taken := 0
	var winnerID string
	for i, item := range outcomes {
		if item.err == nil && item.result.Broken {
			won++
			winnerID = ids[i]
			if len(item.result.Inventory.Slots) != 1 || item.result.Inventory.Slots[0].Quantity != 4 {
				t.Fatalf("bag %+v", item.result.Inventory)
			}
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
	if got := world.woodQuantity(t, winnerID); got != 4 {
		t.Fatalf("winner wood %d", got)
	}
	loser := first
	if loser == winnerID {
		loser = second
	}
	if got := world.woodQuantity(t, loser); got != 0 {
		t.Fatalf("loser wood %d", got)
	}
	world.assertLayer(t, tileX, tileY, mapstate.LayerTree, nil)
	world.assertLayer(t, tileX, tileY, mapstate.LayerDurability, nil)
}

func TestStrikeSplitsWoodAcrossSlots(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 4, 8)
	ctx := context.Background()
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
		SELECT $1::uuid, $2::uuid, 0, id, 97 FROM items WHERE slug = 'wood'
	`, world.gameID, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
		SELECT $1::uuid, $2::uuid, slot.index::smallint, items.id, items.max_stack
		FROM generate_series(1, 18) AS slot(index)
		JOIN items ON items.slug = 'flower-0'
	`, world.gameID, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, 4, 8, 'durability', '1')
	`, world.gameID); err != nil {
		t.Fatal(err)
	}

	result, err := world.svc.Strike(ctx, world.gameID, playerID, 4, 8, "tree")
	if err != nil || !result.Broken {
		t.Fatalf("err %v broken %v", err, result.Broken)
	}
	if got := world.woodQuantity(t, playerID); got != 101 {
		t.Fatalf("wood %d", got)
	}
}

func TestStrikeRejectsWoodWhenNoSlotCanTakeTheRemainder(t *testing.T) {
	world := newWorld(t)
	playerID := world.player(t)
	world.stand(t, playerID, 9, 1)
	ctx := context.Background()
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
		SELECT $1::uuid, $2::uuid, 0, id, 97 FROM items WHERE slug = 'wood'
	`, world.gameID, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
		SELECT $1::uuid, $2::uuid, slot.index::smallint, items.id, items.max_stack
		FROM generate_series(1, 19) AS slot(index)
		JOIN items ON items.slug = 'flower-0'
	`, world.gameID, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := world.pool.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, 9, 1, 'durability', '1')
	`, world.gameID); err != nil {
		t.Fatal(err)
	}

	_, err := world.svc.Strike(ctx, world.gameID, playerID, 9, 1, "tree")
	if !errors.Is(err, ErrBagFull) {
		t.Fatalf("err %v", err)
	}
	world.assertLayer(t, 9, 1, mapstate.LayerDurability, strPtr("1"))
	if got := world.woodQuantity(t, playerID); got != 97 {
		t.Fatalf("wood %d", got)
	}
}

func (w *worldFix) woodQuantity(t *testing.T, playerID string) int {
	t.Helper()
	return w.itemQuantity(t, playerID, "wood")
}

func (w *worldFix) stoneQuantity(t *testing.T, playerID string) int {
	t.Helper()
	return w.itemQuantity(t, playerID, "stone")
}

func (w *worldFix) itemQuantity(t *testing.T, playerID, slug string) int {
	t.Helper()
	var quantity int
	if err := w.pool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(s.quantity), 0)
		FROM inventory_slots s
		JOIN items i ON i.id = s.item_id
		WHERE s.game_id = $1::uuid AND s.player_id = $2::uuid AND i.slug = $3
	`, w.gameID, playerID, slug).Scan(&quantity); err != nil {
		t.Fatal(err)
	}
	return quantity
}

func (w *worldFix) layerValue(t *testing.T, x, y int, layer string) (bool, *string) {
	t.Helper()
	var value *string
	err := w.pool.QueryRow(context.Background(), `
		SELECT value FROM map_cells
		WHERE game_id = $1::uuid AND x = $2 AND y = $3 AND layer = $4
	`, w.gameID, x, y, layer).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return true, value
}

func (w *worldFix) assertLayer(t *testing.T, x, y int, layer string, want *string) {
	t.Helper()
	present, got := w.layerValue(t, x, y, layer)
	if !present {
		t.Fatalf("missing %s at %d,%d", layer, x, y)
	}
	if want == nil {
		if got != nil {
			t.Fatalf("%s %q", layer, *got)
		}
		return
	}
	if got == nil || *got != *want {
		t.Fatalf("%s %v want %s", layer, got, *want)
	}
}
