ALTER TABLE map_cells DROP CONSTRAINT map_cells_layer;

ALTER TABLE map_cells
    ADD CONSTRAINT map_cells_layer
    CHECK (layer IN ('terrain', 'tree', 'flora', 'feature', 'durability'));

INSERT INTO items (slug, name, icon_key, description, max_stack) VALUES
    ('wood', 'Древесина', 'wood', '', 99)
ON CONFLICT (slug) DO NOTHING;
