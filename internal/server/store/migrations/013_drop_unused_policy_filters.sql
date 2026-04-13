-- +goose Up
-- Issue #156: Drop unused policy filter columns added in 010_deployment_engine.sql.
--
-- The policies table has two sets of filter columns serving different subsystems:
--
--   008_policy_engine.sql columns (KEPT — used by policy dry-run evaluator):
--     selection_mode, min_severity, cve_ids, package_regex, exclude_packages
--     Used in: internal/server/policy/evaluator.go
--
--   010_deployment_engine.sql columns:
--     severity_filter   (KEPT — used by deployment evaluator via ListPatchesForPolicyFilters)
--     classification_filter  (DROPPED — never referenced in application code)
--     product_filter         (DROPPED — never referenced in application code)

ALTER TABLE policies DROP COLUMN IF EXISTS classification_filter;
ALTER TABLE policies DROP COLUMN IF EXISTS product_filter;

-- +goose Down
ALTER TABLE policies ADD COLUMN classification_filter TEXT[];
ALTER TABLE policies ADD COLUMN product_filter TEXT[];
