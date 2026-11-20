"""Verify extracted release archives and compare native platform results."""

import argparse
import hashlib
import json
from pathlib import Path
import platform
import re
import stat
import subprocess
import tarfile
import tempfile
import zipfile

from jsonschema import Draft202012Validator


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def extract(archive_path, destination):
    destination = destination.resolve()
    if archive_path.suffix == ".zip":
        with zipfile.ZipFile(archive_path) as archive:
            for entry in archive.infolist():
                target = (destination / entry.filename).resolve()
                require(
                    target.is_relative_to(destination),
                    "archive path escapes destination",
                )
                require(
                    not stat.S_ISLNK(entry.external_attr >> 16),
                    "archive contains a symlink",
                )
            archive.extractall(destination)
    else:
        with tarfile.open(archive_path) as archive:
            archive.extractall(destination, filter="data")


def check_package(dist, version, result_path):
    system = platform.system().lower()
    require(system in ("windows", "linux"), "package checks support Windows and Linux")
    expected_archives = {
        f"grantdiff_{version}_windows_amd64.zip",
        f"grantdiff_{version}_linux_amd64.tar.gz",
    }
    checksums = {}
    for line in (dist / "SHA256SUMS").read_text(encoding="utf-8").splitlines():
        digest, name = line.split()
        require(
            name in expected_archives and name not in checksums,
            "unexpected checksum entry",
        )
        require(
            hashlib.sha256((dist / name).read_bytes()).hexdigest() == digest,
            f"checksum mismatch for {name}",
        )
        checksums[name] = digest
    require(set(checksums) == expected_archives, "missing release checksum")
    extension = ".zip" if system == "windows" else ".tar.gz"
    archive_path = dist / f"grantdiff_{version}_{system}_amd64{extension}"
    hashes = {}
    with tempfile.TemporaryDirectory(prefix="grantdiff package ") as temporary:
        # Exercise Unicode and spaces in the executable and input paths.
        root = Path(temporary) / "发布 验证"
        root.mkdir()
        extract(archive_path, root)
        executable = root / ("grantdiff.exe" if system == "windows" else "grantdiff")
        validator = Draft202012Validator(
            json.loads((root / "docs/report.schema.json").read_text(encoding="utf-8"))
        )

        def run(arguments, expected=0):
            response = subprocess.run(
                [str(executable), *arguments], cwd=root, capture_output=True, timeout=60
            )
            require(
                response.returncode == expected,
                f"{arguments} returned {response.returncode}, expected {expected}\n{response.stderr.decode('utf-8', errors='replace')}",
            )
            return response.stdout

        require(
            run(["version"]).decode().strip() == version, "package version mismatch"
        )
        for command in ("diff", "query", "catalog", "snapshot"):
            run([command, "--help"])
        scenarios = {
            "retained": [
                "--identities",
                "examples/retained/identities.yaml",
                "--catalog",
                "examples/retained/catalog.yaml",
            ],
            "crd": [],
            "external-dns": ["--catalog", "examples/external-dns/catalog.yaml"],
            "kube-state-metrics": [
                "--catalog",
                "examples/kube-state-metrics/catalog.json",
            ],
        }
        for scenario, options in scenarios.items():
            base = [
                "diff",
                "--before",
                f"examples/{scenario}/before",
                "--after",
                f"examples/{scenario}/after",
                *options,
            ]
            for compact in (False, True):
                extra = ["--compact"] if compact else []
                for report_format in ("json", "markdown", "text"):
                    output = run([*base, *extra, "--format", report_format])
                    key = f"{scenario}/{compact}/{report_format}"
                    hashes[key] = hashlib.sha256(output).hexdigest()
                    if report_format == "json":
                        report = json.loads(output)
                        validator.validate(report)
                        require(
                            report["scope"]["calculation"] == "complete",
                            f"incomplete {key}",
                        )
                        if scenario == "external-dns":
                            require(
                                len(report["changes"]) == (3 if compact else 9),
                                "wrong Secret addition ranges",
                            )
                        if scenario == "kube-state-metrics":
                            removed = [
                                change
                                for change in report["changes"]
                                if change["type"] == "removed"
                            ]
                            require(
                                removed
                                and all(
                                    change["representativeRequest"]["resource"]
                                    == "secrets"
                                    for change in removed
                                ),
                                "wrong Secret revocation",
                            )
                    if report_format == "markdown":
                        expected_file = (
                            root
                            / f"examples/{scenario}/expected{'-compact' if compact else ''}.md"
                        )
                        if expected_file.exists():
                            require(
                                output
                                == expected_file.read_bytes().replace(b"\r\n", b"\n"),
                                f"expected report differs for {key}",
                            )
                if scenario in ("external-dns", "kube-state-metrics"):
                    failure = "high" if scenario == "external-dns" else "removed"
                    summary = run(
                        [
                            *base,
                            *extra,
                            "--summary",
                            "--format",
                            "markdown",
                            "--fail-on",
                            failure,
                        ],
                        1,
                    )
                    require(
                        b"<details>" not in summary, "summary contains expanded ranges"
                    )
                    hashes[f"{scenario}/{compact}/summary"] = hashlib.sha256(
                        summary
                    ).hexdigest()
            limited = json.loads(
                run([*base, "--compact", "--max-classes", "1", "--format", "json"], 3)
            )
            validator.validate(limited)
            require(
                any(
                    item["code"] == "budget_exceeded" for item in limited["diagnostics"]
                ),
                "missing budget diagnostic",
            )
        catalog = run(["catalog", "--input", "examples/crd/after"])
        require(
            json.loads(catalog)["resources"][0]["name"] == "widgets",
            "CRD catalog differs",
        )
        hashes["catalog"] = hashlib.sha256(catalog).hexdigest()
        query = [
            "query",
            "--input",
            "examples/retained/after",
            "--identities",
            "examples/retained/identities.yaml",
            "--catalog",
            "examples/retained/catalog.yaml",
            "--identity",
            "report-bot",
            "--verb",
            "get",
            "--resource",
            "configmaps",
            "--namespace",
            "finance",
            "--format",
            "json",
        ]
        for name, expected in (("billing-config", "granted"), ("other", "not_granted")):
            require(
                json.loads(run([*query, "--name", name]))["decision"]["result"]
                == expected,
                "wrong query decision",
            )
        invalid = json.loads(run(query, 3))
        require(
            invalid["decision"]["result"] == "unknown", "invalid query must be unknown"
        )
        run(["diff", "--format", "invalid"], 2)
        (root / "invalid-kubeconfig").write_text("invalid: [", encoding="utf-8")
        run(
            [
                "snapshot",
                "--context",
                "invalid",
                "--kubeconfig",
                "invalid-kubeconfig",
                "--output",
                "snapshot",
            ],
            4,
        )
        encoded = root / "含空格 输入.yaml"
        text = (root / "examples/retained/after/rbac.yaml").read_text(encoding="utf-8")
        encoded.write_bytes(
            b"\xef\xbb\xbf" + text.replace("\n", "\r\n").encode("utf-8")
        )
        encoded.chmod(stat.S_IRUSR | stat.S_IRGRP | stat.S_IROTH)
        try:
            result = json.loads(
                run(
                    [
                        "diff",
                        "--before",
                        "examples/retained/after",
                        "--after",
                        encoded.name,
                        "--identities",
                        "examples/retained/identities.yaml",
                        "--catalog",
                        "examples/retained/catalog.yaml",
                        "--format",
                        "json",
                    ]
                )
            )
            validator.validate(result)
            require(not result["changes"], "BOM/CRLF changed permissions")
        finally:
            encoded.chmod(stat.S_IRUSR | stat.S_IWUSR)
        for document in [
            *root.glob("*.md"),
            *root.glob("docs/*.md"),
            *root.glob("examples/**/README.md"),
        ]:
            for target in re.findall(
                r"\[[^\]]+\]\(([^)]+)\)", document.read_text(encoding="utf-8")
            ):
                if "://" not in target and not target.startswith("#"):
                    require(
                        (document.parent / target.split("#")[0]).exists(),
                        f"broken package link in {document.name}: {target}",
                    )
        for name in (
            "LICENSE",
            "NOTICE",
            "THIRD_PARTY_LICENSES.txt",
            "docs/compatibility.md",
        ):
            require(
                (root / name).stat().st_size > 0, f"missing package document {name}"
            )
    result = dict(
        version=version,
        platform=system,
        architecture="amd64",
        result="passed",
        checksums=checksums,
        outputHashes=hashes,
        checks=[
            "archive checksums",
            "version",
            "command help",
            "four scenarios",
            "JSON Schema 1.0",
            "full and compact output",
            "golden reports",
            "exit codes 0/1/2/3/4",
            "CRD catalog",
            "query decisions",
            "Unicode and space paths",
            "BOM and CRLF",
            "read-only input",
            "documentation links",
            "license notices",
        ],
    )
    result_path.parent.mkdir(parents=True, exist_ok=True)
    result_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(f"{system}/amd64 package verified with {len(hashes)} output fingerprints")


def compare_results(directory):
    results = [
        json.loads(path.read_text(encoding="utf-8"))
        for path in sorted(directory.glob("package-*.json"))
    ]
    require(
        len(results) == 2
        and {result["platform"] for result in results} == {"windows", "linux"},
        "both native platform results are required",
    )
    left, right = results
    require(
        all(result["result"] == "passed" for result in results),
        "a package check failed",
    )
    require(
        left["version"] == right["version"] and left["checksums"] == right["checksums"],
        "platforms tested different packages",
    )
    require(
        left["outputHashes"] == right["outputHashes"], "native platform outputs differ"
    )
    print(
        f"Windows/Linux outputs match across {len(left['outputHashes'])} fingerprints for {left['version']}"
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dist", type=Path, default=Path("dist"))
    parser.add_argument("--version")
    parser.add_argument("--result", type=Path)
    parser.add_argument("--compare", type=Path)
    args = parser.parse_args()
    if args.compare:
        compare_results(args.compare)
    else:
        if not args.version or not args.result:
            parser.error("--version and --result are required")
        check_package(args.dist.resolve(), args.version, args.result.resolve())
