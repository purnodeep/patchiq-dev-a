import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
  Button,
  Input,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@patchiq/ui';
import { toast } from 'sonner';

interface AddFeedFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const defaultValues = {
  name: '',
  fullName: '',
  feedIcon: 'shield',
  iconColor: 'blue',
  sourceUrl: '',
  syncIntervalHours: 24,
  description: '',
  initialTotalEntries: 0,
  errorRate: 0,
};

export const AddFeedForm = ({ open, onOpenChange }: AddFeedFormProps) => {
  const [values, setValues] = useState(defaultValues);

  const set = <K extends keyof typeof defaultValues>(key: K, value: (typeof defaultValues)[K]) => {
    setValues((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onOpenChange(false);
    setValues(defaultValues);
    toast('Feed creation is not yet supported.', { description: 'Backend endpoint pending.' });
  };

  const handleCancel = () => {
    onOpenChange(false);
    setValues(defaultValues);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px] bg-card border">
        <DialogHeader>
          <DialogTitle className="text-foreground">Add Feed</DialogTitle>
          <DialogDescription className="text-muted-foreground text-sm">
            Configure a new vulnerability or patch data feed source.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {/* Feed Name */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Feed Name
            </label>
            <Input
              placeholder="e.g., NVD, KEV, MSRC"
              value={values.name}
              onChange={(e) => set('name', e.target.value)}
              className="bg-background border text-foreground"
            />
          </div>

          {/* Full Name */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Full Name
            </label>
            <Input
              placeholder="e.g., National Vulnerability Database"
              value={values.fullName}
              onChange={(e) => set('fullName', e.target.value)}
              className="bg-background border text-foreground"
            />
          </div>

          {/* Feed Icon + Icon Color */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Feed Icon
              </label>
              <Select value={values.feedIcon} onValueChange={(v) => set('feedIcon', v)}>
                <SelectTrigger className="bg-background border text-foreground">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="bg-card border">
                  <SelectItem value="shield">🛡 Shield</SelectItem>
                  <SelectItem value="circle-red">🔴 Circle-Red</SelectItem>
                  <SelectItem value="circle-purple">🟣 Circle-Purple</SelectItem>
                  <SelectItem value="circle-orange">🟠 Circle-Orange</SelectItem>
                  <SelectItem value="gear">⚙ Gear</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Icon Color
              </label>
              <Select value={values.iconColor} onValueChange={(v) => set('iconColor', v)}>
                <SelectTrigger className="bg-background border text-foreground">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="bg-card border">
                  <SelectItem value="blue">Blue</SelectItem>
                  <SelectItem value="red">Red</SelectItem>
                  <SelectItem value="purple">Purple</SelectItem>
                  <SelectItem value="orange">Orange</SelectItem>
                  <SelectItem value="gray">Gray</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Feed Source URL */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Feed Source URL
            </label>
            <Input
              placeholder="https://api.example.com/feed"
              value={values.sourceUrl}
              onChange={(e) => set('sourceUrl', e.target.value)}
              className="bg-background border text-foreground font-mono text-sm"
            />
          </div>

          {/* Sync Interval + Initial Total Entries + Error Rate */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Sync Interval (h)
              </label>
              <Input
                type="number"
                min={1}
                value={values.syncIntervalHours}
                onChange={(e) => set('syncIntervalHours', Number(e.target.value))}
                className="bg-background border text-foreground"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Initial Entries
              </label>
              <Input
                type="number"
                min={0}
                value={values.initialTotalEntries}
                onChange={(e) => set('initialTotalEntries', Number(e.target.value))}
                className="bg-background border text-foreground"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Error Rate (%)
              </label>
              <Input
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={values.errorRate}
                onChange={(e) => set('errorRate', Number(e.target.value))}
                className="bg-background border text-foreground"
              />
            </div>
          </div>

          {/* Description */}
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Description
            </label>
            <textarea
              placeholder="Brief description of the feed"
              value={values.description}
              onChange={(e) => set('description', e.target.value)}
              rows={3}
              className="w-full rounded-md border border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-[var(--accent)] resize-none"
            />
          </div>

          <DialogFooter className="pt-2">
            <Button type="button" variant="outline" onClick={handleCancel}>
              Cancel
            </Button>
            <Button type="submit">Add Feed</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
