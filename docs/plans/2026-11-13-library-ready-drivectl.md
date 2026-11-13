# Library-Ready `drivectl` (`pkg/drive`, `pkg/discovery`, & Embeddable `mcp`)

Make `github.com/ghchinoy/drivectl` directly importable as a clean, stateless Go library by external services (such as `cuj-suggester` running on Cloud Run / Corp Run) and embeddable in third-party MCP / Genkit hosts, while preserving full backward compatibility for the `drivectl` CLI.

## Goals & Adjustments

1. **Promote `internal/drive` $\rightarrow$ `pkg/drive` and `internal/discovery` $\rightarrow$ `pkg/discovery`:**
   - Move packages out of `internal/` so external Go modules can import `github.com/ghchinoy/drivectl/pkg/drive` and `github.com/ghchinoy/drivectl/pkg/discovery`.
   - Add `discovery.CallMethod(ctx, client, endpoint, payload)` and `(*RestDescription).CallMethod(...)`.
   - Fix `BuildRequest` / `prepareQueryOrBody` so `location == "query"` parameters (from both `method.Parameters` and `doc.Parameters`) are extracted onto the URL query string before serializing the JSON request body on `POST`/`PUT`/`PATCH` methods (`method.Request != nil`).
2. **Add Stateless Token-Source, Bearer-Token & Service Constructors in `pkg/drive/auth.go`:**
   - Add `Services` struct (`Client`, `Drive`, `Docs`, `Sheets`, `Slides`).
   - Add `NewClientFromTokenSource`, `NewClientFromBearerToken`, `NewDriveService`, `NewDocsService`, `NewSheetsService`, `NewSlidesService`, and `NewServicesFromClient` with zero disk I/O and zero stdout output.
3. **Use `encoding/csv.Writer` in `pkg/drive/sheets.go`:**
   - Replace `strings.Join(csvRow, ",")` in `FetchSheetAsCSV` with `encoding/csv.NewWriter` for proper RFC 4180 escaping of commas, quotes, and multi-line cells.
4. **Add First-Class Google Slides Helpers (`pkg/drive/slides.go`):**
   - Add `FetchPresentation`, `FetchPresentationAsMarkdown`, and `RenderPresentationAsMarkdown` using `google.golang.org/api/slides/v1`.
   - Add `"slides.v1.presentations.get"` to `dynamicTools` in `mcp/server.go`.
5. **Make `mcp/server.go` Embeddable (`RegisterTools`):**
   - Export `ClientProvider`, `RegisterOption`, `WithDiscoveryHTTPClient`, and `RegisterTools(ctx, s, clientProvider, opts...)` so external MCP servers or Genkit hosts can mount `drivectl`'s tools in-process without package globals or local log files.
