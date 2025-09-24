import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './app.jsx';

console.log('Renderer starting...');

function initializeApp() {
    const container = document.getElementById('root');
    if (!container) {
        console.error('No root element found');
        return;
    }

    const root = createRoot(container);
    root.render(React.createElement(App));
    console.log('React app rendered successfully');
}

// Ensure DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeApp);
} else {
    initializeApp();
}
