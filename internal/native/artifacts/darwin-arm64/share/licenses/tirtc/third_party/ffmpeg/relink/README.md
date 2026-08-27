# FFmpeg static relink kit

Each recipe rebuilds one Runtime binary that statically absorbs FFmpeg. Extract `source/ffmpeg-source.tar.gz`, modify FFmpeg if desired, and rebuild with the toolchain recorded by the build manifest:

```sh
python3 source/rebuild_ffmpeg.py --source-dir <extracted>/ffmpeg/src --output-dir <ffmpeg-build> [--toolchain-bin-dir <matching-toolchain-bin>] [--sysroot <matching-sdk-sysroot>]
```

Then relink each Runtime output:

```sh
python3 relink.py --recipe <platform>/<output>/recipe.json --ffmpeg-lib-dir <modified-ffmpeg-lib-dir> --output-dir <output-dir> [--toolchain-bin-dir <matching-toolchain-bin>]
```

The replacement directory must contain every archive listed by the recipe. Use the toolchain identity recorded in recipe.json; if that toolchain moved, pass its bin directory explicitly. Run the corresponding Runtime consumer smoke before distributing the result.
