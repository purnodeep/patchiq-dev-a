-- +goose Up
-- Grant UPDATE permission on compliance_scores to patchiq_app.
-- Missing from 019_compliance.sql which only granted SELECT, INSERT, DELETE.
-- Required by UpdateEndpointScoresForRun and UpdateEndpointScoreByID queries.
GRANT UPDATE ON compliance_scores TO patchiq_app;

-- +goose Down
REVOKE UPDATE ON compliance_scores FROM patchiq_app;
