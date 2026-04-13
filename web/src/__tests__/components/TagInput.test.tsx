import { render, screen, fireEvent } from '@testing-library/react';
import { TagInput } from '../../components/TagInput';

describe('TagInput', () => {
  it('renders existing tags', () => {
    render(<TagInput value={['CVE-2024-001', 'CVE-2024-002']} onChange={() => {}} />);
    expect(screen.getByText('CVE-2024-001')).toBeInTheDocument();
    expect(screen.getByText('CVE-2024-002')).toBeInTheDocument();
  });

  it('adds tag on Enter', () => {
    const onChange = vi.fn();
    render(<TagInput value={[]} onChange={onChange} />);
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'CVE-2024-003' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledWith(['CVE-2024-003']);
  });

  it('removes tag on X click', () => {
    const onChange = vi.fn();
    render(<TagInput value={['tag1', 'tag2']} onChange={onChange} />);
    const removeButtons = screen.getAllByRole('button');
    fireEvent.click(removeButtons[0]);
    expect(onChange).toHaveBeenCalledWith(['tag2']);
  });

  it('does not add duplicate tags', () => {
    const onChange = vi.fn();
    render(<TagInput value={['existing']} onChange={onChange} />);
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'existing' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).not.toHaveBeenCalled();
  });
});
