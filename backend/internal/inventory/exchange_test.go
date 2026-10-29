package inventory

import "testing"

func TestPlanExchangePutsChestInSlotFreedByExactWood(t *testing.T) {
	slots := fullBag(map[int]Slot{
		4: {Index: 4, Quantity: 4, Item: woodItem},
	})
	before := slots[4].Quantity
	next, err := planExchange(slots, []Cost{{Slug: "wood", Quantity: 4}}, chestItem, 1)
	if err != nil {
		t.Fatal(err)
	}
	if slots[4].Quantity != before {
		t.Fatal("input was modified")
	}
	if len(next) != Capacity || next[4].Item.ID != "chest" || next[4].Quantity != 1 {
		t.Fatalf("slot 4 %+v len %d", next[4], len(next))
	}
}

func TestPlanExchangeRejectsPartialWoodWhenBagStaysFull(t *testing.T) {
	slots := fullBag(map[int]Slot{
		4: {Index: 4, Quantity: 5, Item: woodItem},
	})
	if _, err := planExchange(slots, []Cost{{Slug: "wood", Quantity: 4}}, chestItem, 1); err != ErrBagFull {
		t.Fatalf("err %v", err)
	}
}

func TestPlanExchangeStacksResultBeforeTakingAnEmptySlot(t *testing.T) {
	slots := fullBag(map[int]Slot{
		0: {Index: 0, Quantity: 98, Item: Item{ID: "stone", Slug: "stone", MaxStack: 99}},
		1: {Index: 1, Quantity: 5, Item: woodItem},
	})
	next, err := planExchange(slots, []Cost{{Slug: "wood", Quantity: 1}}, Item{ID: "stone", Slug: "stone", MaxStack: 99}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if next[0].Quantity != 99 || next[1].Quantity != 4 || next[1].Item.ID != "wood" {
		t.Fatalf("stacks %+v %+v", next[0], next[1])
	}
}

func TestPlanExchangeTakesWoodFromTheLowestIndex(t *testing.T) {
	slots := []Slot{
		{Index: 2, Quantity: 10, Item: woodItem},
		{Index: 5, Quantity: 10, Item: woodItem},
	}
	next, err := planExchange(slots, []Cost{{Slug: "wood", Quantity: 4}}, chestItem, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 3 || next[0].Index != 0 || next[0].Item.ID != "chest" || next[0].Quantity != 1 {
		t.Fatalf("chest %+v", next)
	}
	if next[1].Index != 2 || next[1].Quantity != 6 || next[2].Index != 5 || next[2].Quantity != 10 {
		t.Fatalf("wood %+v", next)
	}
}

func TestPlanExchangeRejectsWhenWoodIsShort(t *testing.T) {
	slots := []Slot{{Index: 0, Quantity: 3, Item: woodItem}}
	if _, err := planExchange(slots, []Cost{{Slug: "wood", Quantity: 4}}, chestItem, 1); err != ErrShort {
		t.Fatalf("err %v", err)
	}
}

var (
	woodItem  = Item{ID: "wood", Slug: "wood", MaxStack: 99}
	chestItem = Item{ID: "chest", Slug: "chest", MaxStack: 1}
)

func fullBag(replace map[int]Slot) []Slot {
	slots := make([]Slot, 0, Capacity)
	for index := 0; index < Capacity; index++ {
		if slot, ok := replace[index]; ok {
			slots = append(slots, slot)
			continue
		}
		slots = append(slots, Slot{
			Index:    index,
			Quantity: 1,
			Item:     Item{ID: "flower", Slug: "flower-0", MaxStack: 99},
		})
	}
	return slots
}
