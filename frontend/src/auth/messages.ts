import type { AuthErrorCode } from './types';

export const authErrorMessage: Record<AuthErrorCode, string> = {
  invalid_body: 'Проверьте поля формы',
  invalid_login: 'Логин: 3–24 символа, латиница, цифры и _',
  invalid_password: 'Пароль должен быть не короче 8 символов',
  login_taken: 'Этот логин уже занят',
  invalid_credentials: 'Неверный логин или пароль',
  unauthorized: 'Нужно войти снова',
  unknown: 'Не получилось выполнить запрос',
};
