CREATE TABLE map_cells (
    game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    x INT NOT NULL,
    y INT NOT NULL,
    layer TEXT NOT NULL,
    value TEXT,
    PRIMARY KEY (game_id, x, y, layer),
    CONSTRAINT map_cells_layer CHECK (layer IN ('terrain', 'tree', 'flora', 'feature'))
);

INSERT INTO items (slug, name, icon_key, description, max_stack) VALUES
    ('flower-0', 'Цветок I', 'flower-0', '', 99),
    ('flower-1', 'Цветок II', 'flower-1', '', 99),
    ('flower-2', 'Цветок III', 'flower-2', '', 99),
    ('flower-3', 'Цветок IV', 'flower-3', '', 99)
ON CONFLICT (slug) DO NOTHING;
