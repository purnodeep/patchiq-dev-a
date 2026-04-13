#!/usr/bin/env python3
"""Import CVE JSON files into the Server's cves table (tenant-scoped).

Usage:
    python3 scripts/import-cves-server.py docs/cves-files/cves_*.json
"""

import json
import subprocess
import sys
import os

BATCH_SIZE = 500
TENANT_ID = "00000000-0000-0000-0000-000000000001"

# Server uses title-case attack vectors (CHECK constraint)
AV_MAP = {"N": "Network", "A": "Adjacent", "L": "Local", "P": "Physical"}

def extract_attack_vector(vector_string):
    for part in vector_string.split("/"):
        if part.startswith("AV:"):
            return AV_MAP.get(part[3:], None)
    return None

def parse_cve(cve):
    cve_id = cve.get("id", "")
    if not cve_id:
        return None

    descriptions = cve.get("descriptions", [])
    desc = ""
    if descriptions:
        if isinstance(descriptions[0], str):
            desc = descriptions[0]
        elif isinstance(descriptions[0], dict):
            for d in descriptions:
                if d.get("lang") == "en":
                    desc = d.get("value", "")
                    break
            if not desc:
                desc = descriptions[0].get("value", "")

    metrics = cve.get("metrics", {})
    cvss_score = None
    cvss_vector = ""
    attack_vector = None
    base_severity = ""

    for key in ["cvssMetricV31", "cvssMetricV40", "cvssMetricV2"]:
        metric_list = metrics.get(key, [])
        if not metric_list:
            continue
        chosen = metric_list[0]
        for m in metric_list:
            if m.get("type") == "Primary":
                chosen = m
                break
        cvss_data = chosen.get("cvssData", {})
        cvss_score = cvss_data.get("baseScore")
        cvss_vector = cvss_data.get("vectorString", "")
        base_severity = cvss_data.get("baseSeverity", "").lower()
        attack_vector = extract_attack_vector(cvss_vector)
        break

    severity_aliases = {"important": "high", "moderate": "medium", "negligible": "low", "informational": "none"}
    if base_severity in severity_aliases:
        base_severity = severity_aliases[base_severity]
    if base_severity not in ("critical", "high", "medium", "low", "none"):
        base_severity = "medium"

    cwe_id = ""
    for w in cve.get("weaknesses", []):
        for d in w.get("description", []):
            if d.get("lang") == "en" and d.get("value", "") not in ("", "NVD-CWE-noinfo", "NVD-CWE-Other"):
                cwe_id = d["value"]
                break
        if cwe_id:
            break

    refs = []
    for r in cve.get("references", []):
        if isinstance(r, str):
            refs.append({"url": r, "source": "nvd"})
        elif isinstance(r, dict):
            refs.append({"url": r.get("url", ""), "source": "nvd"})

    return {
        "cve_id": cve_id,
        "severity": base_severity,
        "description": desc,
        "published_at": cve.get("published", ""),
        "cvss_v3_score": cvss_score,
        "cvss_v3_vector": cvss_vector,
        "attack_vector": attack_vector,
        "cwe_id": cwe_id,
        "external_references": json.dumps(refs),
        "nvd_last_modified": cve.get("lastModified", ""),
    }

def escape_sql(val):
    if val is None:
        return "NULL"
    s = str(val)
    return "'" + s.replace("'", "''").replace("\\", "\\\\") + "'"

def fmt_timestamp(val):
    if not val:
        return "NULL"
    if not val.endswith("Z") and "+" not in val:
        val = val + "+00"
    return escape_sql(val) + "::timestamptz"

def generate_sql_batch(records):
    lines = []
    # Set tenant for RLS
    lines.append(f"BEGIN;")
    lines.append(f"SET LOCAL app.current_tenant_id = '{TENANT_ID}';")
    lines.append("""INSERT INTO cves (
    tenant_id, cve_id, severity, description, published_at,
    cvss_v3_score, cvss_v3_vector, cisa_kev_due_date, exploit_available,
    nvd_last_modified, attack_vector, external_references, cwe_id, source
) VALUES""")

    value_rows = []
    for r in records:
        cvss = str(r["cvss_v3_score"]) if r["cvss_v3_score"] is not None else "NULL"
        av = escape_sql(r["attack_vector"]) if r["attack_vector"] else "NULL"
        value_rows.append(f"""('{TENANT_ID}', {escape_sql(r['cve_id'])}, {escape_sql(r['severity'])},
    {escape_sql(r['description'])}, {fmt_timestamp(r['published_at'])},
    {cvss}, {escape_sql(r['cvss_v3_vector'])}, NULL, false,
    {fmt_timestamp(r['nvd_last_modified'])}, {av},
    {escape_sql(r['external_references'])}::jsonb, {escape_sql(r['cwe_id'])}, 'NVD')""")

    lines.append(",\n".join(value_rows))
    lines.append("""ON CONFLICT (tenant_id, cve_id)
DO UPDATE SET
    severity = EXCLUDED.severity,
    description = EXCLUDED.description,
    published_at = EXCLUDED.published_at,
    cvss_v3_score = EXCLUDED.cvss_v3_score,
    cvss_v3_vector = EXCLUDED.cvss_v3_vector,
    exploit_available = EXCLUDED.exploit_available,
    nvd_last_modified = EXCLUDED.nvd_last_modified,
    attack_vector = EXCLUDED.attack_vector,
    external_references = EXCLUDED.external_references,
    cwe_id = EXCLUDED.cwe_id,
    source = EXCLUDED.source,
    updated_at = now();""")
    lines.append("COMMIT;")
    return "\n".join(lines)

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/import-cves-server.py <json-file> [json-file ...]", file=sys.stderr)
        sys.exit(1)

    db_name = os.environ.get("DB_NAME_SERVER", "patchiq_dev_production")
    pg_host = os.environ.get("PGHOST", "localhost")
    pg_port = os.environ.get("PGPORT", "5832")
    pg_user = os.environ.get("PGUSER", "patchiq")

    all_records = []
    for filepath in sys.argv[1:]:
        print(f"Reading {filepath}...", file=sys.stderr)
        with open(filepath) as f:
            data = json.load(f)
        cves = data if isinstance(data, list) else [v["cve"] for v in data.get("vulnerabilities", [])]
        for cve in cves:
            rec = parse_cve(cve)
            if rec:
                all_records.append(rec)

    print(f"Parsed {len(all_records)} CVEs total", file=sys.stderr)

    total_imported = 0
    for i in range(0, len(all_records), BATCH_SIZE):
        batch = all_records[i:i + BATCH_SIZE]
        sql = generate_sql_batch(batch)

        result = subprocess.run(
            ["psql", "-h", pg_host, "-p", pg_port, "-U", pg_user, "-d", db_name, "-v", "ON_ERROR_STOP=1"],
            input=sql, capture_output=True, text=True,
            env={**os.environ, "PGPASSWORD": os.environ.get("PGPASSWORD", "patchiq")},
        )

        if result.returncode != 0:
            print(f"ERROR at batch {i // BATCH_SIZE + 1}: {result.stderr}", file=sys.stderr)
            print(f"First CVE in batch: {batch[0]['cve_id']}", file=sys.stderr)
            sys.exit(1)

        total_imported += len(batch)
        print(f"  Imported {total_imported}/{len(all_records)} CVEs...", file=sys.stderr)

    print(f"Done. {total_imported} CVEs imported into {db_name}.", file=sys.stderr)

if __name__ == "__main__":
    main()
