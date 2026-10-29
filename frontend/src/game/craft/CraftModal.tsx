import { useRef, useState } from 'react';
import { itemIconUrl } from '../inventory/icons';
import type { InventoryView } from '../inventory/InventoryModal';
import { loadInventory } from '../inventory/api';
import type { Inventory } from '../inventory/types';
import { Tooltip } from '../ui/Tooltip';
import { craftRecipe } from './api';
import { craftBlock } from './fit';
import { craftBlockMessage } from './messages';
import type { Recipe } from './types';

export type RecipeView =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; recipes: Recipe[] };

type CraftModalProps = {
  gameId: string;
  recipes: RecipeView;
  inventory: InventoryView;
  onClose: () => void;
  onInventory: (inventory: Inventory) => void;
};

export function CraftModal({ gameId, recipes, inventory, onClose, onInventory }: CraftModalProps) {
  const [selectedSlug, setSelectedSlug] = useState<string | null>(null);
  const [serverError, setServerError] = useState('');
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);

  const readyRecipes = recipes.status === 'ready' ? recipes.recipes : [];
  const selected = readyRecipes.find((recipe) => recipe.slug === selectedSlug) ?? readyRecipes[0] ?? null;
  const bag = inventory.status === 'ready' ? inventory.inventory : null;
  const block = bag && selected ? craftBlock(bag, selected) : null;
  const disabled = busy || !selected || !bag || block !== null;
  const error =
    serverError ||
    (block ? craftBlockMessage[block] : '') ||
    (inventory.status === 'error' ? inventory.message : '');

  async function createSelected() {
    if (!selected || disabled || busyRef.current) {
      return;
    }
    busyRef.current = true;
    setServerError('');
    setBusy(true);
    const result = await craftRecipe(gameId, selected.slug);
    busyRef.current = false;
    setBusy(false);
    if (!result.ok) {
      setServerError(result.message);
      if (result.code === 'bag_full' || result.code === 'missing_ingredients') {
        const fresh = await loadInventory(gameId);
        if (fresh.ok) {
          onInventory(fresh.data);
        }
      }
      return;
    }
    onInventory({ capacity: result.data.capacity, slots: result.data.slots ?? [] });
  }

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <section
        className="panel craft-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="craft-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <h2 id="craft-title">Крафт</h2>
        {recipes.status === 'loading' ? <p>Загрузка рецептов…</p> : null}
        {recipes.status === 'error' ? (
          <p className="form-error" role="alert">
            {recipes.message}
          </p>
        ) : null}
        {recipes.status === 'ready' ? (
          <div className="craft-list" role="listbox" aria-label="Рецепты">
            {readyRecipes.map((recipe) => (
              <button
                key={recipe.slug}
                type="button"
                className={recipe.slug === selected?.slug ? 'craft-row is-selected' : 'craft-row'}
                role="option"
                aria-selected={recipe.slug === selected?.slug}
                onClick={() => {
                  setSelectedSlug(recipe.slug);
                  setServerError('');
                }}
              >
                <span className="craft-ingredients">
                  {recipe.ingredients.map((ingredient) => {
                    const icon = itemIconUrl(ingredient.iconKey);
                    return (
                      <span key={ingredient.slug} className="craft-ingredient">
                        {icon ? <img src={icon} alt="" draggable={false} /> : null}
                        <span>{ingredient.name}</span>
                        <span className="craft-qty">× {ingredient.quantity}</span>
                      </span>
                    );
                  })}
                </span>
                <Tooltip label={recipe.result.name}>
                  <RecipeResult recipe={recipe} />
                </Tooltip>
              </button>
            ))}
          </div>
        ) : null}
        <div className="craft-footer">
          <button type="button" className="craft-create" disabled={disabled} onClick={() => void createSelected()}>
            Создать
          </button>
          {error ? (
            <p className="form-error" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </section>
    </div>
  );
}

function RecipeResult({ recipe }: { recipe: Recipe }) {
  const icon = itemIconUrl(recipe.result.iconKey);
  if (!icon) {
    return <span className="craft-result">{recipe.result.name}</span>;
  }
  return <img className="craft-result" src={icon} alt={recipe.result.name} draggable={false} />;
}
