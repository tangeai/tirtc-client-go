#!/usr/bin/env python3
import argparse, json, os, shlex, shutil, subprocess
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--source-dir", required=True)
parser.add_argument("--output-dir", required=True)
parser.add_argument("--build-manifest")
parser.add_argument("--toolchain-bin-dir")
parser.add_argument("--sysroot")
parser.add_argument("--jobs", type=int, default=max(1, os.cpu_count() or 1))
args = parser.parse_args()

script_path = Path(__file__).resolve()
manifest_path = (
    Path(args.build_manifest).resolve()
    if args.build_manifest
    else script_path.parents[1] / "build-manifest.json"
)
source_root = Path(args.source_dir).resolve()
output_root = Path(args.output_dir).resolve()
build_root = output_root / "build"
stage_root = output_root / "stage"
toolchain_bin_root = Path(args.toolchain_bin_dir).resolve() if args.toolchain_bin_dir else None

if args.jobs < 1:
    raise SystemExit("--jobs must be positive")
if not (source_root / "configure").is_file():
    raise SystemExit(f"FFmpeg configure script is missing: {source_root / 'configure'}")
if output_root.exists() and any(output_root.iterdir()):
    raise SystemExit(f"rebuild output directory must be empty: {output_root}")

manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
toolchain = manifest.get("toolchain", {})
configuration = manifest.get("configuration")
if not isinstance(configuration, str) or not configuration:
    raise SystemExit("build manifest has no FFmpeg configuration")

replacements = {}
if toolchain_bin_root:
    for key in ("cc", "cxx", "ar", "ranlib", "nm", "strip"):
        recorded = toolchain.get(key)
        if not recorded:
            continue
        recorded_parts = shlex.split(recorded)
        if not recorded_parts:
            continue
        candidate = toolchain_bin_root / Path(recorded_parts[-1]).name
        if not candidate.is_file():
            raise SystemExit(f"matching tool is missing: {candidate}")
        replacements[recorded_parts[-1]] = str(candidate)
if args.sysroot:
    recorded_sysroot = toolchain.get("sdk_sysroot")
    if recorded_sysroot:
        replacements[recorded_sysroot] = str(Path(args.sysroot).resolve())

configure_arguments = []
for raw in shlex.split(configuration, posix=True):
    value = raw
    for recorded, replacement in replacements.items():
        value = value.replace(recorded, replacement)
    configure_arguments.append(value)

recorded_make = str(toolchain.get("make", "make"))
make = recorded_make if Path(recorded_make).is_file() else shutil.which(Path(recorded_make).name)
if not make:
    raise SystemExit(f"matching make tool is missing: {recorded_make}")

build_root.mkdir(parents=True)
stage_root.mkdir(parents=True)
environment = os.environ.copy()
environment["PKG_CONFIG_PATH"] = ""
environment["PKG_CONFIG_LIBDIR"] = ""
subprocess.run(
    [str(source_root / "configure"), *configure_arguments],
    cwd=build_root,
    env=environment,
    check=True,
)
subprocess.run([make, f"-j{args.jobs}"], cwd=build_root, env=environment, check=True)
subprocess.run(
    [make, f"DESTDIR={stage_root}", "install-libs", "install-headers"],
    cwd=build_root,
    env=environment,
    check=True,
)

required = {
    "libavformat.a",
    "libavcodec.a",
    "libavutil.a",
    "libswresample.a",
    "libswscale.a",
}
actual = {path.name for path in (stage_root / "lib").glob("*.a")}
if actual != required:
    raise SystemExit(f"rebuilt FFmpeg archive closure mismatch: {sorted(actual)}")
print(stage_root / "lib")
