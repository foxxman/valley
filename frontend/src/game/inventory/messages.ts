import type { InventoryErrorCode } from './types';

export const inventoryErrorMessage: Record<InventoryErrorCode, string> = {
  game_not_found: 'Игра не найдена',
  unauthorized: 'Нужно войти снова',
  unknown: 'Не получилось загрузить инвентарь',
};
