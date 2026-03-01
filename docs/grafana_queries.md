# Grafana Queries

This document contains useful LogQL queries for analyzing the `balance-tracker-web` logs in Grafana.

## Application Logs (JSON + Nginx Access Logs)

The backend applications output structured JSON via `log/slog`. The frontend logs are Nginx access logs formatted as JSON. Since LogQL `line_format` cannot iterate dynamically over all keys, we explicitly check and print known attributes for each log type.

### Combined Frontend / Backend Query

This query parses both the frontend (nginx) and backend (Go) logs, unpacking the JSON payload and conditionally formatting it into a readable single-line format.

```logql
{container=~"balance-tracker-.*"} 
  |~ "\\{" 
  | regexp "(?s)^[^{]*(?P<json_payload>\\{.*)"
  | line_format "{{.json_payload}}"
  | json
  | line_format "{{if .uri}}[FRONTEND] {{.method}} {{.uri}} | Status: {{.status}} | {{.body_bytes_sent}}B | Time: {{.request_time}}s | IP: {{.remote_addr}}{{if .referer}} | Ref: {{.referer}}{{end}} | UA: {{.user_agent}}{{else}}[BACKEND]  {{if .method}}{{.method}} {{.path}} | Status: {{.status}} | Time: {{.duration_ms}}ms | IP: {{.remote_addr}}{{else}}{{.msg}}{{if .error}} | error={{.error}}{{end}}{{if .card}} | card={{.card}}{{end}}{{if .cards_processed}} | cards={{.cards_processed}}{{end}}{{if .transactions_processed}} | txns={{.transactions_processed}}{{end}}{{if .addr}} | addr={{.addr}}{{end}}{{if .timezone}} | tz={{.timezone}}{{end}}{{if .date}} | date={{.date}}{{end}}{{if .due_date}} | due={{.due_date}}{{end}}{{if .amount}} | amount={{.amount}}{{end}}{{if .next_run}} | next_run={{.next_run}}{{end}}{{end}}{{end}}"
```

### Breakdown of the Query Formatting

**Frontend Logs Display Format:**

```
[FRONTEND] METHOD /uri | Status: 200 | 1024B | Time: 0.123s | IP: 127.0.0.1 | Ref: https://source... | UA: Browser_String
```

**Backend HTTP Access Logs Display Format:**

```
[BACKEND]  METHOD /path | Status: 200 | Time: 15ms | IP: 127.0.0.1
```

**Backend General Logs Display Format:**

```
[BACKEND]  Message text | key=value | key=value
```

**Note:** If you add new structured attributes to backend `slog` calls in the Go code, you should update the innermost `{{else}}` branch in the query to include a check for the new attribute: `{{if .new_attr_name}} | new_attr_name={{.new_attr_name}}{{end}}`.
