INSERT INTO items (slug, name, icon_key, description, max_stack) VALUES
    ('stone', 'Камень', 'stone', '', 99)
ON CONFLICT (slug) DO NOTHING;
