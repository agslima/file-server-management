from __future__ import annotations

from pathlib import PurePosixPath
from typing import Iterable


# Keep these patterns in sync with .github/workflows/ci-pr-security-scan.yaml
FILTERS = {
    "backend": [
        "backend/composer.json",
        "backend/composer.lock",
    ],
    "frontend": [
        "frontend/package.json",
        "frontend/package-lock.json",
        # Add yarn/pnpm lockfiles here if you ever use them
    ],
    "file_engine": [
        "file-engine/go.mod",
        "file-engine/go.sum",
    ],
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

    return p.match(pattern)


def matches_any(path: str, patterns: Iterable[str]) -> bool:
    return any(matches_pattern(path, pattern) for pattern in patterns)


def compute_outputs(changed_files: list[str]) -> dict:
    outputs = {
        "backend": any(matches_any(f, FILTERS["backend"]) for f in changed_files),
        "frontend": any(matches_any(f, FILTERS["frontend"]) for f in changed_files),
        "file_engine": any(matches_any(f, FILTERS["file_engine"]) for f in changed_files),
        "docker": any(matches_any(f, FILTERS["docker"]) for f in changed_files),
        "docker_files": sorted([f for f in changed_files if matches_any(f, DOCKER_FILES_GLOBS)]),
    }
    return outputs


def run_case(name: str, changed: list[str], expected: dict) -> None:
    got = compute_outputs(changed)
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
            "backend composer lock change",
            ["backend/composer.lock"],
            dict(backend=True, frontend=False, file_engine=False, docker=False, docker_files=[]),
        ),
        (
            "frontend lock only change (critical regression test)",
            ["frontend/package-lock.json"],
            dict(backend=False, frontend=True, file_engine=False, docker=False, docker_files=[]),
        ),
        (
            "go mod/sum change",
            ["file-engine/go.mod", "file-engine/go.sum"],
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
            dict(backend=False, frontend=False, file_engine=False, docker=True, docker_files=["file-engine/api/Dockerfile"]),
        ),
        (
            "dockerfile wildcard (Dockerfile.gen) change",
            ["file-engine/Dockerfile.gen"],
            dict(backend=False, frontend=False, file_engine=False, docker=True, docker_files=["file-engine/Dockerfile.gen"]),
        ),
        (
            "multiple areas",
            ["backend/composer.lock", "frontend/package.json", "file-engine/worker/Dockerfile", "docker-compose.yml"],
            dict(
                backend=True,
                frontend=True,
                file_engine=False,  # no go files changed
                docker=True,
                docker_files=["docker-compose.yml", "file-engine/worker/Dockerfile"],
            ),
        ),
    ]

    for name, changed, expected in cases:
        run_case(name, changed, expected)

    print("OK: change detection + docker matrix inputs behave as expected.")


if __name__ == "__main__":
    main()
