import type { MapErrorCode } from './types';

export const mapErrorMessage: Record<MapErrorCode, string> = {
  game_not_found: 'Игра не найдена',
  unauthorized: 'Нужно войти снова',
  already_taken: 'Этот цветок уже сорван',
  bag_full: 'Рюкзак полон',
  not_in_reach: 'Слишком далеко',
  invalid_kind: 'Неизвестный цветок',
  unknown: 'Не получилось загрузить карту',
};
