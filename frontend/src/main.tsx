import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { App } from './app/App';
import './app/styles.css';
import { SessionProvider } from './auth/session';

const root = document.querySelector<HTMLElement>('#app');

if (!root) {
  throw new Error('Root element #app not found');
}

createRoot(root).render(
  <BrowserRouter>
    <SessionProvider>
      <App />
    </SessionProvider>
  </BrowserRouter>,
);
