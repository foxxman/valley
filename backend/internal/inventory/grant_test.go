package inventory

import "testing"

func TestPlanGrantFillsStackThenEmptySlot(t *testing.T) {
	slots := []Slot{{
		Index:    0,
		Quantity: 97,
		Item:     Item{ID: "wood", MaxStack: 99},
	}}
	for index := 1; index <= 18; index++ {
		slots = append(slots, Slot{
			Index:    index,
			Quantity: 99,
			Item:     Item{ID: "flower", MaxStack: 99},
		})
	}
	places, err := planGrant(slots, "wood", 99, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 2 || places[0].index != 0 || places[0].add != 2 || places[0].insert {
		t.Fatalf("first %+v", places)
	}
	if places[1].index != 19 || places[1].add != 2 || !places[1].insert {
		t.Fatalf("second %+v", places)
	}
}

func TestPlanGrantRejectsWhenRemainderHasNoSlot(t *testing.T) {
	slots := make([]Slot, 0, Capacity)
	slots = append(slots, Slot{Index: 0, Quantity: 97, Item: Item{ID: "wood", MaxStack: 99}})
	for index := 1; index < Capacity; index++ {
		slots = append(slots, Slot{Index: index, Quantity: 99, Item: Item{ID: "flower", MaxStack: 99}})
	}
	if _, err := planGrant(slots, "wood", 99, 4); err != ErrBagFull {
		t.Fatalf("err %v", err)
	}
}

func TestPlanGrantUsesOneEmptySlotForASingleItem(t *testing.T) {
	places, err := planGrant(nil, "flower", 99, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 1 || places[0].index != 0 || places[0].add != 1 || !places[0].insert {
		t.Fatalf("places %+v", places)
	}
}
