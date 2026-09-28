import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import App from './App';

describe('App Component', () => {
  it('renders the AI Filmmaker title and foundation message', () => {
    render(<App />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('AI Filmmaker');
    expect(screen.getByText('Foundation ready.')).toBeInTheDocument();
  });
});
