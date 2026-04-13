#!/usr/bin/env python3
"""Import CVE JSON files (NVD 2.0 format) into the Hub's cve_feeds table.

Usage:
    python3 scripts/import-cves.py docs/cves-files/cves_*.json

Connects to the Hub database via psql using PGHOST/PGPORT/PGUSER/PGPASSWORD
env vars (set by .env or manually).
"""

import json
import subprocess
import sys
import os
import re

BATCH_SIZE = 500

def extract_attack_vector(vector_string):
    av_map = {"N": "NETWORK", "A": "ADJACENT_NETWORK", "L": "LOCAL", "P": "PHYSICAL"}
    for part in vector_string.split("/"):
        if part.startswith("AV:"):
            return av_map.get(part[3:], "")
    return ""

def parse_cve(cve):
    cve_id = cve.get("id", "")
    if not cve_id:
        return None

    # Description — flat string array in this format
    desc = ""
    descriptions = cve.get("descriptions", [])
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

    # CVSS metrics
    metrics = cve.get("metrics", {})
    cvss_score = None
    cvss_vector = ""
    attack_vector = ""
    base_severity = ""

    for key in ["cvssMetricV31", "cvssMetricV40", "cvssMetricV2"]:
        metric_list = metrics.get(key, [])
        if not metric_list:
            continue
        # Prefer Primary type
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

    # Severity normalization
    severity_aliases = {"important": "high", "moderate": "medium", "negligible": "low", "informational": "none"}
    if base_severity in severity_aliases:
        base_severity = severity_aliases[base_severity]
    if base_severity not in ("critical", "high", "medium", "low", "none"):
        base_severity = "medium"  # fallback

    # CWE
    cwe_id = ""
    for w in cve.get("weaknesses", []):
        for d in w.get("description", []):
            lang = d.get("lang", "")
            val = d.get("value", "")
            if lang == "en" and val and val not in ("NVD-CWE-noinfo", "NVD-CWE-Other"):
                cwe_id = val
                break
        if cwe_id:
            break

    # References — can be flat strings or objects
    refs = []
    for r in cve.get("references", []):
        if isinstance(r, str):
            refs.append({"url": r, "source": "nvd"})
        elif isinstance(r, dict):
            refs.append({"url": r.get("url", ""), "source": "nvd"})

    # Published
    published = cve.get("published", "")
    last_modified = cve.get("lastModified", "")

    return {
        "cve_id": cve_id,
        "severity": base_severity,
        "description": desc,
        "published_at": published,
        "source": "nist",
        "cvss_v3_score": cvss_score,
        "cvss_v3_vector": cvss_vector,
        "attack_vector": attack_vector,
        "cwe_id": cwe_id,
        "external_references": json.dumps(refs),
        "nvd_last_modified": last_modified,
    }

def escape_sql(val):
    if val is None:
        return "NULL"
    s = str(val)
    return "'" + s.replace("'", "''").replace("\\", "\\\\") + "'"

def fmt_timestamp(val):
    if not val:
        return "NULL"
    # NVD timestamps may lack timezone — append UTC
    if not val.endswith("Z") and "+" not in val and not val.endswith(")"):
        val = val + "+00"
    return escape_sql(val) + "::timestamptz"

def generate_sql_batch(records):
    lines = []
    lines.append("""INSERT INTO cve_feeds (
    cve_id, severity, description, published_at, source,
    cvss_v3_score, cvss_v3_vector, attack_vector, cwe_id,
    external_references, nvd_last_modified, exploit_known, in_kev
) VALUES""")

    value_rows = []
    for r in records:
        cvss = str(r["cvss_v3_score"]) if r["cvss_v3_score"] is not None else "NULL"
        value_rows.append(f"""({escape_sql(r['cve_id'])}, {escape_sql(r['severity'])}, {escape_sql(r['description'])},
    {fmt_timestamp(r['published_at'])}, {escape_sql(r['source'])},
    {cvss}, {escape_sql(r['cvss_v3_vector'])}, {escape_sql(r['attack_vector'])}, {escape_sql(r['cwe_id'])},
    {escape_sql(r['external_references'])}::jsonb, {fmt_timestamp(r['nvd_last_modified'])},
    false, false)""")

    lines.append(",\n".join(value_rows))
    lines.append("""ON CONFLICT (cve_id) DO UPDATE SET
    severity           = COALESCE(NULLIF(EXCLUDED.severity, ''), cve_feeds.severity),
    description        = COALESCE(EXCLUDED.description, cve_feeds.description),
    published_at       = COALESCE(EXCLUDED.published_at, cve_feeds.published_at),
    source             = COALESCE(NULLIF(EXCLUDED.source, ''), cve_feeds.source),
    cvss_v3_score      = COALESCE(EXCLUDED.cvss_v3_score, cve_feeds.cvss_v3_score),
    cvss_v3_vector     = COALESCE(NULLIF(EXCLUDED.cvss_v3_vector, ''), cve_feeds.cvss_v3_vector),
    attack_vector      = COALESCE(NULLIF(EXCLUDED.attack_vector, ''), cve_feeds.attack_vector),
    cwe_id             = COALESCE(NULLIF(EXCLUDED.cwe_id, ''), cve_feeds.cwe_id),
    external_references = CASE
        WHEN EXCLUDED.external_references IS NOT NULL AND EXCLUDED.external_references != '[]'::jsonb
        THEN EXCLUDED.external_references
        ELSE cve_feeds.external_references
    END,
    nvd_last_modified  = COALESCE(EXCLUDED.nvd_last_modified, cve_feeds.nvd_last_modified),
    updated_at         = now();""")

    return "\n".join(lines)

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 scripts/import-cves.py <json-file> [json-file ...]", file=sys.stderr)
        sys.exit(1)

    db_name = os.environ.get("DB_NAME_HUB", "patchiq_hub_dev_production")
    pg_host = os.environ.get("PGHOST", "localhost")
    pg_port = os.environ.get("PGPORT", "5832")
    pg_user = os.environ.get("PGUSER", "patchiq")

    all_records = []
    for filepath in sys.argv[1:]:
        print(f"Reading {filepath}...", file=sys.stderr)
        with open(filepath) as f:
            data = json.load(f)

        if isinstance(data, dict) and "vulnerabilities" in data:
            cves = [v["cve"] for v in data["vulnerabilities"]]
        elif isinstance(data, list):
            cves = data
        else:
            print(f"Unknown format in {filepath}", file=sys.stderr)
            continue

        for cve in cves:
            rec = parse_cve(cve)
            if rec:
                all_records.append(rec)

    print(f"Parsed {len(all_records)} CVEs total", file=sys.stderr)

    if not all_records:
        print("No records to import", file=sys.stderr)
        sys.exit(1)

    # Process in batches via psql
    total_imported = 0
    for i in range(0, len(all_records), BATCH_SIZE):
        batch = all_records[i:i + BATCH_SIZE]
        sql = generate_sql_batch(batch)

        result = subprocess.run(
            ["psql", "-h", pg_host, "-p", pg_port, "-U", pg_user, "-d", db_name, "-v", "ON_ERROR_STOP=1"],
            input=sql,
            capture_output=True,
            text=True,
            env={**os.environ, "PGPASSWORD": os.environ.get("PGPASSWORD", "patchiq_dev")},
        )

        if result.returncode != 0:
            print(f"ERROR at batch {i // BATCH_SIZE + 1}: {result.stderr}", file=sys.stderr)
            # Print first failing SQL for debugging
            print(f"First CVE in batch: {batch[0]['cve_id']}", file=sys.stderr)
            sys.exit(1)

        total_imported += len(batch)
        print(f"  Imported {total_imported}/{len(all_records)} CVEs...", file=sys.stderr)

    print(f"Done. {total_imported} CVEs imported into {db_name}.", file=sys.stderr)

if __name__ == "__main__":
    main()
