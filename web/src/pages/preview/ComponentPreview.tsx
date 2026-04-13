import { useState } from 'react';
import { AlertBanner, RingChart, SparklineChart, Button } from '@patchiq/ui';
import { Monitor, ShieldAlert, Rocket, CheckCircle, Package, Inbox } from 'lucide-react';
import { PageHeader } from '../../components/PageHeader';
import { StatCard } from '../../components/StatCard';
import { FilterBar, FilterPill, FilterSeparator, FilterSearch } from '../../components/FilterBar';
import { SeverityBadge } from '../../components/SeverityBadge';
import { StatusBadge } from '../../components/StatusBadge';
import { EmptyState } from '../../components/EmptyState';

const sectionStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 16,
};

const sectionHeadingStyle: React.CSSProperties = {
  fontFamily: 'var(--font-sans)',
  fontSize: 16,
  fontWeight: 600,
  color: 'var(--text-primary)',
  margin: 0,
};

export const ComponentPreview = () => {
  const [alertVisible, setAlertVisible] = useState(true);
  const [activePill, setActivePill] = useState('all');
  const [search, setSearch] = useState('');

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'var(--bg-page)',
        padding: 32,
        display: 'flex',
        flexDirection: 'column',
        gap: 40,
      }}
    >
      <h1
        style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 28,
          fontWeight: 700,
          color: 'var(--text-emphasis)',
          margin: 0,
          letterSpacing: '-0.02em',
        }}
      >
        Component Preview — PIQ-232
      </h1>

      {/* 1. PageHeader */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>1. PageHeader</h2>
        <PageHeader
          breadcrumbs={[
            { label: 'Home', href: '/' },
            { label: 'Inventory', href: '/patches' },
            { label: 'Patches' },
          ]}
          title="Patches"
          count={156}
          actions={
            <>
              <Button variant="outline" size="sm">
                Export
              </Button>
              <Button size="sm">Create Deployment</Button>
            </>
          }
        />
      </section>

      {/* 2. StatCard */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>2. StatCard</h2>
        <div style={{ display: 'grid', gap: 12, gridTemplateColumns: 'repeat(4, 1fr)' }}>
          <StatCard
            icon={Monitor}
            iconColor="bg-green-500/10 text-green-500"
            value={247}
            label="Endpoints Online"
            trend={{ direction: 'up', percentage: '2.1%', context: 'vs last week' }}
            visualization={<RingChart value={247} max={260} color="var(--accent)" label="/ 260" />}
          />
          <StatCard
            icon={ShieldAlert}
            iconColor="bg-red-500/10 text-red-500"
            value={23}
            label="Critical CVEs"
            trend={{ direction: 'up', percentage: '5.3%', context: 'vs last week' }}
            visualization={
              <SparklineChart data={[3, 5, 8, 12, 15, 18, 23]} color="var(--signal-critical)" />
            }
          />
          <StatCard
            icon={Rocket}
            iconColor="bg-emerald-500/10 text-emerald-500"
            value={12}
            label="Active Deployments"
            trend={{ direction: 'down', percentage: '1.2%', context: 'vs last week' }}
          />
          <StatCard
            icon={CheckCircle}
            iconColor="bg-amber-500/10 text-amber-500"
            value="87%"
            label="Compliance Rate"
            visualization={
              <RingChart value={87} max={100} color="var(--signal-warning)" label="score" />
            }
          />
        </div>
      </section>

      {/* 3. FilterBar */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>3. FilterBar</h2>
        <FilterBar>
          <FilterPill
            label="All"
            count={156}
            active={activePill === 'all'}
            onClick={() => setActivePill('all')}
          />
          <FilterPill
            label="Critical"
            count={12}
            active={activePill === 'critical'}
            variant="critical"
            onClick={() => setActivePill('critical')}
          />
          <FilterPill
            label="High"
            count={28}
            active={activePill === 'high'}
            variant="high"
            onClick={() => setActivePill('high')}
          />
          <FilterPill
            label="Medium"
            count={45}
            active={activePill === 'medium'}
            variant="medium"
            onClick={() => setActivePill('medium')}
          />
          <FilterPill
            label="Low"
            count={71}
            active={activePill === 'low'}
            variant="low"
            onClick={() => setActivePill('low')}
          />
          <FilterSeparator />
          <FilterSearch
            value={search}
            onChange={setSearch}
            placeholder="Search patches (KB, USN, RHSA...)"
          />
        </FilterBar>
      </section>

      {/* 4. AlertBanner */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>4. AlertBanner</h2>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {alertVisible && (
            <AlertBanner
              severity="warning"
              message="3 critical patches overdue SLA"
              detail="2 deployments failed in last hour"
              onDismiss={() => setAlertVisible(false)}
            />
          )}
          <AlertBanner
            severity="critical"
            message="Production outage detected"
            detail="db-primary-02 unreachable"
          />
          <AlertBanner
            severity="info"
            message="Catalog sync completed"
            detail="142 new patches available"
          />
          {!alertVisible && (
            <button
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 0,
                fontFamily: 'var(--font-sans)',
                fontSize: 12,
                color: 'var(--accent)',
                textDecoration: 'underline',
              }}
              onClick={() => setAlertVisible(true)}
            >
              Show dismissed alert again
            </button>
          )}
        </div>
      </section>

      {/* 5. SeverityBadge */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>5. SeverityBadge</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <SeverityBadge severity="critical" />
          <SeverityBadge severity="high" />
          <SeverityBadge severity="medium" />
          <SeverityBadge severity="low" />
          <SeverityBadge severity="none" />
        </div>
      </section>

      {/* 6. StatusBadge */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>6. StatusBadge</h2>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <p
            style={{
              margin: 0,
              fontSize: 11,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            Endpoint statuses:
          </p>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <StatusBadge status="online" />
            <StatusBadge status="offline" />
            <StatusBadge status="degraded" />
          </div>
          <p
            style={{
              margin: '8px 0 0',
              fontSize: 11,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            Deployment statuses:
          </p>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <StatusBadge status="running" />
            <StatusBadge status="completed" />
            <StatusBadge status="failed" />
            <StatusBadge status="pending" />
            <StatusBadge status="cancelled" />
          </div>
          <p
            style={{
              margin: '8px 0 0',
              fontSize: 11,
              color: 'var(--text-muted)',
              fontFamily: 'var(--font-mono)',
            }}
          >
            Policy statuses:
          </p>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <StatusBadge status="enforce" />
            <StatusBadge status="audit" />
            <StatusBadge status="disabled" />
          </div>
        </div>
      </section>

      {/* 7. RingChart */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>7. RingChart</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <RingChart value={247} max={260} color="var(--accent)" label="/ 260" />
          <RingChart
            value={87}
            max={100}
            color="var(--signal-warning)"
            label="score"
            size={80}
            thickness={8}
          />
          <RingChart value={23} max={100} color="var(--signal-critical)" label="critical" />
          <RingChart value={100} max={100} color="var(--accent)" label="done" />
        </div>
      </section>

      {/* 8. SparklineChart */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>8. SparklineChart</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <SparklineChart data={[3, 5, 2, 8, 12, 9, 15, 18, 23]} color="var(--accent)" />
          <SparklineChart data={[20, 18, 15, 12, 8, 5, 3]} color="var(--signal-critical)" />
          <SparklineChart data={[5, 8, 3, 12, 7, 15, 10]} color="var(--accent)" />
          <SparklineChart
            data={[1, 1, 2, 3, 5, 8, 13, 21]}
            color="var(--signal-warning)"
            width={100}
            height={40}
          />
        </div>
      </section>

      {/* 9. EmptyState */}
      <section style={sectionStyle}>
        <h2 style={sectionHeadingStyle}>9. EmptyState</h2>
        <div
          style={{
            borderRadius: 8,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
          }}
        >
          <EmptyState
            icon={Inbox}
            title="No deployments found"
            description="Create your first deployment to start patching endpoints."
            action={{ label: 'Create Deployment', onClick: () => alert('Create clicked') }}
          />
        </div>
        <div
          style={{
            borderRadius: 8,
            border: '1px solid var(--border)',
            background: 'var(--bg-card)',
          }}
        >
          <EmptyState
            icon={Package}
            title="No patches match your filters"
            description="Try adjusting your search or filter criteria."
          />
        </div>
      </section>
    </div>
  );
};
