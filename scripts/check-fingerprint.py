#!/usr/bin/env python3
"""Verify Go client TLS and HTTP/2 fingerprints against upstream fixtures."""

from __future__ import annotations

import argparse
import os
import pathlib
import queue
import re
import shutil
import socket
import subprocess
import sys
import threading
import time
from typing import Iterable

import yaml


ROOT = pathlib.Path(__file__).resolve().parents[1]
UPSTREAM_TESTS = ROOT / "third_party" / "curl-impersonate" / "tests"
SIGNATURES = UPSTREAM_TESTS / "signatures"
SSL = UPSTREAM_TESTS / "ssl"

sys.path.insert(0, str(UPSTREAM_TESTS))
from signature import HTTP2Signature, TLSClientHelloSignature  # noqa: E402


DEFAULT_EXPECTED_SIGNATURES = {
    "chrome": "chrome_116.0.5845.180_win10",
    "chrome116": "chrome_116.0.5845.180_win10",
    "firefox": "firefox_117.0.1_win10",
    "ff117": "firefox_117.0.1_win10",
}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", default=os.environ.get("GO_CURL_IMPERSONATE_TEST_PROFILE", "chrome"))
    parser.add_argument("--expected-signature")
    parser.add_argument("--skip-tls", action="store_true")
    parser.add_argument("--skip-http2", action="store_true")
    parser.add_argument("--curl-binary", help="diagnose a curl-impersonate wrapper instead of the Go client")
    parser.add_argument("--compare-curl-binary", help="compare the Go client against a curl-impersonate wrapper")
    parser.add_argument("--timeout", type=float, default=15.0)
    args = parser.parse_args()
    if args.curl_binary and args.compare_curl_binary:
        raise SystemExit("--curl-binary and --compare-curl-binary are mutually exclusive")

    expected_name = args.expected_signature or DEFAULT_EXPECTED_SIGNATURES.get(args.profile)
    if not expected_name:
        raise SystemExit(f"no default expected signature for profile {args.profile!r}; pass --expected-signature")

    docs = load_signatures(SIGNATURES)
    if expected_name not in docs:
        raise SystemExit(f"missing expected signature fixture: {expected_name}")
    expected = docs[expected_name]["signature"]

    if not args.skip_tls:
        verify_tls(
            args.profile,
            expected_name,
            expected,
            args.timeout,
            args.curl_binary,
            args.compare_curl_binary,
        )
    if not args.skip_http2:
        verify_http2(
            args.profile,
            expected_name,
            expected,
            args.timeout,
            args.curl_binary,
            args.compare_curl_binary,
        )
    return 0


def load_signatures(path: pathlib.Path) -> dict:
    docs = {}
    for file in path.glob("*.yaml"):
        with file.open() as stream:
            docs.update({doc["name"]: doc for doc in yaml.safe_load_all(stream) if doc})
    return docs


def verify_tls(
    profile: str,
    expected_name: str,
    expected: dict,
    timeout: float,
    curl_binary: str | None,
    compare_curl_binary: str | None,
) -> None:
    allow_permutation = expected.get("options", {}).get("tls_permute_extensions", False)
    if compare_curl_binary:
        go_sig = TLSClientHelloSignature.from_bytes(capture_tls_probe(profile, timeout, curl_binary=None))
        curl_sig = TLSClientHelloSignature.from_bytes(capture_tls_probe(profile, timeout, curl_binary=compare_curl_binary))
        equals, reason = go_sig.equals(curl_sig, allow_tls_permutation=allow_permutation, reason=True)
        if not equals:
            raise SystemExit(f"Go TLS fingerprint differs from {compare_curl_binary}: {reason}")
        print(f"TLS fingerprint matches curl wrapper {compare_curl_binary}")
        return

    record = capture_tls_probe(profile, timeout, curl_binary)
    actual_sig = TLSClientHelloSignature.from_bytes(record)
    expected_sig = TLSClientHelloSignature.from_dict(expected["tls_client_hello"])
    equals, reason = actual_sig.equals(
        expected_sig,
        allow_tls_permutation=allow_permutation,
        reason=True,
    )
    if not equals:
        raise SystemExit(f"TLS fingerprint mismatch for {expected_name}: {reason}")
    print(f"TLS fingerprint matches {expected_name}")


def capture_tls_probe(profile: str, timeout: float, curl_binary: str | None) -> bytes:
    captured = queue.Queue()
    ready = queue.Queue()
    thread = threading.Thread(target=capture_tls_record, args=(captured, ready), daemon=True)
    thread.start()
    port = ready.get(timeout=timeout)
    url = f"https://localhost:{port}/"
    run_probe(profile, url, timeout, allow_request_error=True, curl_binary=curl_binary, tls_verify=True)
    return captured.get(timeout=timeout)


def capture_tls_record(captured: queue.Queue, ready: queue.Queue) -> None:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        listener.bind(("127.0.0.1", 0))
        listener.listen(1)
        ready.put(listener.getsockname()[1])
        conn, _ = listener.accept()
        with conn:
            header = recv_exact(conn, 5)
            length = int.from_bytes(header[3:5], "big")
            captured.put(header + recv_exact(conn, length))


def recv_exact(conn: socket.socket, size: int) -> bytes:
    chunks = bytearray()
    while len(chunks) < size:
        chunk = conn.recv(size - len(chunks))
        if not chunk:
            raise RuntimeError(f"connection closed after {len(chunks)} of {size} bytes")
        chunks.extend(chunk)
    return bytes(chunks)


def verify_http2(
    profile: str,
    expected_name: str,
    expected: dict,
    timeout: float,
    curl_binary: str | None,
    compare_curl_binary: str | None,
) -> None:
    if compare_curl_binary:
        go_sig = capture_http2_signature(profile, timeout, curl_binary=None)
        curl_sig = capture_http2_signature(profile, timeout, curl_binary=compare_curl_binary)
        equals, reason = go_sig.equals(curl_sig, reason=True)
        if not equals:
            raise SystemExit(f"Go HTTP/2 fingerprint differs from {compare_curl_binary}: {reason}")
        print(f"HTTP/2 fingerprint matches curl wrapper {compare_curl_binary}")
        return

    actual_sig = capture_http2_signature(profile, timeout, curl_binary)
    expected_sig = HTTP2Signature.from_dict(expected["http2"])
    equals, reason = actual_sig.equals(expected_sig, reason=True)
    if not equals:
        raise SystemExit(f"HTTP/2 fingerprint mismatch for {expected_name}: {reason}")
    print(f"HTTP/2 fingerprint matches {expected_name}")


def capture_http2_signature(profile: str, timeout: float, curl_binary: str | None) -> HTTP2Signature:
    nghttpd = shutil.which("nghttpd")
    if nghttpd is None:
        raise SystemExit("missing nghttpd; install the nghttp2 server package or pass --skip-http2")
    port = free_port()
    proc = subprocess.Popen(
        [
            nghttpd,
            "-v",
            str(port),
            str(SSL / "server.key"),
            str(SSL / "server.crt"),
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )
    lines: queue.Queue[str] = queue.Queue()
    reader = threading.Thread(target=read_lines, args=(proc, lines), daemon=True)
    reader.start()
    try:
        wait_for_nghttpd(lines, port, timeout)
        run_probe(
            profile,
            f"https://localhost:{port}/",
            timeout,
            allow_request_error=False,
            curl_binary=curl_binary,
            tls_verify=False,
        )
        output = collect_lines(lines, timeout=2.0)
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=3)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=3)

    pseudo_headers, headers = parse_nghttpd_output(output)
    return HTTP2Signature(pseudo_headers, headers)


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def read_lines(proc: subprocess.Popen, lines: queue.Queue[str]) -> None:
    assert proc.stdout is not None
    for line in proc.stdout:
        lines.put(line.rstrip())


def wait_for_nghttpd(lines: queue.Queue[str], port: int, timeout: float) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            line = lines.get(timeout=0.1)
        except queue.Empty:
            continue
        if f"listen 0.0.0.0:{port}" in line or f"listen 127.0.0.1:{port}" in line or f"listen [::]:{port}" in line:
            return
    raise SystemExit("nghttpd did not start in time")


def collect_lines(lines: queue.Queue[str], timeout: float) -> list[str]:
    deadline = time.monotonic() + timeout
    output = []
    while time.monotonic() < deadline:
        try:
            output.append(lines.get(timeout=0.1))
        except queue.Empty:
            continue
    return output


def parse_nghttpd_output(lines: Iterable[str]) -> tuple[list[str], list[str]]:
    stream_id = None
    saved = list(lines)
    for line in saved:
        match = re.search(r"recv HEADERS frame.*stream_id=(\d+)", line)
        if match:
            stream_id = match.group(1)
            break
    if stream_id is None:
        raise SystemExit("failed to find HTTP/2 HEADERS frame in nghttpd output")

    pseudo_headers = []
    headers = []
    for line in saved:
        match = re.search(rf"recv \(stream_id={stream_id}\) (.*)", line)
        if not match:
            continue
        header = match.group(1)
        if header.startswith(":"):
            match = re.match(r"(:\w+):", header)
            if match:
                pseudo_headers.append(match.group(1))
        else:
            headers.append(header)
    return pseudo_headers, headers


def run_probe(
    profile: str,
    url: str,
    timeout: float,
    allow_request_error: bool,
    curl_binary: str | None,
    tls_verify: bool,
) -> None:
    if curl_binary:
        run_curl_probe(curl_binary, url, timeout, allow_request_error, tls_verify)
        return
    run_go_probe(profile, url, timeout, allow_request_error, tls_verify)


def run_curl_probe(curl_binary: str, url: str, timeout: float, allow_request_error: bool, tls_verify: bool) -> None:
    command = [
        curl_binary,
        "--max-time",
        str(timeout),
        "-o",
        os.devnull,
    ]
    if not tls_verify:
        command.append("-k")
    command.append(url)
    result = subprocess.run(
        command,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=timeout + 1,
        check=False,
    )
    if result.returncode != 0 and not allow_request_error:
        raise SystemExit(
            "curl fingerprint probe failed\n"
            f"stdout:\n{result.stdout}\n"
            f"stderr:\n{result.stderr}"
        )


def run_go_probe(profile: str, url: str, timeout: float, allow_request_error: bool, tls_verify: bool) -> None:
    env = go_probe_env(profile)
    command = [
        "go",
        "run",
        "-tags=integration native",
        "./cmd/go-curl-impersonate",
        "-profile",
        profile,
        f"-tls-verify={str(tls_verify).lower()}",
        "-url",
        url,
    ]
    if allow_request_error:
        command.append("-allow-request-error")
    result = subprocess.run(
        command,
        cwd=ROOT,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=timeout,
        check=False,
    )
    if result.returncode != 0:
        raise SystemExit(
            "go fingerprint probe failed\n"
            f"stdout:\n{result.stdout}\n"
            f"stderr:\n{result.stderr}"
        )


def go_probe_env(profile: str) -> dict[str, str]:
    env = os.environ.copy()
    if env.get("CGO_CFLAGS") and env.get("CGO_LDFLAGS"):
        return env
    package = pkg_config_package(profile)
    cflags = subprocess.check_output(["pkg-config", "--cflags", package], text=True).strip()
    libs = subprocess.check_output(["pkg-config", "--libs", package], text=True).strip()
    env.setdefault("CGO_CFLAGS", cflags)
    env.setdefault("CGO_LDFLAGS", libs)
    lib_dirs = [flag[2:] for flag in libs.split() if flag.startswith("-L")]
    if lib_dirs:
        existing = env.get("LD_LIBRARY_PATH")
        env["LD_LIBRARY_PATH"] = ":".join(lib_dirs + ([existing] if existing else []))
    return env


def pkg_config_package(profile: str) -> str:
    if profile.startswith("ff") or profile.startswith("firefox"):
        return "libcurl-impersonate-ff"
    return "libcurl-impersonate-chrome"


if __name__ == "__main__":
    raise SystemExit(main())
