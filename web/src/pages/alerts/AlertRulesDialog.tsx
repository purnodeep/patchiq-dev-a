import { useState, useRef, useMemo, useCallback } from 'react';
import {
  Search,
  ArrowLeft,
  Lock,
  Trash2,
  Plus,
  AlertTriangle,
  CheckCircle2,
  Eye,
  Sparkles,
  ChevronRight,
  RotateCcw,
} from 'lucide-react';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  Button,
  Input,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
  Switch,
} from '@patchiq/ui';
import { useCan } from '../../app/auth/AuthContext';
import {
  useAlertRules,
  useCreateAlertRule,
  useUpdateAlertRule,
  useDeleteAlertRule,
} from '../../api/hooks/useAlerts';
import { getEventInfo, getEventsByCategory, EVENT_CATEGORIES } from './event-catalog';
import type { EventTypeInfo, FieldInfo } from './event-catalog';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AlertRulesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface AlertRule {
  id: string;
  event_type: string;
  severity: string;
  category: string;
  title_template: string;
  description_template: string;
  enabled: boolean;
}

interface BuilderForm {
  event_type: string;
  severity: string;
  category: string;
  title_template: string;
  description_template: string;
  enabled: boolean;
}

type ViewMode = 'gallery' | 'builder';
type CategoryFilter = 'all' | 'deployments' | 'cves' | 'agents' | 'compliance' | 'system';

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'var(--signal-critical)',
  warning: 'var(--signal-warning)',
  info: 'var(--signal-healthy)',
};

const SEVERITY_ICONS: Record<string, typeof AlertTriangle> = {
  critical: AlertTriangle,
  warning: AlertTriangle,
  info: CheckCircle2,
};

const CATEGORY_TABS: { id: CategoryFilter; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'deployments', label: 'Deployments' },
  { id: 'cves', label: 'CVEs' },
  { id: 'agents', label: 'Agents' },
  { id: 'compliance', label: 'Compliance' },
  { id: 'system', label: 'System' },
];

function renderPreview(template: string, fields: FieldInfo[]): string {
  let result = template;
  for (const f of fields) {
    result = result.replaceAll(`{{.${f.name}}}`, f.sample);
  }
  return result;
}

// ---------------------------------------------------------------------------
// Main Sheet
// ---------------------------------------------------------------------------

export function AlertRulesDialog({ open, onOpenChange }: AlertRulesDialogProps) {
  const { data: rules, isLoading } = useAlertRules();
  const createRule = useCreateAlertRule();
  const updateRule = useUpdateAlertRule();
  const deleteRule = useDeleteAlertRule();

  const [view, setView] = useState<ViewMode>('gallery');
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [builderForm, setBuilderForm] = useState<BuilderForm>({
    event_type: '',
    severity: 'warning',
    category: '',
    title_template: '',
    description_template: '',
    enabled: true,
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all');
  const [hoveredRuleId, setHoveredRuleId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  const titleRef = useRef<HTMLInputElement>(null);
  const descRef = useRef<HTMLTextAreaElement>(null);

  const ruleList: AlertRule[] = Array.isArray(rules) ? (rules as AlertRule[]) : [];

  const filteredRules = useMemo(() => {
    let list = ruleList;
    if (categoryFilter !== 'all') {
      list = list.filter((r) => r.category === categoryFilter);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter((r) => {
        const info = getEventInfo(r.event_type);
        return (
          r.event_type.toLowerCase().includes(q) || (info?.label ?? '').toLowerCase().includes(q)
        );
      });
    }
    return list;
  }, [ruleList, categoryFilter, searchQuery]);

  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = { all: ruleList.length };
    for (const r of ruleList) {
      counts[r.category] = (counts[r.category] ?? 0) + 1;
    }
    return counts;
  }, [ruleList]);

  const currentEventInfo: EventTypeInfo | undefined = builderForm.event_type
    ? getEventInfo(builderForm.event_type)
    : undefined;

  const duplicateWarning = useMemo(() => {
    if (editingRuleId) return false;
    if (!builderForm.event_type) return false;
    return ruleList.some((r) => r.event_type === builderForm.event_type);
  }, [editingRuleId, builderForm.event_type, ruleList]);

  // -- Navigation --

  const openBuilderForCreate = useCallback(() => {
    setEditingRuleId(null);
    setBuilderForm({
      event_type: '',
      severity: 'warning',
      category: '',
      title_template: '',
      description_template: '',
      enabled: true,
    });
    setDeleteConfirmId(null);
    setView('builder');
  }, []);

  const openBuilderForEdit = useCallback((rule: AlertRule) => {
    setEditingRuleId(rule.id);
    setBuilderForm({
      event_type: rule.event_type,
      severity: rule.severity,
      category: rule.category,
      title_template: rule.title_template,
      description_template: rule.description_template,
      enabled: rule.enabled,
    });
    setDeleteConfirmId(null);
    setView('builder');
  }, []);

  const backToGallery = useCallback(() => {
    setView('gallery');
    setEditingRuleId(null);
    setDeleteConfirmId(null);
  }, []);

  // -- Builder form --

  function handleEventTypeChange(eventType: string) {
    const info = getEventInfo(eventType);
    if (info) {
      setBuilderForm({
        event_type: eventType,
        severity: info.defaultSeverity,
        category: info.category,
        title_template: info.defaultTitle,
        description_template: info.defaultDescription,
        enabled: builderForm.enabled,
      });
    } else {
      setBuilderForm((prev) => ({ ...prev, event_type: eventType }));
    }
  }

  function handleResetToDefault() {
    const info = getEventInfo(builderForm.event_type);
    if (info) {
      setBuilderForm((prev) => ({
        ...prev,
        severity: info.defaultSeverity,
        title_template: info.defaultTitle,
        description_template: info.defaultDescription,
      }));
    }
  }

  function insertFieldAtCursor(
    fieldName: string,
    ref: React.RefObject<HTMLInputElement | HTMLTextAreaElement | null>,
    formKey: 'title_template' | 'description_template',
  ) {
    const el = ref.current;
    const token = `{{.${fieldName}}}`;
    if (el) {
      const start = el.selectionStart ?? el.value.length;
      const end = el.selectionEnd ?? start;
      const before = el.value.slice(0, start);
      const after = el.value.slice(end);
      const newValue = before + token + after;
      setBuilderForm((prev) => ({ ...prev, [formKey]: newValue }));
      requestAnimationFrame(() => {
        el.focus();
        const newPos = start + token.length;
        el.setSelectionRange(newPos, newPos);
      });
    } else {
      setBuilderForm((prev) => ({
        ...prev,
        [formKey]: prev[formKey] + token,
      }));
    }
  }

  function handleToggleEnabled(rule: AlertRule, enabled: boolean) {
    updateRule.mutate({
      id: rule.id,
      event_type: rule.event_type,
      severity: rule.severity,
      category: rule.category,
      title_template: rule.title_template,
      description_template: rule.description_template,
      enabled,
    });
  }

  function handleSave() {
    if (editingRuleId) {
      updateRule.mutate({ id: editingRuleId, ...builderForm }, { onSuccess: backToGallery });
    } else {
      createRule.mutate(builderForm, { onSuccess: backToGallery });
    }
  }

  function handleDelete() {
    if (!editingRuleId) return;
    deleteRule.mutate(editingRuleId, { onSuccess: backToGallery });
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        style={{
          width: 860,
          maxWidth: '92vw',
          padding: 0,
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {view === 'gallery' ? (
          <GalleryView
            rules={filteredRules}
            ruleList={ruleList}
            isLoading={isLoading}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            categoryFilter={categoryFilter}
            onCategoryChange={setCategoryFilter}
            categoryCounts={categoryCounts}
            hoveredRuleId={hoveredRuleId}
            onHoverRule={setHoveredRuleId}
            onCreateCustom={openBuilderForCreate}
            onEditRule={openBuilderForEdit}
            onToggleEnabled={handleToggleEnabled}
          />
        ) : (
          <BuilderView
            form={builderForm}
            isEditing={editingRuleId !== null}
            eventInfo={currentEventInfo}
            duplicateWarning={duplicateWarning}
            deleteConfirmId={deleteConfirmId}
            editingRuleId={editingRuleId}
            isSaving={createRule.isPending || updateRule.isPending}
            isDeleting={deleteRule.isPending}
            saveError={createRule.isError || updateRule.isError}
            titleRef={titleRef}
            descRef={descRef}
            onBack={backToGallery}
            onFormChange={(field, value) => setBuilderForm((prev) => ({ ...prev, [field]: value }))}
            onEventTypeChange={handleEventTypeChange}
            onSeverityChange={(sev) => setBuilderForm((prev) => ({ ...prev, severity: sev }))}
            onInsertField={insertFieldAtCursor}
            onResetToDefault={handleResetToDefault}
            onSave={handleSave}
            onDelete={handleDelete}
            onDeleteConfirm={setDeleteConfirmId}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

// ===========================================================================
// Gallery View
// ===========================================================================

interface GalleryViewProps {
  rules: AlertRule[];
  ruleList: AlertRule[];
  isLoading: boolean;
  searchQuery: string;
  onSearchChange: (q: string) => void;
  categoryFilter: CategoryFilter;
  onCategoryChange: (c: CategoryFilter) => void;
  categoryCounts: Record<string, number>;
  hoveredRuleId: string | null;
  onHoverRule: (id: string | null) => void;
  onCreateCustom: () => void;
  onEditRule: (rule: AlertRule) => void;
  onToggleEnabled: (rule: AlertRule, enabled: boolean) => void;
}

function GalleryView({
  rules,
  isLoading,
  searchQuery,
  onSearchChange,
  categoryFilter,
  onCategoryChange,
  categoryCounts,
  hoveredRuleId,
  onHoverRule,
  onCreateCustom,
  onEditRule,
  onToggleEnabled,
}: GalleryViewProps) {
  const can = useCan();
  return (
    <>
      <SheetHeader style={{ padding: '16px 56px 12px 20px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
          }}
        >
          <div style={{ minWidth: 0 }}>
            <SheetTitle style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-emphasis)' }}>
              Alert Rules
            </SheetTitle>
            <SheetDescription
              style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 2 }}
            >
              Configure which events generate alerts and their severity.
            </SheetDescription>
          </div>
          <Button
            size="sm"
            onClick={onCreateCustom}
            disabled={!can('alerts', 'manage')}
            title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
            style={{ flexShrink: 0 }}
          >
            <Plus style={{ width: 13, height: 13, marginRight: 4 }} />
            New Rule
          </Button>
        </div>
      </SheetHeader>

      {/* Search + Categories */}
      <div style={{ padding: '0 16px 0', display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
          <Search
            style={{
              position: 'absolute',
              left: 10,
              width: 14,
              height: 14,
              color: 'var(--text-muted)',
              pointerEvents: 'none',
            }}
          />
          <Input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search rules..."
            style={{ width: '100%', height: 32, fontSize: 12, paddingLeft: 32 }}
          />
        </div>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {CATEGORY_TABS.map((tab) => {
            const isActive = categoryFilter === tab.id;
            const count = categoryCounts[tab.id] ?? 0;
            return (
              <button
                key={tab.id}
                onClick={() => onCategoryChange(tab.id)}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  padding: '3px 9px',
                  fontSize: 11,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  background: isActive ? 'var(--bg-elevated)' : 'transparent',
                  border: `1px solid ${isActive ? 'var(--border-hover)' : 'var(--border)'}`,
                  borderRadius: 'var(--radius-full, 9999px)',
                  cursor: 'pointer',
                  transition: 'all 0.12s ease',
                  whiteSpace: 'nowrap',
                }}
              >
                {tab.label}
                <span
                  style={{
                    fontSize: 10,
                    fontFamily: 'var(--font-mono)',
                    color: isActive ? 'var(--text-secondary)' : 'var(--text-faint)',
                  }}
                >
                  {count}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Rules list */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '8px 0', marginTop: 4 }}>
        {isLoading && (
          <div style={{ padding: '24px 20px', color: 'var(--text-muted)', fontSize: 13 }}>
            Loading rules...
          </div>
        )}

        {!isLoading && rules.length === 0 && (
          <div style={{ padding: '32px 20px', textAlign: 'center' }}>
            <div style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 4 }}>
              No rules match your search.
            </div>
          </div>
        )}

        {rules.map((rule) => {
          const info = getEventInfo(rule.event_type);
          const sevColor = SEVERITY_COLORS[rule.severity] ?? SEVERITY_COLORS.info;
          const SevIcon = SEVERITY_ICONS[rule.severity] ?? CheckCircle2;
          const isHovered = hoveredRuleId === rule.id;

          return (
            <div
              key={rule.id}
              onMouseEnter={() => onHoverRule(rule.id)}
              onMouseLeave={() => onHoverRule(null)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '9px 16px',
                borderBottom: '1px solid var(--border-faint, var(--border))',
                cursor: 'pointer',
                background: isHovered ? 'var(--bg-inset)' : 'transparent',
                transition: 'background 0.1s',
              }}
              onClick={() => onEditRule(rule)}
            >
              {/* Severity icon */}
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 6,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: `color-mix(in srgb, ${sevColor} 8%, transparent)`,
                  color: sevColor,
                  flexShrink: 0,
                }}
              >
                <SevIcon style={{ width: 13, height: 13 }} />
              </div>

              {/* Labels */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 13,
                    color: 'var(--text-emphasis)',
                    fontWeight: 500,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {info?.label ?? rule.event_type}
                </div>
                <div
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: 'var(--text-muted)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {rule.event_type}
                </div>
              </div>

              {/* Severity badge */}
              <span
                style={{
                  fontSize: 10,
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.04em',
                  background: `color-mix(in srgb, ${sevColor} 8%, transparent)`,
                  color: sevColor,
                  padding: '2px 7px',
                  borderRadius: 'var(--radius-full, 9999px)',
                  flexShrink: 0,
                }}
              >
                {rule.severity}
              </span>

              {/* Edit arrow */}
              <ChevronRight
                style={{
                  width: 14,
                  height: 14,
                  color: 'var(--text-faint)',
                  opacity: isHovered ? 1 : 0,
                  transition: 'opacity 0.1s',
                  flexShrink: 0,
                }}
              />

              {/* Toggle */}
              <div
                onClick={(e) => {
                  e.stopPropagation();
                  if (can('alerts', 'manage')) onToggleEnabled(rule, !rule.enabled);
                }}
                title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
                style={{ flexShrink: 0, opacity: !can('alerts', 'manage') ? 0.5 : undefined }}
              >
                <Switch
                  checked={rule.enabled}
                  disabled={!can('alerts', 'manage')}
                  onCheckedChange={() => undefined}
                />
              </div>
            </div>
          );
        })}
      </div>
    </>
  );
}

// ===========================================================================
// Builder View — form + inline live preview (stacked, same panel)
// ===========================================================================

interface BuilderViewProps {
  form: BuilderForm;
  isEditing: boolean;
  eventInfo: EventTypeInfo | undefined;
  duplicateWarning: boolean;
  deleteConfirmId: string | null;
  editingRuleId: string | null;
  isSaving: boolean;
  isDeleting: boolean;
  saveError: boolean;
  titleRef: React.RefObject<HTMLInputElement | null>;
  descRef: React.RefObject<HTMLTextAreaElement | null>;
  onBack: () => void;
  onFormChange: (field: keyof BuilderForm, value: string | boolean) => void;
  onEventTypeChange: (eventType: string) => void;
  onSeverityChange: (sev: string) => void;
  onInsertField: (
    fieldName: string,
    ref: React.RefObject<HTMLInputElement | HTMLTextAreaElement | null>,
    formKey: 'title_template' | 'description_template',
  ) => void;
  onResetToDefault: () => void;
  onSave: () => void;
  onDelete: () => void;
  onDeleteConfirm: (id: string | null) => void;
}

function BuilderView({
  form,
  isEditing,
  eventInfo,
  duplicateWarning,
  deleteConfirmId,
  editingRuleId,
  isSaving,
  isDeleting,
  saveError,
  titleRef,
  descRef,
  onBack,
  onFormChange,
  onEventTypeChange,
  onSeverityChange,
  onInsertField,
  onResetToDefault,
  onSave,
  onDelete,
  onDeleteConfirm,
}: BuilderViewProps) {
  const can = useCan();
  const fields = eventInfo?.fields ?? [];
  const previewTitle = form.title_template ? renderPreview(form.title_template, fields) : '';
  const previewDesc = form.description_template
    ? renderPreview(form.description_template, fields)
    : '';

  const sevColor = SEVERITY_COLORS[form.severity] ?? SEVERITY_COLORS.info;
  const SevIcon = SEVERITY_ICONS[form.severity] ?? CheckCircle2;

  const sectionLabel: React.CSSProperties = {
    fontFamily: 'var(--font-mono)',
    fontSize: 9,
    fontWeight: 600,
    letterSpacing: '0.08em',
    textTransform: 'uppercase',
    color: 'var(--text-faint)',
    marginBottom: 8,
  };

  const fieldLabel: React.CSSProperties = {
    fontSize: 11,
    color: 'var(--text-secondary)',
    marginBottom: 6,
    fontWeight: 500,
  };

  return (
    <>
      {/* Header */}
      <SheetHeader style={{ padding: '16px 20px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <button
            onClick={onBack}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 28,
              height: 28,
              background: 'none',
              border: '1px solid var(--border)',
              borderRadius: 6,
              cursor: 'pointer',
              color: 'var(--text-muted)',
              flexShrink: 0,
            }}
          >
            <ArrowLeft style={{ width: 14, height: 14 }} />
          </button>
          <div style={{ flex: 1 }}>
            <SheetTitle style={{ fontSize: 15, fontWeight: 600, color: 'var(--text-emphasis)' }}>
              {isEditing ? 'Edit Alert Rule' : 'Create Alert Rule'}
            </SheetTitle>
            <SheetDescription style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 1 }}>
              {isEditing
                ? 'Modify this alert rule configuration.'
                : 'Define when and how alerts are generated.'}
            </SheetDescription>
          </div>
        </div>
      </SheetHeader>

      {/* Scrollable body — form + preview stacked */}
      <div style={{ flex: 1, overflowY: 'auto', display: 'flex' }}>
        {/* Form column */}
        <div
          style={{
            flex: 1,
            minWidth: 0,
            padding: '12px 20px',
            display: 'flex',
            flexDirection: 'column',
            gap: 20,
          }}
        >
          {/* Duplicate warning */}
          {duplicateWarning && (
            <div
              style={{
                padding: '8px 12px',
                borderRadius: 6,
                background: 'color-mix(in srgb, var(--signal-warning) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-warning) 30%, transparent)',
                fontSize: 12,
                color: 'var(--signal-warning)',
              }}
            >
              A rule for this event type already exists.
            </div>
          )}

          {/* Event Type */}
          <div>
            <div style={sectionLabel}>Event Type</div>
            <Select value={form.event_type} onValueChange={onEventTypeChange} disabled={isEditing}>
              <SelectTrigger style={{ height: 36, fontSize: 12 }}>
                <SelectValue placeholder="Select an event type..." />
              </SelectTrigger>
              <SelectContent>
                {EVENT_CATEGORIES.map((cat) => {
                  const events = getEventsByCategory(cat.id);
                  if (events.length === 0) return null;
                  return (
                    <SelectGroup key={cat.id}>
                      <SelectLabel
                        style={{
                          fontSize: 10,
                          fontWeight: 600,
                          textTransform: 'uppercase',
                          letterSpacing: '0.06em',
                        }}
                      >
                        {cat.label}
                      </SelectLabel>
                      {events.map((evt) => (
                        <SelectItem key={evt.type} value={evt.type}>
                          <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11,
                                color: 'var(--text-muted)',
                              }}
                            >
                              {evt.type}
                            </span>
                            <span style={{ fontSize: 12, color: 'var(--text-primary)' }}>
                              {evt.label}
                            </span>
                          </span>
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  );
                })}
              </SelectContent>
            </Select>
            {eventInfo && (
              <p
                style={{
                  fontSize: 11,
                  color: 'var(--text-muted)',
                  margin: 0,
                  marginTop: 6,
                  lineHeight: 1.5,
                }}
              >
                {eventInfo.description}
              </p>
            )}
          </div>

          {/* Classification */}
          {form.event_type && (
            <div>
              <div style={sectionLabel}>Classification</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {/* Severity pills */}
                <div>
                  <div style={fieldLabel}>Severity</div>
                  <div style={{ display: 'flex', gap: 6 }}>
                    {(['critical', 'warning', 'info'] as const).map((sev) => {
                      const isActive = form.severity === sev;
                      const color = SEVERITY_COLORS[sev];
                      return (
                        <button
                          key={sev}
                          onClick={() => onSeverityChange(sev)}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                            padding: '5px 12px',
                            fontSize: 12,
                            fontWeight: isActive ? 600 : 400,
                            color: isActive ? color : 'var(--text-muted)',
                            background: isActive
                              ? `color-mix(in srgb, ${color} 8%, transparent)`
                              : 'transparent',
                            border: `1px solid ${isActive ? `color-mix(in srgb, ${color} 25%, transparent)` : 'var(--border)'}`,
                            borderRadius: 8,
                            cursor: 'pointer',
                            textTransform: 'capitalize',
                            transition: 'all 0.12s ease',
                          }}
                        >
                          <span
                            style={{
                              width: 7,
                              height: 7,
                              borderRadius: '50%',
                              background: color,
                              opacity: isActive ? 1 : 0.4,
                            }}
                          />
                          {sev}
                        </button>
                      );
                    })}
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                  {/* Category badge */}
                  <div>
                    <div style={fieldLabel}>Category</div>
                    <span
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 4,
                        padding: '3px 10px',
                        fontSize: 12,
                        color: 'var(--text-secondary)',
                        background: 'var(--bg-elevated)',
                        border: '1px solid var(--border)',
                        borderRadius: 'var(--radius-full, 9999px)',
                      }}
                    >
                      <Lock style={{ width: 10, height: 10, color: 'var(--text-faint)' }} />
                      {form.category}
                    </span>
                  </div>

                  {/* Enabled toggle */}
                  <div>
                    <div style={fieldLabel}>Status</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <Switch
                        checked={form.enabled}
                        onCheckedChange={(checked) => onFormChange('enabled', checked)}
                      />
                      <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                        {form.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Alert Content */}
          {form.event_type && (
            <div>
              <div style={sectionLabel}>Alert Content</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                <div>
                  <div style={fieldLabel}>Alert Title</div>
                  <FieldChips
                    fields={fields}
                    onInsert={(name) => onInsertField(name, titleRef, 'title_template')}
                  />
                  <Input
                    ref={titleRef}
                    value={form.title_template}
                    onChange={(e) => onFormChange('title_template', e.target.value)}
                    placeholder="Alert title with {{.field}} placeholders"
                    style={{ fontFamily: 'var(--font-mono)', fontSize: 12, height: 32 }}
                  />
                </div>
                <div>
                  <div style={fieldLabel}>Description</div>
                  <FieldChips
                    fields={fields}
                    onInsert={(name) => onInsertField(name, descRef, 'description_template')}
                  />
                  <textarea
                    ref={descRef}
                    value={form.description_template}
                    onChange={(e) => onFormChange('description_template', e.target.value)}
                    placeholder="Alert description with {{.field}} placeholders"
                    rows={3}
                    style={{
                      width: '100%',
                      fontFamily: 'var(--font-mono)',
                      fontSize: 12,
                      padding: '8px 10px',
                      borderRadius: 6,
                      border: '1px solid var(--border)',
                      background: 'var(--bg-card)',
                      color: 'var(--text-primary)',
                      resize: 'vertical',
                      outline: 'none',
                      lineHeight: 1.5,
                      boxSizing: 'border-box',
                    }}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Delete zone */}
          {isEditing && (
            <div style={{ borderTop: '1px solid var(--border)', paddingTop: 14 }}>
              {deleteConfirmId === editingRuleId ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                    Delete permanently?
                  </span>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={onDelete}
                    disabled={isDeleting || !can('alerts', 'manage')}
                    title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
                    style={{ fontSize: 11 }}
                  >
                    {isDeleting ? 'Deleting...' : 'Confirm'}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onDeleteConfirm(null)}
                    style={{ fontSize: 11 }}
                  >
                    Cancel
                  </Button>
                </div>
              ) : (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!can('alerts', 'manage')}
                  title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
                  onClick={() => onDeleteConfirm(editingRuleId)}
                  style={{
                    color: 'var(--signal-critical)',
                    borderColor: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  }}
                >
                  <Trash2 style={{ width: 12, height: 12, marginRight: 4 }} />
                  Delete Rule
                </Button>
              )}
            </div>
          )}

          {saveError && (
            <div style={{ fontSize: 12, color: 'var(--signal-critical)' }}>
              Failed to save. Please try again.
            </div>
          )}
        </div>

        {/* Live preview sidebar — same pattern as ImpactPreview in DeploymentWizard */}
        {form.event_type && (
          <div
            style={{
              width: 260,
              flexShrink: 0,
              borderLeft: '1px solid var(--border)',
              background: 'var(--bg-page)',
              padding: '16px 14px',
              display: 'flex',
              flexDirection: 'column',
              gap: 14,
              overflowY: 'auto',
            }}
          >
            {/* Preview heading */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Eye style={{ width: 12, height: 12, color: 'var(--text-faint)' }} />
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9,
                  fontWeight: 600,
                  letterSpacing: '0.08em',
                  textTransform: 'uppercase',
                  color: 'var(--text-faint)',
                }}
              >
                Live Preview
              </span>
            </div>

            {/* Mini alert card */}
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                overflow: 'hidden',
              }}
            >
              <div style={{ height: 3, background: sevColor, transition: 'background 0.15s' }} />
              <div style={{ padding: '10px 12px', display: 'flex', gap: 8 }}>
                <div
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 6,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: `color-mix(in srgb, ${sevColor} 8%, transparent)`,
                    color: sevColor,
                    flexShrink: 0,
                  }}
                >
                  <SevIcon style={{ width: 14, height: 14 }} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      fontSize: 12,
                      fontWeight: 600,
                      color: 'var(--text-emphasis)',
                      lineHeight: 1.3,
                      marginBottom: 3,
                    }}
                  >
                    {previewTitle || 'Title preview'}
                  </div>
                  <div
                    style={{
                      fontSize: 11,
                      color: 'var(--text-secondary)',
                      lineHeight: 1.4,
                      marginBottom: 6,
                    }}
                  >
                    {previewDesc || 'Description preview'}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                    <span
                      style={{
                        fontSize: 9,
                        fontWeight: 600,
                        fontFamily: 'var(--font-mono)',
                        textTransform: 'uppercase',
                        background: `color-mix(in srgb, ${sevColor} 8%, transparent)`,
                        color: sevColor,
                        padding: '1px 6px',
                        borderRadius: 'var(--radius-full, 9999px)',
                      }}
                    >
                      {form.severity}
                    </span>
                    <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>
                      {form.category}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Template reference */}
            <div
              style={{
                background: 'var(--bg-card)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                padding: '10px 12px',
              }}
            >
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 9,
                  fontWeight: 600,
                  letterSpacing: '0.08em',
                  textTransform: 'uppercase',
                  color: 'var(--text-faint)',
                  marginBottom: 8,
                }}
              >
                Templates
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div>
                  <span
                    style={{
                      fontSize: 9,
                      fontWeight: 500,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    title
                  </span>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-secondary)',
                      background: 'var(--bg-inset)',
                      borderRadius: 4,
                      padding: '3px 6px',
                      marginTop: 2,
                      wordBreak: 'break-all',
                      lineHeight: 1.4,
                    }}
                  >
                    {form.title_template || '—'}
                  </div>
                </div>
                <div>
                  <span
                    style={{
                      fontSize: 9,
                      fontWeight: 500,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    description
                  </span>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-secondary)',
                      background: 'var(--bg-inset)',
                      borderRadius: 4,
                      padding: '3px 6px',
                      marginTop: 2,
                      wordBreak: 'break-all',
                      lineHeight: 1.4,
                    }}
                  >
                    {form.description_template || '—'}
                  </div>
                </div>
              </div>

              {/* Field reference */}
              {fields.length > 0 && (
                <div style={{ marginTop: 10, borderTop: '1px solid var(--border)', paddingTop: 8 }}>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 9,
                      fontWeight: 600,
                      letterSpacing: '0.08em',
                      textTransform: 'uppercase',
                      color: 'var(--text-faint)',
                      marginBottom: 6,
                    }}
                  >
                    Fields
                  </div>
                  {fields.map((f) => (
                    <div
                      key={f.name}
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        gap: 4,
                        fontSize: 10,
                        marginBottom: 3,
                      }}
                    >
                      <code style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)' }}>
                        {f.name}
                      </code>
                      <span style={{ color: 'var(--text-faint)', fontSize: 9 }}>{f.type}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div
              style={{
                fontSize: 9,
                color: 'var(--text-faint)',
                fontStyle: 'italic',
                display: 'flex',
                alignItems: 'center',
                gap: 4,
              }}
            >
              <Sparkles style={{ width: 10, height: 10 }} />
              Sample data preview
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div
        style={{
          padding: '10px 20px',
          borderTop: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <Button
          size="sm"
          onClick={onSave}
          disabled={isSaving || !form.event_type || !can('alerts', 'manage')}
          title={!can('alerts', 'manage') ? "You don't have permission" : undefined}
        >
          {isSaving ? 'Saving...' : 'Save Rule'}
        </Button>
        <Button size="sm" variant="outline" onClick={onBack}>
          Cancel
        </Button>
        {isEditing && (
          <Button size="sm" variant="outline" onClick={onResetToDefault} style={{ fontSize: 11 }}>
            <RotateCcw style={{ width: 11, height: 11, marginRight: 4 }} />
            Reset
          </Button>
        )}
      </div>
    </>
  );
}

// ===========================================================================
// Field Chips
// ===========================================================================

function FieldChips({
  fields,
  onInsert,
}: {
  fields: FieldInfo[];
  onInsert: (fieldName: string) => void;
}) {
  if (fields.length === 0) {
    return (
      <div
        style={{ fontSize: 11, color: 'var(--text-faint)', marginBottom: 6, fontStyle: 'italic' }}
      >
        No dynamic fields available for this event type
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 6 }}>
      {fields.map((f) => (
        <button
          key={f.name}
          onClick={() => onInsert(f.name)}
          title={`${f.type} — ${f.description} (e.g., ${f.sample})`}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
            padding: '2px 8px',
            borderRadius: 6,
            background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
            color: 'var(--accent)',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            cursor: 'pointer',
            border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
            transition: 'all 0.1s ease',
          }}
        >
          <Plus style={{ width: 10, height: 10 }} />
          {f.name}
        </button>
      ))}
    </div>
  );
}
