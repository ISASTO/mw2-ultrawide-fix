# MW2 Ultrawide Fix

A tiny Windows utility that fixes ultrawide aspect ratios in **Call of Duty: Modern Warfare 2 (2009)** after the 2026 64-bit PC update.

The old widescreen tools stopped working when `iw4sp.exe` changed to 64-bit. This utility patches the aspect-ratio float directly in the current single-player / Spec Ops executable.

## Use it

1. Download `MW2-Ultrawide-Fix.exe` from the latest GitHub Release.
2. Put it in your MW2 folder, next to `iw4sp.exe`.
   - Usually: `Steam\steamapps\common\Call of Duty Modern Warfare 2\`
3. Close MW2 if it is running.
4. Open `MW2-Ultrawide-Fix.exe`.
5. Confirm the detected resolution, or choose/type a different one.
6. Click **Apply Fix**.

The program automatically creates `iw4sp.exe.ultrawide-backup` before changing anything. Click **Restore Original** at any time to undo the patch.

## Tested

Confirmed working on the September 2026 64-bit MW2 update at **3440x1440** in Spec Ops.

Common ultrawide resolutions such as 2560x1080, 3440x1440, 3840x1600, 3840x1080, and 5120x1440 are included in the dropdown. You can also type another resolution manually.

## What it changes

MW2 stores the standard 16:9 aspect ratio as a 32-bit floating-point value in `iw4sp.exe`:

```text
39 8E E3 3F
```

That represents approximately `1.7777778` (16:9). The utility calculates `width / height` for the selected resolution and replaces every matching aspect-ratio value in the executable. The current 64-bit build contains two matches.

For 3440x1440, for example:

```text
39 8E E3 3F  ->  8E E3 18 40
1.7777778        2.3888889
```

If Steam updates the game, the executable may be replaced. Just run the utility again. If the new executable no longer contains the expected pattern, the program stops without modifying it.

## Safety / backups

- The original executable is backed up automatically.
- Existing patched installs can be repatched for a different resolution using the original backup.
- If Steam installs a different game build, the utility detects that the old backup no longer matches and takes a fresh backup before patching.
- The program refuses to patch non-ultrawide (16:9 or narrower) resolutions.

## Building

Requires Go 1.23 or newer.

```powershell
go test ./...
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -trimpath -ldflags="-H windowsgui -s -w" -o MW2-Ultrawide-Fix.exe .
```

The executable is unsigned, so Windows SmartScreen may show a warning on first launch. The full source is here so the build can be inspected or reproduced.

## License

MIT
