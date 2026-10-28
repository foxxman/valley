package gather

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"game/backend/internal/game"
	"game/backend/internal/inventory"
	"game/backend/internal/mapstate"
)

const (
	gameID   = "33333333-3333-3333-3333-333333333333"
	playerID = "11111111-1111-1111-1111-111111111111"
)

func TestGatherRejectsKindBeforeIO(t *testing.T) {
	games := &fakeGames{}
	begin := &failBegin{t: t}
	_, err := NewService(begin, idleCells{t: t}, idleBags{t}, games, fakePresence{}).Gather(
		context.Background(), gameID, playerID, 1, 1, "lily",
	)
	if !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("err %v", err)
	}
	if games.calls != 0 || begin.calls != 0 {
		t.Fatalf("games %d begins %d", games.calls, begin.calls)
	}
}

func TestGatherMissingGame(t *testing.T) {
	begin := &failBegin{t: t}
	_, err := NewService(begin, idleCells{t: t}, idleBags{t}, &fakeGames{err: game.ErrNotFound}, fakePresence{ok: true}).Gather(
		context.Background(), gameID, playerID, 1, 1, "flower-0",
	)
	if !errors.Is(err, game.ErrNotFound) {
		t.Fatalf("err %v", err)
	}
	if begin.calls != 0 {
		t.Fatalf("begins %d", begin.calls)
	}
}

func TestGatherNotInReach(t *testing.T) {
	begin := &failBegin{t: t}
	cells := idleCells{t: t}
	svc := NewService(begin, cells, idleBags{t}, &fakeGames{}, fakePresence{})

	_, err := svc.Gather(context.Background(), gameID, playerID, 0, 0, "flower-0")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("missing presence %v", err)
	}

	svc.presence = fakePresence{x: 10, y: 10, ok: true}
	_, err = svc.Gather(context.Background(), gameID, playerID, 1, 1, "flower-2")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("diagonal %v", err)
	}

	svc.presence = fakePresence{x: float64(2*tileSize + 1), y: float64(4 * tileSize), ok: true}
	_, err = svc.Gather(context.Background(), gameID, playerID, 4, 4, "flower-1")
	if !errors.Is(err, ErrNotInReach) {
		t.Fatalf("two away %v", err)
	}
	if begin.calls != 0 || cells.claims != 0 {
		t.Fatalf("begins %d claims %d", begin.calls, cells.claims)
	}
}

func TestInReach(t *testing.T) {
	pixelX := float64(4*tileSize + tileSize - 1)
	pixelY := float64(4 * tileSize)
	allowed := map[[2]int]bool{
		{4, 4}: true,
		{5, 4}: true,
		{3, 4}: true,
		{4, 5}: true,
		{4, 3}: true,
	}
	blocked := [][2]int{{5, 5}, {6, 4}, {4, 6}, {2, 4}, {4, 2}}
	for tile, want := range allowed {
		if inReach(pixelX, pixelY, tile[0], tile[1]) != want {
			t.Fatalf("tile %v", tile)
		}
	}
	for _, tile := range blocked {
		if inReach(pixelX, pixelY, tile[0], tile[1]) {
			t.Fatalf("blocked %v", tile)
		}
	}
	if tileOf(-1) != -1 {
		t.Fatalf("negative tile %d", tileOf(-1))
	}
	if !inReach(-1, -1, -1, -1) || inReach(-1, -1, -1, 0) == false {
		t.Fatal("negative reach")
	}
	if inReach(-1, -1, -1, 1) {
		t.Fatal("negative diagonal")
	}
}

type fakeGames struct {
	err   error
	calls int
}

func (f *fakeGames) Find(context.Context, string) (game.Game, error) {
	f.calls++
	if f.err != nil {
		return game.Game{}, f.err
	}
	return game.Game{ID: gameID}, nil
}

type fakePresence struct {
	x  float64
	y  float64
	ok bool
}

func (f fakePresence) Position(string, string) (float64, float64, bool) {
	return f.x, f.y, f.ok
}

type failBegin struct {
	t     *testing.T
	calls int
}

func (f *failBegin) Begin(context.Context) (pgx.Tx, error) {
	f.calls++
	f.t.Fatal("transaction started")
	return nil, errors.New("begin")
}

type idleCells struct {
	t      *testing.T
	claims int
}

func (c idleCells) List(context.Context, string) ([]mapstate.Cell, error) {
	return []mapstate.Cell{}, nil
}

func (c idleCells) ClaimFlora(context.Context, pgx.Tx, string, int, int) (mapstate.FloraClaim, error) {
	c.claims++
	c.t.Fatal("claim")
	return mapstate.FloraClaim{}, nil
}

type idleBags struct {
	t *testing.T
}

func (b idleBags) Grant(context.Context, pgx.Tx, string, string, string) error {
	b.t.Fatal("grant")
	return nil
}

func (b idleBags) ListTx(context.Context, pgx.Tx, string, string) ([]inventory.Slot, error) {
	b.t.Fatal("list")
	return nil, nil
}
