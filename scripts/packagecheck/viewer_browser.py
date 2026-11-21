"""Exercise the offline report viewer in Chromium using the packaged CLI."""

import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import time

from playwright.sync_api import expect, sync_playwright


def verify(executable, root, screenshot):
    with tempfile.TemporaryDirectory(prefix="grantdiff viewer ") as temporary:
        work = Path(temporary)

        def run(*arguments, code=0):
            result = subprocess.run(
                [str(executable), *map(str, arguments)],
                cwd=root,
                capture_output=True,
                text=True,
                encoding="utf-8",
                timeout=60,
            )
            if result.returncode != code:
                raise RuntimeError(f"{arguments}: {result.returncode}\n{result.stderr}")
            return result.stdout

        def diff(example, *options, code=0):
            directory = f"examples/{example}"
            return run(
                "diff",
                "--before",
                f"{directory}/before",
                "--after",
                f"{directory}/after",
                "--format",
                "json",
                *options,
                code=code,
            )

        external_args = ("--catalog", "examples/external-dns/catalog.yaml", "--compact")
        external = diff("external-dns", *external_args)
        retained = diff(
            "retained",
            "--catalog",
            "examples/retained/catalog.yaml",
            "--identities",
            "examples/retained/identities.yaml",
        )
        incomplete = diff("external-dns", *external_args, "--max-classes", "1", code=3)
        narrowing = diff(
            "kube-state-metrics",
            "--catalog",
            "examples/kube-state-metrics/catalog.json",
            "--compact",
        )
        report = work / "权限报告.json"
        report.write_text(external, encoding="utf-8")
        page_file = work / "review.html"
        blank_file = work / "blank.html"
        run("view", "--input", report, "--output", page_file)
        run("view", "--output", blank_file)

        escaped = json.loads(external)
        markup = '</script><img src="https://example.test/probe" onerror="window.reportExecuted=true"> & evidence'
        escaped["changes"][0]["subject"] = markup
        escaped["changes"][0]["afterEvidence"][0]["ruleSource"]["file"] = markup
        escaped_input = work / "escaped.json"
        escaped_input.write_text(json.dumps(escaped), encoding="utf-8")
        escaped_page = work / "escaped.html"
        run("view", "--input", escaped_input, "--output", escaped_page)

        with sync_playwright() as playwright:
            browser = playwright.chromium.launch()
            context = browser.new_context(
                locale="en-US", viewport={"width": 1440, "height": 1050}, offline=True
            )
            page = context.new_page()
            errors, network = [], []
            console_errors = []
            page.on("pageerror", lambda error: errors.append(str(error)))
            page.on(
                "console",
                lambda message: (
                    console_errors.append(message.text)
                    if message.type == "error"
                    else None
                ),
            )
            page.on(
                "request",
                lambda request: (
                    network.append(request.url)
                    if request.url.startswith(("http:", "https:"))
                    else None
                ),
            )
            page.goto(page_file.as_uri())
            rows = page.locator(".change-row")
            expect(rows).to_have_count(3)
            page.locator("#risk").select_option("high")
            expect(rows).to_have_count(3)
            page.locator("#identity").select_option(index=1)
            expect(rows).to_have_count(3)
            page.locator("#namespace").fill("finance")
            expect(rows).to_have_count(3)
            page.locator("#cluster-scope").check()
            expect(rows).to_have_count(2)
            page.locator("#reset").click()
            page.locator("#search").fill("get")
            expect(rows).to_have_count(1)
            expect(page.locator("#detail .decision").first).to_contain_text(
                "Not granted"
            )
            expect(page.locator("#detail .decision").last).to_contain_text("Granted")
            expect(page.locator("#detail .path[open]")).to_contain_text(
                "ClusterRole//external-dns"
            )
            expect(page.locator("#detail .source").last).to_contain_text(
                "external-dns-clusterrole.yaml"
            )
            page.locator("#detail .path summary").click()
            expect(page.locator("#detail .path[open]")).to_have_count(0)
            page.locator("#detail .path summary").click()
            page.locator("#detail .example summary").click()
            expect(page.locator("#detail .example pre")).to_be_visible()
            page.locator("#reset").click()
            page.locator('[data-type="removed"]').click()
            expect(rows).to_have_count(0)
            expect(page.locator("#rows")).to_contain_text("No matching changes")
            page.locator("#reset").click()
            page.locator("#language").click()
            expect(page.locator("html")).to_have_attribute("lang", "zh-CN")
            expect(page.locator("#detail")).to_contain_text("已授予")
            page.locator("#reset").click()
            if screenshot:
                page.screenshot(path=screenshot, full_page=True)

            page.goto(blank_file.as_uri())
            expect(page.locator("#welcome")).to_be_visible()

            def select(name, data):
                page.locator("#file").set_input_files(
                    {
                        "name": name,
                        "mimeType": "application/json",
                        "buffer": data.encode("utf-8"),
                    }
                )

            select("retained.json", retained)
            expect(rows).to_have_count(1)
            expect(page.locator("#detail")).to_contain_text(
                "RoleBinding/finance/group-read"
            )
            expect(page.locator("#detail .decision").first).to_contain_text("Granted")
            expect(page.locator("#detail .decision").last).to_contain_text("Granted")
            select("invalid.json", '{"schemaVersion":"9.0"}')
            expect(page.get_by_role("alert")).to_be_visible()
            expect(rows).to_have_count(1)
            select("incomplete.json", incomplete)
            expect(page.get_by_role("alert")).to_be_hidden()
            expect(page.locator("#incomplete")).to_be_visible()
            expect(page.locator("#scope-panel")).to_contain_text("budget_exceeded")

            select("narrowing.json", narrowing)
            page.locator('[data-type="removed"]').click()
            expect(rows).to_have_count(2)
            expect(page.locator("#detail .decision").last).to_contain_text(
                "Not granted"
            )
            page.locator("#risk").select_option("high")
            expect(rows).to_have_count(0)
            select(
                "no-changes.json", json.dumps(dict(json.loads(external), changes=[]))
            )
            expect(page.locator("#rows")).to_contain_text("No permission changes")
            uncertain = json.loads(external)
            uncertain["changes"] = []
            uncertain["scope"]["identityCompleteness"] = "incomplete"
            select("uncertain.json", json.dumps(uncertain))
            expect(page.locator("#incomplete")).to_be_visible()
            expect(page.locator("#rows")).to_contain_text("No changes recorded yet")

            # Exercise the exact, complement and URL selectors with different
            # witnesses so filtering must use the range rather than its example.
            selected_report = json.loads(external)
            item = selected_report["changes"][0]
            item["range"]["namespace"] = {"except": ["", "finance"]}
            item["representativeRequest"]["namespace"] = "grantdiff-other"
            selected_report["changes"] = [item]
            select("excluded.json", json.dumps(selected_report))
            page.locator("#namespace").fill("finance")
            expect(rows).to_have_count(0)
            page.locator("#namespace").fill("apps")
            expect(rows).to_have_count(1)
            url_report = json.loads(external)
            url_change = url_report["changes"][0]
            url_change["range"] = {
                "verb": {"value": "get"},
                "url": {
                    "prefix": "/healthz/",
                    "except": ["/healthz/live"],
                    "exceptPrefixes": ["/healthz/private/"],
                },
            }
            url_change["representativeRequest"] = {
                "verb": "get",
                "apiGroup": "",
                "url": "/healthz/ready",
            }
            url_change["risk"] = []
            url_report["changes"] = [url_change]
            select("url.json", json.dumps(url_report))
            expect(page.locator("#detail .range-table")).to_contain_text(
                "/healthz/private/"
            )
            page.locator("#cluster-scope").check()
            expect(rows).to_have_count(0)

            invalid = json.loads(external)
            invalid["changes"][0]["range"].pop("name")
            select("invalid-range.json", json.dumps(invalid))
            expect(page.get_by_role("alert")).to_be_visible()
            invalid = json.loads(external)
            invalid["changes"][0]["risk"][0]["severity"] = "invented"
            select("invalid-risk.json", json.dumps(invalid))
            expect(page.get_by_role("alert")).to_contain_text("unsupported value")

            page.goto(escaped_page.as_uri())
            expect(rows.first).to_contain_text(markup)
            expect(page.locator("#detail img")).to_have_count(0)
            assert page.evaluate("window.reportExecuted") is None
            select("bom.json", "\ufeff" + retained)
            expect(rows).to_have_count(1)
            expect(page.locator("#report-meta")).to_contain_text("bom.json")

            many_paths = json.loads(external)
            evidence = many_paths["changes"][0]["afterEvidence"][0]
            evidence["aggregationPath"] = [
                "ClusterRole//aggregate",
                "ClusterRole//nested",
            ]
            many_paths["changes"][0]["afterEvidence"] = [
                dict(evidence, binding=f"ClusterRoleBinding//path-{index}")
                for index in range(25)
            ]
            select("paths.json", json.dumps(many_paths))
            expect(page.locator("#detail .path")).to_have_count(20)
            expect(page.locator("#detail .path").first).to_contain_text(
                "ClusterRole//nested"
            )
            page.get_by_role("button", name="Show more paths").click()
            expect(page.locator("#detail .path")).to_have_count(25)

            large = json.loads(external)
            original = large["changes"][0]
            large["changes"] = [
                dict(original, id=f"{index:020x}", subject=f"reviewer-{index}")
                for index in range(81)
            ]
            markup = "<b>reviewed input</b> & evidence"
            large["changes"][0]["subject"] = markup
            select("many.json", json.dumps(large))
            expect(rows).to_have_count(40)
            expect(rows.first).to_contain_text(markup)
            expect(rows.first.locator("b")).to_have_count(0)
            page.locator("#next").click()
            expect(page.locator("#page-count")).to_have_text("2 / 3")
            expect(page.locator("#detail")).to_contain_text("reviewer-40")
            page.locator("#next").click()
            expect(rows).to_have_count(1)
            expect(page.locator("#next")).to_be_disabled()
            page.locator("#previous").click()
            expect(rows).to_have_count(40)
            page.locator("#language").click()
            expect(page.locator("#page-count")).to_have_text("2 / 3")
            page.locator("#search").fill("reviewer-80")
            expect(rows).to_have_count(1)
            expect(page.locator("#detail")).to_contain_text("reviewer-80")
            page.locator("#reset").click()
            page.set_viewport_size({"width": 390, "height": 844})
            assert page.evaluate(
                "document.documentElement.scrollWidth <= innerWidth"
            ), "mobile layout overflows"
            rows.first.focus()
            page.keyboard.press("Enter")
            expect(page.locator("#detail")).to_contain_text(markup)
            if screenshot:
                select("external-dns.json", external)
                expect(rows).to_have_count(3)
                page.evaluate("window.scrollTo(0, 0)")
                assert page.evaluate(
                    "document.documentElement.scrollWidth <= innerWidth"
                ), "evidence layout overflows"
                page.screenshot(
                    path=screenshot.with_name(screenshot.stem + "-mobile.png"),
                    full_page=True,
                )

            large["changes"] = [
                dict(original, id=f"{index:020x}", subject=f"bulk-{index}")
                for index in range(5000)
            ]
            started = time.monotonic()
            select("bulk.json", json.dumps(large))
            expect(page.locator("#report-meta")).to_contain_text(
                "bulk.json", timeout=30000
            )
            expect(rows).to_have_count(40)
            page.locator("#search").fill("bulk-4999")
            expect(rows).to_have_count(1)
            expect(page.locator("#detail")).to_contain_text("bulk-4999")
            bulk_elapsed = round((time.monotonic() - started) * 1000)
            assert not errors, errors
            assert not network, network
            assert not console_errors, console_errors
            browser_version = browser.version
            browser.close()
    print("Offline viewer passed Chromium interaction checks")
    return dict(
        browser="Chromium",
        browserVersion=browser_version,
        result="passed",
        offline=True,
        bulkRanges=5000,
        bulkLoadAndFilterMilliseconds=bulk_elapsed,
        checks=[
            "embedded and selected reports",
            "identity/type/risk/search filters",
            "namespace complements",
            "URL prefix exclusions",
            "before/after evidence",
            "path expansion",
            "source locations",
            "incomplete and empty reports",
            "invalid schema rejection",
            "literal HTML-sensitive data",
            "language switching",
            "pagination",
            "keyboard and mobile layout",
            "5000-range report",
            "no network requests or console errors",
        ],
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--executable", required=True, type=Path)
    parser.add_argument("--root", type=Path, default=Path.cwd())
    parser.add_argument("--screenshot", type=Path)
    parser.add_argument("--result", type=Path)
    args = parser.parse_args()
    result = verify(args.executable.resolve(), args.root.resolve(), args.screenshot)
    if args.result:
        args.result.parent.mkdir(parents=True, exist_ok=True)
        args.result.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
