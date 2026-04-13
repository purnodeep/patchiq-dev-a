import { Card, CardContent, CardHeader, CardTitle } from '@patchiq/ui';
import { Check } from 'lucide-react';

interface DeploymentPipelineWidgetProps {
  stages: string[];
  currentStage: number;
}

function stageCircleClass(i: number, currentStage: number): string {
  const base = 'h-8 w-8 rounded-full flex items-center justify-center text-xs font-medium border-2';
  if (i < currentStage) return `${base} bg-emerald-500/10 border-emerald-500 text-emerald-500`;
  if (i === currentStage)
    return `${base} bg-purple-500/10 border-purple-500 text-purple-500 animate-pulse`;
  return `${base} bg-muted/10 border-muted text-muted-foreground`;
}

export function DeploymentPipelineWidget({ stages, currentStage }: DeploymentPipelineWidgetProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Deployment Pipeline</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          {stages.map((stage, i) => (
            <div key={stage} className="flex items-center">
              <div className="flex flex-col items-center gap-1">
                <div className={stageCircleClass(i, currentStage)}>
                  {i < currentStage ? <Check className="h-3.5 w-3.5" /> : i + 1}
                </div>
                <span
                  className={`text-[9px] ${i <= currentStage ? 'text-foreground' : 'text-muted-foreground'}`}
                >
                  {stage}
                </span>
              </div>
              {i < stages.length - 1 && (
                <div
                  className={`h-0.5 w-5 mx-1 ${i < currentStage ? 'bg-emerald-500' : 'bg-muted'}`}
                />
              )}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
