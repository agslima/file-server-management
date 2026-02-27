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
    """
    Determine whether a POSIX-style file path matches a workflow glob pattern, with special handling for Dockerfile and directory-scoped globs.
    
    Parameters:
        path (str): The file path to test (interpreted as a POSIX path).
        pattern (str): A workflow-style glob pattern. Supports:
            - "**/Dockerfile*": matches any filename that starts with "Dockerfile".
            - "<dir>/**": matches the directory itself or any descendant under that directory.
            - other glob patterns: matched using PurePosixPath.match.
    
    Returns:
        bool: `True` if the path matches the pattern according to the described rules, `False` otherwise.
    """
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

    return p.match(pattern)


def matches_any(path: str, patterns: Iterable[str]) -> bool:
    """
    Determines whether a path matches any of the provided glob patterns.
    
    Parameters:
        path (str): The file path to test.
        patterns (Iterable[str]): Iterable of glob patterns to test against (e.g. "**/Dockerfile*", "docker-compose.yml").
    
    Returns:
        true if any pattern matches the path, false otherwise.
    """
    return any(matches_pattern(path, pattern) for pattern in patterns)


def compute_outputs(
    changed_files: list[str],
    event_name: str = "pull_request",
    repo_files: list[str] | None = None,
) -> dict:
    """
    Determine which project areas are affected by a set of changed files and compute the list of Docker-related files for CI workflows.
    
    Parameters:
        changed_files (list[str]): Paths of files that changed in the current run.
        event_name (str): The triggering event name; when "workflow_dispatch", all areas are considered affected and docker files are collected from `repo_files` if provided. Defaults to "pull_request".
        repo_files (list[str] | None): Optional list of all repository file paths; used as the source of docker file discovery when `event_name` is "workflow_dispatch". If omitted, `changed_files` are used.
    
    Returns:
        dict: A mapping with keys:
            - "backend" (bool): True if backend-related files are affected (or always True for "workflow_dispatch").
            - "frontend" (bool): True if frontend-related files are affected (or always True for "workflow_dispatch").
            - "file_engine" (bool): True if file-engine-related files are affected (or always True for "workflow_dispatch").
            - "docker" (bool): True if docker-related files are affected (or always True for "workflow_dispatch").
            - "docker_files" (list[str]): Sorted list of paths matching Docker-related globs; when `event_name` is "workflow_dispatch" this list is derived from `repo_files` if provided, otherwise from `changed_files`.
    """
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
    """
    Run a single test case by calling compute_outputs and asserting its outputs match the expected results.
    
    Parameters:
    	name (str): Descriptive name for the test case used in assertion messages.
    	changed (list[str]): List of changed file paths to pass to compute_outputs.
    	expected (dict): Expected outputs with keys 'backend', 'frontend', 'file_engine', 'docker' (booleans) and 'docker_files' (list of paths).
    	event_name (str): Event name to pass to compute_outputs (defaults to "pull_request"); when "workflow_dispatch" behavior differs as documented by compute_outputs.
    	repo_files (list[str] | None): Optional repository file list used by compute_outputs when event_name is "workflow_dispatch".
    
    Raises:
    	AssertionError: If any of the area booleans or the docker_files list does not equal the expected value; the assertion message includes the test name, the key, and the expected vs. actual values.
    """
    got = compute_outputs(changed, event_name=event_name, repo_files=repo_files)
    # Compare booleans
    for key in ("backend", "frontend", "file_engine", "docker"):
        assert got[key] == expected[key], f"{name}: {key} expected {expected[key]} got {got[key]}"
    # Compare docker file list
    assert got["docker_files"] == expected["docker_files"], (
        f"{name}: docker_files expected {expected['docker_files']} got {got['docker_files']}"
    )


def main() -> None:
    """
    Execute a suite of self-tests that validate the change-detection and Docker-file aggregation logic.
    
    Runs predefined test cases (including workflow_dispatch scenarios) via run_case and prints a success message when all assertions pass.
    """
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
