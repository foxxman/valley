import { itemIconUrl } from './icons';
import type { Inventory, InventorySlot } from './types';

export type InventoryView =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'ready'; inventory: Inventory };

type InventoryModalProps = {
  view: InventoryView;
  onClose: () => void;
};

export function InventoryModal({ view, onClose }: InventoryModalProps) {
  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <section
        className="panel inventory-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="inventory-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <h2 id="inventory-title">Инвентарь</h2>
        {view.status === 'loading' ? <p>Загрузка инвентаря…</p> : null}
        {view.status === 'error' ? (
          <p className="form-error" role="alert">
            {view.message}
          </p>
        ) : null}
        {view.status === 'ready' ? <InventoryGrid inventory={view.inventory} /> : null}
        <button type="button" onClick={onClose}>
          Закрыть
        </button>
      </section>
    </div>
  );
}

function InventoryGrid({ inventory }: { inventory: Inventory }) {
  const occupied = new Map(inventory.slots.map((slot) => [slot.index, slot]));
  const cells = Array.from({ length: inventory.capacity }, (_, index) => occupied.get(index) ?? null);

  return (
    <div className="inventory-grid" role="list">
      {cells.map((slot, index) => (
        <InventoryCell key={index} slot={slot} />
      ))}
    </div>
  );
}

function InventoryCell({ slot }: { slot: InventorySlot | null }) {
  if (!slot) {
    return <div className="inventory-cell" role="listitem" />;
  }

  const icon = itemIconUrl(slot.item.iconKey);
  const title = slot.item.description ? `${slot.item.name}\n${slot.item.description}` : slot.item.name;

  return (
    <div className="inventory-cell" role="listitem" title={title}>
      {icon ? <img src={icon} alt={slot.item.name} /> : <span>{slot.item.name}</span>}
      {slot.quantity > 1 ? <span className="inventory-count">{slot.quantity}</span> : null}
    </div>
  );
}
