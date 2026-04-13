interface SectionHeaderProps {
  title: string;
  action?: React.ReactNode;
}

export function SectionHeader({ title, action }: SectionHeaderProps) {
  return (
    <div className="flex items-center justify-between">
      <h3 className="text-[13px] font-bold">{title}</h3>
      {action}
    </div>
  );
}
