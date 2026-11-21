# Offline interactive report viewer

GrantDiff `1.1.0-preview.1` includes an English/Chinese viewer for schema 1.0 diff reports. The executable writes one HTML file containing the interface, styles, report schema and optional report data. Open the file directly in a browser. All report loading and interaction happen locally.

## Open a report

From an extracted archive, use `./grantdiff` on Linux or `./grantdiff.exe` on Windows. From the source checkout, use `go run ./cmd/grantdiff`.

```text
grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --compact --format json --output report.json
grantdiff view --input report.json --output review.html
```

Open `review.html` to explore the report. `grantdiff view --output review.html` creates an empty viewer with a file picker. The Open JSON report button can switch to another report, including a JSON file saved by an earlier GrantDiff version using schema 1.0. UTF-8 BOM is accepted.

Release archives also contain `viewer.html`, ready to open and select a report.

The output filename uses `.html`. An existing output can be replaced, and the input and output must refer to separate files. The command returns 0 when it generates the page and 2 for invalid arguments, an unsupported report envelope or a file error. The browser validates the full schema and range structure before showing report content. It displays the report's calculation status, scope and risk findings. Run `diff` with the intended `--fail-on` condition when using the result as a CI gate.

## Explore the evidence

- Choose added, removed, retained, path-changed or indeterminate ranges in the sidebar. Counts describe request ranges.
- Combine the identity, risk and text filters to focus the list. Search covers report subjects, request selectors, risk messages and authorization paths.
- Enter a namespace to match the complete namespace selector, including complement exclusions. Empty namespace requests include the cluster and cross-namespace requests represented by the report. URL ranges appear with namespace filtering cleared.
- Select a row to compare before/after decisions and inspect each dimension of the full request range. A representative request is available as a separate example.
- Expand authorization paths to follow the matched subject, binding, aggregated roles and source role. Binding and rule filenames, document numbers and line numbers remain visible as evidence.
- Review the input scope and diagnostics. An incomplete calculation, identity set or policy set receives a persistent notice, including when no changed ranges were recorded.

Lists paginate at 40 changes. Evidence paths are displayed in batches of 20. Language can be switched without losing the active filters. Keyboard controls use standard buttons, inputs and disclosure elements, and the layout adapts to narrow viewports.

## Report handling

The generated page works offline with embedded scripts and styles. A content security policy restricts execution to the embedded content and blocks network connections. Report strings are rendered as text. Imported reports must satisfy the embedded schema 1.0 contract and supported range structure. An invalid selected file produces an error while the previously loaded report remains identified in the heading.

An HTML file generated with `--input` contains the complete supplied report, including identities, source paths and diagnostics. Sharing that HTML shares those details. A newly selected report and the active filters live in browser memory. Reopening the HTML restores its embedded report or the empty viewer.

The JSON input limit is 128 MiB. The preview has automated offline Chromium checks on the packaged executable, including a synthetic 5,000-range report, invalid input, literal HTML-sensitive strings, pagination, source evidence and mobile layout. Larger reports depend on browser memory and device capacity. Browser availability and rendering support are recorded in the validation record.

[Compatibility](compatibility.md) · [Validation](validation.md)
