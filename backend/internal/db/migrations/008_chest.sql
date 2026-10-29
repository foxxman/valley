INSERT INTO items (slug, name, icon_key, description, max_stack) VALUES
    ('chest', 'Сундук', 'chest', '', 1)
ON CONFLICT (slug) DO NOTHING;
