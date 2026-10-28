CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    icon_key TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    max_stack INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT items_max_stack_range CHECK (max_stack BETWEEN 1 AND 999)
);

CREATE TABLE inventory_slots (
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    slot_index SMALLINT NOT NULL,
    item_id UUID NOT NULL REFERENCES items (id),
    quantity INT NOT NULL,
    PRIMARY KEY (game_id, player_id, slot_index),
    CONSTRAINT inventory_slots_index_range CHECK (slot_index BETWEEN 0 AND 19),
    CONSTRAINT inventory_slots_quantity_positive CHECK (quantity >= 1)
);
