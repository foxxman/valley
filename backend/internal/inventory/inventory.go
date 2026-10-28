package inventory

// Capacity is how many cells a bag has.
// The same bound is CHECK (slot_index BETWEEN 0 AND 19) in 004_inventory.sql.
const Capacity = 20

type Item struct {
	ID          string
	Slug        string
	Name        string
	IconKey     string
	Description string
	MaxStack    int
}

type Slot struct {
	Index    int
	Item     Item
	Quantity int
}

type Inventory struct {
	Capacity int
	Slots    []Slot
}
