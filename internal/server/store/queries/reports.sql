-- name: CreateReportGeneration :one
INSERT INTO report_generations (id, tenant_id, report_type, format, name, filters, created_by, expires_at)
VALUES (@id, @tenant_id, @report_type, @format, @name, @filters::jsonb, @created_by, @expires_at)
RETURNING *;

-- name: UpdateReportStatus :exec
UPDATE report_generations
SET status = @status,
    file_path = @file_path,
    file_size_bytes = @file_size_bytes,
    checksum_sha256 = @checksum_sha256,
    row_count = @row_count,
    error_message = @error_message,
    completed_at = @completed_at
WHERE id = @id AND tenant_id = @tenant_id;

-- name: GetReportGeneration :one
SELECT * FROM report_generations
WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListReportGenerations :many
SELECT * FROM report_generations
WHERE tenant_id = @tenant_id
  AND (@status::text = '' OR status = @status)
  AND (@report_type::text = '' OR report_type = @report_type)
  AND (@format::text = '' OR format = @format)
  AND (
    @cursor_created_at::timestamptz IS NULL
    OR (created_at, id) < (@cursor_created_at, @cursor_id::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: CountReportGenerations :one
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE status = 'completed') AS completed,
    count(*) FILTER (WHERE status = 'generating') AS generating,
    count(*) FILTER (WHERE status = 'failed') AS failed,
    count(*) FILTER (WHERE status = 'pending') AS pending
FROM report_generations
WHERE tenant_id = @tenant_id
  AND (@status::text = '' OR status = @status)
  AND (@report_type::text = '' OR report_type = @report_type)
  AND (@format::text = '' OR format = @format);

-- name: CountReportGenerationsToday :one
SELECT count(*) FROM report_generations
WHERE tenant_id = @tenant_id
  AND created_at >= CURRENT_DATE;

-- name: DeleteReportGeneration :exec
DELETE FROM report_generations
WHERE id = @id AND tenant_id = @tenant_id;

-- name: DeleteExpiredReports :many
DELETE FROM report_generations
WHERE tenant_id = @tenant_id
  AND expires_at < now()
  AND status = 'completed'
RETURNING file_path;
