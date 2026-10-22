import type { GameErrorCode } from './types';

export const gameErrorMessage: Record<GameErrorCode, string> = {
  invalid_body: 'Проверьте поля формы',
  invalid_name: 'Название: от 1 до 40 символов',
  game_not_found: 'Игра не найдена',
  forbidden: 'Менять название может только владелец',
  unauthorized: 'Нужно войти снова',
  unknown: 'Не получилось выполнить запрос',
};
