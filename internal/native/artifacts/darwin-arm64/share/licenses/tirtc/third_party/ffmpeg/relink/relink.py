#!/usr/bin/env python3
import argparse, hashlib, json, shutil, subprocess
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--recipe", required=True)
parser.add_argument("--ffmpeg-lib-dir", required=True)
parser.add_argument("--output-dir", required=True)
parser.add_argument("--toolchain-bin-dir")
args = parser.parse_args()
recipe_path = Path(args.recipe).resolve()
recipe = json.loads(recipe_path.read_text(encoding="utf-8"))
input_root = recipe_path.parent / "inputs"
ffmpeg_root = Path(args.ffmpeg_lib_dir).resolve()
output_root = Path(args.output_dir).resolve()
output_root.mkdir(parents=True, exist_ok=True)
toolchain_bin_root = Path(args.toolchain_bin_dir).resolve() if args.toolchain_bin_dir else None

def sha256_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()

for name in recipe["required_ffmpeg_archives"]:
    if not (ffmpeg_root / name).is_file():
        raise SystemExit(f"missing replacement FFmpeg archive: {ffmpeg_root / name}")
for item in recipe["inputs"]:
    bundled_input = input_root / item["path"]
    if not bundled_input.is_file() or sha256_file(bundled_input) != item["sha256"]:
        raise SystemExit(f"relink input integrity mismatch: {bundled_input}")

values = {
    "{INPUT_ROOT}": str(input_root),
    "{FFMPEG_LIB_DIR}": str(ffmpeg_root),
    "{OUTPUT_DIR}": str(output_root),
}
result = output_root / recipe["output_name"]
result.unlink(missing_ok=True)
for command in recipe["commands"]:
    recorded = command["tool"]
    tool = recorded if Path(recorded).is_file() else None
    if not tool and toolchain_bin_root:
        candidate = toolchain_bin_root / command["tool_basename"]
        tool = str(candidate) if candidate.is_file() else None
    if not tool and not Path(recorded).is_absolute():
        tool = shutil.which(command["tool_basename"])
    if not tool:
        raise SystemExit(
            f"missing recorded relink tool: {recorded}; "
            "provide --toolchain-bin-dir for a relocated matching toolchain"
        )
    arguments = []
    for raw in command["arguments"]:
        value = raw
        for marker, replacement in values.items():
            value = value.replace(marker, replacement)
        arguments.append(value)
    subprocess.run([tool, *arguments], check=True)

if not result.is_file():
    raise SystemExit(f"relink command did not produce: {result}")
print(result)
