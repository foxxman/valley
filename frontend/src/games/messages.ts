import type { GameErrorCode } from './types';

export const gameErrorMessage: Record<GameErrorCode, string> = {
  invalid_body: 'Проверьте поля формы',
  invalid_name: 'Название: от 1 до 40 символов',
  invalid_seed: 'Сид: целое число от 0 до 4 294 967 295',
  game_not_found: 'Игра не найдена',
  forbidden: 'Менять название может только владелец',
  unauthorized: 'Нужно войти снова',
  unknown: 'Не получилось выполнить запрос',
};
