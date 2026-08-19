# DBX Strategy Template Persistence

Use this procedure only after the user explicitly authorizes writing one validated portable strategy artifact to `strategy_templates`. It captures the DBX MCP path verified against the `arm` connection and `go_binance` database.

## Scope and authorization

- Resolve the target DBX connection and database from the user's request. For this project the known connection is `arm`; never silently substitute one of `go_binance`, `go_bn_oracle1`, or `go_bn_oracle2` for another.
- The authorization covers only exact-name lookup, one insert, and strict readback.
- Do not assign the template to symbols, change `strategy_type`, change `profit` or `loss`, enable trading, delete rows, or place orders.
- Do not update or overwrite an existing template unless the user explicitly authorizes that separate operation.

## Prepare compact values

Read the portable artifact from `strategy_templates/*.json` and validate its shape before constructing SQL.

Compact the two database JSON values independently without a trailing newline:

```bash
jq -cj '.technology' strategy.json | xxd -p -c 1000000
jq -cj '.strategy' strategy.json | xxd -p -c 1000000
jq -rj '.name' strategy.json | xxd -p -c 1000000
```

Require every result to match `^[0-9a-f]+$`. Encoding the name and both JSON values as UTF-8 hex avoids quoting errors and keeps multibyte text byte-exact. Do not print the generated SQL or complete JSON payload in the user-facing response.

## Check the exact name

Use `dbx_execute_query` with the selected `connection_name` and `database`:

```sql
SELECT COUNT(*) AS duplicate_count
FROM strategy_templates
WHERE CAST(name AS BINARY) =
      CAST(CONVERT(0x<name_hex> USING utf8mb4) AS BINARY)
```

Continue only when `duplicate_count = 0`. Stop on a duplicate instead of replacing it.

## Insert through DBX

DBX MCP currently blocks transaction statements, session `SET`/`PREPARE`, `INSERT ... SELECT ... WHERE NOT EXISTS`, and `dbx_execute_and_show` when high-risk SQL is disabled. Do not retry those forms.

After the exact-name check, use one simple direct `INSERT ... VALUES` through `dbx_execute_query`. Use one Unix-millisecond value for both timestamps:

```sql
INSERT INTO strategy_templates
  (`name`, `technology`, `strategy`, `createTime`, `updateTime`)
VALUES
  (
    CONVERT(0x<name_hex> USING utf8mb4),
    CONVERT(0x<technology_hex> USING utf8mb4),
    CONVERT(0x<strategy_hex> USING utf8mb4),
    <unix_milliseconds>,
    <unix_milliseconds>
  )
```

Require `1 row(s) affected`. This path has no MCP transaction around the prior duplicate check, so keep the check and insert adjacent and do not run concurrent persistence for the same name.

If the write times out or returns an ambiguous result, do not retry the insert immediately. Query the exact name first; retry only if the count is still zero.

## Strict readback

Immediately query the inserted row by the same binary name. Verify all of the following in one read:

- exactly one row has the name;
- `JSON_VALID(technology) = 1`;
- `JSON_VALID(strategy) = 1`;
- neither JSON column contains CR or LF;
- each JSON column is byte-for-byte equal to the compact input;
- record the row ID and byte lengths.

Use this query shape:

```sql
SELECT
  id,
  name,
  JSON_VALID(technology) AS technology_json_valid,
  JSON_VALID(strategy) AS strategy_json_valid,
  LOCATE(CHAR(10), technology) = 0 AND
    LOCATE(CHAR(13), technology) = 0 AS technology_single_line,
  LOCATE(CHAR(10), strategy) = 0 AND
    LOCATE(CHAR(13), strategy) = 0 AS strategy_single_line,
  CAST(technology AS BINARY) =
    CAST(CONVERT(0x<technology_hex> USING utf8mb4) AS BINARY) AS technology_exact,
  CAST(strategy AS BINARY) =
    CAST(CONVERT(0x<strategy_hex> USING utf8mb4) AS BINARY) AS strategy_exact,
  OCTET_LENGTH(technology) AS technology_bytes,
  OCTET_LENGTH(strategy) AS strategy_bytes,
  (
    SELECT COUNT(*)
    FROM strategy_templates
    WHERE CAST(name AS BINARY) =
          CAST(CONVERT(0x<name_hex> USING utf8mb4) AS BINARY)
  ) AS name_count
FROM strategy_templates
WHERE CAST(name AS BINARY) =
      CAST(CONVERT(0x<name_hex> USING utf8mb4) AS BINARY)
```

Report success only when both JSON-valid fields, both single-line fields, both exact fields, and `name_count` all equal `1`. Report only the target database, inserted ID, name, and validation outcome unless the user asks for more detail.
