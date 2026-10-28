package game

import (
	"context"
	"testing"
	"time"
)

func TestCreateTrimsName(t *testing.T) {
	store := &fakeStore{}
	created, err := NewService(store).Create(context.Background(), ownerID, "  Долина  ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Долина" || created.OwnerID != ownerID {
		t.Fatalf("created %+v", created)
	}
}

func TestCreateKeepsSeed(t *testing.T) {
	store := &fakeStore{}
	seed := int64(42)
	created, err := NewService(store).Create(context.Background(), ownerID, "Долина", &seed)
	if err != nil || created.WorldSeed != 42 || store.gotSeed != 42 {
		t.Fatalf("created %+v seed %d err %v", created, store.gotSeed, err)
	}
}

func TestCreateGeneratesSeed(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	svc.newSeed = func() (int64, error) { return 7, nil }
	created, err := svc.Create(context.Background(), ownerID, "Долина", nil)
	if err != nil || created.WorldSeed != 7 || store.gotSeed != 7 {
		t.Fatalf("created %+v seed %d err %v", created, store.gotSeed, err)
	}
}

func TestCreateRejectsSeed(t *testing.T) {
	store := &fakeStore{}
	tooBig := MaxWorldSeed + 1
	negative := int64(-1)
	for _, seed := range []int64{tooBig, negative} {
		_, err := NewService(store).Create(context.Background(), ownerID, "Долина", &seed)
		if err != ErrInvalidSeed {
			t.Fatalf("seed %d err %v", seed, err)
		}
	}
	if store.creates != 0 {
		t.Fatalf("store calls %d", store.creates)
	}
}

func TestCreateRejectsBlankName(t *testing.T) {
	_, err := NewService(&fakeStore{}).Create(context.Background(), ownerID, "  ", nil)
	if err != ErrInvalidName {
		t.Fatalf("err %v", err)
	}
}

func TestRenameOwnerAndStranger(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)
	created, err := svc.Create(context.Background(), ownerID, "Долина", nil)
	if err != nil {
		t.Fatal(err)
	}

	renamed, err := svc.Rename(context.Background(), ownerID, created.ID, "Новая")
	if err != nil || renamed.Name != "Новая" {
		t.Fatalf("renamed %+v err %v", renamed, err)
	}

	_, err = svc.Rename(context.Background(), "22222222-2222-2222-2222-222222222222", created.ID, "Чужая")
	if err != ErrForbidden {
		t.Fatalf("err %v", err)
	}
	if store.byID[created.ID].Name != "Новая" {
		t.Fatal("stranger rename changed the name")
	}
}

func TestFindRejectsPartialID(t *testing.T) {
	store := &fakeStore{}
	_, err := NewService(store).Find(context.Background(), "11111111")
	if err != ErrNotFound {
		t.Fatalf("err %v", err)
	}
	if store.findCalls != 0 {
		t.Fatalf("store calls %d", store.findCalls)
	}
}

func TestListOwnedReturnsEmptySlice(t *testing.T) {
	found, err := NewService(&fakeStore{}).ListOwned(context.Background(), ownerID)
	if err != nil || found == nil || len(found) != 0 {
		t.Fatalf("found %#v err %v", found, err)
	}
}

const ownerID = "11111111-1111-1111-1111-111111111111"

type fakeStore struct {
	byID      map[string]Game
	findCalls int
	creates   int
	gotSeed   int64
}

func (f *fakeStore) Create(_ context.Context, ownerID, name string, worldSeed int64) (Game, error) {
	f.creates++
	f.gotSeed = worldSeed
	created := Game{
		ID:         "33333333-3333-3333-3333-333333333333",
		Name:       name,
		OwnerID:    ownerID,
		OwnerLogin: "fox",
		WorldSeed:  worldSeed,
		CreatedAt:  time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	}
	if f.byID == nil {
		f.byID = map[string]Game{}
	}
	f.byID[created.ID] = created
	return created, nil
}

func (f *fakeStore) FindByID(_ context.Context, id string) (Game, error) {
	f.findCalls++
	found, ok := f.byID[id]
	if !ok {
		return Game{}, ErrNotFound
	}
	return found, nil
}

func (f *fakeStore) ListByOwner(_ context.Context, ownerID string) ([]Game, error) {
	var owned []Game
	for _, item := range f.byID {
		if item.OwnerID == ownerID {
			owned = append(owned, item)
		}
	}
	return owned, nil
}

func (f *fakeStore) UpdateName(_ context.Context, id, name string) (Game, error) {
	found, ok := f.byID[id]
	if !ok {
		return Game{}, ErrNotFound
	}
	found.Name = name
	f.byID[id] = found
	return found, nil
}
