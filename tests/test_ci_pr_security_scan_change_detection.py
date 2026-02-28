from __future__ import annotations

from pathlib import PurePosixPath
from typing import Iterable


# Keep these patterns in sync with .github/workflows/ci-pr-security-scan.yaml
FILTERS = {
    "backend": ["backend/**"],
    "frontend": ["frontend/**"],
    "file_engine": ["file-engine/**"],
    "docker": [
        "**/Dockerfile*",
        "docker-compose.yml",
    ],
}

# Used for docker_files_json collection (should match your second changed-files call)
DOCKER_FILES_GLOBS = [
    "**/Dockerfile*",
    "docker-compose.yml",
]


def matches_pattern(path: str, pattern: str) -> bool:
    p = PurePosixPath(path)

    # PurePosixPath.match("**/Dockerfile*") does not match a root-level
    # "Dockerfile*" path, so handle this workflow glob explicitly.
    if pattern == "**/Dockerfile*":
        return p.name.startswith("Dockerfile")

    # PurePosixPath.match("<scope>/**") does not include deeper descendants
    # consistently, so handle directory-scoped workflow globs explicitly.
    if pattern.endswith("/**"):
        prefix = pattern[: -len("/**")]
        return str(p) == prefix or str(p).startswith(f"{prefix}/")

    has_glob_meta = any(ch in pattern for ch in ("*", "?", "[", "]"))
    if "/" not in pattern and not has_glob_meta:
        return p.name == pattern and len(p.parts) == 1

    return p.match(pattern)


def matches_any(path: str, patterns: Iterable[str]) -> bool:
    return any(matches_pattern(path, pattern) for pattern in patterns)


def compute_outputs(
    changed_files: list[str],
    event_name: str = "pull_request",
    repo_files: list[str] | None = None,
) -> dict:
    if event_name == "workflow_dispatch":
        docker_scan_files = repo_files if repo_files is not None else changed_files
        return {
            "backend": True,
            "frontend": True,
            "file_engine": True,
            "docker": True,
            "docker_files": sorted([f for f in docker_scan_files if matches_any(f, DOCKER_FILES_GLOBS)]),
        }

    outputs = {
        "backend": any(matches_any(f, FILTERS["backend"]) for f in changed_files),
        "frontend": any(matches_any(f, FILTERS["frontend"]) for f in changed_files),
        "file_engine": any(matches_any(f, FILTERS["file_engine"]) for f in changed_files),
        "docker": any(matches_any(f, FILTERS["docker"]) for f in changed_files),
        "docker_files": sorted([f for f in changed_files if matches_any(f, DOCKER_FILES_GLOBS)]),
    }
    return outputs


def run_case(
    name: str,
    changed: list[str],
    expected: dict,
    event_name: str = "pull_request",
    repo_files: list[str] | None = None,
) -> None:
    got = compute_outputs(changed, event_name=event_name, repo_files=repo_files)
    # Compare booleans
    for key in ("backend", "frontend", "file_engine", "docker"):
        assert got[key] == expected[key], f"{name}: {key} expected {expected[key]} got {got[key]}"
    # Compare docker file list
    assert got["docker_files"] == expected["docker_files"], (
        f"{name}: docker_files expected {expected['docker_files']} got {got['docker_files']}"
    )


def main() -> None:
    cases = [
        (
            "no relevant changes",
            ["README.md", "docs/guide.md"],
            dict(backend=False, frontend=False, file_engine=False, docker=False, docker_files=[]),
        ),
        (
            "backend scoped change",
            ["backend/README.md"],
            dict(backend=True, frontend=False, file_engine=False, docker=False, docker_files=[]),
        ),
        (
            "frontend scoped change",
            ["frontend/README.md"],
            dict(backend=False, frontend=True, file_engine=False, docker=False, docker_files=[]),
        ),
        (
            "file-engine scoped change",
            ["file-engine/README.md"],
            dict(backend=False, frontend=False, file_engine=True, docker=False, docker_files=[]),
        ),
        (
            "root docker-compose change",
            ["docker-compose.yml"],
            dict(backend=False, frontend=False, file_engine=False, docker=True, docker_files=["docker-compose.yml"]),
        ),
        (
            "root Dockerfile change",
            ["Dockerfile"],
            dict(backend=False, frontend=False, file_engine=False, docker=True, docker_files=["Dockerfile"]),
        ),
        (
            "nested dockerfile change",
            ["file-engine/api/Dockerfile"],
            dict(backend=False, frontend=False, file_engine=True, docker=True, docker_files=["file-engine/api/Dockerfile"]),
        ),
        (
            "dockerfile wildcard (Dockerfile.gen) change",
            ["file-engine/Dockerfile.gen"],
            dict(backend=False, frontend=False, file_engine=True, docker=True, docker_files=["file-engine/Dockerfile.gen"]),
        ),
        (
            "multiple areas",
            ["backend/README.md", "frontend/README.md", "file-engine/worker/Dockerfile", "docker-compose.yml"],
            dict(
                backend=True,
                frontend=True,
                file_engine=True,
                docker=True,
                docker_files=["docker-compose.yml", "file-engine/worker/Dockerfile"],
            ),
        ),
    ]

    for name, changed, expected in cases:
        run_case(name, changed, expected)

    run_case(
        "workflow_dispatch forces all areas true",
        ["README.md"],
        dict(backend=True, frontend=True, file_engine=True, docker=True, docker_files=["Dockerfile", "docker-compose.yml", "frontend/Dockerfile"]),
        event_name="workflow_dispatch",
        repo_files=["README.md", "Dockerfile", "docker-compose.yml", "frontend/Dockerfile"],
    )

    run_case(
        "workflow_dispatch remains docker true even without docker files",
        ["README.md"],
        dict(backend=True, frontend=True, file_engine=True, docker=True, docker_files=[]),
        event_name="workflow_dispatch",
        repo_files=["README.md", "docs/guide.md"],
    )

    print("OK: change detection + docker matrix inputs behave as expected.")


if __name__ == "__main__":
    main()
