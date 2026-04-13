// @patchiq/ui — shared component library
export { cn } from './lib/utils';
export { ThemeProvider, useTheme, ACCENT_PRESETS, type ThemeMode } from './theme';
export { Button, buttonVariants } from './components/ui/button';
export {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
  CardAction,
} from './components/ui/card';
export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogTrigger,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from './components/ui/dialog';
export { Badge, badgeVariants } from './components/ui/badge';
export { Input } from './components/ui/input';
export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
  SelectScrollUpButton,
  SelectScrollDownButton,
} from './components/ui/select';
export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuGroup,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuRadioGroup,
} from './components/ui/dropdown-menu';
export { Tabs, TabsList, TabsTrigger, TabsContent } from './components/ui/tabs';
export { Skeleton } from './components/ui/skeleton';
export { Toaster } from './components/ui/sonner';
export { RingChart } from './components/ui/ring-chart';
export type { RingChartProps } from './components/ui/ring-chart';
export { SparklineChart } from './components/ui/sparkline-chart';
export type { SparklineChartProps } from './components/ui/sparkline-chart';
export { AlertBanner } from './components/ui/alert-banner';
export type { AlertBannerProps, AlertSeverity } from './components/ui/alert-banner';
export { DataTable } from './components/ui/data-table';
export {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInput,
  SidebarInset,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
  useSidebar,
} from './components/ui/sidebar';
export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from './components/ui/tooltip';
export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
} from './components/ui/sheet';
export { Switch } from './components/ui/switch';
export { Separator } from './components/ui/separator';
export { Avatar, AvatarImage, AvatarFallback } from './components/ui/avatar';
export { Progress } from './components/ui/progress';
export { Collapsible, CollapsibleTrigger, CollapsibleContent } from './components/ui/collapsible';
export { RingGauge } from './components/ring-gauge';
export type { RingGaugeProps } from './components/ring-gauge';
export { StatCard } from './components/stat-card';
export type { StatCardProps, StatCardTrend } from './components/stat-card';
export { SeverityText } from './components/severity-text';
export type { SeverityTextProps } from './components/severity-text';
export { MonoTag } from './components/mono-tag';
export type { MonoTagProps } from './components/mono-tag';
export { PageHeader } from './components/page-header';
export type { PageHeaderProps } from './components/page-header';
export { EmptyState } from './components/empty-state';
export type { EmptyStateProps, EmptyStateAction } from './components/empty-state';
export { ErrorState } from './components/error-state';
export type { ErrorStateProps } from './components/error-state';
export { SkeletonCard } from './components/skeleton-card';
export type { SkeletonCardProps } from './components/skeleton-card';
export { DotMosaic } from './components/dot-mosaic';
export type { DotMosaicProps, DotMosaicItem, RiskLevel } from './components/dot-mosaic';
export { ThemeConfigurator } from './components/theme-configurator';
export { RouteErrorBoundary } from './components/route-error-boundary';
