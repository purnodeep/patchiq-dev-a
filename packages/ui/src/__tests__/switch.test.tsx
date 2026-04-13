import { render } from '@testing-library/react';
import { Switch } from '../components/ui/switch';

test('Switch applies red track class when danger=true and checked', () => {
  const { container } = render(<Switch checked danger onCheckedChange={() => {}} />);
  const root = container.querySelector('[data-state="checked"]');
  expect(root?.className).toContain('data-[state=checked]:bg-red-500');
});

test('Switch uses primary color when danger=false', () => {
  const { container } = render(<Switch checked onCheckedChange={() => {}} />);
  const root = container.querySelector('[data-state="checked"]');
  expect(root?.className).toContain('data-[state=checked]:bg-primary');
});
