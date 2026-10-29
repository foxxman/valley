import type { CraftErrorCode } from './types';

export const craftErrorMessage: Record<CraftErrorCode, string> = {
  game_not_found: 'Игра не найдена',
  unauthorized: 'Нужно войти снова',
  missing_ingredients: 'Не хватает ингредиентов',
  bag_full: 'Нет места в рюкзаке',
  unknown_recipe: 'Неизвестный рецепт',
  unknown: 'Не получилось создать предмет',
};

export const craftBlockMessage = {
  missing: 'Не хватает ингредиентов',
  full: 'Нет места в рюкзаке',
} as const;
