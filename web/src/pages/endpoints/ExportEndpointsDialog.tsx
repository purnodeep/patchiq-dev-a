import { useState } from 'react';
import {
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@patchiq/ui';
import { downloadEndpointExport } from './export-csv';

interface ExportEndpointsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  filteredCount?: number;
  filters: {
    status?: string;
    os_family?: string;
    tag_id?: string;
    search?: string;
  };
}

export function ExportEndpointsDialog({
  open,
  onOpenChange,
  filteredCount,
  filters,
}: ExportEndpointsDialogProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const hasFilters =
    !!filters.status || !!filters.os_family || !!filters.tag_id || !!filters.search;

  const handleExport = async (filtered: boolean) => {
    setError(null);
    setLoading(true);
    try {
      await downloadEndpointExport(filtered ? filters : {});
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle style={{ fontFamily: 'var(--font-display)' }}>Export Endpoints</DialogTitle>
        </DialogHeader>

        <div className="space-y-3 text-sm text-muted-foreground">
          <p>Download a CSV file of endpoint data.</p>
          {hasFilters && (
            <div className="rounded-md border border-border bg-muted/40 px-3 py-2 text-xs">
              <span className="font-medium text-foreground">Active filters: </span>
              {[
                filters.status && `Status: ${filters.status}`,
                filters.os_family && `OS: ${filters.os_family}`,
                filters.search && `Search: ${filters.search}`,
              ]
                .filter(Boolean)
                .join(' · ')}
            </div>
          )}
          {error && <p className="text-destructive">{error}</p>}
        </div>

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          {hasFilters && (
            <Button variant="outline" onClick={() => handleExport(true)} disabled={loading}>
              {loading
                ? 'Exporting…'
                : `Export Filtered${filteredCount != null ? ` (${filteredCount})` : ''}`}
            </Button>
          )}
          <Button onClick={() => handleExport(false)} disabled={loading}>
            {loading ? 'Exporting…' : 'Export All'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
