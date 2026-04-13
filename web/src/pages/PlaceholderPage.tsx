import { Card, CardHeader, CardTitle, CardDescription } from '@patchiq/ui';

interface PlaceholderPageProps {
  title: string;
}

export const PlaceholderPage = ({ title }: PlaceholderPageProps) => (
  <div className="p-6">
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>Coming in M1</CardDescription>
      </CardHeader>
    </Card>
  </div>
);
